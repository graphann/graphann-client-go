package graphann

import (
	"context"
	"net/url"
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
// Restores the named backup into req.DestIndex within the path tenant.
// backupID is percent-encoded because backup IDs contain slashes.
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
