package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when an entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicateSKU is returned when a SKU is already used by another detail.
	ErrDuplicateSKU = errors.New("sku already exists")
)

// ValidationError describes an invalid field.
type ValidationError struct {
	Field   string
	Message string
}

// NewValidationError builds a ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Message)
}
