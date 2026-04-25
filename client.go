package graphann

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Version is the SDK version. Embedded in the User-Agent.
const Version = "0.1.0"

// defaultUserAgent is computed once.
var defaultUserAgent = fmt.Sprintf(
	"graphann-go/%s (%s; %s/%s)",
	Version, runtime.Version(), runtime.GOOS, runtime.GOARCH,
)

// maxResponseBodyBytes caps every response body so a malicious server
// cannot OOM the client. 50 MiB matches the server's outbound cap on
// embedding responses.
const maxResponseBodyBytes = 50 << 20

// gzipThreshold is the size at which request bodies are compressed.
const gzipThreshold = 64 << 10 // 64 KiB

// authHeaderTenant is the canonical tenant ID header.
const authHeaderTenant = "X-Tenant-ID"

// authHeaderAPIKey is the canonical API key header.
const authHeaderAPIKey = "X-API-Key" //nolint:gosec // header name, not a credential

// requestIDHeader is echoed by the server.
const requestIDHeader = "X-Request-ID"

// Client is the GraphANN HTTP client. It is safe for concurrent use.
type Client struct {
	cfg config

	httpClient *http.Client
	cache      *queryCache
	sf         *sfGroup

	rngMu sync.Mutex
	rng   *rand.Rand
}

// NewClient builds a Client with the supplied options. WithBaseURL is
// required; other options have sensible defaults.
func NewClient(opts ...Option) (*Client, error) {
	cfg := config{
		baseURL:   "",
		userAgent: defaultUserAgent,
		retry:     DefaultRetryPolicy(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	if cfg.baseURL == "" {
		return nil, errors.Join(ErrConfig, errors.New("base url is required"))
	}
	if _, err := url.Parse(cfg.baseURL); err != nil {
		return nil, errors.Join(ErrConfig, err)
	}
	cfg.baseURL = strings.TrimRight(cfg.baseURL, "/")

	hc := cfg.httpClient
	if hc == nil {
		hc = newHardenedHTTPClient(cfg.tlsConfig, 30*time.Second, 16)
	}

	c := &Client{
		cfg:        cfg,
		httpClient: hc,
		cache:      newQueryCache(cfg.cacheMaxEntries, cfg.cacheTTL),
		sf:         newSFGroup(cfg.singleflightWindow),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec // jitter only, no security relevance
	}
	return c, nil
}

// Close releases resources held by the Client. Currently a no-op when a
// caller-supplied http.Client is in use; otherwise it idle-closes the
// internal Transport.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.httpClient != nil && c.cfg.httpClient == nil {
		if t, ok := c.httpClient.Transport.(*http.Transport); ok {
			t.CloseIdleConnections()
		}
	}
	return nil
}

// newHardenedHTTPClient mirrors the embedding/server.go pattern from the
// GraphANN repo: independent dial / TLS / response-header timeouts so a
// single slow upstream cannot pin the overall budget.
func newHardenedHTTPClient(tlsCfg *tls.Config, timeout time.Duration, maxIdleConnsPerHost int) *http.Client {
	if maxIdleConnsPerHost < 4 {
		maxIdleConnsPerHost = 4
	}
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       tlsCfg,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: t,
	}
}

// =============================================================================
// Internal request plumbing
// =============================================================================

// requestOpts tunes the request envelope.
type requestOpts struct {
	// query is appended to the URL.
	query url.Values
	// extraHeaders are merged onto the outgoing request.
	extraHeaders http.Header
	// noRetry forces a single attempt regardless of the policy.
	noRetry bool
}

// do performs an HTTP request with retry/backoff and decodes JSON into
// out. body and out may be nil. method is uppercase. path begins with /.
func (c *Client) do(ctx context.Context, method, path string, body any, out any, opts *requestOpts) error {
	if ctx == nil {
		return errors.Join(ErrConfig, errors.New("nil context"))
	}
	policy := c.cfg.retry
	if opts != nil && opts.noRetry {
		policy = RetryPolicy{MaxAttempts: 1}
	}
	maxAttempts := policy.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			retryAfter := retryAfterFromError(lastErr)
			c.rngMu.Lock()
			d := policy.backoff(attempt-1, retryAfter, c.rng)
			c.rngMu.Unlock()
			if err := sleepCtx(ctx, d); err != nil {
				return err
			}
		}
		err := c.attemptOnce(ctx, method, path, body, out, opts)
		if err == nil {
			return nil
		}
		lastErr = err
		if !policy.shouldRetry(err) {
			return err
		}
	}
	return lastErr
}

// attemptOnce performs a single HTTP attempt. It does not retry.
func (c *Client) attemptOnce(ctx context.Context, method, path string, body any, out any, opts *requestOpts) error {
	start := time.Now()

	target, err := c.urlFor(path, opts)
	if err != nil {
		return err
	}

	var (
		bodyReader io.Reader
		bodyBytes  []byte
		gzEncoded  bool
	)
	if body != nil {
		bb, mErr := json.Marshal(body)
		if mErr != nil {
			return errors.Join(ErrConfig, mErr)
		}
		bodyBytes = bb
		if len(bb) > gzipThreshold {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			if _, werr := gz.Write(bb); werr != nil {
				return errors.Join(ErrConfig, werr)
			}
			if cerr := gz.Close(); cerr != nil {
				return errors.Join(ErrConfig, cerr)
			}
			bodyReader = &buf
			gzEncoded = true
		} else {
			bodyReader = bytes.NewReader(bb)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
	if err != nil {
		return wrapNet(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
		if gzEncoded {
			req.Header.Set("Content-Encoding", "gzip")
			req.Header.Del("Content-Length")
		}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if c.cfg.userAgent != "" {
		req.Header.Set("User-Agent", c.cfg.userAgent)
	}
	if c.cfg.tenantID != "" {
		req.Header.Set(authHeaderTenant, c.cfg.tenantID)
	}
	if c.cfg.apiKey != "" {
		req.Header.Set(authHeaderAPIKey, c.cfg.apiKey)
	}
	if opts != nil {
		for k, vs := range opts.extraHeaders {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.observe(method, path, 0, time.Since(start), wrapNet(err))
		return wrapNet(err)
	}
	defer func() { _ = resp.Body.Close() }()

	rdr := io.Reader(resp.Body)
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, gzErr := gzip.NewReader(io.LimitReader(resp.Body, maxResponseBodyBytes))
		if gzErr != nil {
			perr := wrapNet(gzErr)
			c.observe(method, path, resp.StatusCode, time.Since(start), perr)
			return perr
		}
		defer func() { _ = gz.Close() }()
		rdr = gz
	}
	limited := io.LimitReader(rdr, maxResponseBodyBytes)
	respBody, readErr := io.ReadAll(limited)
	if readErr != nil {
		perr := wrapNet(readErr)
		c.observe(method, path, resp.StatusCode, time.Since(start), perr)
		return perr
	}

	requestID := resp.Header.Get(requestIDHeader)

	if resp.StatusCode >= 400 {
		apiErr := decodeAPIError(resp, respBody, requestID)
		c.observe(method, path, resp.StatusCode, time.Since(start), apiErr)
		return apiErr
	}

	if out != nil && len(respBody) > 0 && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(respBody, out); err != nil {
			perr := errors.Join(ErrServer, err)
			c.observe(method, path, resp.StatusCode, time.Since(start), perr)
			return perr
		}
	}

	c.observe(method, path, resp.StatusCode, time.Since(start), nil)
	return nil
}

// observe forwards a request observation to the configured metrics hook.
func (c *Client) observe(method, path string, status int, d time.Duration, err error) {
	if c.cfg.metrics == nil {
		return
	}
	c.cfg.metrics.ObserveRequest(method, path, status, d, err)
}

// decodeAPIError parses the server error envelope and synthesises an
// APIError with the appropriate sentinel and Retry-After.
func decodeAPIError(resp *http.Response, body []byte, requestID string) *APIError {
	var env errorEnvelope
	_ = json.Unmarshal(body, &env)
	code := env.Error.Code
	msg := env.Error.Message
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	return newAPIError(resp.StatusCode, code, msg, env.Error.Details, requestID, retryAfter)
}

// parseRetryAfter understands integer-seconds and HTTP-date forms.
func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// urlFor builds a fully-qualified URL for path, joining onto baseURL and
// applying any query parameters from opts.
func (c *Client) urlFor(path string, opts *requestOpts) (string, error) {
	if path == "" {
		return "", errors.Join(ErrConfig, errors.New("empty path"))
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u, err := url.Parse(c.cfg.baseURL + path)
	if err != nil {
		return "", errors.Join(ErrConfig, err)
	}
	if opts != nil && len(opts.query) > 0 {
		q := u.Query()
		for k, vs := range opts.query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}
