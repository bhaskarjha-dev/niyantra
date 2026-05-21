package core

import "fmt"

// ProviderError is a structured error from a provider operation.
type ProviderError struct {
	Provider  string // Provider ID
	Op        string // Operation: "fetch", "auth", "discover", "parse"
	Err       error  // Underlying error
	Retryable bool   // Whether this error is transient and worth retrying
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %s: %v", e.Provider, e.Op, e.Err)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// NewProviderError creates a ProviderError.
func NewProviderError(provider, op string, err error, retryable bool) *ProviderError {
	return &ProviderError{
		Provider:  provider,
		Op:        op,
		Err:       err,
		Retryable: retryable,
	}
}
