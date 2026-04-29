package graphann

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// Test helpers
// =============================================================================

// mockHandler is a small fake server. routes is a map of "METHOD path" to
// the response handler. Anything unmatched returns 404.
type mockHandler struct {
	routes map[string]http.HandlerFunc
	calls  int64
}

func newMockHandler(routes map[string]http.HandlerFunc) *mockHandler {
	return &mockHandler{routes: routes}
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&h.calls, 1)
	key := r.Method + " " + r.URL.Path
	hh, ok := h.routes[key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"Route not found"}}`))
		return
	}
	hh(w, r)
}

func (h *mockHandler) Calls() int64 { return atomic.LoadInt64(&h.calls) }

// newTestClient builds a Client pointed at the given test server, with
// retries disabled by default so unit tests run quickly.
func newTestClient(t *testing.T, ts *httptest.Server, opts ...Option) *Client {
	t.Helper()
	all := []Option{
		WithBaseURL(ts.URL),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	}
	all = append(all, opts...)
	c, err := NewClient(all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

// =============================================================================
// Construction tests
// =============================================================================

func TestNewClient_RequiresBaseURL(t *testing.T) {
	_, err := NewClient()
	if err == nil {
		t.Fatal("expected error for missing base url")
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("expected ErrConfig, got %v", err)
	}
}

func TestNewClient_AcceptsOptions(t *testing.T) {
	c, err := NewClient(
		WithBaseURL("http://example.test"),
		WithUserAgent("custom/1.0"),
		WithAPIKey("t_abc", "key_xyz"),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 5}),
		WithSingleflight(100*time.Millisecond),
		WithQueryCache(64, time.Minute),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.cfg.userAgent != "custom/1.0" {
		t.Errorf("user agent: %q", c.cfg.userAgent)
	}
	if c.cfg.tenantID != "t_abc" || c.cfg.apiKey != "key_xyz" {
		t.Errorf("auth not set")
	}
	if c.cache == nil {
		t.Errorf("cache nil despite enabled")
	}
	if c.sf == nil {
		t.Errorf("singleflight nil despite enabled")
	}
}

// =============================================================================
// Health
// =============================================================================

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, HealthResponse{Status: "healthy"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if got.Status != "healthy" {
		t.Errorf("status: %q", got.Status)
	}
}

// =============================================================================
// Tenants
// =============================================================================

func TestCreateTenant_SendsAuthHeaders(t *testing.T) {
	var gotTenantHeader, gotKeyHeader, gotUA string
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			gotTenantHeader = r.Header.Get(authHeaderTenant)
			gotKeyHeader = r.Header.Get(authHeaderAPIKey)
			gotUA = r.Header.Get("User-Agent")
			writeJSON(t, w, 201, Tenant{ID: "t_new", Name: "x"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithAPIKey("t_admin", "K"))
	if _, err := c.CreateTenant(context.Background(), CreateTenantRequest{Name: "x"}); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if gotTenantHeader != "t_admin" {
		t.Errorf("tenant hdr: %q", gotTenantHeader)
	}
	if gotKeyHeader != "K" {
		t.Errorf("api key hdr: %q", gotKeyHeader)
	}
	if !strings.HasPrefix(gotUA, "graphann-go/") {
		t.Errorf("user agent: %q", gotUA)
	}
}

func TestListTenants(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, ListTenantsResponse{
				Tenants: []Tenant{{ID: "t_1", Name: "one"}, {ID: "t_2", Name: "two"}},
				Total:   2,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	r, err := c.ListTenants(context.Background())
	if err != nil {
		t.Fatalf("ListTenants: %v", err)
	}
	if r.Total != 2 || len(r.Tenants) != 2 {
		t.Errorf("got %+v", r)
	}
}

// =============================================================================
// Errors
// =============================================================================

func TestError_NotFound_WrapsSentinel(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants/t_missing": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 404, map[string]any{
				"error": map[string]any{"code": "not_found", "message": "Tenant not found"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.GetTenant(context.Background(), "t_missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("not ErrNotFound: %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("not APIError: %v", err)
	}
	if apiErr.Status != 404 || apiErr.Code != "not_found" {
		t.Errorf("apiErr: %+v", apiErr)
	}
}

func TestError_RateLimited_HasRetryAfterSeconds(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "7")
			writeJSON(t, w, 429, map[string]any{
				"error": map[string]any{"code": "rate_limited", "message": "slow down"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("not ErrRateLimited: %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("not APIError: %v", err)
	}
	if apiErr.RetryAfter != 7*time.Second {
		t.Errorf("retry-after: %v", apiErr.RetryAfter)
	}
}

func TestError_RateLimited_HTTPDate(t *testing.T) {
	target := time.Now().Add(3 * time.Second).UTC()
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", target.Format(http.TimeFormat))
			writeJSON(t, w, 503, map[string]any{
				"error": map[string]any{"code": "service_unavailable", "message": "retry"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("not APIError: %v", err)
	}
	if apiErr.RetryAfter <= 0 || apiErr.RetryAfter > 5*time.Second {
		t.Errorf("retry-after: %v", apiErr.RetryAfter)
	}
}

func TestError_Validation_DistinguishedFromBadRequest(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 400, map[string]any{
				"error": map[string]any{"code": "validation_error", "message": "bad name"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.CreateTenant(context.Background(), CreateTenantRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("not ErrValidation: %v", err)
	}
	if errors.Is(err, ErrBadRequest) {
		t.Errorf("validation should not match ErrBadRequest")
	}
}

// =============================================================================
// Retries + backoff
// =============================================================================

func TestRetry_429_ThenSuccess(t *testing.T) {
	var n int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt64(&n, 1) == 1 {
				w.Header().Set("Retry-After", "0")
				writeJSON(t, w, 429, map[string]any{
					"error": map[string]any{"code": "rate_limited", "message": "slow"},
				})
				return
			}
			writeJSON(t, w, 200, HealthResponse{Status: "healthy"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithRetryPolicy(RetryPolicy{
		MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 10 * time.Millisecond,
	}))
	// Health uses noRetry; use ListTenants instead which retries.
	srv2 := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt64(&n, 1) == 1 {
				w.Header().Set("Retry-After", "0")
				writeJSON(t, w, 429, map[string]any{
					"error": map[string]any{"code": "rate_limited", "message": "slow"},
				})
				return
			}
			writeJSON(t, w, 200, ListTenantsResponse{Tenants: []Tenant{}, Total: 0})
		},
	}))
	defer srv2.Close()
	atomic.StoreInt64(&n, 0)
	c2 := newTestClient(t, srv2, WithRetryPolicy(RetryPolicy{
		MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 10 * time.Millisecond,
	}))
	if _, err := c2.ListTenants(context.Background()); err != nil {
		t.Fatalf("ListTenants: %v", err)
	}
	if got := atomic.LoadInt64(&n); got != 2 {
		t.Errorf("calls: %d, want 2", got)
	}
	_ = c
}

func TestRetry_NotApplied_For4xxBadRequest(t *testing.T) {
	var n int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
			writeJSON(t, w, 400, map[string]any{
				"error": map[string]any{"code": "bad_request", "message": "bad"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithRetryPolicy(RetryPolicy{
		MaxAttempts: 3, InitialBackoff: time.Millisecond,
	}))
	_, err := c.CreateTenant(context.Background(), CreateTenantRequest{Name: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt64(&n); got != 1 {
		t.Errorf("attempts: %d, want 1", got)
	}
}

// =============================================================================
// Search + cache + singleflight
// =============================================================================

func TestSearch_PassesQueryAndK(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			writeJSON(t, w, 200, SearchResponse{
				Results: []SearchResult{{ID: "c1", Text: "hello", Score: 0.91}},
				Total:   1,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	res, err := c.Search(context.Background(), "t", "i", SearchRequest{Query: "hello", K: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Total != 1 || res.Results[0].Score != 0.91 {
		t.Errorf("got %+v", res)
	}
	if !strings.Contains(string(gotBody), `"query":"hello"`) {
		t.Errorf("body: %s", gotBody)
	}
}

func TestSearch_QueryCache_HitsServerOnce(t *testing.T) {
	var n int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
			writeJSON(t, w, 200, SearchResponse{Total: 0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithQueryCache(10, 30*time.Second))
	for i := 0; i < 3; i++ {
		if _, err := c.Search(context.Background(), "t", "i", SearchRequest{Query: "q", K: 5}); err != nil {
			t.Fatalf("Search %d: %v", i, err)
		}
	}
	if got := atomic.LoadInt64(&n); got != 1 {
		t.Errorf("server calls: %d, want 1", got)
	}
}

func TestSearch_QueryCache_TTLExpiry(t *testing.T) {
	var n int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
			writeJSON(t, w, 200, SearchResponse{Total: 0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithQueryCache(10, 5*time.Millisecond))
	if _, err := c.Search(context.Background(), "t", "i", SearchRequest{Query: "q"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := c.Search(context.Background(), "t", "i", SearchRequest{Query: "q"}); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt64(&n); got != 2 {
		t.Errorf("server calls: %d, want 2", got)
	}
}

func TestSearch_Singleflight_CoalescesConcurrent(t *testing.T) {
	var n int64
	gate := make(chan struct{})
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
			<-gate
			writeJSON(t, w, 200, SearchResponse{Total: 1})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithSingleflight(50*time.Millisecond))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Search(context.Background(), "t", "i", SearchRequest{Query: "q"})
		}()
	}
	// Allow the goroutines to coalesce, then unblock.
	time.Sleep(20 * time.Millisecond)
	close(gate)
	wg.Wait()
	if got := atomic.LoadInt64(&n); got != 1 {
		t.Errorf("server calls: %d, want 1", got)
	}
}

// =============================================================================
// Documents
// =============================================================================

func TestAddDocuments_Roundtrip(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/documents": func(w http.ResponseWriter, r *http.Request) {
			var req AddDocumentsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(t, w, 400, map[string]any{"error": map[string]any{"code": "bad_request", "message": err.Error()}})
				return
			}
			ids := make([]string, len(req.Documents))
			for i := range req.Documents {
				ids[i] = fmt.Sprintf("chunk-%d", i)
			}
			writeJSON(t, w, 201, AddDocumentsResponse{
				Added:    len(req.Documents),
				IndexID:  "i",
				ChunkIDs: ids,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	res, err := c.AddDocuments(context.Background(), "t", "i", AddDocumentsRequest{
		Documents: []Document{{ID: "doc1", Text: "abc"}, {ID: "doc2", Text: "def"}},
	})
	if err != nil {
		t.Fatalf("AddDocuments: %v", err)
	}
	if res.Added != 2 || len(res.ChunkIDs) != 2 {
		t.Errorf("got %+v", res)
	}
}

// =============================================================================
// Pagination
// =============================================================================

func TestListJobs_Pagination(t *testing.T) {
	pages := [][]Job{
		{{JobID: "job_a", Kind: "reembed", Status: JobStatusRunning}},
		{{JobID: "job_b", Kind: "reembed", Status: JobStatusCompleted}},
	}
	cursors := []string{"c1", ""}
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			cursor := r.URL.Query().Get("cursor")
			page := 0
			if cursor == "c1" {
				page = 1
			}
			writeJSON(t, w, 200, ListJobsResponse{
				Jobs:       pages[page],
				NextCursor: cursors[page],
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	it := c.ListJobs(context.Background(), ListJobsRequest{})
	all, err := it.All(context.Background())
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 2 || all[0].JobID != "job_a" || all[1].JobID != "job_b" {
		t.Errorf("got %+v", all)
	}
}

// =============================================================================
// Cluster
// =============================================================================

func TestGetClusterHealth(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/cluster/health": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, ClusterHealthResponse{
				Status: "ok", ClusterSize: 3, AliveNodes: 3, RaftHasLeader: true,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.GetClusterHealth(context.Background())
	if err != nil {
		t.Fatalf("ClusterHealth: %v", err)
	}
	if got.Status != "ok" || got.AliveNodes != 3 {
		t.Errorf("got %+v", got)
	}
}

// =============================================================================
// Settings
// =============================================================================

func TestUpdateLLMSettings_UnwrapsEnvelope(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"PATCH /v1/orgs/o/llm-settings": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, UpdateLLMSettingsResponse{
				Message: "ok",
				OrgID:   "o",
				Settings: LLMSettings{
					Provider: "openai", Model: "gpt-4o-mini", APIKey: "***1234",
				},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.UpdateLLMSettings(context.Background(), "o", LLMSettings{Provider: "openai", Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("UpdateLLMSettings: %v", err)
	}
	if got.Provider != "openai" || got.Model != "gpt-4o-mini" {
		t.Errorf("got %+v", got)
	}
}

func TestGetLLMSettings_HitsCanonicalPath(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/orgs/o/llm-settings": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, LLMSettings{Provider: "openai", Model: "gpt-4o-mini"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.GetLLMSettings(context.Background(), "o")
	if err != nil {
		t.Fatalf("GetLLMSettings: %v", err)
	}
	if got.Provider != "openai" || got.Model != "gpt-4o-mini" {
		t.Errorf("got %+v", got)
	}
}

func TestDeleteLLMSettings_HitsCanonicalPath(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"DELETE /v1/orgs/o/llm-settings": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, DeleteLLMSettingsResponse{
				Message: "reset",
				OrgID:   "o",
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.DeleteLLMSettings(context.Background(), "o")
	if err != nil {
		t.Fatalf("DeleteLLMSettings: %v", err)
	}
	if got.OrgID != "o" || got.Message != "reset" {
		t.Errorf("got %+v", got)
	}
}

// =============================================================================
// UpsertResource
// =============================================================================

func TestUpsertResource_Create(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"PUT /v1/tenants/t/indexes/i/resources/res_1": func(w http.ResponseWriter, r *http.Request) {
			var req UpsertResourceRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(t, w, 400, map[string]any{"error": map[string]any{"code": "bad_request", "message": err.Error()}})
				return
			}
			writeJSON(t, w, 200, UpsertResourceResponse{
				ResourceID:       "res_1",
				ChunksAdded:      3,
				ChunksTombstoned: 0,
				Operation:        "create",
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.UpsertResource(context.Background(), "t", "i", "res_1", UpsertResourceRequest{
		Text: "hello world",
	})
	if err != nil {
		t.Fatalf("UpsertResource: %v", err)
	}
	if got.ResourceID != "res_1" || got.Operation != "create" || got.ChunksAdded != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestUpsertResource_Update(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"PUT /v1/tenants/t/indexes/i/resources/res_1": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, UpsertResourceResponse{
				ResourceID:       "res_1",
				ChunksAdded:      2,
				ChunksTombstoned: 3,
				Operation:        "update",
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.UpsertResource(context.Background(), "t", "i", "res_1", UpsertResourceRequest{
		Text:     "updated text",
		Metadata: map[string]string{"source": "api"},
	})
	if err != nil {
		t.Fatalf("UpsertResource: %v", err)
	}
	if got.Operation != "update" || got.ChunksTombstoned != 3 {
		t.Errorf("got %+v", got)
	}
}

// =============================================================================
// CompactIndex 409 handling
// =============================================================================

func TestCompactIndex_409_IsErrCompactInProgress(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/compact": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 409, map[string]any{
				"error": map[string]any{"code": "compact_in_progress", "message": "Compaction already running"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.CompactIndex(context.Background(), "t", "i")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCompactInProgress) {
		t.Errorf("want ErrCompactInProgress, got %v", err)
	}
}

// =============================================================================
// CreateIndex with compression + approximate fields
// =============================================================================

func TestCreateIndex_CompressionAndApproximate(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes": func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			compression := "pq"
			approx := true
			writeJSON(t, w, 201, Index{
				ID:          "i_new",
				TenantID:    "t",
				Name:        "pq-index",
				Status:      IndexStatusEmpty,
				Compression: "pq",
				Approximate: true,
			})
			_ = compression
			_ = approx
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	comp := "pq"
	approx := true
	got, err := c.CreateIndex(context.Background(), "t", CreateIndexRequest{
		Name:        "pq-index",
		Compression: &comp,
		Approximate: &approx,
	})
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if got.Compression != "pq" || !got.Approximate {
		t.Errorf("got %+v", got)
	}
	if !strings.Contains(string(gotBody), `"compression":"pq"`) {
		t.Errorf("body missing compression: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), `"approximate":true`) {
		t.Errorf("body missing approximate: %s", gotBody)
	}
}

// =============================================================================
// SearchFilter.Equals
// =============================================================================

func TestSearch_FilterEquals(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			writeJSON(t, w, 200, SearchResponse{Total: 0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.Search(context.Background(), "t", "i", SearchRequest{
		Query: "q",
		Filter: SearchFilter{
			Equals: map[string]string{"env": "prod", "team": "eng"},
		},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !strings.Contains(string(gotBody), `"equals"`) {
		t.Errorf("body missing equals: %s", gotBody)
	}
}

// =============================================================================
// CleanupOrphans (admin-only)
// =============================================================================

func TestCleanupOrphans_HappyPath(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/admin/cleanup-orphans": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, CleanupOrphansResponse{
				Removed:    []string{"/data/tenants/t/indexes/i/seg.old"},
				FreedBytes: 4096,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("CleanupOrphans: %v", err)
	}
	if got.FreedBytes != 4096 || len(got.Removed) != 1 {
		t.Errorf("got %+v", got)
	}
}

// =============================================================================
// Metrics hook
// =============================================================================

func TestMetricsHook_FiresOnEachRequest(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /health": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, HealthResponse{Status: "healthy"})
		},
	}))
	defer srv.Close()

	type call struct {
		method, path string
		status       int
	}
	var (
		mu    sync.Mutex
		calls []call
	)
	hook := MetricsHookFunc(func(method, path string, status int, d time.Duration, err error) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call{method, path, status})
	})
	c := newTestClient(t, srv, WithMetricsHook(hook))
	if _, err := c.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("calls: %d", len(calls))
	}
	if calls[0].method != "GET" || calls[0].path != "/health" || calls[0].status != 200 {
		t.Errorf("call: %+v", calls[0])
	}
}

// =============================================================================
// Compression on large bodies
// =============================================================================

func TestRequestBody_GzippedAboveThreshold(t *testing.T) {
	bigText := strings.Repeat("A", 100*1024) // 100 KiB
	var gotEnc string
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/documents": func(w http.ResponseWriter, r *http.Request) {
			gotEnc = r.Header.Get("Content-Encoding")
			// We don't decode here — the server normally would. Just
			// drain the body so the reader doesn't error on the client.
			_, _ = io.Copy(io.Discard, r.Body)
			writeJSON(t, w, 201, AddDocumentsResponse{Added: 1, IndexID: "i"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if _, err := c.AddDocuments(context.Background(), "t", "i", AddDocumentsRequest{
		Documents: []Document{{Text: bigText}},
	}); err != nil {
		t.Fatalf("AddDocuments: %v", err)
	}
	if gotEnc != "gzip" {
		t.Errorf("Content-Encoding: %q, want gzip", gotEnc)
	}
}

// =============================================================================
// Cache: nil-safe operations
// =============================================================================

func TestQueryCache_NilSafe(t *testing.T) {
	var c *queryCache
	if v, ok := c.Get("k"); ok || v != nil {
		t.Error("nil Get should return false/nil")
	}
	c.Set("k", &SearchResponse{})
	c.Purge()
	if c.Len() != 0 {
		t.Error("nil Len should be 0")
	}
}

func TestSFGroup_NilSafe(t *testing.T) {
	var s *sfGroup
	v, shared, err := s.Do("k", func() (any, error) { return 1, nil })
	if shared || err != nil || v.(int) != 1 {
		t.Errorf("nil Do unexpected: %v %v %v", v, shared, err)
	}
}

// =============================================================================
// Retry attempts cap
// =============================================================================

func TestRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	var n int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
			writeJSON(t, w, 503, map[string]any{
				"error": map[string]any{"code": "service_unavailable", "message": "down"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithRetryPolicy(RetryPolicy{
		MaxAttempts: 4, InitialBackoff: time.Millisecond,
	}))
	_, err := c.ListTenants(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt64(&n); got != 4 {
		t.Errorf("attempts: %d, want 4", got)
	}
}

// =============================================================================
// Network error sentinel
// =============================================================================

func TestNetworkError_WrapsErrNetwork(t *testing.T) {
	c, err := NewClient(
		WithBaseURL("http://127.0.0.1:1"), // closed
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = c.Close() }()
	_, err = c.ListTenants(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNetwork) {
		t.Errorf("not ErrNetwork: %v", err)
	}
}

// =============================================================================
// Context cancellation
// =============================================================================

func TestContext_CancellationStops(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			writeJSON(t, w, 200, ListTenantsResponse{})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := c.ListTenants(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// =============================================================================
// Error envelope without "error" key: still surfaces useful message
// =============================================================================

func TestError_BareStatus_NoEnvelope(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte("plain text fail"))
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.ListTenants(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrServer) {
		t.Errorf("not ErrServer: %v", err)
	}
}

// =============================================================================
// Pagination: all-in-one helper for ListDocuments
// =============================================================================

func TestListDocuments_Iter(t *testing.T) {
	pages := [][]ListDocumentEntry{
		{{ID: "1"}, {ID: "2"}},
		{{ID: "3"}},
	}
	cursors := []string{"c1", ""}
	var nthCall int
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants/t/indexes/i/documents": func(w http.ResponseWriter, r *http.Request) {
			cur := r.URL.Query().Get("cursor")
			page := 0
			if cur == "c1" {
				page = 1
			}
			nthCall++
			writeJSON(t, w, 200, ListDocumentsResponse{
				Documents:  pages[page],
				NextCursor: cursors[page],
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	it := c.ListDocuments(context.Background(), "t", "i", ListDocumentsRequest{Prefix: "foo:", Limit: 2})
	all, err := it.All(context.Background())
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("got %d, want 3", len(all))
	}
	if nthCall != 2 {
		t.Errorf("server calls: %d, want 2", nthCall)
	}
}

// =============================================================================
// Search + cache: distinct-key isolation
// =============================================================================

func TestSearch_CacheIsolatedByKey(t *testing.T) {
	var n int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
			writeJSON(t, w, 200, SearchResponse{Total: 0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithQueryCache(10, time.Minute))
	for i := 0; i < 3; i++ {
		_, _ = c.Search(context.Background(), "t", "i", SearchRequest{Query: fmt.Sprintf("q%d", i)})
	}
	if got := atomic.LoadInt64(&n); got != 3 {
		t.Errorf("server calls: %d, want 3", got)
	}
}

// =============================================================================
// Error formatting
// =============================================================================

func TestAPIError_Format(t *testing.T) {
	e := newAPIError(404, "not_found", "Index not found", nil, "req-1", 0)
	got := e.Error()
	if !strings.Contains(got, "404") || !strings.Contains(got, "not_found") || !strings.Contains(got, "req-1") {
		t.Errorf("error format: %q", got)
	}
}

// =============================================================================
// Body too large surfaces ErrPayloadTooLarge
// =============================================================================

func TestError_PayloadTooLarge(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/documents": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 413, map[string]any{
				"error": map[string]any{"code": "payload_too_large", "message": "Body too large"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.AddDocuments(context.Background(), "t", "i", AddDocumentsRequest{
		Documents: []Document{{Text: "x"}},
	})
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Errorf("not ErrPayloadTooLarge: %v", err)
	}
}

// =============================================================================
// strconv import sanity (avoid unused import in editor reorders)
// =============================================================================

var _ = strconv.Itoa
