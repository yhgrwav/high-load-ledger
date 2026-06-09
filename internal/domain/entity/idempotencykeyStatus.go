package entity

type IdempotencyKeyStatus string

const (
	UNKNOWN    IdempotencyKeyStatus = "unknown"
	UNUSED     IdempotencyKeyStatus = "unused"
	INPROGRESS IdempotencyKeyStatus = "inProgress"
	COMPLETED  IdempotencyKeyStatus = "completed"
)
