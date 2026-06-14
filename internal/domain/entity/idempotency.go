package entity

type IdempotencyStatus string

const (
	IDEMPOTENCY_STATUS_UNSPECIFIED IdempotencyStatus = "unspecified"
	IDEMPOTENCY_IN_PROCESS         IdempotencyStatus = "in process"
	IDEMPOTENCY_MISS               IdempotencyStatus = "miss"
	IDEMPOTENCY_COMPLETED          IdempotencyStatus = "completed"
)
