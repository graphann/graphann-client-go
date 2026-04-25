package graphann

import "time"

// Tenant represents a GraphANN tenant.
type Tenant struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at,omitempty"`
	IndexCount int               `json:"index_count,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// CreateTenantRequest is the body for POST /v1/tenants.
type CreateTenantRequest struct {
	// ID is optional. When provided, tenant creation is idempotent: the
	// existing tenant with this ID is returned. When omitted, the server
	// generates a fresh ID.
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// ListTenantsResponse is the body returned by GET /v1/tenants.
type ListTenantsResponse struct {
	Tenants []Tenant `json:"tenants"`
	Total   int      `json:"total"`
}

// DeleteTenantResponse is the body returned by DELETE /v1/tenants/{id}.
type DeleteTenantResponse struct {
	Deleted  bool   `json:"deleted"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

// IndexStatus enumerates lifecycle states reported by the server.
type IndexStatus string

const (
	IndexStatusEmpty    IndexStatus = "empty"
	IndexStatusBuilding IndexStatus = "building"
	IndexStatusReady    IndexStatus = "ready"
	IndexStatusError    IndexStatus = "error"
)

// Index describes a single index belonging to a tenant.
type Index struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Status      IndexStatus `json:"status"`
	NumDocs     int         `json:"num_docs"`
	NumChunks   int         `json:"num_chunks"`
	Dimension   int         `json:"dimension"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
	Path        string      `json:"path,omitempty"`
}

// CreateIndexRequest is the body for POST /v1/tenants/{tid}/indexes.
type CreateIndexRequest struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateIndexRequest is the body for PATCH /v1/tenants/{tid}/indexes/{iid}.
type UpdateIndexRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ListIndexesResponse is the body returned by GET /v1/tenants/{tid}/indexes.
type ListIndexesResponse struct {
	Indexes []Index `json:"indexes"`
	Total   int     `json:"total"`
}

// IndexStatusResponse is the body returned by GET .../indexes/{iid}/status.
type IndexStatusResponse struct {
	IndexID string      `json:"index_id"`
	Status  IndexStatus `json:"status"`
	Error   *string     `json:"error,omitempty"`
}

// HealthResponse is the body returned by GET /health.
type HealthResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Document is the input shape for AddDocuments / Import.
type Document struct {
	// ID is the optional client-supplied external identifier.
	ID string `json:"id,omitempty"`
	// Text is the document body. Required.
	Text string `json:"text"`
	// Metadata is arbitrary JSON metadata stored with the document.
	Metadata any `json:"metadata,omitempty"`
	// Upsert, when true, deletes any chunks with this ExternalID before
	// re-indexing. Use to make ingest idempotent by external key.
	Upsert bool `json:"upsert,omitempty"`
	// ExpiresAt, when non-nil, marks chunks as eligible for GC after the
	// timestamp.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// RBAC metadata for filtering search results.
	RepoID    string `json:"repo_id,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
}

// AddDocumentsRequest is the body for POST .../documents.
type AddDocumentsRequest struct {
	Documents []Document `json:"documents"`
}

// AddDocumentsResponse is the response from POST .../documents.
type AddDocumentsResponse struct {
	Added    int    `json:"added"`
	IndexID  string `json:"index_id"`
	ChunkIDs []int  `json:"chunk_ids,omitempty"`
}

// ImportDocumentsRequest is the body for POST .../import.
type ImportDocumentsRequest struct {
	Documents []Document `json:"documents"`
}

// ImportDocumentsResponse is the response from POST .../import.
type ImportDocumentsResponse struct {
	IndexID      string `json:"index_id"`
	Imported     int    `json:"imported"`
	DocumentIDs  []int  `json:"document_ids"`
	PendingTotal int    `json:"pending_total"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

// PendingStatusResponse is the body returned by GET .../pending.
type PendingStatusResponse struct {
	IndexID      string `json:"index_id"`
	PendingCount int    `json:"pending_count"`
}

// ProcessPendingResponse is the body returned by POST .../process.
type ProcessPendingResponse struct {
	IndexID       string `json:"index_id"`
	Processed     int    `json:"processed"`
	ChunksCreated int    `json:"chunks_created"`
	ChunkIDs      []int  `json:"chunk_ids,omitempty"`
}

// ClearPendingResponse is the body returned by DELETE .../pending.
type ClearPendingResponse struct {
	IndexID string `json:"index_id"`
	Cleared int    `json:"cleared,omitempty"`
	Status  string `json:"status"`
}

// BulkDeleteDocumentsRequest is the body for DELETE .../documents.
type BulkDeleteDocumentsRequest struct {
	DocumentIDs []int `json:"document_ids"`
}

// BulkDeleteDocumentsResponse is the body returned by DELETE .../documents.
type BulkDeleteDocumentsResponse struct {
	IndexID          string      `json:"index_id"`
	DocumentsDeleted int         `json:"documents_deleted"`
	ChunksDeleted    int         `json:"chunks_deleted"`
	DeletedPerDoc    map[int]int `json:"deleted_per_doc,omitempty"`
}

// BulkDeleteByExternalIDsRequest is the body for DELETE .../documents/by-external-id.
type BulkDeleteByExternalIDsRequest struct {
	ExternalIDs []string `json:"external_ids"`
}

// BulkDeleteByExternalIDsResponse is the response from the endpoint above.
type BulkDeleteByExternalIDsResponse struct {
	IndexID          string         `json:"index_id"`
	DocumentsDeleted int            `json:"documents_deleted"`
	ChunksDeleted    int            `json:"chunks_deleted"`
	DeletedPerID     map[string]int `json:"deleted_per_id,omitempty"`
}

// DocumentResponse is the body returned by GET .../documents/{docID}.
type DocumentResponse struct {
	IndexID     string          `json:"index_id"`
	DocumentID  int             `json:"document_id"`
	ExternalID  string          `json:"external_id,omitempty"`
	Chunks      []DocumentChunk `json:"chunks"`
	TotalChunks int             `json:"total_chunks"`
}

// DocumentChunk is one chunk in a DocumentResponse.
type DocumentChunk struct {
	ChunkID    int    `json:"chunk_id"`
	UUID       string `json:"uuid,omitempty"`
	Text       string `json:"text,omitempty"`
	ChunkIndex int    `json:"chunk_index"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
	RepoID     string `json:"repo_id,omitempty"`
	FilePath   string `json:"file_path,omitempty"`
	CommitSHA  string `json:"commit_sha,omitempty"`
}

// DeleteDocumentResponse is the body returned by DELETE .../documents/{docID}.
type DeleteDocumentResponse struct {
	DeletedChunks int    `json:"deleted_chunks"`
	DocumentID    int    `json:"document_id"`
	IndexID       string `json:"index_id"`
}

// ListDocumentsResponse is the body returned by GET .../documents.
type ListDocumentsResponse struct {
	Documents  []ListDocumentEntry `json:"documents"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

// ListDocumentEntry is one document in a ListDocumentsResponse.
type ListDocumentEntry struct {
	ID       string         `json:"id"`
	Text     string         `json:"text,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SearchFilter narrows search results by metadata.
type SearchFilter struct {
	// RepoIDs filters chunks to the named repositories.
	RepoIDs []string `json:"repo_ids,omitempty"`
	// ExcludeExternalIDs strips chunks whose ExternalID is in the list.
	ExcludeExternalIDs []string `json:"exclude_external_ids,omitempty"`
	// MetadataFilter requires each key/value to match the chunk's
	// sidecar metadata exactly.
	MetadataFilter map[string]any `json:"metadata_filter,omitempty"`
}

// SearchRequest is the body for POST .../search and friends.
type SearchRequest struct {
	Query  string       `json:"query,omitempty"`
	Vector []float32    `json:"vector,omitempty"`
	K      int          `json:"k,omitempty"`
	Filter SearchFilter `json:"filter,omitempty"`
}

// SearchResult is one hit in a SearchResponse.
type SearchResult struct {
	ID       string  `json:"id"`
	Text     string  `json:"text,omitempty"`
	Score    float32 `json:"score"`
	Metadata any     `json:"metadata,omitempty"`
}

// SearchResponse is the body returned by .../search endpoints.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// LiveStatsResponse is the body returned by GET .../live-stats.
type LiveStatsResponse struct {
	IndexID       string `json:"index_id"`
	IsLive        bool   `json:"is_live"`
	BaseChunks    int    `json:"base_chunks,omitempty"`
	DeltaChunks   int    `json:"delta_chunks,omitempty"`
	TotalChunks   int    `json:"total_chunks,omitempty"`
	DeletedChunks int    `json:"deleted_chunks,omitempty"`
	LiveChunks    int    `json:"live_chunks,omitempty"`
	Documents     int    `json:"documents,omitempty"`
	Dimension     int    `json:"dimension,omitempty"`
	IsDirty       bool   `json:"is_dirty,omitempty"`
	NumChunks     int    `json:"num_chunks,omitempty"`
	NumDocs       int    `json:"num_docs,omitempty"`
}

// CompactResponse is the body returned by POST .../compact.
type CompactResponse struct {
	IndexID string `json:"index_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ClearIndexResponse is the body returned by POST .../clear.
type ClearIndexResponse struct {
	IndexID string `json:"index_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ChunkResponse is the body returned by GET .../chunks/{chunkID}.
type ChunkResponse struct {
	ChunkID    int    `json:"chunk_id"`
	Text       string `json:"text,omitempty"`
	DocumentID int    `json:"document_id"`
	ChunkIndex int    `json:"chunk_index"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
}

// DeleteChunksRequest is the body for DELETE .../chunks/{chunkID}.
type DeleteChunksRequest struct {
	ChunkIDs []int `json:"chunk_ids"`
}

// DeleteChunksResponse is the body returned by DELETE .../chunks/{chunkID}.
type DeleteChunksResponse struct {
	Deleted int    `json:"deleted"`
	IndexID string `json:"index_id"`
}

// SwitchEmbeddingModelRequest is the body for the hot-model-switch
// endpoint.
type SwitchEmbeddingModelRequest struct {
	Backend          string `json:"embedding_backend"`
	Model            string `json:"model"`
	Dimension        int    `json:"dimension"`
	EndpointOverride string `json:"endpoint_override,omitempty"`
	APIKey           string `json:"api_key,omitempty"`
}

// SwitchEmbeddingModelResponse is the response from PATCH
// .../embedding-model.
type SwitchEmbeddingModelResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// JobStatus enumerates job lifecycle states.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// JobProgress is the progress snapshot embedded in Job.
type JobProgress struct {
	ChunksDone  int `json:"chunks_done"`
	ChunksTotal int `json:"chunks_total"`
}

// Job is the public projection of a server-side job.
type Job struct {
	JobID       string      `json:"job_id"`
	Kind        string      `json:"kind"`
	TenantID    string      `json:"tenant_id"`
	IndexID     string      `json:"index_id"`
	Status      JobStatus   `json:"status"`
	Progress    JobProgress `json:"progress"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// ListJobsRequest is the query for GET /v1/jobs and tenant-scoped
// equivalents.
type ListJobsRequest struct {
	// TenantID, when set, scopes to a tenant. Empty means cross-tenant
	// (admin-only).
	TenantID string `json:"-"`
	Status   JobStatus
	Cursor   string
	Limit    int
}

// ListJobsResponse is the body returned by GET /v1/jobs.
type ListJobsResponse struct {
	Jobs       []Job  `json:"jobs"`
	Total      int    `json:"total"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ClusterNode is one entry in a cluster node listing.
type ClusterNode struct {
	ID       string    `json:"id"`
	Addr     string    `json:"addr"`
	Zone     string    `json:"zone,omitempty"`
	State    string    `json:"state"`
	LastSeen time.Time `json:"last_seen"`
}

// ListClusterNodesResponse is the body returned by GET /v1/cluster/nodes.
type ListClusterNodesResponse struct {
	Nodes  []ClusterNode `json:"nodes"`
	Leader string        `json:"leader"`
}

// ClusterShard is one entry in a cluster shard listing.
type ClusterShard struct {
	ID            string            `json:"id"`
	Primary       string            `json:"primary"`
	Replicas      []string          `json:"replicas"`
	ZonePlacement map[string]string `json:"zone_placement,omitempty"`
}

// ListClusterShardsResponse is the body returned by GET /v1/cluster/shards.
type ListClusterShardsResponse struct {
	Shards []ClusterShard `json:"shards"`
	RF     int            `json:"rf"`
}

// ClusterHealthResponse is the body returned by GET /v1/cluster/health.
type ClusterHealthResponse struct {
	Status                string `json:"status"`
	ClusterSize           int    `json:"cluster_size"`
	AliveNodes            int    `json:"alive_nodes"`
	RaftHasLeader         bool   `json:"raft_has_leader"`
	UnderReplicatedShards int    `json:"under_replicated_shards"`
}

// LLMSettings holds organization-level LLM configuration.
type LLMSettings struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	APIKey      string  `json:"api_key,omitempty"`
	BaseURL     string  `json:"base_url,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

// UpdateLLMSettingsResponse is the body returned by PATCH /v1/orgs/{org}/llm-settings.
type UpdateLLMSettingsResponse struct {
	Message  string      `json:"message"`
	OrgID    string      `json:"org_id"`
	Settings LLMSettings `json:"settings"`
}

// DeleteLLMSettingsResponse is the body returned by DELETE /v1/orgs/{org}/llm-settings.
type DeleteLLMSettingsResponse struct {
	Message  string      `json:"message"`
	OrgID    string      `json:"org_id"`
	Settings LLMSettings `json:"settings,omitempty"`
}

// BuildIndexResponse is the body returned by POST .../indexes/{iid}/build.
type BuildIndexResponse struct {
	IndexID string `json:"index_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// CleanupOrphansResponse is the body returned by POST /v1/admin/cleanup-orphans.
type CleanupOrphansResponse struct {
	Removed    []string `json:"removed"`
	FreedBytes int64    `json:"freed_bytes"`
}

// MultiSearchRequest is the body for POST /v1/orgs/{org}/users/{user}/search.
type MultiSearchRequest struct {
	Query             string   `json:"query"`
	K                 int      `json:"k,omitempty"`
	Sources           []string `json:"sources,omitempty"`
	EfSearch          int      `json:"ef_search,omitempty"`
	IncludeText       bool     `json:"include_text,omitempty"`
	StartTime         int64    `json:"start_time,omitempty"`
	EndTime           int64    `json:"end_time,omitempty"`
	DistanceThreshold float32  `json:"distance_threshold,omitempty"`
}

// MultiSearchResult is one hit in a MultiSearchResponse.
type MultiSearchResult struct {
	ChunkID    int            `json:"chunk_id"`
	Text       string         `json:"text,omitempty"`
	Distance   float32        `json:"distance"`
	SourceType string         `json:"source_type"`
	RepoID     string         `json:"repo_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  int64          `json:"created_at,omitempty"`
	Shared     bool           `json:"shared,omitempty"`
}

// MultiSearchResponse is the body returned by POST .../search (org).
type MultiSearchResponse struct {
	Results []MultiSearchResult `json:"results"`
	Total   int                 `json:"total"`
	Query   string              `json:"query,omitempty"`
	OrgID   string              `json:"org_id,omitempty"`
	UserID  string              `json:"user_id,omitempty"`
}

// SyncDocumentsRequest is the body for POST /v1/orgs/{org}/documents.
type SyncDocumentsRequest struct {
	UserID     string         `json:"user_id"`
	SourceType string         `json:"source_type"`
	Shared     bool           `json:"shared"`
	Documents  []SyncDocument `json:"documents"`
}

// SyncDocument is one entry in a SyncDocumentsRequest.
type SyncDocument struct {
	ResourceID string            `json:"resource_id,omitempty"`
	Text       string            `json:"text"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SyncDocumentsResponse is the body returned by POST /v1/orgs/{org}/documents.
type SyncDocumentsResponse struct {
	Synced     int    `json:"synced"`
	OrgID      string `json:"org_id"`
	UserID     string `json:"user_id"`
	SourceType string `json:"source_type"`
	IndexType  string `json:"index_type"`
}

// APIKey is the public projection of a tenant API key.
type APIKey struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Prefix    string    `json:"prefix,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	// Plaintext is set by CreateAPIKey on the response and ONLY then —
	// the server returns the secret exactly once and it is not
	// recoverable.
	Plaintext string `json:"key,omitempty"`
}

// CreateAPIKeyRequest is the body for POST /v1/tenants/{tid}/api-keys.
type CreateAPIKeyRequest struct {
	Name string `json:"name"`
	// UserID is optional; when set, ties the key to a user.
	UserID string `json:"user_id,omitempty"`
}

// ListAPIKeysResponse is the body returned by GET /v1/tenants/{tid}/api-keys.
type ListAPIKeysResponse struct {
	Keys  []APIKey `json:"keys"`
	Total int      `json:"total"`
}
