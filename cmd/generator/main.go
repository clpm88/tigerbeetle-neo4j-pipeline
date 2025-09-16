package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"tigerbeetle-neo4j/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	tb "github.com/tigerbeetle/tigerbeetle-go"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

const (
	NumAccounts = 10
	Ledger      = 1
)

// Prometheus metrics
var (
	transfersGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tigerbeetle_transfers_generated_total",
			Help: "Total number of transfers generated",
		},
		[]string{"status"}, // success, error
	)
	
	transferAmount = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "tigerbeetle_transfer_amount_dollars",
			Help:    "Distribution of transfer amounts in dollars",
			Buckets: prometheus.LinearBuckets(0, 100, 11), // 0-1000 in 100 dollar buckets
		},
	)
	
	accountsCreated = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tigerbeetle_accounts_created_total",
			Help: "Total number of accounts created",
		},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(transfersGenerated)
	prometheus.MustRegister(transferAmount)
	prometheus.MustRegister(accountsCreated)
}

func main() {
	cfg := config.Load()
	
	log.Printf("Starting Transaction Generator...")
	log.Printf("TigerBeetle Address: %s", cfg.TigerBeetle.Address)
	
	// Start metrics HTTP server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Starting metrics server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Printf("Metrics server failed: %v", err)
		}
	}()

	// Connect to TigerBeetle
	clusterID := types.ToUint128(0) // Cluster ID 0 for single node
	client, err := tb.NewClient(clusterID, []string{cfg.TigerBeetle.Address})
	if err != nil {
		log.Fatalf("Failed to create TigerBeetle client: %v", err)
	}
	defer client.Close()

	log.Println("Connected to TigerBeetle")

	// Create accounts on startup
	if err := createAccounts(client); err != nil {
		log.Fatalf("Failed to create accounts: %v", err)
	}

	log.Printf("Created %d accounts", NumAccounts)

	// Generate transfers in infinite loop
	// Use timestamp as base to ensure unique IDs across runs
	transferID := uint64(time.Now().Unix())
	
	for {
		// Generate random transfer
		fromAccount := uint64(rand.Intn(NumAccounts) + 1)
		toAccount := uint64(rand.Intn(NumAccounts) + 1)
		
		// Ensure different accounts
		for toAccount == fromAccount {
			toAccount = uint64(rand.Intn(NumAccounts) + 1)
		}
		
		amount := uint64(rand.Intn(1000) + 1)
		
		transfer := types.Transfer{
			ID:              types.ToUint128(transferID),
			DebitAccountID:  types.ToUint128(fromAccount),
			CreditAccountID: types.ToUint128(toAccount),
			Amount:          types.ToUint128(amount),
			Ledger:          Ledger,
			Code:            718, // Transfer code as specified in outline
		}

		// Submit transfer
		results, err := client.CreateTransfers([]types.Transfer{transfer})
		if err != nil {
			log.Printf("ERROR: Failed to create transfer %d: %v", transferID, err)
			transfersGenerated.WithLabelValues("error").Inc()
		} else if len(results) > 0 {
			log.Printf("ERROR: Transfer %d failed with result: %v", transferID, results[0])
			transfersGenerated.WithLabelValues("error").Inc()
		} else {
			log.Printf("SUCCESS: Transfer %d - $%d from account %d to account %d", 
				transferID, amount, fromAccount, toAccount)
			transfersGenerated.WithLabelValues("success").Inc()
			transferAmount.Observe(float64(amount))
		}

		transferID++
		time.Sleep(1 * time.Second)
	}
}

func createAccounts(client tb.Client) error {
	// First, check which accounts already exist
	accountIDs := make([]types.Uint128, NumAccounts)
	for i := 0; i < NumAccounts; i++ {
		accountIDs[i] = types.ToUint128(uint64(i + 1))
	}
	
	existingAccounts, err := client.LookupAccounts(accountIDs)
	if err != nil {
		return fmt.Errorf("failed to lookup existing accounts: %w", err)
	}
	
	// Create a set of existing account IDs for quick lookup
	existingIDs := make(map[string]bool)
	for _, account := range existingAccounts {
		id := account.ID.String()
		existingIDs[id] = true
	}
	
	// Only create accounts that don't exist
	var accountsToCreate []types.Account
	for i := 0; i < NumAccounts; i++ {
		accountID := uint64(i + 1)
		accountIDStr := types.ToUint128(accountID).String()
		if !existingIDs[accountIDStr] {
			accountsToCreate = append(accountsToCreate, types.Account{
				ID:             types.ToUint128(accountID),
				DebitsPending:  types.ToUint128(0),
				DebitsPosted:   types.ToUint128(0),
				CreditsPending: types.ToUint128(0),
				CreditsPosted:  types.ToUint128(0),
				UserData128:    types.ToUint128(0),
				UserData64:     0,
				UserData32:     0,
				Reserved:       0,
				Ledger:         Ledger,
				Code:           1, // Account code
				Flags:          0,
				Timestamp:      0,
			})
		}
	}
	
	log.Printf("Found %d existing accounts, creating %d new accounts", 
		len(existingAccounts), len(accountsToCreate))
	
	// Update accounts created metric
	accountsCreated.Set(float64(len(existingAccounts) + len(accountsToCreate)))
	
	// Create only the missing accounts
	if len(accountsToCreate) > 0 {
		results, err := client.CreateAccounts(accountsToCreate)
		if err != nil {
			return fmt.Errorf("create accounts error: %w", err)
		}
		
		if len(results) > 0 {
			return fmt.Errorf("account creation failed with results: %v", results)
		}
	}

	return nil
}
