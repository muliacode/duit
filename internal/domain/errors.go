package domain

import "errors"

// Sentinel errors returned by repositories and services. Callers (the Wails
// App layer) map these to user-facing messages; nothing below this layer
// knows about HTTP/JSON/UI concerns.
var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation failed")
)

// ValidationError carries which field failed and why, so the frontend can
// show an inline error next to the right input instead of a generic banner.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
