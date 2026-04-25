package entity

import (
	"github.com/google/uuid"
)

type Account struct {
	ID       uuid.UUID
	Balance  int64
	Currency Currency
}
