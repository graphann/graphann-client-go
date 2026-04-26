package graphann

import (
	"context"
	"net/url"
	"strconv"
)

// AddDocuments calls POST /v1/tenants/{tid}/indexes/{iid}/documents.
func (c *Client) AddDocuments(ctx context.Context, tenantID, indexID string, req AddDocumentsRequest) (*AddDocumentsResponse, error) {
	var out AddDocumentsResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/documents", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ImportDocuments calls POST /v1/tenants/{tid}/indexes/{iid}/import.
// The server queues and processes in the background — the call returns
// immediately.
func (c *Client) ImportDocuments(ctx context.Context, tenantID, indexID string, req ImportDocumentsRequest) (*ImportDocumentsResponse, error) {
	var out ImportDocumentsResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/import", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPendingStatus calls GET /v1/tenants/{tid}/indexes/{iid}/pending.
func (c *Client) GetPendingStatus(ctx context.Context, tenantID, indexID string) (*PendingStatusResponse, error) {
	var out PendingStatusResponse
	if err := c.do(ctx, "GET", indexBasePath(tenantID, indexID)+"/pending", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ProcessPending calls POST /v1/tenants/{tid}/indexes/{iid}/process.
func (c *Client) ProcessPending(ctx context.Context, tenantID, indexID string) (*ProcessPendingResponse, error) {
	var out ProcessPendingResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/process", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ClearPending calls DELETE /v1/tenants/{tid}/indexes/{iid}/pending.
func (c *Client) ClearPending(ctx context.Context, tenantID, indexID string) (*ClearPendingResponse, error) {
	var out ClearPendingResponse
	if err := c.do(ctx, "DELETE", indexBasePath(tenantID, indexID)+"/pending", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDocument calls GET /v1/tenants/{tid}/indexes/{iid}/documents/{docID}.
func (c *Client) GetDocument(ctx context.Context, tenantID, indexID string, docID int) (*DocumentResponse, error) {
	var out DocumentResponse
	path := indexBasePath(tenantID, indexID) + "/documents/" + strconv.Itoa(docID)
	if err := c.do(ctx, "GET", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDocument calls DELETE /v1/tenants/{tid}/indexes/{iid}/documents/{docID}.
func (c *Client) DeleteDocument(ctx context.Context, tenantID, indexID string, docID int) (*DeleteDocumentResponse, error) {
	var out DeleteDocumentResponse
	path := indexBasePath(tenantID, indexID) + "/documents/" + strconv.Itoa(docID)
	if err := c.do(ctx, "DELETE", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// BulkDeleteDocuments calls DELETE /v1/tenants/{tid}/indexes/{iid}/documents.
func (c *Client) BulkDeleteDocuments(ctx context.Context, tenantID, indexID string, req BulkDeleteDocumentsRequest) (*BulkDeleteDocumentsResponse, error) {
	var out BulkDeleteDocumentsResponse
	if err := c.do(ctx, "DELETE", indexBasePath(tenantID, indexID)+"/documents", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// BulkDeleteByExternalIDs calls DELETE
// /v1/tenants/{tid}/indexes/{iid}/documents/by-external-id.
func (c *Client) BulkDeleteByExternalIDs(ctx context.Context, tenantID, indexID string, req BulkDeleteByExternalIDsRequest) (*BulkDeleteByExternalIDsResponse, error) {
	var out BulkDeleteByExternalIDsResponse
	if err := c.do(ctx, "DELETE", indexBasePath(tenantID, indexID)+"/documents/by-external-id", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDocumentsRequest narrows GET .../documents by prefix and paginates.
type ListDocumentsRequest struct {
	Prefix string
	Limit  int
}

// ListDocuments calls GET /v1/tenants/{tid}/indexes/{iid}/documents and
// returns an iterator over the prefix-paginated stream.
func (c *Client) ListDocuments(ctx context.Context, tenantID, indexID string, req ListDocumentsRequest) *Iter[ListDocumentEntry] {
	fetch := func(ctx context.Context, cursor string) ([]ListDocumentEntry, string, error) {
		q := url.Values{}
		if req.Prefix != "" {
			q.Set("prefix", req.Prefix)
		}
		if req.Limit > 0 {
			q.Set("limit", strconv.Itoa(req.Limit))
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var out ListDocumentsResponse
		if err := c.do(ctx, "GET", indexBasePath(tenantID, indexID)+"/documents", nil, &out, &requestOpts{query: q}); err != nil {
			return nil, "", err
		}
		return out.Documents, out.NextCursor, nil
	}
	return newIter[ListDocumentEntry](fetch)
}

// GetChunk calls GET /v1/tenants/{tid}/indexes/{iid}/chunks/{chunkID}.
func (c *Client) GetChunk(ctx context.Context, tenantID, indexID string, chunkID int) (*ChunkResponse, error) {
	var out ChunkResponse
	path := indexBasePath(tenantID, indexID) + "/chunks/" + strconv.Itoa(chunkID)
	if err := c.do(ctx, "GET", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteChunks calls DELETE /v1/tenants/{tid}/indexes/{iid}/chunks/{chunkID}.
// chunkID is unused by the server but required by the route — we pass 0
// when DeleteChunksRequest.ChunkIDs is the source of truth.
func (c *Client) DeleteChunks(ctx context.Context, tenantID, indexID string, req DeleteChunksRequest) (*DeleteChunksResponse, error) {
	var out DeleteChunksResponse
	// The server route is /chunks/{chunkID} but the body carries the
	// list. Pass 0 as a placeholder; the route only matches the prefix.
	path := indexBasePath(tenantID, indexID) + "/chunks/0"
	if err := c.do(ctx, "DELETE", path, req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
