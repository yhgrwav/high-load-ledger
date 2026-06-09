package entity

import "bytes"

type IdempotencyState int

const (
	IdempotencyUnused IdempotencyState = iota
	IdempotencyInProgress
	IdempotencyCompleted
)

// IdempotencyInProgressMarker — значение в Redis, пока запрос обрабатывается.
var IdempotencyInProgressMarker = []byte("IN_PROGRESS")

type IdempotencyEntry struct {
	State    IdempotencyState
	Response []byte
}

func ParseIdempotencyValue(raw []byte) IdempotencyEntry {
	if bytes.Equal(raw, IdempotencyInProgressMarker) {
		return IdempotencyEntry{State: IdempotencyInProgress}
	}
	return IdempotencyEntry{State: IdempotencyCompleted, Response: raw}
}
