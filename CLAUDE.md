# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

### Go services (`services/ingestion`, `services/billing-api`)

```bash
# Build all services + Flink fat JAR
make build

# Run all tests with race detector
make test

# Run a single Go test
cd services/ingestion  && go test ./internal/handler/... -run TestIngestEvent -v
cd services/billing-api && go test ./internal/pricing/...  -run TestTieredPricing -v

# Lint (requires golangci-lint)
make lint

# Run locally (env vars required — see below)
cd services/ingestion  && go run ./cmd/server
cd services/billing-api && go run ./cmd/server
```

Required env vars:
- `services/ingestion`: `KAFKA_BROKERS` (comma-separated), `REDIS_URL` (host:port)
- `services/billing-api`: `DATABASE_URL` (pgx DSN)

### Flink aggregator (`services/flink-aggregator`, Java 11 / Maven)

```bash
cd services/flink-aggregator
mvn clean package -DskipTests     # produces target/flink-aggregator-1.0.0.jar
mvn test
mvn test -Dtest=WindowAggregatorTest
```

### Full local stack

```bash
make docker-build                  # build all five images
docker-compose up -d               # Kafka (3-node KRaft), Citus (coord + 2 workers), Redis, services, Flink
docker-compose logs -f ingestion

make chaos-test                    # kill kafka-2 mid-stream; verify zero data loss
make bench                         # throughput benchmark — reports events/sec and P99 latency
```

After the Citus workers start, register them and distribute the schema:
```bash
docker-compose exec citus-coordinator psql -U billing -c \
  "SELECT citus_add_node('citus-worker-1', 5432); SELECT citus_add_node('citus-worker-2', 5432);"
docker-compose exec citus-coordinator psql -U billing -f /docker-entrypoint-initdb.d/02_citus_distribution.sql
```

## Architecture

### Data flow

```
Client
  │  POST /v1/events
  ▼
Ingestion Service  ──SETNX idempotency:{tenantId}:{eventId}──►  Redis
  │  Kafka transactional producer
  ▼
Kafka  (topic: billing.events, 64 partitions)
  ▼
Flink Aggregator  ──2-phase commit JDBC sink──►  Citus coordinator
                                                        │
                                               (sharded on tenant_id)
                                                        ▲
                                              Billing API  ◄──  Client
                                              GET /v1/tenants/{id}/invoice
```

### Service map

| Service | Directory | Port | Language |
|---|---|---|---|
| Ingestion | `services/ingestion` | 8080 | Go 1.22 |
| Billing API | `services/billing-api` | 8081 | Go 1.22 |
| Flink Aggregator | `services/flink-aggregator` | — (Flink UI: 8082) | Java 11 / Flink 1.18 |
| PostgreSQL + Citus | `infra/postgres-citus` | 5432 | Citus 12.1 |
| Kafka | `infra/kafka` | 9092 | Kafka 3.7 KRaft |

### Exactly-once semantics

**Ingestion → Kafka**: `internal/redis/idempotency.go` calls `SETNX idempotency:{tenantID}:{eventID}` (24 h TTL) before publishing. Only if that lock is acquired does `internal/kafka/producer.go` publish inside a Kafka transaction (`transactional.id = ingestion-{podName}`). Retried HTTP requests return 200 immediately after the Redis check without re-publishing.

**Kafka → Citus**: Flink checkpoints every 30 s (`EXACTLY_ONCE` mode). The Kafka source sets `isolation.level=read_committed` to skip aborted transactions. The JDBC sink uses an `ON CONFLICT … DO UPDATE` upsert so checkpoint replays are idempotent. On TaskManager restart, Flink rewinds to the last successful checkpoint — no double-counting.

### Tenant isolation in Citus

`usage_events`, `usage_aggregates`, and `invoices` are distributed on `tenant_id`. `tenants` and `pricing_plans` are reference tables (replicated to every worker). All three distributed tables are co-located on the same shards (`colocate_with => 'usage_events'`), enabling `JOIN`s without network hops between workers.

**Every Citus query must have `WHERE tenant_id = $1` as the first predicate.** Omitting it causes a scatter query across all shards.

### Flink windowing (`services/flink-aggregator/src/main/java/com/billing/AggregationJob.java`)

`buildPipeline()` is called three times to produce 1-minute, 1-hour, and 1-day tumbling windows in a single job. All three share the same `DataStream<UsageEvent>` source. Key points:

- Watermark strategy: 30 s bounded-out-of-orderness + 1 min idleness timeout.
- `allowedLateness(Time.minutes(5))`: events arriving ≤5 min past the watermark update the existing aggregate via the upsert; events later than that are routed to `LateEventTag.TAG` (dead-letter side output).
- Window results are never retroactively deleted — the dead-letter topic is the audit trail for dropped late events.

### Pricing model (`services/billing-api/internal/pricing/engine.go`)

`pricing_plans.config` is a JSONB column. Three types are supported without schema changes:

```jsonc
// per_request
{"type": "per_request", "unit_price_cents": 1}

// tiered  (Stripe-style: charged per tier in order, nil up_to = unlimited)
{"type": "tiered", "tiers": [{"up_to": 10000, "unit_price_cents": 2}, {"up_to": null, "unit_price_cents": 1}]}

// seat_based
{"type": "seat_based", "seat_price_cents": 5000}
```

`engine.Calculate(configJSON, quantity, seats)` resolves the config at invoice time — no code change needed for new pricing models as long as `engine.go` is extended.

### Observability

OpenTelemetry is initialized in `internal/telemetry/otel.go` in both Go services. Trace context (`traceparent`) is propagated through Kafka record headers in `producer.go` and extracted by Flink. Key Prometheus metrics:

- `billing_events_ingested_total{tenant_id, status}` — counter
- `billing_aggregation_lag_seconds` — histogram (Flink window close → Citus write)
- `billing_late_events_total{window_duration}` — counter

### Kubernetes scaling

`billing.events` has 64 partitions. The ingestion `HorizontalPodAutoscaler` scales between 4 and 64 replicas. Each new pod claims additional partitions — throughput scales linearly. Flink TaskManagers are scaled separately; each slot handles one key-group of the keyed stream.

## Key invariants

- Never publish to Kafka without first acquiring the Redis idempotency lock — the lock is the only dedup gate between the HTTP layer and Kafka.
- Never disable Flink checkpointing. It is the recovery mechanism; disabling it means a TaskManager crash can cause double-counting.
- All Citus queries targeting distributed tables must start with `tenant_id = $1`. Anything else scatter-gathers across all shards.
- `transactional.id` for each ingestion pod must be globally unique — the pod name (injected via `POD_NAME` env var) ensures this across rolling deployments.
- Late events (>5 min past watermark) are dropped from aggregates and go to the dead-letter Kafka topic `billing.late-events`. Aggregate rows are never updated after the lateness window closes.
