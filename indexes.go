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
// Note: at v0.1 of the server this endpoint is documented as not
// persisted (returns 501). The client surface is provided for forward
// compatibility — callers should be ready for ErrNotImplemented.
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
