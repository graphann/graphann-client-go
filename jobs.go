package graphann

import (
	"context"
	"net/url"
	"strconv"
)

// SwitchEmbeddingModel calls PATCH
// /v1/tenants/{tid}/indexes/{iid}/embedding-model. The server enqueues
// an async reembed job and returns the job id immediately.
func (c *Client) SwitchEmbeddingModel(ctx context.Context, tenantID, indexID string, req SwitchEmbeddingModelRequest) (*SwitchEmbeddingModelResponse, error) {
	path := indexBasePath(tenantID, indexID) + "/embedding-model"
	var out SwitchEmbeddingModelResponse
	if err := c.do(ctx, "PATCH", path, req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetJob calls GET /v1/jobs/{jobID}.
func (c *Client) GetJob(ctx context.Context, jobID string) (*Job, error) {
	var out Job
	path := "/v1/jobs/" + url.PathEscape(jobID)
	if err := c.do(ctx, "GET", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListJobs returns an iterator over GET /v1/jobs (or
// /v1/tenants/{tid}/jobs when req.TenantID is set). Status and Limit
// are optional; the iterator handles cursor pagination internally.
func (c *Client) ListJobs(ctx context.Context, req ListJobsRequest) *Iter[Job] {
	fetch := func(ctx context.Context, cursor string) ([]Job, string, error) {
		q := url.Values{}
		if req.Status != "" {
			q.Set("status", string(req.Status))
		}
		if req.Limit > 0 {
			q.Set("limit", strconv.Itoa(req.Limit))
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		path := "/v1/jobs"
		if req.TenantID != "" {
			path = "/v1/tenants/" + url.PathEscape(req.TenantID) + "/jobs"
		}
		var out ListJobsResponse
		if err := c.do(ctx, "GET", path, nil, &out, &requestOpts{query: q}); err != nil {
			return nil, "", err
		}
		return out.Jobs, out.NextCursor, nil
	}
	return newIter[Job](fetch)
}
