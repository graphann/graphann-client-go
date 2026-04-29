package graphann

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors. All API errors wrap one of these so callers can write
//
//	if errors.Is(err, graphann.ErrNotFound) { ... }
//
// while still retrieving the full APIError via errors.As.
var (
	// ErrUnauthorized is returned for HTTP 401.
	ErrUnauthorized = errors.New("graphann: unauthorized")
	// ErrForbidden is returned for HTTP 403.
	ErrForbidden = errors.New("graphann: forbidden")
	// ErrNotFound is returned for HTTP 404.
	ErrNotFound = errors.New("graphann: not found")
	// ErrConflict is returned for HTTP 409.
	ErrConflict = errors.New("graphann: conflict")
	// ErrCompactInProgress is returned when POST .../compact responds 409,
	// indicating a compaction is already running. Safe to retry after a delay.
	ErrCompactInProgress = errors.New("graphann: compaction already in progress")
	// ErrPayloadTooLarge is returned for HTTP 413.
	ErrPayloadTooLarge = errors.New("graphann: payload too large")
	// ErrRateLimited is returned for HTTP 429. The retry-after duration
	// is preserved on the wrapping APIError.
	ErrRateLimited = errors.New("graphann: rate limited")
	// ErrIndexNotReady is returned for HTTP 503 when the server reports
	// index_not_ready or index_building.
	ErrIndexNotReady = errors.New("graphann: index not ready")
	// ErrServer is returned for HTTP 5xx that aren't ErrIndexNotReady.
	ErrServer = errors.New("graphann: server error")
	// ErrNetwork is returned for transport-level failures (DNS, TCP, TLS,
	// timeouts) where no HTTP response was received.
	ErrNetwork = errors.New("graphann: network error")
	// ErrBadRequest is returned for HTTP 400.
	ErrBadRequest = errors.New("graphann: bad request")
	// ErrValidation is returned for server-side validation failures.
	ErrValidation = errors.New("graphann: validation failed")
	// ErrNotImplemented is returned for HTTP 501.
	ErrNotImplemented = errors.New("graphann: not implemented")
	// ErrConfig is returned at construction time for invalid SDK options.
	ErrConfig = errors.New("graphann: invalid configuration")
)

// APIError is the structured error returned by the GraphANN HTTP API.
// It implements error and the standard Unwrap()/Is interfaces so callers
// can mix errors.Is sentinel checks with errors.As field access.
type APIError struct {
	// Status is the HTTP status code.
	Status int
	// Code is the server's error code (e.g. "not_found", "rate_limited").
	Code string
	// Message is the human-readable error message.
	Message string
	// Details is any structured detail payload returned by the server.
	Details any
	// RequestID is the X-Request-ID echoed by the server, when present.
	RequestID string
	// RetryAfter is the parsed Retry-After header on 429/503. Zero when
	// absent or unparseable.
	RetryAfter time.Duration
	// sentinel is the wrapped sentinel from the var block above.
	sentinel error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.RequestID != "" {
		return fmt.Sprintf("graphann: %d %s: %s (request_id=%s)", e.Status, e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("graphann: %d %s: %s", e.Status, e.Code, e.Message)
}

// Unwrap returns the sentinel so errors.Is works against the package
// errors.
func (e *APIError) Unwrap() error { return e.sentinel }

// Is reports whether target matches the wrapped sentinel.
func (e *APIError) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	return errors.Is(e.sentinel, target)
}

// newAPIError constructs an APIError with the appropriate sentinel for
// the HTTP status / server code.
func newAPIError(status int, code, message string, details any, requestID string, retryAfter time.Duration) *APIError {
	return &APIError{
		Status:     status,
		Code:       code,
		Message:    message,
		Details:    details,
		RequestID:  requestID,
		RetryAfter: retryAfter,
		sentinel:   sentinelFor(status, code),
	}
}

// sentinelFor maps HTTP status / server code to the sentinel exported
// from this package. Status takes precedence; code is used to disambiguate
// 503 (index_not_ready vs server_error) and 400 (bad_request vs
// validation_error).
func sentinelFor(status int, code string) error {
	switch status {
	case 400:
		if code == "validation_error" {
			return ErrValidation
		}
		return ErrBadRequest
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 409:
		if code == "compact_in_progress" {
			return ErrCompactInProgress
		}
		return ErrConflict
	case 413:
		return ErrPayloadTooLarge
	case 429:
		return ErrRateLimited
	case 501:
		return ErrNotImplemented
	case 503:
		if code == "index_not_ready" || code == "index_building" {
			return ErrIndexNotReady
		}
		return ErrServer
	}
	if status >= 500 {
		return ErrServer
	}
	return nil
}

// errorEnvelope is the JSON shape the server uses for error payloads:
//
//	{ "error": { "code": "...", "message": "...", "details": ... } }
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details,omitempty"`
	} `json:"error"`
}

// netError wraps a transport-level error so callers can errors.Is against
// ErrNetwork.
type netError struct{ err error }

func (n *netError) Error() string { return "graphann: network: " + n.err.Error() }
func (n *netError) Unwrap() error { return n.err }
func (n *netError) Is(target error) bool {
	return target == ErrNetwork
}

func wrapNet(err error) error {
	if err == nil {
		return nil
	}
	return &netError{err: err}
}
