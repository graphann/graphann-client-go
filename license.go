package graphann

import (
	"context"
	"net/url"
	"strconv"
)

// GetLicenseStatus calls GET /v1/license/status. Unauthenticated, like
// Health — this is the operator's self-serve rebind surface;
// LicenseStatus.CurrentFingerprint is what to paste into the licensing
// panel. Only mounted when the server was started with a license
// manager; the route is absent (404, ErrNotFound) otherwise, never 501.
func (c *Client) GetLicenseStatus(ctx context.Context) (*LicenseStatus, error) {
	var out LicenseStatus
	if err := c.do(ctx, "GET", "/v1/license/status", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetLicenseAudit calls GET /v1/license/audit. Returns this node's own
// local license state-transition history, newest first — not a
// cross-customer staff-panel audit trail. Same unauthenticated posture as
// GetLicenseStatus.
//
// limit caps the number of events returned; 0 (or any value above 1000)
// is clamped server-side to 1000.
//
// Note: on the provider-read-failure path the server returns a flat
// {"error": "<message>"} body instead of the standard error envelope, so
// the resulting APIError.Message may read as a generic "Internal Server
// Error" rather than the provider's own message — this is a server-side
// quirk, not a client bug.
func (c *Client) GetLicenseAudit(ctx context.Context, limit int) ([]LicenseAuditEvent, error) {
	var opts *requestOpts
	if limit > 0 {
		opts = &requestOpts{query: url.Values{"limit": []string{strconv.Itoa(limit)}}}
	}
	var out []LicenseAuditEvent
	if err := c.do(ctx, "GET", "/v1/license/audit", nil, &out, opts); err != nil {
		return nil, err
	}
	return out, nil
}
