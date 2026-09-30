package graphann

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// errBody is the server's standard error envelope.
func errBody(code, msg string) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "message": msg}}
}

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return got
}

func TestGetTenantQuota(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants/t_1/quota": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{"max_storage_gb": 10, "used_storage_gb": 1.5})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.GetTenantQuota(context.Background(), "t_1")
	if err != nil {
		t.Fatalf("GetTenantQuota: %v", err)
	}
	if got.MaxStorageGB != 10 || got.UsedStorageGB != 1.5 {
		t.Errorf("got %+v", got)
	}
}

func TestGetTenantQuota_NotFound(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(nil))
	defer srv.Close()
	c := newTestClient(t, srv)
	if _, err := c.GetTenantQuota(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateTenantQuota_SendsZeroToRemoveQuota(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"PUT /v1/tenants/t_1/quota": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 200, map[string]any{"max_storage_gb": 0, "used_storage_gb": 2.0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.UpdateTenantQuota(context.Background(), "t_1", UpdateTenantQuotaRequest{MaxStorageGB: 0})
	if err != nil {
		t.Fatalf("UpdateTenantQuota: %v", err)
	}
	// 0 means "no quota" and must reach the server rather than be omitted.
	if v, ok := body["max_storage_gb"]; !ok || v != float64(0) {
		t.Errorf("request body: %+v", body)
	}
	if got.MaxStorageGB != 0 || got.UsedStorageGB != 2.0 {
		t.Errorf("got %+v", got)
	}
}

func TestUpdateTenantQuota_ForbiddenWithoutAdminToken(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"PUT /v1/tenants/t_1/quota": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 403, errBody("forbidden", "admin token required"))
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.UpdateTenantQuota(context.Background(), "t_1", UpdateTenantQuotaRequest{MaxStorageGB: 5})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestUpdateTenantQuota_PurgesQueryCache(t *testing.T) {
	var searches int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&searches, 1)
			writeJSON(t, w, 200, SearchResponse{})
		},
		"PUT /v1/tenants/t/quota": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{"max_storage_gb": 1, "used_storage_gb": 0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithQueryCache(10, time.Minute))
	ctx := context.Background()
	req := SearchRequest{Query: "q", K: 3}

	for i := 0; i < 2; i++ {
		if _, err := c.Search(ctx, "t", "i", req); err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt64(&searches); got != 1 {
		t.Fatalf("searches before update: %d, want 1", got)
	}
	if _, err := c.UpdateTenantQuota(ctx, "t", UpdateTenantQuotaRequest{MaxStorageGB: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(ctx, "t", "i", req); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt64(&searches); got != 2 {
		t.Errorf("searches after update: %d, want 2", got)
	}
}

func TestGetAPIKeyStatus(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/api-key-status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"tenants": []map[string]any{
					{"tenant_id": "t_acme", "tenant_name": "Acme", "active_keys": 3, "locked": true},
					{"tenant_id": "t_open", "tenant_name": "Open", "active_keys": 0, "locked": false},
				},
				"strict_auth":            true,
				"admin_token_configured": true,
				"metrics_public":         false,
				"tenant_key_enforcement": true,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.GetAPIKeyStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAPIKeyStatus: %v", err)
	}
	want := &APIKeyStatusResponse{
		Tenants: []APIKeyStatusTenantRow{
			{TenantID: "t_acme", TenantName: "Acme", ActiveKeys: 3, Locked: true},
			{TenantID: "t_open", TenantName: "Open"},
		},
		StrictAuth:           true,
		AdminTokenConfigured: true,
		TenantKeyEnforcement: true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetAPIKeyStatus_Forbidden(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/api-key-status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 403, errBody("forbidden", "admin only"))
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if _, err := c.GetAPIKeyStatus(context.Background()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

func TestListAllBackups_PaginatesWithFiltersOnEveryPage(t *testing.T) {
	created := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	var queries []map[string]string
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/backups": func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			queries = append(queries, map[string]string{
				"tenant_id": q.Get("tenant_id"), "index_id": q.Get("index_id"),
				"limit": q.Get("limit"), "cursor": q.Get("cursor"),
			})
			row := func(id string) AdminBackupRow {
				return AdminBackupRow{
					ID: id, TenantID: "t_1", TenantName: "One", IndexID: "i_1", IndexName: "Idx",
					CreatedAt: created, TotalSize: 1 << 33, NumChunks: 7,
				}
			}
			if q.Get("cursor") == "" {
				writeJSON(t, w, 200, AdminBackupList{Backups: []AdminBackupRow{row("b2")}, NextCursor: "cur1", Total: 2})
				return
			}
			writeJSON(t, w, 200, AdminBackupList{Backups: []AdminBackupRow{row("b1")}, Total: 2})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	all, err := c.ListAllBackups(context.Background(), ListAllBackupsRequest{TenantID: "t_1", IndexID: "i_1", Limit: 1}).All(context.Background())
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 2 || all[0].ID != "b2" || all[1].ID != "b1" {
		t.Fatalf("rows: %+v", all)
	}
	if all[0].TotalSize != 1<<33 || all[0].NumChunks != 7 || !all[0].CreatedAt.Equal(created) {
		t.Errorf("row fields: %+v", all[0])
	}
	wantQueries := []map[string]string{
		{"tenant_id": "t_1", "index_id": "i_1", "limit": "1", "cursor": ""},
		{"tenant_id": "t_1", "index_id": "i_1", "limit": "1", "cursor": "cur1"},
	}
	if !reflect.DeepEqual(queries, wantQueries) {
		t.Errorf("queries: %+v", queries)
	}
}

func TestListAllBackups_OmitsUnsetFilters(t *testing.T) {
	var rawQuery string
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/backups": func(w http.ResponseWriter, r *http.Request) {
			rawQuery = r.URL.RawQuery
			writeJSON(t, w, 200, AdminBackupList{Backups: []AdminBackupRow{}})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	all, err := c.ListAllBackups(context.Background(), ListAllBackupsRequest{}).All(context.Background())
	if err != nil || len(all) != 0 {
		t.Fatalf("got %v, %v", all, err)
	}
	if rawQuery != "" {
		t.Errorf("query: %q", rawQuery)
	}
}

func TestListAllBackups_ErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/backups": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 400, errBody("bad_request", "limit must be an integer between 1 and 1000"))
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	it := c.ListAllBackups(context.Background(), ListAllBackupsRequest{Limit: 5000})
	if it.Next(context.Background()) {
		t.Fatal("Next returned true on error")
	}
	if !errors.Is(it.Err(), ErrBadRequest) {
		t.Fatalf("want ErrBadRequest, got %v", it.Err())
	}
}

func TestGetBackupStatus_Disabled(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/backups/status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{"enabled": false, "keep": 0, "runs_ok": 0, "runs_failed": 0})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.GetBackupStatus(context.Background())
	if err != nil {
		t.Fatalf("GetBackupStatus: %v", err)
	}
	if got.Enabled || got.NextRun != nil || got.LastRun != nil || got.LastSuccess != nil || got.LastRunOK != nil {
		t.Errorf("disabled status carries data: %+v", got)
	}
}

func TestGetBackupStatus_Enabled(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/backups/status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"enabled": true, "schedule": "0 2 * * *", "keep": 7, "interval_seconds": 86400,
				"next_run": "2026-08-02T02:00:00Z", "last_run": "2026-08-01T02:00:00Z",
				"last_success": "2026-08-01T02:00:00Z", "last_run_ok": true,
				"runs_ok": 12, "runs_failed": 1,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.GetBackupStatus(context.Background())
	if err != nil {
		t.Fatalf("GetBackupStatus: %v", err)
	}
	if !got.Enabled || got.Keep != 7 || got.RunsOK != 12 || got.RunsFailed != 1 {
		t.Errorf("counters: %+v", got)
	}
	if got.IntervalSeconds != 86400 {
		t.Errorf("interval: %d", got.IntervalSeconds)
	}
	if got.LastRunOK == nil || !*got.LastRunOK {
		t.Errorf("last_run_ok: %+v", got.LastRunOK)
	}
	if got.NextRun == nil || !got.NextRun.Equal(time.Date(2026, 8, 2, 2, 0, 0, 0, time.UTC)) {
		t.Errorf("next_run: %+v", got.NextRun)
	}
}

func TestRestoreBackup_SendsNameAndMapsErrors(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t_1/backups/t_1/i_1/b1/restore": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 200, RestoreBackupResponse{})
		},
		"POST /v1/tenants/t_1/backups/dup/restore": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 409, errBody("conflict", "exists"))
		},
		"POST /v1/tenants/t_1/backups/old/restore": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 422, errBody("unprocessable_entity", "no settings"))
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	ctx := context.Background()

	name := "Restored copy"
	if _, err := c.RestoreBackup(ctx, "t_1", "t_1/i_1/b1", RestoreBackupRequest{DestIndex: "i_new", Name: &name}); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}
	if body["dest_index"] != "i_new" || body["name"] != "Restored copy" {
		t.Errorf("body: %+v", body)
	}

	if _, err := c.RestoreBackup(ctx, "t_1", "dup", RestoreBackupRequest{DestIndex: "i_1"}); !errors.Is(err, ErrConflict) {
		t.Errorf("409: want ErrConflict, got %v", err)
	}
	_, err := c.RestoreBackup(ctx, "t_1", "old", RestoreBackupRequest{DestIndex: "i_2"})
	if !errors.Is(err, ErrUnprocessable) {
		t.Errorf("422: want ErrUnprocessable, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != ErrorCodeUnprocessable {
		t.Errorf("422 code: %+v", apiErr)
	}
}

func TestRestoreBackup_OmitsNameWhenUnset(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/backups/b/restore": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 200, RestoreBackupResponse{})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if _, err := c.RestoreBackup(context.Background(), "t", "b", RestoreBackupRequest{DestIndex: "i_x"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["name"]; ok {
		t.Errorf("name sent when unset: %+v", body)
	}
}

func TestInsufficientStorage_MapsToSentinelAndIsNotRetried(t *testing.T) {
	var calls int64
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/documents": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&calls, 1)
			writeJSON(t, w, 507, errBody(ErrorCodeInsufficientStorage, "quota reached"))
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv, WithRetryPolicy(RetryPolicy{MaxAttempts: 3}))

	_, err := c.AddDocuments(context.Background(), "t", "i", AddDocumentsRequest{Documents: []Document{{Text: "x"}}})
	if !errors.Is(err, ErrInsufficientStorage) {
		t.Fatalf("want ErrInsufficientStorage, got %v", err)
	}
	if errors.Is(err, ErrServer) {
		t.Error("507 must not match ErrServer, which is retried")
	}
	if got := atomic.LoadInt64(&calls); got != 1 {
		t.Errorf("calls: %d, want 1", got)
	}
}

func TestReady_ShuttingDownReasonSurfaced(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /ready": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 503, map[string]string{"status": "not ready", "reason": "shutting down"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.Ready(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}
	if apiErr.Status != 503 || apiErr.Message != "shutting down" {
		t.Errorf("got %+v", apiErr)
	}
}

func TestCreateIndex_SendsChunkOverrides(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 201, map[string]any{
				"id": "i_1", "tenant_id": "t", "name": "n", "status": "pending",
				"chunk_size": 1024, "chunk_overlap": 100,
				"vectors_external": true, "embedding_backend": "openai", "embedding_model": "m",
				"embedding_dimension": 1536, "created_by": "u_1", "metadata": map[string]string{"k": "v"},
				"last_compacted_at": "2026-08-01T00:00:00Z",
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.CreateIndex(context.Background(), "t", CreateIndexRequest{Name: "n", ChunkSize: 1024, ChunkOverlap: 100})
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if body["chunk_size"] != float64(1024) || body["chunk_overlap"] != float64(100) {
		t.Errorf("body: %+v", body)
	}
	if got.ChunkSize != 1024 || got.ChunkOverlap != 100 || got.Status != IndexStatusPending {
		t.Errorf("index: %+v", got)
	}
	if !got.VectorsExternal || got.EmbeddingBackend != "openai" || got.EmbeddingModel != "m" ||
		got.EmbeddingDimension != 1536 || got.CreatedBy != "u_1" || got.Metadata["k"] != "v" ||
		got.LastCompactedAt.IsZero() {
		t.Errorf("index extras: %+v", got)
	}
}

func TestCreateIndex_OmitsChunkOverridesByDefault(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 201, Index{ID: "i_1"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if _, err := c.CreateIndex(context.Background(), "t", CreateIndexRequest{Name: "n"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"chunk_size", "chunk_overlap"} {
		if _, ok := body[k]; ok {
			t.Errorf("%s sent by default: %+v", k, body)
		}
	}
}

func TestAddDocuments_ParsesWarnings(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/documents": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"added": 1, "index_id": "i", "chunk_ids": []string{"c1"},
				"warnings": []string{"document d1 exceeds the embedder token budget"},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.AddDocuments(context.Background(), "t", "i", AddDocumentsRequest{Documents: []Document{{ID: "d1", Text: "x"}}})
	if err != nil {
		t.Fatalf("AddDocuments: %v", err)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "document d1 exceeds the embedder token budget" {
		t.Errorf("warnings: %+v", got.Warnings)
	}
}

func TestSearch_SendsGroupBy(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/indexes/i/search": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 200, SearchResponse{})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.Search(context.Background(), "t", "i", SearchRequest{Query: "q", GroupBy: SearchGroupByDocument, MaxPerDoc: 2}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if body["group_by"] != "document" || body["max_per_doc"] != float64(2) {
		t.Errorf("body: %+v", body)
	}

	if _, err := c.Search(context.Background(), "t", "i", SearchRequest{Query: "plain"}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, k := range []string{"group_by", "max_per_doc"} {
		if _, ok := body[k]; ok {
			t.Errorf("%s sent by default: %+v", k, body)
		}
	}
}

func TestCreateAPIKey_SendsRole(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/tenants/t/api-keys": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 201, map[string]any{"id": "k", "name": "n", "user_id": "u_new", "plaintext": "p", "created_at": "x"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if _, err := c.CreateAPIKey(context.Background(), "t", CreateAPIKeyRequest{UserID: "u_new", Name: "n", Role: APIKeyRoleAdmin}); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if body["role"] != "admin" {
		t.Errorf("body: %+v", body)
	}

	if _, err := c.CreateAPIKey(context.Background(), "t", CreateAPIKeyRequest{UserID: "u_1", Name: "n"}); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if _, ok := body["role"]; ok {
		t.Errorf("role sent when unset: %+v", body)
	}
}

func TestGetIndexStatus_EmbedSpaceFields(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants/t/indexes/i/status": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"index_id": "i", "status": "ready", "error": "",
				"embed_space_state": "empty", "embed_fingerprint": "", "server_embed_fingerprint": "fp_s",
				"embed_space_policy": "warn", "embed_space_detail": "no vectors",
				"embed_space_checked_at": "2026-08-10T12:00:00Z",
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.GetIndexStatus(context.Background(), "t", "i")
	if err != nil {
		t.Fatalf("GetIndexStatus: %v", err)
	}
	if got.EmbedSpaceState != EmbedSpaceEmpty || got.ServerEmbedFingerprint != "fp_s" ||
		got.EmbedSpacePolicy != "warn" || got.EmbedSpaceDetail != "no vectors" || got.EmbedSpaceCheckedAt == nil {
		t.Errorf("got %+v", got)
	}
}

func TestGetEmbedSpaceAdmin_EmptyStateAndNumChunks(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/admin/embed-space": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"counts": map[string]int{
					"verified": 1, "mismatch": 0, "unknown": 0, "external_unverified": 0,
					"external_agree": 0, "external_diverged": 0, "empty": 1,
				},
				"indexes": []map[string]any{
					{"index_id": "i_full", "tenant_id": "t", "state": "verified", "num_chunks": 15000},
					{"index_id": "i_new", "tenant_id": "t", "state": "empty", "num_chunks": 0},
				},
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	got, err := c.GetEmbedSpaceAdmin(context.Background())
	if err != nil {
		t.Fatalf("GetEmbedSpaceAdmin: %v", err)
	}
	if got.Counts == nil || len(*got.Counts) != 7 || (*got.Counts)["empty"] != 1 {
		t.Errorf("counts: %+v", got.Counts)
	}
	rows := *got.Indexes
	if *rows[0].NumChunks != 15000 || *rows[1].NumChunks != 0 || string(*rows[1].State) != string(EmbedSpaceEmpty) {
		t.Errorf("rows: %+v", rows)
	}
	if !rows[1].State.Valid() {
		t.Error("empty state not a known enum member")
	}
}

func TestLiveStats_ParsesSizeFields(t *testing.T) {
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"GET /v1/tenants/t/indexes/i/live-stats": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"index_id": "i", "is_live": true, "frozen_chunks": 3, "base_bytes": 1 << 34,
				"delta_bytes": 10, "embedding_sidecar_bytes": 20, "raw_bytes": 30, "index_size_bytes": 40,
				"compression_ratio": 4.5, "base_deleted_chunks": 2, "base_deleted_fraction": 0.25,
				"rebuild_recommended": true,
			})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	got, err := c.GetLiveStats(context.Background(), "t", "i")
	if err != nil {
		t.Fatalf("GetLiveStats: %v", err)
	}
	if got.FrozenChunks != 3 || got.BaseBytes != 1<<34 || got.DeltaBytes != 10 || got.EmbeddingSidecarBytes != 20 ||
		got.RawBytes != 30 || got.IndexSizeBytes != 40 || got.CompressionRatio != 4.5 ||
		got.BaseDeletedChunks != 2 || got.BaseDeletedFraction != 0.25 || !got.RebuildRecommended {
		t.Errorf("got %+v", got)
	}
}

func TestUpsertResource_SendsMetadataFields(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"PUT /v1/tenants/t/indexes/i/resources/r1": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 200, UpsertResourceResponse{ResourceID: "r1", Operation: "create"})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.UpsertResource(context.Background(), "t", "i", "r1", UpsertResourceRequest{
		Text: "body", ExternalID: "ext", RepoID: "repo", FilePath: "a.go", CommitSHA: "abc",
		SourceType: "code", OwnerUserID: "u_1", Title: "T", URL: "https://x",
	})
	if err != nil {
		t.Fatalf("UpsertResource: %v", err)
	}
	want := map[string]any{
		"text": "body", "external_id": "ext", "repo_id": "repo", "file_path": "a.go", "commit_sha": "abc",
		"source_type": "code", "owner_user_id": "u_1", "title": "T", "url": "https://x",
	}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("body: %+v", body)
	}
}

func TestMultiSearch_SendsMaxResults(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(newMockHandler(map[string]http.HandlerFunc{
		"POST /v1/orgs/o/users/u/search": func(w http.ResponseWriter, r *http.Request) {
			body = decodeBody(t, r)
			writeJSON(t, w, 200, MultiSearchResponse{})
		},
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if _, err := c.MultiSearch(context.Background(), "o", "u", MultiSearchRequest{Query: "q", MaxResults: 9}); err != nil {
		t.Fatalf("MultiSearch: %v", err)
	}
	if body["max_results"] != float64(9) {
		t.Errorf("body: %+v", body)
	}
}
