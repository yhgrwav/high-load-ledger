package entity

import "errors"

var (
	ErrAccountNotFound = errors.New("account not found")

	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidTxType       = errors.New("invalid transaction type")
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrSameAccountTransfer = errors.New("source and destination accounts must be different")
	ErrInvalidCurrency     = errors.New("currency code is invalid or empty")
	ErrEmptyIdempotencyKey = errors.New("idempotency key is required")
	ErrInvalidUUID         = errors.New("UUID is invalid")

	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrCurrencyMismatch  = errors.New("account currency does not match transfer currency")
)
