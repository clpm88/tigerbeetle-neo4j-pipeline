package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"tigerbeetle-neo4j/pkg/config"
	"tigerbeetle-neo4j/pkg/models"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting CDC Connector...")
	log.Printf("TigerBeetle Address: %s", cfg.TigerBeetle.Address)
	log.Printf("Redpanda Brokers: %s", strings.Join(cfg.Redpanda.Brokers, ","))
	log.Printf("Redpanda Topic: %s", cfg.Redpanda.Topic)

	// Connect to TigerBeetle
	clusterID := types.ToUint128(0) // Cluster ID 0 for single node
	client, err := tb.NewClient(clusterID, []string{cfg.TigerBeetle.Address})
	if err != nil {
		log.Fatalf("Failed to create TigerBeetle client: %v", err)
	}
	defer client.Close()

	log.Println("Connected to TigerBeetle")

	// Create Kafka client
	kafkaClient, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Redpanda.Brokers...),
	)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	log.Println("Connected to Redpanda")

	// Start CDC loop
	log.Println("Starting CDC processing...")
	
	// Note: CDC functionality in TigerBeetle requires specific setup.
	// For now, we'll implement a simple polling mechanism for transfers.
	// In production, you would use the GetChangeEvents API with proper filtering.
	
	log.Println("WARNING: Using polling mode instead of CDC for this demo")
	log.Println("In production, configure TigerBeetle CDC and use GetChangeEvents API")
	
	// Keep track of processed transfers
	processedTransfers := make(map[string]bool)
	
	for {
		// Query recent transfers (this is a simplified approach)
		// In production, you would use proper CDC with GetChangeEvents
		filter := types.QueryFilter{
			UserData128: types.ToUint128(0),
			UserData64:  0,
			UserData32:  0,
			Ledger:      1,
			Code:        718,
			TimestampMin: 0,
			TimestampMax: 0,
			Limit:       100,
			Flags:       0, // QueryFilterFlags as uint32
		}
		
		transfers, err := client.QueryTransfers(filter)
		if err != nil {
			log.Printf("ERROR: Failed to query transfers: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, transfer := range transfers {
			transferIDStr := uint128ToString(transfer.ID)
			
			// Skip if already processed
			if processedTransfers[transferIDStr] {
				continue
			}
			
			// Convert TigerBeetle Transfer to our JSON model
			jsonTransfer := models.Transfer{
				ID:              transferIDStr,
				DebitAccountID:  uint128ToString(transfer.DebitAccountID),
				CreditAccountID: uint128ToString(transfer.CreditAccountID),
				Amount:          uint128ToUint64(transfer.Amount),
				Ledger:          transfer.Ledger,
				Code:            transfer.Code,
			}

			// Marshal to JSON
			jsonData, err := json.Marshal(jsonTransfer)
			if err != nil {
				log.Printf("ERROR: Failed to marshal transfer %s to JSON: %v", jsonTransfer.ID, err)
				continue
			}

			// Create Kafka record
			record := &kgo.Record{
				Topic: cfg.Redpanda.Topic,
				Key:   []byte(jsonTransfer.ID),
				Value: jsonData,
			}

			// Produce to Redpanda
			kafkaClient.Produce(context.Background(), record, func(record *kgo.Record, err error) {
				if err != nil {
					log.Printf("ERROR: Failed to produce transfer %s to Redpanda: %v", jsonTransfer.ID, err)
				} else {
					log.Printf("SUCCESS: Forwarded transfer %s to Redpanda topic '%s'", 
						jsonTransfer.ID, cfg.Redpanda.Topic)
					processedTransfers[transferIDStr] = true
				}
			})
		}
		
		time.Sleep(2 * time.Second) // Poll every 2 seconds
	}
}

// uint128ToString converts a Uint128 to a string representation
func uint128ToString(u types.Uint128) string {
	// For simplicity in demo, we'll just use the string representation
	// In production, you might want more sophisticated conversion
	return u.String()
}

// uint128ToUint64 converts a Uint128 to uint64 (for amount values)
func uint128ToUint64(u types.Uint128) uint64 {
	// For demo purposes, we assume amounts fit in uint64
	// In production, you might need proper big number handling
	bigInt := u.BigInt()
	return bigInt.Uint64()
}
