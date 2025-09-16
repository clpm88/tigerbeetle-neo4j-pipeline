.PHONY: help build run-generator run-cdc run-sink setup-infra create-topic clean monitoring monitoring-up monitoring-down

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build all services"
	@echo "  run-generator  - Run the transaction generator"
	@echo "  run-cdc        - Run the CDC connector"
	@echo "  run-sink       - Run the Neo4j sink"
	@echo "  setup-infra    - Format TigerBeetle and start infrastructure"
	@echo "  create-topic   - Create the Redpanda transactions topic"
	@echo "  monitoring     - Start monitoring stack (Prometheus + Grafana)"
	@echo "  monitoring-up  - Start monitoring stack"
	@echo "  monitoring-down- Stop monitoring stack"
	@echo "  clean          - Stop infrastructure and remove volumes"

# Create bin directory
bin:
	mkdir -p bin

# Build all services
build: bin
	go build -o bin/generator ./cmd/generator
	go build -o bin/cdc-connector ./cmd/cdc-connector
	go build -o bin/neo4j-sink ./cmd/neo4j-sink

# Run individual services
run-generator: bin/generator
	./bin/generator

run-cdc: bin/cdc-connector
	./bin/cdc-connector

run-sink: bin/neo4j-sink
	./bin/neo4j-sink

# Infrastructure setup
setup-infra:
	@echo "Formatting TigerBeetle database..."
	docker run --privileged --rm -v tigerbeetle-neo4j_tigerbeetle_data:/data ghcr.io/tigerbeetle/tigerbeetle:latest format --development --cluster=0 --replica=0 --replica-count=1 /data/tigerbeetle.tigerbeetle
	@echo "Starting infrastructure..."
	docker compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Infrastructure is ready!"

# Create Redpanda topic
create-topic:
	docker exec redpanda rpk topic create transactions --partitions 3 --replicas 1

# Monitoring
monitoring: monitoring-up

monitoring-up:
	@echo "Starting monitoring stack..."
	docker compose up -d prometheus grafana
	@echo "Monitoring available at:"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana:    http://localhost:3001 (admin/admin)"

monitoring-down:
	@echo "Stopping monitoring stack..."
	docker compose stop prometheus grafana

# Clean up
clean:
	docker compose down -v
	rm -rf bin/

# Ensure individual binaries exist
bin/generator: 
	go build -o bin/generator ./cmd/generator

bin/cdc-connector:
	go build -o bin/cdc-connector ./cmd/cdc-connector

bin/neo4j-sink:
	go build -o bin/neo4j-sink ./cmd/neo4j-sink