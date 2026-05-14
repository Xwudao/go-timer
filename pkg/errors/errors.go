package errors

import (
	"errors"
	"fmt"
)

// Error codes
const (
	CodeNotFound       = 1
	CodeAlreadyExists  = 2
	CodeSystemd        = 3
	CodeConfig         = 4
	CodePermission     = 5
	CodeInvalidInput   = 6
	CodeNotInitialized = 7
)

// TimerdError is a structured error with a code.
type TimerdError struct {
	Code    int
	Message string
	Cause   error
}

func (e *TimerdError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *TimerdError) Unwrap() error {
	return e.Cause
}

// New creates a new TimerdError.
func New(code int, message string) *TimerdError {
	return &TimerdError{Code: code, Message: message}
}

// Wrap wraps an existing error with a code and message.
func Wrap(code int, message string, cause error) *TimerdError {
	return &TimerdError{Code: code, Message: message, Cause: cause}
}

// NotFound returns a not-found error.
func NotFound(name string) *TimerdError {
	return New(CodeNotFound, fmt.Sprintf("job %q not found", name))
}

// AlreadyExists returns an already-exists error.
func AlreadyExists(name string) *TimerdError {
	return New(CodeAlreadyExists, fmt.Sprintf("job %q already exists", name))
}

// SystemdError wraps a systemd operation error.
func SystemdError(op string, cause error) *TimerdError {
	return Wrap(CodeSystemd, fmt.Sprintf("systemd operation %q failed", op), cause)
}

// ConfigError wraps a config operation error.
func ConfigError(op string, cause error) *TimerdError {
	return Wrap(CodeConfig, fmt.Sprintf("config operation %q failed", op), cause)
}

// NotInitialized returns an error when the config directory is not initialized.
func NotInitialized() *TimerdError {
	return New(CodeNotInitialized, "timerd is not initialized; run 'timerd init' first")
}

// IsNotFound checks if an error is a not-found error.
func IsNotFound(err error) bool {
	var te *TimerdError
	if errors.As(err, &te) {
		return te.Code == CodeNotFound
	}
	return false
}

// IsAlreadyExists checks if an error is an already-exists error.
func IsAlreadyExists(err error) bool {
	var te *TimerdError
	if errors.As(err, &te) {
		return te.Code == CodeAlreadyExists
	}
	return false
}
