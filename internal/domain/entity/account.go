package entity

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID        uuid.UUID
	Balance   int64
	Currency  Currency
	UpdatedAt time.Time
}
