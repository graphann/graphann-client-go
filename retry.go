package graphann

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy describes the SDK's retry behaviour for transient failures.
// A zero RetryPolicy disables retries.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts (including the first).
	// 0 or 1 disables retries.
	MaxAttempts int
	// InitialBackoff is the base delay before the second attempt.
	InitialBackoff time.Duration
	// MaxBackoff caps the per-attempt delay after exponential growth.
	MaxBackoff time.Duration
	// JitterFraction is the fraction of the computed backoff added as
	// uniform random jitter. Recommended 0.2; clamped to [0, 1].
	JitterFraction float64
	// RetryOn lets callers extend the default classifier. Return true to
	// retry, false to abort. err is the SDK-classified error.
	RetryOn func(err error) bool
}

// DefaultRetryPolicy returns a sensible default: 3 attempts, 100 ms base
// backoff, 5 s max, 20% jitter. Retries on ErrRateLimited, ErrServer,
// ErrIndexNotReady, and ErrNetwork.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		JitterFraction: 0.2,
	}
}

// shouldRetry returns true if the error is one of the well-known
// transient classes, plus the optional caller-supplied RetryOn hook.
func (p RetryPolicy) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if p.RetryOn != nil && p.RetryOn(err) {
		return true
	}
	switch {
	case errors.Is(err, ErrRateLimited):
		return true
	case errors.Is(err, ErrServer):
		return true
	case errors.Is(err, ErrIndexNotReady):
		return true
	case errors.Is(err, ErrNetwork):
		return true
	}
	return false
}

// backoff computes the delay for the given attempt index (0-based, so
// attempt=0 means "before the second try"). Honours retryAfter from the
// server when provided (for 429 / 503).
func (p RetryPolicy) backoff(attempt int, retryAfter time.Duration, rng *rand.Rand) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	if p.InitialBackoff <= 0 {
		return 0
	}
	exp := math.Pow(2, float64(attempt))
	d := time.Duration(float64(p.InitialBackoff) * exp)
	if p.MaxBackoff > 0 && d > p.MaxBackoff {
		d = p.MaxBackoff
	}
	jf := p.JitterFraction
	if jf < 0 {
		jf = 0
	}
	if jf > 1 {
		jf = 1
	}
	if jf > 0 {
		jitter := time.Duration(rng.Float64() * jf * float64(d))
		d += jitter
	}
	return d
}

// retryAfterFromError extracts the Retry-After hint from an APIError, if
// present. Returns 0 otherwise.
func retryAfterFromError(err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

// sleepCtx waits for d or until ctx is done.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
