.PHONY: build test lint docker-build docker-push chaos-test bench clean

REGISTRY ?= ghcr.io/juandanieldev29
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO_SERVICES = ingestion billing-api

build:
	@for svc in $(GO_SERVICES); do \
		echo "==> building $$svc"; \
		cd services/$$svc && go build ./... && cd ../..; \
	done
	@echo "==> building flink-aggregator"
	cd services/flink-aggregator && mvn clean package -DskipTests -q

test:
	@for svc in $(GO_SERVICES); do \
		echo "==> testing $$svc"; \
		cd services/$$svc && go test ./... -race -timeout 60s && cd ../..; \
	done
	@echo "==> testing flink-aggregator"
	cd services/flink-aggregator && mvn test -q

lint:
	@for svc in $(GO_SERVICES); do \
		echo "==> linting $$svc"; \
		cd services/$$svc && golangci-lint run ./... && cd ../..; \
	done

docker-build:
	docker build -t $(REGISTRY)/ingestion:$(VERSION)        services/ingestion
	docker build -t $(REGISTRY)/billing-api:$(VERSION)      services/billing-api
	docker build -t $(REGISTRY)/flink-aggregator:$(VERSION) services/flink-aggregator
	docker build -t $(REGISTRY)/postgres-citus:$(VERSION)   infra/postgres-citus
	docker build -t $(REGISTRY)/kafka:$(VERSION)            infra/kafka

docker-push:
	@for img in ingestion billing-api flink-aggregator postgres-citus kafka; do \
		docker push $(REGISTRY)/$$img:$(VERSION); \
	done

chaos-test:
	@echo "Starting chaos: killing kafka-2 mid-stream and verifying zero data loss"
	./scripts/chaos-kafka.sh

bench:
	cd services/ingestion && go test ./benchmark/... -bench=. -benchtime=30s -benchmem

clean:
	@for svc in $(GO_SERVICES); do \
		cd services/$$svc && go clean ./... && cd ../..; \
	done
	cd services/flink-aggregator && mvn clean -q
