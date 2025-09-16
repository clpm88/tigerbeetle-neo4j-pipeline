# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

This is a real-time financial graph analysis pipeline built in Go that demonstrates Change Data Capture (CDC) patterns. The system processes financial transactions from TigerBeetle database and loads them into Neo4j for graph analysis, showcasing a modern approach to separating transactional and analytical workloads.

## Architecture

The system consists of three independent Go microservices connected via streaming:

1. **Transaction Generator** (`cmd/generator`) - Creates sample accounts and generates random transfers in TigerBeetle
2. **CDC Connector** (`cmd/cdc-connector`) - Reads transfers from TigerBeetle and publishes them to Redpanda (currently using polling, not true CDC)  
3. **Neo4j Sink** (`cmd/neo4j-sink`) - Consumes messages from Redpanda and creates account/transaction graphs in Neo4j

**Data Flow**: `TigerBeetle → CDC Connector → Redpanda → Neo4j Sink → Neo4j`

## Key Technologies

- **Go 1.23+** with modules
- **TigerBeetle** (high-performance accounting database)
- **Redpanda** (Kafka-compatible streaming platform)
- **Neo4j** (graph database)
- **Docker Compose** for orchestration

**Core Dependencies**:
- `github.com/tigerbeetle/tigerbeetle-go` - TigerBeetle client
- `github.com/twmb/franz-go` - Kafka/Redpanda client
- `github.com/neo4j/neo4j-go-driver/v5` - Neo4j driver

## Essential Commands

### Setup and Infrastructure
```bash
# Initial setup
go mod tidy

# OPTION 1: Using Neo4j AuraDB (recommended)
# 1. Update .env file with your AuraDB credentials:
#    NEO4J_URI=neo4j+s://YOUR_INSTANCE.databases.neo4j.io
#    NEO4J_USERNAME=neo4j
#    NEO4J_PASSWORD=your_password
# 2. Test AuraDB connection and setup:
./setup-auradb.sh

# OPTION 2: Using local Neo4j (uncomment neo4j service in docker-compose.yml first)
# Format TigerBeetle database (required first time)
docker run --rm -v tigerbeetle-neo4j_tigerbeetle_data:/data ghcr.io/tigerbeetle/tigerbeetle:latest format --cluster=0 --replica=0 /data/tigerbeetle.tigerbeetle

# Start infrastructure services (TigerBeetle + Redpanda + Neo4j if using local)
docker-compose up -d

# Create Redpanda topic
docker exec redpanda rpk topic create transactions --partitions 3 --replicas 1
```

### Building and Running
```bash
# Build all services
make build
# OR manually:
go build -o bin/generator ./cmd/generator
go build -o bin/cdc-connector ./cmd/cdc-connector  
go build -o bin/neo4j-sink ./cmd/neo4j-sink

# Run services (in separate terminals)
make run-generator    # or ./bin/generator
make run-cdc         # or ./bin/cdc-connector
make run-sink        # or ./bin/neo4j-sink

# Complete setup workflow
make setup-infra && make create-topic && make build
```

### Testing and Development
```bash
# Check service health
docker-compose ps

# View Redpanda messages
# Web UI: http://localhost:8080
docker exec redpanda rpk topic consume transactions --print-keys

# Check Neo4j data
# Web UI: http://localhost:7474 (neo4j/password)
docker exec neo4j cypher-shell -u neo4j -p password "MATCH (a:Account)-[r:SENT_TO]->(b:Account) RETURN count(*)"

# Clean up
make clean  # Stops services and removes data volumes
```

### Running Single Tests
```bash
# Test specific package
go test ./pkg/config -v
go test ./pkg/models -v

# Test with coverage
go test -cover ./...
```

## Configuration System

The application uses a centralized configuration system in `pkg/config/config.go` that loads from environment variables with sensible defaults:

**TigerBeetle**: `TIGERBEETLE_ADDRESS` (default: "3000")
**Redpanda**: `REDPANDA_BROKERS` (default: "localhost:19092"), `REDPANDA_TOPIC` (default: "transactions"), `REDPANDA_CONSUMER_GROUP` (default: "neo4j-sink-group")
**Neo4j**: `NEO4J_URI` (default: "bolt://localhost:7687"), `NEO4J_USERNAME` (default: "neo4j"), `NEO4J_PASSWORD` (default: "password")

All services use `config.Load()` to initialize their settings consistently.

**AuraDB Setup**: Create a `.env` file in the project root with your AuraDB credentials:
```bash
NEO4J_URI=neo4j+s://YOUR_INSTANCE.databases.neo4j.io
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your_auradb_password
```

## Code Organization

```
cmd/                     # Main applications (one per microservice)
├── generator/          # Transaction generator service
├── cdc-connector/      # CDC connector service  
└── neo4j-sink/         # Neo4j sink service
pkg/                     # Shared packages
├── config/             # Configuration management
└── models/             # Data models (Transfer struct)
docker/                 # Docker configuration files
```

## Important Implementation Notes

### Data Model
The `pkg/models/Transfer` struct is the canonical JSON representation used in Redpanda messages. It mirrors TigerBeetle's Transfer but uses JSON-friendly types (string for Uint128 IDs).

### CDC Limitation
The current CDC connector uses polling (`client.QueryTransfers`) rather than true CDC due to TigerBeetle's CDC setup requirements. In production, this should use `GetChangeEvents` API with proper change stream filtering.

### Idempotent Neo4j Operations
The Neo4j sink uses `MERGE` for accounts to prevent duplicates but `CREATE` for relationships, allowing multiple transfers between the same accounts.

### Error Handling Pattern
All services follow consistent patterns:
- Fatal errors for connection failures
- Logged errors for operational issues
- Graceful degradation where possible

## Neo4j Queries

**View transaction graph**:
```cypher
MATCH (a:Account)-[r:SENT_TO]->(b:Account) 
RETURN a, r, b 
LIMIT 25
```

**Account balances (derived)**:
```cypher
MATCH (a:Account)
OPTIONAL MATCH (a)-[out:SENT_TO]->()
OPTIONAL MATCH ()-[in:SENT_TO]->(a)
RETURN a.id, 
       COALESCE(sum(in.amount), 0) AS credits,
       COALESCE(sum(out.amount), 0) AS debits
```

## Development Workflow

1. **AuraDB Setup**: Use the `.env` file and `setup-auradb.sh` script to configure and test your Neo4j AuraDB connection
2. **Local Development**: Use `docker-compose up -d` for infrastructure, run Go services locally for faster iteration
3. **Service Independence**: Each service can be restarted independently without affecting others (thanks to Redpanda buffering)
4. **Testing**: Services log success/error messages extensively for debugging

## Troubleshooting

- **TigerBeetle connection**: Ensure database is formatted and container is healthy
- **Redpanda issues**: Check `docker exec redpanda rpk cluster health`
- **Neo4j connectivity**: Verify with `docker exec neo4j cypher-shell -u neo4j -p password "RETURN 1"`
- **Message flow**: Monitor Redpanda Console at http://localhost:8080 for message throughput