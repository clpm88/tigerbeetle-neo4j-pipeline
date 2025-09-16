package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"tigerbeetle-neo4j/pkg/config"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

const (
	NumAccounts = 10
	Ledger      = 1
)

func main() {
	cfg := config.Load()
	
	log.Printf("Starting Transaction Generator...")
	log.Printf("TigerBeetle Address: %s", cfg.TigerBeetle.Address)

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
		} else if len(results) > 0 {
			log.Printf("ERROR: Transfer %d failed with result: %v", transferID, results[0])
		} else {
			log.Printf("SUCCESS: Transfer %d - $%d from account %d to account %d", 
				transferID, amount, fromAccount, toAccount)
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
