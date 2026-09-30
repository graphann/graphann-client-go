package graphann

import (
	"context"
	"net/url"
)

// CreateTenant calls POST /v1/tenants. When req.ID is set, the call is
// idempotent: an existing tenant with that ID is returned.
func (c *Client) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
	var out Tenant
	if err := c.do(ctx, "POST", "/v1/tenants", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTenant calls GET /v1/tenants/{tenantID}.
func (c *Client) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	var out Tenant
	path := "/v1/tenants/" + url.PathEscape(tenantID)
	if err := c.do(ctx, "GET", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTenants calls GET /v1/tenants.
func (c *Client) ListTenants(ctx context.Context) (*ListTenantsResponse, error) {
	var out ListTenantsResponse
	if err := c.do(ctx, "GET", "/v1/tenants", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteTenant calls DELETE /v1/tenants/{tenantID}. Idempotent at the
// route level: returns ErrNotFound if the tenant does not exist.
func (c *Client) DeleteTenant(ctx context.Context, tenantID string) (*DeleteTenantResponse, error) {
	var out DeleteTenantResponse
	path := "/v1/tenants/" + url.PathEscape(tenantID)
	if err := c.do(ctx, "DELETE", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// tenantQuotaPath returns /v1/tenants/{tenantID}/quota.
func tenantQuotaPath(tenantID string) string {
	return "/v1/tenants/" + url.PathEscape(tenantID) + "/quota"
}

// GetTenantQuota calls GET /v1/tenants/{tenantID}/quota. MaxStorageGB 0
// means the tenant has no quota.
func (c *Client) GetTenantQuota(ctx context.Context, tenantID string) (*TenantQuotaResponse, error) {
	var out TenantQuotaResponse
	if err := c.do(ctx, "GET", tenantQuotaPath(tenantID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateTenantQuota calls PUT /v1/tenants/{tenantID}/quota. A
// MaxStorageGB of 0 removes the quota; ingest over the quota fails with
// ErrInsufficientStorage. It needs the node admin token when the node
// requires one, otherwise it returns ErrForbidden.
func (c *Client) UpdateTenantQuota(ctx context.Context, tenantID string, req UpdateTenantQuotaRequest) (*TenantQuotaResponse, error) {
	var out TenantQuotaResponse
	if err := c.do(ctx, "PUT", tenantQuotaPath(tenantID), req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
