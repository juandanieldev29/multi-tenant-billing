-- Must run on the coordinator node before adding workers.
CREATE EXTENSION IF NOT EXISTS citus;

-- Reference table: replicated to all Citus workers for zero-network joins
CREATE TABLE tenants (
    id         TEXT        PRIMARY KEY,
    name       TEXT        NOT NULL,
    seats      INTEGER     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reference table: pricing config stored as JSONB to support arbitrary models
-- without schema migrations. Supported types: per_request, tiered, seat_based.
CREATE TABLE pricing_plans (
    tenant_id    TEXT        NOT NULL REFERENCES tenants(id),
    event_type   TEXT        NOT NULL,
    config       JSONB       NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, event_type, effective_at)
);

-- Distributed table: raw ingested events (append-only audit log)
CREATE TABLE usage_events (
    id          BIGSERIAL,
    tenant_id   TEXT             NOT NULL,
    event_id    TEXT             NOT NULL,
    event_type  TEXT             NOT NULL,
    quantity    DOUBLE PRECISION NOT NULL,
    resource_id TEXT,
    occurred_at TIMESTAMPTZ      NOT NULL,
    ingested_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE      (tenant_id, event_id)   -- dedup at storage layer
);

-- Distributed table: Flink aggregation output
-- ON CONFLICT upsert means late-arriving events safely update existing windows.
CREATE TABLE usage_aggregates (
    id              BIGSERIAL,
    tenant_id       TEXT             NOT NULL,
    event_type      TEXT             NOT NULL,
    window_start    TIMESTAMPTZ      NOT NULL,
    window_end      TIMESTAMPTZ      NOT NULL,
    window_duration TEXT             NOT NULL,  -- '1 minute', '1 hour', '1 day'
    total_quantity  DOUBLE PRECISION NOT NULL,
    event_count     BIGINT           NOT NULL,
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, event_type, window_start, window_duration)
);

-- Distributed table: finalized invoices
CREATE TABLE invoices (
    id           BIGSERIAL,
    tenant_id    TEXT        NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    total_cents  BIGINT      NOT NULL,
    line_items   JSONB       NOT NULL DEFAULT '[]',
    status       TEXT        NOT NULL DEFAULT 'draft',  -- draft | finalized | void
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

-- Supporting indexes (created on all shards automatically by Citus)
CREATE INDEX ON usage_aggregates (tenant_id, window_start, window_duration);
CREATE INDEX ON usage_events     (tenant_id, occurred_at);
CREATE INDEX ON invoices         (tenant_id, period_start, status);
