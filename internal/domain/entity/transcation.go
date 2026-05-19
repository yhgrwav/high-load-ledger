package entity

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID             uuid.UUID
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Currency       Currency
	Amount         int64
	IdempotencyKey uuid.UUID
	CreatedAt      time.Time
}

type TransactionRequest struct {
	IdempotencyKey uuid.UUID
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Currency       Currency
	Amount         int64
}
