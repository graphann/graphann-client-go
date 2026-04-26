# graphann-go

Official Go client SDK for the [GraphANN](https://graphann.com) vector
database.

- Module: `github.com/graphann/graphann-client-go`
- Status: `v0.1.0` — bootstrapping
- License: see [`./LICENSE`](./LICENSE)

The SDK speaks the v1 HTTP API and exposes idiomatic Go types for
tenant, index, document, search, job, cluster, and settings operations.

## Install

```bash
go get github.com/graphann/graphann-client-go@v0.1.0
```

## 5-line quickstart

```go
c, _ := graphann.NewClient(graphann.WithBaseURL("http://localhost:38888"))
defer c.Close()
idx, _ := c.CreateIndex(context.Background(), "t_demo", graphann.CreateIndexRequest{Name: "demo"})
_, _ = c.AddDocuments(context.Background(), "t_demo", idx.ID, graphann.AddDocumentsRequest{Documents: []graphann.Document{{Text: "hello"}}})
res, _ := c.Search(context.Background(), "t_demo", idx.ID, graphann.SearchRequest{Query: "hello", K: 5})
```

A full end-to-end example (tenant + index + ingest + search + hot model
swap) lives at [`examples/quickstart/main.go`](./examples/quickstart/main.go).

## Public surface

```go
type Client struct { /* opaque */ }

func NewClient(opts ...Option) (*Client, error)

// Construction options
type Option func(*config) error
func WithBaseURL(url string) Option
func WithAPIKey(tenantID, apiKey string) Option
func WithHTTPClient(*http.Client) Option
func WithTLSConfig(*tls.Config) Option
func WithRetryPolicy(RetryPolicy) Option
func WithUserAgent(string) Option
func WithLogger(slog.Handler) Option
func WithSingleflight(window time.Duration) Option
func WithQueryCache(maxEntries int, ttl time.Duration) Option
func WithMetricsHook(MetricsHook) Option

// Health
func (c *Client) Health(ctx context.Context) (*HealthResponse, error)
func (c *Client) Ready(ctx context.Context)  (*HealthResponse, error)

// Tenants
func (c *Client) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error)
func (c *Client) GetTenant   (ctx context.Context, tenantID string)         (*Tenant, error)
func (c *Client) ListTenants (ctx context.Context)                          (*ListTenantsResponse, error)
func (c *Client) DeleteTenant(ctx context.Context, tenantID string)         (*DeleteTenantResponse, error)

// Indexes
func (c *Client) CreateIndex   (ctx context.Context, tenantID string, req CreateIndexRequest)              (*Index, error)
func (c *Client) GetIndex      (ctx context.Context, tenantID, indexID string)                             (*Index, error)
func (c *Client) ListIndexes   (ctx context.Context, tenantID string)                                      (*ListIndexesResponse, error)
func (c *Client) UpdateIndex   (ctx context.Context, tenantID, indexID string, req UpdateIndexRequest)     (*Index, error)
func (c *Client) DeleteIndex   (ctx context.Context, tenantID, indexID string)                             error
func (c *Client) GetIndexStatus(ctx context.Context, tenantID, indexID string)                             (*IndexStatusResponse, error)
func (c *Client) GetLiveStats  (ctx context.Context, tenantID, indexID string)                             (*LiveStatsResponse, error)
func (c *Client) CompactIndex  (ctx context.Context, tenantID, indexID string)                             (*CompactResponse, error)
func (c *Client) ClearIndex    (ctx context.Context, tenantID, indexID string)                             (*ClearIndexResponse, error)

// Documents
func (c *Client) AddDocuments         (ctx context.Context, tenantID, indexID string, req AddDocumentsRequest)         (*AddDocumentsResponse, error)
func (c *Client) ImportDocuments      (ctx context.Context, tenantID, indexID string, req ImportDocumentsRequest)      (*ImportDocumentsResponse, error)
func (c *Client) GetPendingStatus     (ctx context.Context, tenantID, indexID string)                                  (*PendingStatusResponse, error)
func (c *Client) ProcessPending       (ctx context.Context, tenantID, indexID string)                                  (*ProcessPendingResponse, error)
func (c *Client) ClearPending         (ctx context.Context, tenantID, indexID string)                                  (*ClearPendingResponse, error)
func (c *Client) GetDocument          (ctx context.Context, tenantID, indexID string, docID int)                       (*DocumentResponse, error)
func (c *Client) DeleteDocument       (ctx context.Context, tenantID, indexID string, docID int)                       (*DeleteDocumentResponse, error)
func (c *Client) BulkDeleteDocuments  (ctx context.Context, tenantID, indexID string, req BulkDeleteDocumentsRequest)  (*BulkDeleteDocumentsResponse, error)
func (c *Client) BulkDeleteByExternalIDs(ctx context.Context, tenantID, indexID string, req BulkDeleteByExternalIDsRequest) (*BulkDeleteByExternalIDsResponse, error)
func (c *Client) ListDocuments        (ctx context.Context, tenantID, indexID string, req ListDocumentsRequest)        *Iter[ListDocumentEntry]
func (c *Client) GetChunk             (ctx context.Context, tenantID, indexID string, chunkID int)                     (*ChunkResponse, error)
func (c *Client) DeleteChunks         (ctx context.Context, tenantID, indexID string, req DeleteChunksRequest)         (*DeleteChunksResponse, error)

// Search
func (c *Client) Search       (ctx context.Context, tenantID, indexID string, req SearchRequest)      (*SearchResponse, error)
func (c *Client) SearchText   (ctx context.Context, tenantID, indexID string, req SearchRequest)      (*SearchResponse, error)
func (c *Client) SearchVector (ctx context.Context, tenantID, indexID string, req SearchRequest)      (*SearchResponse, error)
func (c *Client) MultiSearch  (ctx context.Context, orgID, userID string, req MultiSearchRequest)     (*MultiSearchResponse, error)
func (c *Client) SyncDocuments(ctx context.Context, orgID string, req SyncDocumentsRequest)           (*SyncDocumentsResponse, error)
func (c *Client) ListUserIndexes  (ctx context.Context, orgID, userID string)                         (*ListIndexesResponse, error)
func (c *Client) ListSharedIndexes(ctx context.Context, orgID string)                                 (*ListIndexesResponse, error)

// Async jobs
func (c *Client) SwitchEmbeddingModel(ctx context.Context, tenantID, indexID string, req SwitchEmbeddingModelRequest) (*SwitchEmbeddingModelResponse, error)
func (c *Client) GetJob             (ctx context.Context, jobID string)                                                (*Job, error)
func (c *Client) ListJobs           (ctx context.Context, req ListJobsRequest)                                         *Iter[Job]

// Cluster (read-only)
func (c *Client) GetClusterNodes (ctx context.Context) (*ListClusterNodesResponse, error)
func (c *Client) GetClusterShards(ctx context.Context) (*ListClusterShardsResponse, error)
func (c *Client) GetClusterHealth(ctx context.Context) (*ClusterHealthResponse, error)

// LLM settings (per organization)
func (c *Client) GetLLMSettings   (ctx context.Context, orgID string)                          (*LLMSettings, error)
func (c *Client) UpdateLLMSettings(ctx context.Context, orgID string, settings LLMSettings)    (*LLMSettings, error)
func (c *Client) DeleteLLMSettings(ctx context.Context, orgID string)                          (*DeleteLLMSettingsResponse, error)

// API key management (admin-only; forward-looking — see apikey.go)
func (c *Client) CreateAPIKey(ctx context.Context, tenantID string, req CreateAPIKeyRequest) (*APIKey, error)
func (c *Client) RevokeAPIKey(ctx context.Context, tenantID, keyID string)                    error
func (c *Client) ListAPIKeys (ctx context.Context, tenantID string)                          (*ListAPIKeysResponse, error)
```

## Errors

Every operation returns either nil or a wrapped `*APIError`. Use
sentinel matchers:

```go
res, err := c.Search(ctx, t, i, req)
switch {
case errors.Is(err, graphann.ErrRateLimited):
    var ae *graphann.APIError
    errors.As(err, &ae)
    time.Sleep(ae.RetryAfter)
case errors.Is(err, graphann.ErrIndexNotReady):
    // back off and try again later
case errors.Is(err, graphann.ErrUnauthorized):
    // refresh credentials
case err != nil:
    return err
}
```

Sentinels: `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`,
`ErrConflict`, `ErrPayloadTooLarge`, `ErrRateLimited`,
`ErrIndexNotReady`, `ErrServer`, `ErrNetwork`, `ErrBadRequest`,
`ErrValidation`, `ErrNotImplemented`, `ErrConfig`.

## Production knobs

- **Hardened HTTP client**: tuned `DialContext`, `TLSHandshakeTimeout`,
  `ResponseHeaderTimeout`, `ExpectContinueTimeout`,
  `MaxIdleConnsPerHost`, `IdleConnTimeout`. Override with
  `WithHTTPClient` for instrumented transports.
- **Retries with jitter**: exponential backoff, honours `Retry-After`
  on 429/503 (integer-seconds and HTTP-date forms). Configure with
  `WithRetryPolicy`. Default: 3 attempts, 100ms base, 5s cap, 20% jitter.
- **Body cap**: every response body is read through
  `io.LimitReader(resp.Body, 50 MiB)` to bound memory usage.
- **gzip**: request bodies above 64 KiB are gzip-compressed; the SDK
  honours `Accept-Encoding: gzip` on responses.
- **User-Agent**: `graphann-go/0.1.0 (go1.21+; goos/goarch)` by default.
- **Singleflight**: opt-in with `WithSingleflight(window)`. Concurrent
  identical search requests are coalesced; the result stays sticky for
  `window` after completion.
- **Query cache**: opt-in with `WithQueryCache(max, ttl)`. LRU + TTL
  cache of `*SearchResponse`, keyed on
  `(tenantID, indexID, request fingerprint)`.
- **Metrics hook**: `WithMetricsHook` registers a callback invoked on
  every request — useful for Prometheus counters or OpenTelemetry spans.

## Testing

Unit tests stub the HTTP layer with `httptest.NewServer`:

```bash
go test -race -count=1 ./...
```

Integration tests are gated by the `integration` build tag and require
`GRAPHANN_BASE_URL`. Optional: `GRAPHANN_TENANT_ID`, `GRAPHANN_API_KEY`.

```bash
GRAPHANN_BASE_URL=http://localhost:38888 \
GRAPHANN_TENANT_ID=t_demo GRAPHANN_API_KEY=k \
go test -tags integration -count=1 ./...
```

## License

Commercial — see [`./LICENSE`](./LICENSE).
