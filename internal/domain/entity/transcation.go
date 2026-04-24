package entity

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus int16

const (
	STATUS_UNKNOWN TransactionStatus = iota
	STATUS_PENDING
	STATUS_COMPLETED
	STATUS_FAIILED
)

type Transaction struct {
	ID             uuid.UUID
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Currency       Currency
	Amount         int64
	IdempotencyKey uuid.UUID
	Status         TransactionStatus
	CreatedAt      time.Time
}
