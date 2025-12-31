package config

import "fmt"

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error [%s]: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ProjectNotFoundError indicates no SDBX project was found
type ProjectNotFoundError struct {
	StartPath string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("not in a sdbx project directory (searched from: %s)", e.StartPath)
}

// IsProjectNotFoundError checks if an error is a ProjectNotFoundError
func IsProjectNotFoundError(err error) bool {
	_, ok := err.(*ProjectNotFoundError)
	return ok
}
