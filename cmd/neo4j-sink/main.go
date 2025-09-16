package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"tigerbeetle-neo4j/pkg/config"
	"tigerbeetle-neo4j/pkg/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting Neo4j Sink...")
	log.Printf("Redpanda Brokers: %s", strings.Join(cfg.Redpanda.Brokers, ","))
	log.Printf("Redpanda Topic: %s", cfg.Redpanda.Topic)
	log.Printf("Consumer Group: %s", cfg.Redpanda.ConsumerGroup)
	log.Printf("Neo4j URI: %s", cfg.Neo4j.URI)

	// Connect to Neo4j
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4j.URI,
		neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""),
	)
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close(context.Background())

	// Test Neo4j connection
	err = driver.VerifyConnectivity(context.Background())
	if err != nil {
		log.Fatalf("Failed to verify Neo4j connectivity: %v", err)
	}

	log.Println("Connected to Neo4j")

	// Create Kafka consumer client
	kafkaClient, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Redpanda.Brokers...),
		kgo.ConsumerGroup(cfg.Redpanda.ConsumerGroup),
		kgo.ConsumeTopics(cfg.Redpanda.Topic),
	)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	log.Println("Connected to Redpanda as consumer")

	// Start consuming messages
	log.Println("Starting message consumption...")
	
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	for {
		// Poll for messages
		fetches := kafkaClient.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				log.Printf("ERROR: Fetch error: %v", err)
			}
			continue
		}

		// Process each record
		fetches.EachRecord(func(record *kgo.Record) {
			// Unmarshal JSON message
			var transfer models.Transfer
			if err := json.Unmarshal(record.Value, &transfer); err != nil {
				log.Printf("ERROR: Failed to unmarshal message: %v", err)
				return
			}

			// Write to Neo4j
			if err := writeTransferToNeo4j(ctx, session, transfer); err != nil {
				log.Printf("ERROR: Failed to write transfer %s to Neo4j: %v", transfer.ID, err)
				return
			}

			log.Printf("SUCCESS: Written transfer %s to Neo4j graph (from:%s to:%s amount:%d)", 
				transfer.ID, transfer.DebitAccountID, transfer.CreditAccountID, transfer.Amount)
		})
	}
}

// writeTransferToNeo4j executes the Cypher query to create accounts and transaction relationship
func writeTransferToNeo4j(ctx context.Context, session neo4j.SessionWithContext, transfer models.Transfer) error {
	cypher := `
		MERGE (from:Account {id: $debitId})
		MERGE (to:Account {id: $creditId})
		CREATE (from)-[:SENT_TO {amount: $amount, txId: $txId, ledger: $ledger}]->(to)
	`

	parameters := map[string]any{
		"debitId":  transfer.DebitAccountID,
		"creditId": transfer.CreditAccountID,
		"amount":   transfer.Amount,
		"txId":     transfer.ID,
		"ledger":   transfer.Ledger,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, cypher, parameters)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})

	return err
}