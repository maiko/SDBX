package secrets

import "fmt"

// SecretNotConfiguredError indicates a secret file is not properly configured
type SecretNotConfiguredError struct {
	Filename string
}

func (e *SecretNotConfiguredError) Error() string {
	return fmt.Sprintf("secret not configured: %s (file is empty or contains placeholder)", e.Filename)
}

// IsSecretNotConfigured checks if error is SecretNotConfiguredError
func IsSecretNotConfigured(err error) bool {
	_, ok := err.(*SecretNotConfiguredError)
	return ok
}

// ManualSecretError indicates an operation attempted on a user-provided secret
type ManualSecretError struct {
	Filename string
}

func (e *ManualSecretError) Error() string {
	return fmt.Sprintf("secret %s requires manual configuration and cannot be auto-generated", e.Filename)
}

// IsManualSecret checks if error is ManualSecretError
func IsManualSecret(err error) bool {
	_, ok := err.(*ManualSecretError)
	return ok
}
