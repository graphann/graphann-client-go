package graphann

import "context"

// Health calls GET /health. The probe never retries — operators should
// build retry into their own health-check logic.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var out HealthResponse
	if err := c.do(ctx, "GET", "/health", nil, &out, &requestOpts{noRetry: true}); err != nil {
		return nil, err
	}
	return &out, nil
}

// Ready calls GET /ready. Returns the same shape as Health. 503
// responses are surfaced as ErrServer/ErrIndexNotReady — callers should
// inspect Status if they want a less binary signal.
func (c *Client) Ready(ctx context.Context) (*HealthResponse, error) {
	var out HealthResponse
	if err := c.do(ctx, "GET", "/ready", nil, &out, &requestOpts{noRetry: true}); err != nil {
		return nil, err
	}
	return &out, nil
}
