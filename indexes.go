package graphann

import (
	"context"
	"net/url"
)

// indexBasePath returns /v1/tenants/{tid}/indexes/{iid}.
func indexBasePath(tenantID, indexID string) string {
	return "/v1/tenants/" + url.PathEscape(tenantID) + "/indexes/" + url.PathEscape(indexID)
}

// indexCollectionPath returns /v1/tenants/{tid}/indexes.
func indexCollectionPath(tenantID string) string {
	return "/v1/tenants/" + url.PathEscape(tenantID) + "/indexes"
}

// CreateIndex calls POST /v1/tenants/{tenantID}/indexes.
func (c *Client) CreateIndex(ctx context.Context, tenantID string, req CreateIndexRequest) (*Index, error) {
	var out Index
	if err := c.do(ctx, "POST", indexCollectionPath(tenantID), req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetIndex calls GET /v1/tenants/{tenantID}/indexes/{indexID}.
func (c *Client) GetIndex(ctx context.Context, tenantID, indexID string) (*Index, error) {
	var out Index
	if err := c.do(ctx, "GET", indexBasePath(tenantID, indexID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListIndexes calls GET /v1/tenants/{tenantID}/indexes.
func (c *Client) ListIndexes(ctx context.Context, tenantID string) (*ListIndexesResponse, error) {
	var out ListIndexesResponse
	if err := c.do(ctx, "GET", indexCollectionPath(tenantID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateIndex calls PATCH /v1/tenants/{tenantID}/indexes/{indexID}.
//
// Accepts optional Compression and Approximate fields. As of v0.3.0 this
// endpoint is fully functional server-side.
func (c *Client) UpdateIndex(ctx context.Context, tenantID, indexID string, req UpdateIndexRequest) (*Index, error) {
	var out Index
	if err := c.do(ctx, "PATCH", indexBasePath(tenantID, indexID), req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteIndex calls DELETE /v1/tenants/{tenantID}/indexes/{indexID}.
func (c *Client) DeleteIndex(ctx context.Context, tenantID, indexID string) error {
	return c.do(ctx, "DELETE", indexBasePath(tenantID, indexID), nil, nil, nil)
}

// GetIndexStatus calls GET /v1/tenants/{tenantID}/indexes/{indexID}/status.
func (c *Client) GetIndexStatus(ctx context.Context, tenantID, indexID string) (*IndexStatusResponse, error) {
	var out IndexStatusResponse
	if err := c.do(ctx, "GET", indexBasePath(tenantID, indexID)+"/status", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetLiveStats calls GET /v1/tenants/{tenantID}/indexes/{indexID}/live-stats.
func (c *Client) GetLiveStats(ctx context.Context, tenantID, indexID string) (*LiveStatsResponse, error) {
	var out LiveStatsResponse
	if err := c.do(ctx, "GET", indexBasePath(tenantID, indexID)+"/live-stats", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// CompactIndex calls POST /v1/tenants/{tenantID}/indexes/{indexID}/compact.
//
// Returns ErrCompactInProgress (which wraps ErrConflict) when the server
// responds 409 because a compaction is already running. This is retryable —
// callers should back off and try again.
func (c *Client) CompactIndex(ctx context.Context, tenantID, indexID string) (*CompactResponse, error) {
	var out CompactResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/compact", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ClearIndex calls POST /v1/tenants/{tenantID}/indexes/{indexID}/clear.
func (c *Client) ClearIndex(ctx context.Context, tenantID, indexID string) (*ClearIndexResponse, error) {
	var out ClearIndexResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/clear", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpsertResource calls PUT /v1/tenants/{tenantID}/indexes/{indexID}/resources/{resID}.
//
// Atomically replaces all chunks for the given resource ID with the
// re-chunked content of req.Text. If the resource does not exist it is
// created (operation="create"); if it does, old chunks are tombstoned and
// new ones are written (operation="update").
func (c *Client) UpsertResource(ctx context.Context, tenantID, indexID, resourceID string, req UpsertResourceRequest) (*UpsertResourceResponse, error) {
	path := indexBasePath(tenantID, indexID) + "/resources/" + resourceID
	var out UpsertResourceResponse
	if err := c.do(ctx, "PUT", path, req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// CleanupOrphans calls POST /v1/admin/cleanup-orphans. Admin-only on the
// server; sweeps stale compaction artifacts (*.old, *.compact, *.backup,
// *.failed) from every tenant's data directory using a 1h minimum-age
// guard so in-flight compactions are not disturbed.
func (c *Client) CleanupOrphans(ctx context.Context) (*CleanupOrphansResponse, error) {
	var out CleanupOrphansResponse
	if err := c.do(ctx, "POST", "/v1/admin/cleanup-orphans", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// RunIndexGC calls POST /v1/tenants/{tid}/indexes/{iid}/gc. Sweeps every
// document whose sidecar ExpiresAt has passed and returns the count
// reclaimed. Idempotent — calling twice in a row returns 0 the second time.
func (c *Client) RunIndexGC(ctx context.Context, tenantID, indexID string) (*GCResponse, error) {
	var out GCResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/gc", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// RunAdminGC calls POST /v1/admin/gc. Sweeps expired documents across every
// loaded index in one shot. Admin-only.
func (c *Client) RunAdminGC(ctx context.Context) (*GCResponse, error) {
	var out GCResponse
	if err := c.do(ctx, "POST", "/v1/admin/gc", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
