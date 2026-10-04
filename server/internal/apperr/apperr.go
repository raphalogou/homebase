// Package apperr defines the typed errors that map to the API error codes in
// docs/SPEC.md section 1.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is one of the error codes of the API.
type Code string

const (
	Invalid      Code = "invalid"
	Unauthorized Code = "unauthorized"
	NotFound     Code = "not_found"
	TooLarge     Code = "too_large"
	RateLimited  Code = "rate_limited"
)

// Status is the HTTP status for the code.
func (c Code) Status() int {
	switch c {
	case Invalid:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case NotFound:
		return http.StatusNotFound
	case TooLarge:
		return http.StatusRequestEntityTooLarge
	case RateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// Error is an error the client is allowed to see. Reason is a short machine
// value such as "day_full"; Message is a sentence for people. Field names the
// request field the message belongs to, so a form can show it under that field.
type Error struct {
	Code    Code
	Reason  string
	Message string
	Field   string
}

func (e *Error) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%s (%s): %s", e.Code, e.Reason, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New returns an *Error.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf returns an *Error with a formatted message.
func Newf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// WithReason returns an *Error that carries a machine-readable reason.
func WithReason(code Code, reason, message string) *Error {
	return &Error{Code: code, Reason: reason, Message: message}
}

// ForField returns an *Error that belongs to one request field.
func ForField(code Code, field, message string) *Error {
	return &Error{Code: code, Field: field, Message: message}
}

// As finds an *Error in err's chain.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
