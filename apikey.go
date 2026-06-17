package graphann

import (
	"context"
	"net/url"
)

// apiKeysPath returns /v1/tenants/{tenantID}/api-keys, the tenant-scoped
// admin route for API key create/list/revoke.
func apiKeysPath(tenantID string) string {
	return "/v1/tenants/" + url.PathEscape(tenantID) + "/api-keys"
}

// CreateAPIKey calls POST /v1/tenants/{tenantID}/api-keys.
//
// The plaintext key is returned in the response's Plaintext field
// exactly once. The server stores only the argon2id hash; lose the
// plaintext and the only recovery is to rotate the key.
func (c *Client) CreateAPIKey(ctx context.Context, tenantID string, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	var out CreateAPIKeyResponse
	if err := c.do(ctx, "POST", apiKeysPath(tenantID), req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// RevokeAPIKey calls DELETE /v1/tenants/{tenantID}/api-keys/{keyID}.
// Idempotent: returns ErrNotFound when the key is unknown.
func (c *Client) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
	path := apiKeysPath(tenantID) + "/" + url.PathEscape(keyID)
	return c.do(ctx, "DELETE", path, nil, nil, nil)
}

// ListAPIKeys calls GET /v1/tenants/{tenantID}/api-keys.
func (c *Client) ListAPIKeys(ctx context.Context, tenantID string) (*ListAPIKeysResponse, error) {
	var out ListAPIKeysResponse
	if err := c.do(ctx, "GET", apiKeysPath(tenantID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
