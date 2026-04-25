package graphann

import (
	"context"
	"net/url"
)

// Search calls POST /v1/tenants/{tid}/indexes/{iid}/search. Either
// req.Query or req.Vector must be set. Honours the Client's query cache
// and singleflight settings when configured.
func (c *Client) Search(ctx context.Context, tenantID, indexID string, req SearchRequest) (*SearchResponse, error) {
	return c.runSearch(ctx, "/search", tenantID, indexID, req)
}

// SearchText calls POST /v1/tenants/{tid}/indexes/{iid}/search/text.
// req.Query is required; req.Vector is ignored.
func (c *Client) SearchText(ctx context.Context, tenantID, indexID string, req SearchRequest) (*SearchResponse, error) {
	req.Vector = nil
	return c.runSearch(ctx, "/search/text", tenantID, indexID, req)
}

// SearchVector calls POST /v1/tenants/{tid}/indexes/{iid}/search/vector.
// req.Vector is required; req.Query is ignored.
func (c *Client) SearchVector(ctx context.Context, tenantID, indexID string, req SearchRequest) (*SearchResponse, error) {
	req.Query = ""
	return c.runSearch(ctx, "/search/vector", tenantID, indexID, req)
}

// runSearch shared by Search, SearchText, SearchVector. Applies the
// optional cache + singleflight envelopes around the HTTP call.
func (c *Client) runSearch(ctx context.Context, suffix, tenantID, indexID string, req SearchRequest) (*SearchResponse, error) {
	path := indexBasePath(tenantID, indexID) + suffix

	// fingerprint over the request and resource so cache + singleflight
	// don't collide across endpoints.
	key := fingerprint(suffix, tenantID, indexID, req)

	if c.cache != nil && key != "" {
		if cached, ok := c.cache.Get(key); ok {
			return cached, nil
		}
	}

	exec := func() (any, error) {
		var out SearchResponse
		if err := c.do(ctx, "POST", path, req, &out, nil); err != nil {
			return nil, err
		}
		if c.cache != nil && key != "" {
			c.cache.Set(key, &out)
		}
		return &out, nil
	}

	if c.sf != nil && key != "" {
		v, _, err := c.sf.Do(key, exec)
		if err != nil {
			return nil, err
		}
		return v.(*SearchResponse), nil
	}

	v, err := exec()
	if err != nil {
		return nil, err
	}
	return v.(*SearchResponse), nil
}

// MultiSearch calls POST /v1/orgs/{orgID}/users/{userID}/search and
// returns the merged result set across the user's accessible indexes.
func (c *Client) MultiSearch(ctx context.Context, orgID, userID string, req MultiSearchRequest) (*MultiSearchResponse, error) {
	path := "/v1/orgs/" + url.PathEscape(orgID) + "/users/" + url.PathEscape(userID) + "/search"
	var out MultiSearchResponse
	if err := c.do(ctx, "POST", path, req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// SyncDocuments calls POST /v1/orgs/{orgID}/documents.
func (c *Client) SyncDocuments(ctx context.Context, orgID string, req SyncDocumentsRequest) (*SyncDocumentsResponse, error) {
	var out SyncDocumentsResponse
	if err := c.do(ctx, "POST", "/v1/orgs/"+url.PathEscape(orgID)+"/documents", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListUserIndexes calls GET /v1/orgs/{orgID}/users/{userID}/indexes.
func (c *Client) ListUserIndexes(ctx context.Context, orgID, userID string) (*ListIndexesResponse, error) {
	path := "/v1/orgs/" + url.PathEscape(orgID) + "/users/" + url.PathEscape(userID) + "/indexes"
	var out ListIndexesResponse
	if err := c.do(ctx, "GET", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListSharedIndexes calls GET /v1/orgs/{orgID}/shared/indexes.
func (c *Client) ListSharedIndexes(ctx context.Context, orgID string) (*ListIndexesResponse, error) {
	path := "/v1/orgs/" + url.PathEscape(orgID) + "/shared/indexes"
	var out ListIndexesResponse
	if err := c.do(ctx, "GET", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
