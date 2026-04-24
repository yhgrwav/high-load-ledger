package entity

import "github.com/google/uuid"

type Posting struct {
	ID            int64
	TransactionID uuid.UUID
	AccountID     int64
	Amount        int64
}
