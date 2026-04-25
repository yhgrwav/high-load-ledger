package entity

import "errors"

var (
	ErrAccountNotFound = errors.New("account not found")

	ErrTransactionNotFound     = errors.New("transaction not found")
	ErrDuplicateIdempotencyKey = errors.New("idempotency key already exists")
)
