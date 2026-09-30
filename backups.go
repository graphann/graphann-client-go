package graphann

import (
	"context"
	"net/url"
	"strconv"
)

// backupCollectionPath returns /v1/tenants/{tid}/backups.
func backupCollectionPath(tenantID string) string {
	return "/v1/tenants/" + url.PathEscape(tenantID) + "/backups"
}

// CreateBackup calls POST /v1/tenants/{tid}/indexes/{iid}/backups.
// Snapshots the index into the server's configured backup storage.
// Registered only when the server was started with --backup-dir; on a
// registered-but-disabled deployment every backup route returns 501
// (ErrNotImplemented), distinguishing "backups are off" from "route
// missing" (404, ErrNotFound).
func (c *Client) CreateBackup(ctx context.Context, tenantID, indexID string) (*CreateBackupResponse, error) {
	var out CreateBackupResponse
	if err := c.do(ctx, "POST", indexBasePath(tenantID, indexID)+"/backups", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListBackups calls GET /v1/tenants/{tid}/backups. Returns every backup
// summary for the tenant, across all of its indexes.
func (c *Client) ListBackups(ctx context.Context, tenantID string) (*ListBackupsResponse, error) {
	var out ListBackupsResponse
	if err := c.do(ctx, "GET", backupCollectionPath(tenantID), nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// RestoreBackup calls POST /v1/tenants/{tid}/backups/{backupID}/restore.
// Creates a new index named req.DestIndex (a new, unused id; it must start
// with "i_") in the path tenant. req.Name sets its display name and
// defaults to "<source name> (restored)". Errors: ErrConflict when the id
// exists, ErrUnprocessable when a pre-settings backup's source index is
// gone. backupID is percent-encoded because backup IDs contain slashes.
func (c *Client) RestoreBackup(ctx context.Context, tenantID, backupID string, req RestoreBackupRequest) (*RestoreBackupResponse, error) {
	path := backupCollectionPath(tenantID) + "/" + url.PathEscape(backupID) + "/restore"
	var out RestoreBackupResponse
	if err := c.do(ctx, "POST", path, req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBackup calls DELETE /v1/tenants/{tid}/backups/{backupID}.
// backupID is percent-encoded because backup IDs contain slashes.
func (c *Client) DeleteBackup(ctx context.Context, tenantID, backupID string) (*DeleteBackupResponse, error) {
	path := backupCollectionPath(tenantID) + "/" + url.PathEscape(backupID)
	var out DeleteBackupResponse
	if err := c.do(ctx, "DELETE", path, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAllBackups returns an iterator over GET /v1/admin/backups, the
// admin cross-tenant backup list ordered newest first. The iterator follows
// the server's keyset next_cursor, so deleting a backup between pages
// causes no duplicates or skipped rows. Like the other backup routes it
// exists only when the server runs with --backup-dir.
func (c *Client) ListAllBackups(ctx context.Context, req ListAllBackupsRequest) *Iter[AdminBackupRow] {
	fetch := func(ctx context.Context, cursor string) ([]AdminBackupRow, string, error) {
		out, err := c.ListAllBackupsPage(ctx, req, cursor)
		if err != nil {
			return nil, "", err
		}
		return out.Backups, out.NextCursor, nil
	}
	return newIter[AdminBackupRow](fetch)
}

// ListAllBackupsPage fetches one page of GET /v1/admin/backups. An empty
// cursor requests the first page; pass the previous page's NextCursor for
// the next one. The result's Total counts every backup matching the filters,
// which the iterator does not expose.
func (c *Client) ListAllBackupsPage(ctx context.Context, req ListAllBackupsRequest, cursor string) (*AdminBackupList, error) {
	q := url.Values{}
	if req.TenantID != "" {
		q.Set("tenant_id", req.TenantID)
	}
	if req.IndexID != "" {
		q.Set("index_id", req.IndexID)
	}
	if req.Limit > 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	var out AdminBackupList
	if err := c.do(ctx, "GET", "/v1/admin/backups", nil, &out, &requestOpts{query: q}); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBackupStatus calls GET /v1/admin/backups/status. Enabled is false
// when no --backup-cron schedule is configured; the time fields stay nil
// until the scheduler has recorded them.
func (c *Client) GetBackupStatus(ctx context.Context) (*BackupScheduleStatus, error) {
	var out BackupScheduleStatus
	if err := c.do(ctx, "GET", "/v1/admin/backups/status", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
