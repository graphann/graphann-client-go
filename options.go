package graphann

import (
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Option configures a Client at construction time.
type Option func(*config) error

// MetricsHook is invoked once per outbound request. Implementations must
// be safe for concurrent use; the SDK does not synchronize.
type MetricsHook interface {
	// ObserveRequest reports a single request with its outcome.
	ObserveRequest(method, path string, status int, duration time.Duration, err error)
}

// MetricsHookFunc adapts a plain function to the MetricsHook interface.
type MetricsHookFunc func(method, path string, status int, duration time.Duration, err error)

// ObserveRequest implements MetricsHook.
func (f MetricsHookFunc) ObserveRequest(method, path string, status int, duration time.Duration, err error) {
	f(method, path, status, duration, err)
}

// config holds the resolved Client configuration.
type config struct {
	baseURL    string
	apiKey     string
	tenantID   string
	httpClient *http.Client
	tlsConfig  *tls.Config
	retry      RetryPolicy
	userAgent  string
	logger     *slog.Logger

	singleflightWindow time.Duration
	cacheMaxEntries    int
	cacheTTL           time.Duration

	metrics MetricsHook
}

// WithBaseURL sets the API endpoint base URL. The path component is
// preserved (so http://host/prefix is supported), the trailing slash is
// trimmed for joining.
func WithBaseURL(url string) Option {
	return func(c *config) error {
		if url == "" {
			return errors.Join(ErrConfig, errors.New("base url is empty"))
		}
		c.baseURL = url
		return nil
	}
}

// WithAPIKey sets the tenant ID and API key sent on every request. Both
// values are forwarded as the X-Tenant-ID and X-API-Key headers,
// matching the GraphANN auth middleware contract.
func WithAPIKey(tenantID, apiKey string) Option {
	return func(c *config) error {
		c.tenantID = tenantID
		c.apiKey = apiKey
		return nil
	}
}

// WithHTTPClient supplies a pre-built http.Client. If unset, the SDK
// constructs a hardened client (tuned dial, TLS, idle, response
// timeouts). Use this option for custom transports (e.g. Istio mTLS,
// instrumentation).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) error {
		if hc == nil {
			return errors.Join(ErrConfig, errors.New("http client is nil"))
		}
		c.httpClient = hc
		return nil
	}
}

// WithTLSConfig overrides TLS settings on the SDK-built http.Client. It
// is ignored if WithHTTPClient is used.
func WithTLSConfig(t *tls.Config) Option {
	return func(c *config) error {
		c.tlsConfig = t
		return nil
	}
}

// WithRetryPolicy configures retry behaviour for transient errors.
// Zero-valued policy disables retries.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *config) error {
		c.retry = p
		return nil
	}
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *config) error {
		c.userAgent = ua
		return nil
	}
}

// WithLogger installs a slog.Handler for SDK logging. Pass nil to
// disable.
func WithLogger(h slog.Handler) Option {
	return func(c *config) error {
		if h == nil {
			c.logger = nil
			return nil
		}
		c.logger = slog.New(h)
		return nil
	}
}

// WithSingleflight enables coalescing of concurrent identical search
// requests. Setting window to 0 disables singleflight; non-zero values
// keep the singleflight key live for that duration after a hit (so
// downstream callers within the window observe the cached response).
//
// Internally the SDK uses golang.org/x/sync/singleflight; this option
// only toggles its participation in the request path.
func WithSingleflight(window time.Duration) Option {
	return func(c *config) error {
		if window < 0 {
			return errors.Join(ErrConfig, errors.New("singleflight window is negative"))
		}
		c.singleflightWindow = window
		return nil
	}
}

// WithQueryCache enables a client-side LRU + TTL cache of search
// responses. maxEntries=0 disables the cache.
func WithQueryCache(maxEntries int, ttl time.Duration) Option {
	return func(c *config) error {
		if maxEntries < 0 || ttl < 0 {
			return errors.Join(ErrConfig, errors.New("cache parameters cannot be negative"))
		}
		c.cacheMaxEntries = maxEntries
		c.cacheTTL = ttl
		return nil
	}
}

// WithMetricsHook installs a hook called once per request with
// method, path, status code, duration, and the terminal error (nil on
// success).
func WithMetricsHook(h MetricsHook) Option {
	return func(c *config) error {
		c.metrics = h
		return nil
	}
}
