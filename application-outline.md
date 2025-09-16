# Project Specification: Real-Time Financial Graph Analysis Pipeline (v2)

## 1. Project Overview

The goal of this project is to build a real-time data pipeline that processes financial transactions and loads them into a graph database for analysis. The system will simulate financial transactions, capture them using Change Data Capture (CDC), stream them through a message queue, and finally model them as a graph.

This architecture demonstrates a modern approach to separating transactional workloads from analytical workloads, enabling powerful, real-time insights into complex financial networks.

### ## Architecture

The system consists of three independent Go microservices and the necessary infrastructure.



* **Transaction Generator:** A Go application that creates sample accounts and financial transfers in a TigerBeetle database.
* **CDC Connector:** A Go application that reads the stream of committed transfers from TigerBeetle's CDC log and publishes them as JSON messages to a Redpanda topic.
* **Neo4j Sink:** A Go application that consumes JSON messages from the Redpanda topic and writes them into a Neo4j AuraDB instance, creating a graph of accounts and transactions.

---

## 2. Core Technologies

* **Language:** Go (Golang)
* **Transactional Database:** TigerBeetle
* **Streaming Platform:** Redpanda (Kafka-compatible)
* **Graph Database:** Neo4j AuraDB
* **Orchestration:** Docker and Docker Compose for local infrastructure.
* **Go Libraries:**
    * `github.com/tigerbeetle/tigerbeetle-go`: For all TigerBeetle interactions.
    * **`github.com/twmb/franz-go`**: For producing to and consuming from Redpanda.
    * `github.com/neo4j/neo4j-go-driver/v5/neo4j`: For connecting and writing to Neo4j.

---

## 3. Service Specifications

### ### Service 1: Transaction Generator

* **Purpose:** To populate the TigerBeetle database with a continuous stream of sample financial transactions.
* **Directory:** `cmd/generator/main.go`
* **Functionality:**
    1.  Establish a connection to the TigerBeetle cluster.
    2.  On startup, create a predefined set of accounts (e.g., 10 accounts with IDs 1 through 10).
    3.  Enter an infinite loop that, every second, generates a random transfer between two random accounts.
    4.  The transfer amount should also be a random integer (e.g., between 1 and 1000).
    5.  The transfer ID must be a unique `Uint128`; a simple incrementing counter can be used.
    6.  Log the result of each transfer creation (success or failure) to standard output.
* **Configuration:** The TigerBeetle cluster address (`3000`) should be a configurable constant.

### ### Service 2: CDC Connector (TigerBeetle -> Redpanda)

* **Purpose:** To act as a bridge, reliably moving committed transaction data from TigerBeetle to Redpanda.
* **Directory:** `cmd/cdc-connector/main.go`
* **Functionality:**
    1.  Establish a connection to the TigerBeetle cluster.
    2.  Initialize a `franz-go` Kafka client (`kgo.Client`) pointing to the Redpanda seed brokers.
    3.  Implement the TigerBeetle CDC loop (`GetChanges`). The loop should only listen for transfer events (`types.ChangeAccountTransfers | types.ChangeTransferAmount`).
    4.  For each `Transfer` event received from the CDC stream:
        a. Marshal the `tigerbeetle-go/pkg/types.Transfer` struct into a JSON object. The JSON should represent the Uint128 IDs as strings.
        b. Create a new `kgo.Record` with the topic set to `transactions`, the transfer ID string as the key, and the JSON data as the value.
        c. Produce the record using the client's `Produce` method. This is an asynchronous operation.
    5.  Log confirmation for each message successfully sent to Redpanda or handle any produce errors.
* **Configuration:** The TigerBeetle address (`3000`), Redpanda broker address (`localhost:19092`), and topic name (`transactions`) should be configurable constants.

### ### Service 3: Neo4j Sink (Redpanda -> Neo4j)

* **Purpose:** To consume transaction events from Redpanda and build the corresponding graph in Neo4j.
* **Directory:** `cmd/neo4j-sink/main.go`
* **Functionality:**
    1.  Establish a connection to the Neo4j AuraDB instance.
    2.  Initialize a `franz-go` Kafka client (`kgo.Client`) with the consumer group (`GroupID`) and topic (`ConsumeTopics`) options set.
    3.  In an infinite loop, call the client's `PollFetches` method to consume records.
    4.  Iterate over the fetched records. For each record:
        a. Unmarshal the JSON value into a Go struct that mirrors the `Transfer` data.
        b. Execute a single Cypher query against Neo4j to update the graph.
        c. The query must be idempotent, using `MERGE` for accounts to avoid duplicates and `CREATE` for the transaction relationship.
* **Cypher Query to Execute:**
    ```cypher
    MERGE (from:Account {id: $debitId})
    MERGE (to:Account {id: $creditId})
    CREATE (from)-[:SENT_TO {amount: $amount, txId: $txId, ledger: $ledger}]->(to)
    ```
* **Configuration:** Redpanda broker address, topic name, consumer group ID (`neo4j-sink-group`), and Neo4j credentials (URI, User, Password) must be configurable.

---

## 4. Data Models

### ### JSON Message Format (in Redpanda)

The JSON message in the `transactions` topic should have the following structure. `Uint128` values from TigerBeetle must be serialized as strings.

```json
{
  "ID": "12345",
  "DebitAccountID": "1",
  "CreditAccountID": "2",
  "Amount": 1000,
  "Ledger": 1,
  "Code": 718
}

5. Acceptance Criteria
The project is complete when:

The generator service successfully creates accounts and transfers, logging its activity.

The cdc-connector service logs that it is forwarding transfers to Redpanda.

The neo4j-sink service logs that it is writing transactions to Neo4j.

Connecting to the Neo4j AuraDB instance with the Neo4j Browser and running the following query returns a graph of accounts and their transactions:

Cypher

MATCH (a:Account)-[r:SENT_TO]->(b:Account)
RETURN a, r, b
LIMIT 25
