package models

// Transfer represents a financial transfer for JSON serialization
// This mirrors the TigerBeetle Transfer struct but with JSON-friendly types
type Transfer struct {
	ID              string `json:"ID"`
	DebitAccountID  string `json:"DebitAccountID"`
	CreditAccountID string `json:"CreditAccountID"`
	Amount          uint64 `json:"Amount"`
	Ledger          uint32 `json:"Ledger"`
	Code            uint16 `json:"Code"`
}