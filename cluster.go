package graphann

import "context"

// GetClusterNodes calls GET /v1/cluster/nodes (Admin-only on the
// server). Returns ListClusterNodesResponse with members + Raft leader.
func (c *Client) GetClusterNodes(ctx context.Context) (*ListClusterNodesResponse, error) {
	var out ListClusterNodesResponse
	if err := c.do(ctx, "GET", "/v1/cluster/nodes", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetClusterShards calls GET /v1/cluster/shards (Admin-only on the
// server). Returns the shard placement map plus the configured
// replication factor.
func (c *Client) GetClusterShards(ctx context.Context) (*ListClusterShardsResponse, error) {
	var out ListClusterShardsResponse
	if err := c.do(ctx, "GET", "/v1/cluster/shards", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetClusterHealth calls GET /v1/cluster/health.
func (c *Client) GetClusterHealth(ctx context.Context) (*ClusterHealthResponse, error) {
	var out ClusterHealthResponse
	if err := c.do(ctx, "GET", "/v1/cluster/health", nil, &out, &requestOpts{noRetry: true}); err != nil {
		return nil, err
	}
	return &out, nil
}
