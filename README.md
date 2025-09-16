# TigerBeetle Neo4j Real-Time Financial Graph Analysis Pipeline

A real-time data pipeline that processes financial transactions from TigerBeetle database and loads them into a Neo4j graph database for analysis. This project demonstrates a modern approach to separating transactional workloads from analytical workloads using Change Data Capture (CDC).

## Architecture

The system consists of three independent Go microservices:

1. **Transaction Generator** (`cmd/generator`) - Creates sample accounts and generates random financial transfers in TigerBeetle
2. **CDC Connector** (`cmd/cdc-connector`) - Reads committed transfers from TigerBeetle's CDC log and publishes them to Redpanda
3. **Neo4j Sink** (`cmd/neo4j-sink`) - Consumes messages from Redpanda and creates a graph of accounts and transactions in Neo4j

## Technologies Used

- **Language:** Go (Golang)
- **Transactional Database:** TigerBeetle
- **Streaming Platform:** Redpanda (Kafka-compatible)
- **Graph Database:** Neo4j AuraDB or local Neo4j
- **Monitoring:** Prometheus & Grafana
- **Analytics Planning:** PostHog integration
- **Orchestration:** Docker and Docker Compose

## Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- Neo4j AuraDB account (optional - can use local Neo4j)

## Quick Start

### 1. Clone and Setup

```bash
git clone <your-repo>
cd tigerbeetle-neo4j
go mod tidy
```

### 2. Start Infrastructure

Start the local infrastructure (TigerBeetle, Redpanda, Neo4j):

```bash
# First, format TigerBeetle database
docker run --rm -v tigerbeetle-neo4j_tigerbeetle_data:/data ghcr.io/tigerbeetle/tigerbeetle:latest format --cluster=0 --replica=0 /data/tigerbeetle.tigerbeetle

# Start all services
docker-compose up -d
```

Wait for all services to be healthy:

```bash
docker-compose ps
```

### 3. Create Redpanda Topic

```bash
# Create the transactions topic
docker exec redpanda rpk topic create transactions --partitions 3 --replicas 1
```

### 4. Build and Run Services

Build all services:

```bash
go build -o bin/generator ./cmd/generator
go build -o bin/cdc-connector ./cmd/cdc-connector  
go build -o bin/neo4j-sink ./cmd/neo4j-sink
```

Run the services (in separate terminals):

```bash
# Terminal 1 - Start transaction generator
./bin/generator

# Terminal 2 - Start CDC connector  
./bin/cdc-connector

# Terminal 3 - Start Neo4j sink
./bin/neo4j-sink
```

### 5. Verify the Pipeline

#### Check TigerBeetle
The generator should log successful transfers:
```
SUCCESS: Transfer 1 - $543 from account 3 to account 7
```

#### Check Redpanda
The CDC connector should log forwarded messages:
```
SUCCESS: Forwarded transfer 1 to Redpanda topic 'transactions'
```

You can also view messages in Redpanda Console at http://localhost:8080

#### Check Neo4j
The Neo4j sink should log successful writes:
```
SUCCESS: Written transfer 1 to Neo4j graph (from:3 to:7 amount:543)
```

Connect to Neo4j Browser at http://localhost:7474 (user: neo4j, password: password) and run:

```cypher
MATCH (a:Account)-[r:SENT_TO]->(b:Account)
RETURN a, r, b
LIMIT 25
```

## Configuration

All services can be configured using environment variables:

### TigerBeetle
- `TIGERBEETLE_ADDRESS`: TigerBeetle server address (default: "3000")

### Redpanda
- `REDPANDA_BROKERS`: Comma-separated broker addresses (default: "localhost:19092")
- `REDPANDA_TOPIC`: Topic name for transactions (default: "transactions")
- `REDPANDA_CONSUMER_GROUP`: Consumer group for Neo4j sink (default: "neo4j-sink-group")

### Neo4j
- `NEO4J_URI`: Neo4j connection URI (default: "bolt://localhost:7687")
- `NEO4J_USERNAME`: Neo4j username (default: "neo4j")
- `NEO4J_PASSWORD`: Neo4j password (default: "password")

## Monitoring

The system includes comprehensive monitoring with Prometheus and Grafana:

### Start Monitoring Stack

```bash
# Start Prometheus and Grafana
make monitoring

# Or manually:
docker compose up -d prometheus grafana
```

### Access Monitoring Interfaces

- **Prometheus**: http://localhost:9090 - Metrics collection and querying
- **Grafana**: http://localhost:3001 - Dashboards and visualization (admin/admin)

### Available Metrics

**Transaction Generator**:
- `tigerbeetle_transfers_generated_total` - Total transfers by status (success/error)
- `tigerbeetle_transfer_amount_dollars` - Distribution of transfer amounts
- `tigerbeetle_accounts_created_total` - Total accounts in system

**System Metrics**:
- Go runtime metrics (memory, goroutines, GC)
- HTTP request metrics for each service
- Custom business logic metrics

### Metrics Endpoints

- Generator: http://localhost:8081/metrics
- CDC Connector: http://localhost:8082/metrics (planned)
- Neo4j Sink: http://localhost:8083/metrics (planned)

## Using Neo4j AuraDB

To use Neo4j AuraDB instead of local Neo4j:

1. Create a Neo4j AuraDB instance
2. Set environment variables:
```bash
export NEO4J_URI="neo4j+s://your-instance.databases.neo4j.io"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="your-password"
```
3. Comment out the neo4j service in docker-compose.yml

## Data Flow

```
TigerBeetle → CDC Connector → Redpanda → Neo4j Sink → Neo4j
```

1. **Transaction Generator** creates random transfers between 10 accounts in TigerBeetle
2. **CDC Connector** polls TigerBeetle's change stream and publishes transfer events as JSON to Redpanda
3. **Neo4j Sink** consumes messages from Redpanda and creates/updates the account graph in Neo4j

## Message Format

JSON messages in Redpanda have this structure:

```json
{
  "ID": "12345",
  "DebitAccountID": "1", 
  "CreditAccountID": "2",
  "Amount": 1000,
  "Ledger": 1,
  "Code": 718
}
```

## Cypher Query

The Neo4j sink executes this idempotent Cypher query for each transfer:

```cypher
MERGE (from:Account {id: $debitId})
MERGE (to:Account {id: $creditId}) 
CREATE (from)-[:SENT_TO {amount: $amount, txId: $txId, ledger: $ledger}]->(to)
```

## Troubleshooting

### TigerBeetle Connection Issues
- Ensure TigerBeetle container is running and healthy
- Check if the database file was properly formatted

### Redpanda Connection Issues  
- Verify Redpanda is healthy: `docker exec redpanda rpk cluster health`
- Check if topics exist: `docker exec redpanda rpk topic list`

### Neo4j Connection Issues
- Test connection: `docker exec neo4j cypher-shell -u neo4j -p password "RETURN 1"`
- Check Neo4j logs: `docker logs neo4j`

## Development

To make changes to the services:

1. Modify the Go code
2. Rebuild: `go build -o bin/<service> ./cmd/<service>`
3. Restart the specific service

## Future Development

This project serves as a foundation for advanced financial analytics and e-commerce applications. Planned enhancements include:
- Complete monitoring infrastructure (TigerBeetle, Redpanda, Neo4j metrics)
- V2 e-commerce web UI with simulated transactions
- PostHog analytics integration for user behavior tracking
- Advanced graph algorithms and machine learning features

## Project Documentation

- [docs/posthog-integration.md](docs/posthog-integration.md) - PostHog analytics planning
- [WARP.md](WARP.md) - Comprehensive project guide for AI development

## Stopping

```bash
# Stop all services
docker-compose down

# Remove volumes (this will delete all data)
docker-compose down -v
```