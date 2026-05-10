package domain

import "errors"

var (
	// ErrNotFound means the resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict means an optimistic concurrency check failed (version mismatch).
	ErrConflict = errors.New("version conflict")

	// ErrDuplicate means the idempotency key was already used.
	ErrDuplicate = errors.New("duplicate request")

	// ErrInvalidInput means the caller sent bad data.
	ErrInvalidInput = errors.New("invalid input")
)
