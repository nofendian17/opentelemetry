package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents different types of domain errors
type ErrorCode string

const (
	// User related errors
	ErrCodeUserNotFound      ErrorCode = "USER_NOT_FOUND"
	ErrCodeUserAlreadyExists ErrorCode = "USER_ALREADY_EXISTS"
	ErrCodeInvalidUserData   ErrorCode = "INVALID_USER_DATA"

	// Validation errors
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidEmail     ErrorCode = "INVALID_EMAIL"
	ErrCodeInvalidName      ErrorCode = "INVALID_NAME"
	ErrCodeInvalidID        ErrorCode = "INVALID_ID"

	// Repository errors
	ErrCodeRepositoryError ErrorCode = "REPOSITORY_ERROR"
	ErrCodeDatabaseError   ErrorCode = "DATABASE_ERROR"

	// Application errors
	ErrCodeInternalError ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceError  ErrorCode = "SERVICE_ERROR"
)

// DomainError represents a domain-specific error with context
type DomainError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for error wrapping
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for errors.Is
func (e *DomainError) Is(target error) bool {
	var t *DomainError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// WithContext adds context information to the error
func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewDomainError creates a new domain error
func NewDomainError(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// NewDomainErrorWithCause creates a new domain error with an underlying cause
func NewDomainErrorWithCause(code ErrorCode, message string, cause error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// Predefined domain errors
var (
	ErrUserNotFound      = NewDomainError(ErrCodeUserNotFound, "user not found")
	ErrUserAlreadyExists = NewDomainError(ErrCodeUserAlreadyExists, "user already exists")
	ErrInvalidUserData   = NewDomainError(ErrCodeInvalidUserData, "invalid user data")
	ErrValidationFailed  = NewDomainError(ErrCodeValidationFailed, "validation failed")
	ErrInvalidEmail      = NewDomainError(ErrCodeInvalidEmail, "invalid email format")
	ErrInvalidName       = NewDomainError(ErrCodeInvalidName, "invalid name")
	ErrInvalidID         = NewDomainError(ErrCodeInvalidID, "invalid ID")
	ErrRepositoryError   = NewDomainError(ErrCodeRepositoryError, "repository error")
	ErrDatabaseError     = NewDomainError(ErrCodeDatabaseError, "database error")
	ErrInternalError     = NewDomainError(ErrCodeInternalError, "internal error")
	ErrServiceError      = NewDomainError(ErrCodeServiceError, "service error")
)

// IsUserNotFound checks if the error is a user not found error
func IsUserNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound)
}

// IsUserAlreadyExists checks if the error is a user already exists error
func IsUserAlreadyExists(err error) bool {
	return errors.Is(err, ErrUserAlreadyExists)
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrCodeValidationFailed ||
			domainErr.Code == ErrCodeInvalidEmail ||
			domainErr.Code == ErrCodeInvalidName ||
			domainErr.Code == ErrCodeInvalidID ||
			domainErr.Code == ErrCodeInvalidUserData
	}
	return false
}

// IsRepositoryError checks if the error is a repository error
func IsRepositoryError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrCodeRepositoryError ||
			domainErr.Code == ErrCodeDatabaseError
	}
	return false
}
