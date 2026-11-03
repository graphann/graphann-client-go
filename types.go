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
	IndexStatusPending  IndexStatus = "pending"
	IndexStatusBuilding IndexStatus = "building"
	IndexStatusReady    IndexStatus = "ready"
	IndexStatusError    IndexStatus = "error"
	IndexStatusDeleted  IndexStatus = "deleted"
)

// EmbedSpaceState classifies an index's embedding space against the
// serving embedder.
type EmbedSpaceState string

const (
	EmbedSpaceVerified           EmbedSpaceState = "verified"
	EmbedSpaceMismatch           EmbedSpaceState = "mismatch"
	EmbedSpaceUnknown            EmbedSpaceState = "unknown"
	EmbedSpaceExternalUnverified EmbedSpaceState = "external_unverified"
	EmbedSpaceExternalAgree      EmbedSpaceState = "external_agree"
	EmbedSpaceExternalDiverged   EmbedSpaceState = "external_diverged"
	// EmbedSpaceEmpty means the index holds no vectors, so there is
	// nothing to compare. It is not a risk signal and outranks every
	// other state.
	EmbedSpaceEmpty EmbedSpaceState = "empty"
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
	// Compression is the vector compression strategy configured for this index.
	Compression string `json:"compression,omitempty"`
	// Approximate reports whether approximate (HNSW) search is enabled.
	Approximate bool `json:"approximate,omitempty"`
	// ChunkSize is the requested per-index chunker size in bytes; zero means
	// the server default. The effective size can be smaller because the
	// embedding model's token limit caps it.
	ChunkSize int `json:"chunk_size,omitempty"`
	// ChunkOverlap is the requested per-index chunker overlap in bytes; zero
	// means the server default.
	ChunkOverlap int `json:"chunk_overlap,omitempty"`

	// Error is the failure reason when Status is "error".
	Error string `json:"error,omitempty"`
	// LastCompactedAt is the zero time when the index was never compacted.
	LastCompactedAt time.Time         `json:"last_compacted_at,omitempty"`
	CreatedBy       string            `json:"created_by,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	// VectorsExternal reports that the index holds caller-supplied vectors.
	VectorsExternal bool `json:"vectors_external,omitempty"`

	// Per-index embedding override; empty backend means the process embedder.
	EmbeddingBackend   string `json:"embedding_backend,omitempty"`
	EmbeddingModel     string `json:"embedding_model,omitempty"`
	EmbeddingEndpoint  string `json:"embedding_endpoint,omitempty"`
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	EmbeddingAPIKeyEnv string `json:"embedding_api_key_env,omitempty"`
}

// CreateIndexRequest is the body for POST /v1/tenants/{tid}/indexes.
type CreateIndexRequest struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Compression sets the vector compression strategy for the index.
	// One of "scalar", "binary", "pq", "recompute", or "" (server default).
	Compression *string `json:"compression,omitempty"`
	// Approximate, when true, enables approximate (HNSW) search; false uses exact scan.
	// Nil defers to the server default.
	Approximate *bool `json:"approximate,omitempty"`
	// ChunkSize overrides the server chunker size in bytes. The server
	// accepts [64, 8192] and requires ChunkOverlap alongside it; zero for
	// both keeps the server default. It does not rechunk existing data.
	ChunkSize int `json:"chunk_size,omitempty"`
	// ChunkOverlap overrides the chunker overlap in bytes. It must be >= 0
	// and strictly less than ChunkSize.
	ChunkOverlap int `json:"chunk_overlap,omitempty"`
}

// UpdateIndexRequest is the body for PATCH /v1/tenants/{tid}/indexes/{iid}.
type UpdateIndexRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	// Compression updates the vector compression strategy. Same values as CreateIndexRequest.
	Compression *string `json:"compression,omitempty"`
	// Approximate updates the approximate-search flag.
	Approximate *bool `json:"approximate,omitempty"`

	// EmbeddingBackend sets a per-index embedding backend override.
	// Remote backends only — per-index local_onnx is unsupported; use
	// the process embedder for local models. Setting "" clears the
	// override back to the process-wide embedder.
	EmbeddingBackend *string `json:"embedding_backend,omitempty"`
	// EmbeddingModel is the backend model identifier (e.g.
	// "text-embedding-3-small"), used when EmbeddingBackend is set.
	EmbeddingModel *string `json:"embedding_model,omitempty"`
	// EmbeddingDimension is the expected output dimension for the
	// per-index embedding override.
	EmbeddingDimension *int `json:"embedding_dimension,omitempty"`
	// EmbeddingEndpoint overrides the backend base URL. SSRF-validated
	// server-side (public http/https hosts only).
	EmbeddingEndpoint *string `json:"embedding_endpoint,omitempty"`
	// EmbeddingAPIKeyEnv names the ENV VAR holding the backend key —
	// never send the key itself in this field.
	EmbeddingAPIKeyEnv *string `json:"embedding_api_key_env,omitempty"`
}

// ListIndexesResponse is the body returned by GET /v1/tenants/{tid}/indexes.
type ListIndexesResponse struct {
	Indexes []Index `json:"indexes"`
	Total   int     `json:"total"`
	// OrgID and UserID are set by the org-scoped listings.
	OrgID  string `json:"org_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// IndexStatusResponse is the body returned by GET .../indexes/{iid}/status.
type IndexStatusResponse struct {
	IndexID string      `json:"index_id"`
	Status  IndexStatus `json:"status"`
	Error   *string     `json:"error,omitempty"`

	// EmbedSpaceState compares the index's embedding space with the serving
	// embedder's.
	EmbedSpaceState        EmbedSpaceState `json:"embed_space_state,omitempty"`
	EmbedFingerprint       string          `json:"embed_fingerprint,omitempty"`
	ServerEmbedFingerprint string          `json:"server_embed_fingerprint,omitempty"`
	// EmbedSpacePolicy is "warn", "refuse_search" or "refuse_open".
	EmbedSpacePolicy    string     `json:"embed_space_policy,omitempty"`
	EmbedSpaceDetail    string     `json:"embed_space_detail,omitempty"`
	EmbedSpaceCheckedAt *time.Time `json:"embed_space_checked_at,omitempty"`
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

	// Vector is an optional precomputed embedding. When EVERY document in
	// the batch carries a non-empty Vector, the server skips embedding and
	// ingests the vectors directly. Mixed batches (some documents with a
	// vector, some without) are rejected with 400 validation_error.
	//
	// The vector length must match the index dimension once it is set;
	// the first ingest into a fresh index fixes the dimension. Precomputed
	// inserts are idempotent by external ID (upsert semantics), so the
	// per-document Upsert flag is ignored on this path.
	Vector []float32 `json:"vector,omitempty"`

	// VectorB64 is Vector as base64-encoded little-endian float32
	// (numpy `astype('<f4').tobytes()` / `binary.LittleEndian`) — a
	// cheaper-to-decode alternative for bulk loads. Set one of Vector /
	// VectorB64, never both.
	VectorB64 string `json:"vector_b64,omitempty"`

	// Content is an alias for Text; the server accepts either, and Text
	// takes precedence when both are set.
	Content string `json:"content,omitempty"`
}

// AddDocumentsRequest is the body for POST .../documents.
type AddDocumentsRequest struct {
	Documents []Document `json:"documents"`

	// DeferSave, when true, skips the per-batch full-delta save. Data
	// stays in memory (index dirty) but is still searchable. Persist with
	// Client.FlushIndex once the batch sequence is complete.
	DeferSave bool `json:"defer_save,omitempty"`

	// Bulk implies DeferSave and additionally defers the per-node HNSW
	// insert — the delta graph is built once, concurrently, at flush.
	// Bulk-ingested data is NOT searchable until the graph is built,
	// with one safety net: the first search against a pending deferred
	// build transparently triggers it (build-on-read), so searches never
	// silently miss bulk data. Defaults (both false) preserve the
	// per-batch, immediately-searchable behaviour.
	Bulk bool `json:"bulk,omitempty"`
}

// AddDocumentsResponse is the response from POST .../documents.
// ChunkIDs are server-issued UUID strings (store.ChunkID = string),
// not integers.
//
// ExternalIDs is present only when the server minted at least one
// ExternalID (which happens on sharded ingest of ID-less documents —
// ExternalID is the shard routing key). When present it carries one
// entry per submitted document, positionally aligned with the request
// array, mixing minted and client-supplied IDs. Persist these as the
// durable document IDs. Unsharded ingests never mint, so the field is
// absent (nil) there.
type AddDocumentsResponse struct {
	Added       int      `json:"added"`
	IndexID     string   `json:"index_id"`
	ChunkIDs    []string `json:"chunk_ids,omitempty"`
	ExternalIDs []string `json:"external_ids,omitempty"`
	// Warnings are advisory: the request succeeded and every document was
	// ingested. The server sets them on the precomputed-vector path when a
	// document's text exceeds the embedder's token budget, because a later
	// re-embed would truncate it.
	Warnings []string `json:"warnings,omitempty"`
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
// ChunkIDs are server-issued UUID strings (store.ChunkID = string),
// not integers.
type ProcessPendingResponse struct {
	IndexID       string   `json:"index_id"`
	Processed     int      `json:"processed"`
	ChunksCreated int      `json:"chunks_created"`
	ChunkIDs      []string `json:"chunk_ids,omitempty"`
}

// ClearPendingResponse is the body returned by DELETE .../pending.
type ClearPendingResponse struct {
	IndexID string `json:"index_id"`
	Cleared int    `json:"cleared,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
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
	// Equals is a generic metadata pre-filter: every key/value pair must
	// match the chunk's stored metadata (string equality).
	Equals map[string]string `json:"equals,omitempty"`

	// OmitText drops chunk text from every result. Off by default so an
	// existing client sees exactly what it saw before; worth setting
	// when the caller already holds the text (RAG pipelines reading
	// from their own store, reranking stages keyed on ids).
	OmitText bool `json:"omit_text,omitempty"`
}

// SearchRequest is the body for POST .../search and friends.
type SearchRequest struct {
	Query  string       `json:"query,omitempty"`
	Vector []float32    `json:"vector,omitempty"`
	K      int          `json:"k,omitempty"`
	Filter SearchFilter `json:"filter,omitempty"`

	// Rerank, when true, rescores the top-CandidateK HNSW candidates
	// with the server-configured cross-encoder reranker and returns the
	// top-RerankK (or top-K if RerankK is unset). No-op when the server
	// has no reranker configured (--reranker-url unset) — flipping this
	// on a non-rerank-aware deployment is safe.
	//
	// Only applies to the text Query path. Vector-only requests ignore
	// Rerank since the cross-encoder needs raw query text.
	Rerank bool `json:"rerank,omitempty"`

	// CandidateK is the size of the first-stage candidate pool fed to
	// the reranker. Effective only when Rerank is true. Zero defaults
	// to max(4*K, 50). The server clamps to [K, 1000].
	CandidateK int `json:"candidate_k,omitempty"`

	// RerankK is the number of results to return after reranking.
	// Effective only when Rerank is true. Zero defaults to K.
	RerankK int `json:"rerank_k,omitempty"`

	// EfSearch overrides the HNSW search-time beam width (ef) for this
	// query. Zero (or omitted) uses the server default (`--search-ef`,
	// default 64). The server clamps rather than rejects: negative values
	// map to the server default and values above 2000 are capped. Some
	// modes apply local floors on top of the request, and binary/PQ flat
	// scans ignore ef entirely.
	EfSearch int `json:"ef_search,omitempty"`

	// Hybrid fuses dense (semantic) results with a BM25 lexical ranking
	// via Reciprocal Rank Fusion. Applies to text Query searches only;
	// ignored for vector-only requests. False (default) is dense-only.
	// Falls back to dense-only results if the lexical index can't be
	// built, so flipping this on is always safe.
	Hybrid bool `json:"hybrid,omitempty"`

	// VectorB64 is Vector as base64-encoded little-endian float32 — an
	// alternative encoding of the query vector that is cheaper to
	// decode. Mutually exclusive with Vector; set one or the other.
	VectorB64 string `json:"vector_b64,omitempty"`

	// GroupBy collapses results per source document; only
	// SearchGroupByDocument is supported. It requires MaxPerDoc >= 1 and
	// the server answers 400 on a sharded index.
	GroupBy string `json:"group_by,omitempty"`

	// MaxPerDoc caps the chunks returned per document when GroupBy is set.
	// The server ignores it when GroupBy is empty.
	MaxPerDoc int `json:"max_per_doc,omitempty"`

	// RelatedTo is the external ID of a stored document. When set, the
	// request is a related search: leave Query and Vector empty and the
	// server ranks the documents most similar to that document.
	RelatedTo string `json:"related_to,omitempty"`

	// Hops is the length of a related walk (1-3); needs RelatedTo. Zero
	// uses the server default of 1, the direct related search.
	Hops int `json:"hops,omitempty"`

	// Fanout is the number of documents each hop expands (1-50). Needs
	// Hops of 2 or 3. Zero uses the server default of 10.
	Fanout int `json:"fanout,omitempty"`

	// NovelOnly drops documents a direct related search already returns.
	NovelOnly bool `json:"novel_only,omitempty"`

	// Diversity (0-1) weights the diverse frontier pick of a walk. Needs
	// Hops of 2 or 3. Zero expands the Fanout best paths.
	Diversity float32 `json:"diversity,omitempty"`
}

// SearchGroupByDocument is the only supported SearchRequest.GroupBy value.
const SearchGroupByDocument = "document"

// SearchResult is one hit in a SearchResponse.
//
// Score is always the first-stage cosine similarity (higher is better).
// Clients can rely on this single scale regardless of whether the
// request asked for reranking.
//
// RerankScore is non-nil only when the server actually applied the
// cross-encoder reranker to this result — i.e. the request set
// Rerank=true AND the server has a reranker configured AND the
// rerank call succeeded. When non-nil, RerankScore is the reranker's
// native relevance score (different scale from cosine; typically
// roughly -10..10 for bge-reranker-v2-m3) and the result order
// reflects RerankScore. When nil, results are ordered by Score and
// the client should treat the response as a regular non-reranked
// search.
type SearchResult struct {
	ID          string   `json:"id"`
	Text        string   `json:"text,omitempty"`
	Score       float32  `json:"score"`
	RerankScore *float32 `json:"rerank_score,omitempty"`
	Metadata    any      `json:"metadata,omitempty"`

	// Path is the best path from the source to this document, the source
	// left out. Set by a related walk of 2 or 3 hops.
	Path []RelatedHop `json:"path,omitempty"`
	// Support counts the paths that reach this document. Set by a related walk.
	Support int `json:"support,omitempty"`
	// DirectScore is the similarity of this document to the source itself.
	// Set by a related walk.
	DirectScore *float32 `json:"direct_score,omitempty"`
}

// RelatedHop is one step of SearchResult.Path.
type RelatedHop struct {
	// ID is the external ID of the document this hop reached.
	ID    string  `json:"id"`
	Score float32 `json:"score"`
}

// SearchResponse is the body returned by .../search endpoints.
//
// The sharded-search fields (Partial, ShardsTotal, ShardsOK,
// DegradedShards) are populated ONLY when the request was served by the
// scatter-gather path — a clustered deployment where the index has more
// than one shard. Check Partial != nil to detect the sharded path.
//
// Sharded text queries are embedded once on the coordinator. Results are
// deduplicated by external ID before optional coordinator-side reranking.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`

	// Exhaustive is set by a related walk: true when no unexplored path can
	// outscore the k-th result.
	Exhaustive *bool `json:"exhaustive,omitempty"`
	// ShardsProbed is the number of shards the query visited; lower than
	// ShardsTotal when content routing pruned shards.
	ShardsProbed *int `json:"shards_probed,omitempty"`

	// Partial is true when at least one shard contributed nothing.
	Partial *bool `json:"partial,omitempty"`
	// ShardsTotal is the number of shards queried.
	ShardsTotal *int `json:"shards_total,omitempty"`
	// ShardsOK is the number of shards that responded successfully.
	ShardsOK *int `json:"shards_ok,omitempty"`
	// DegradedShards lists the shard IDs that failed or degraded.
	// Present only when non-empty.
	DegradedShards []string `json:"degraded_shards,omitempty"`
	// RerankApplied reports whether the server applied reranking. Nil means
	// the server did not report this field; false explicitly means skipped.
	RerankApplied *bool `json:"rerank_applied,omitempty"`
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

	FrozenChunks          int     `json:"frozen_chunks,omitempty"`
	BaseBytes             int64   `json:"base_bytes,omitempty"`
	DeltaBytes            int64   `json:"delta_bytes,omitempty"`
	EmbeddingSidecarBytes int64   `json:"embedding_sidecar_bytes,omitempty"`
	RawBytes              int64   `json:"raw_bytes,omitempty"`
	IndexSizeBytes        int64   `json:"index_size_bytes,omitempty"`
	CompressionRatio      float64 `json:"compression_ratio,omitempty"`
	BaseDeletedChunks     int     `json:"base_deleted_chunks,omitempty"`
	BaseDeletedFraction   float64 `json:"base_deleted_fraction,omitempty"`
	RebuildRecommended    bool    `json:"rebuild_recommended,omitempty"`
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

// FlushIndexResponse is the body returned by POST .../flush.
type FlushIndexResponse struct {
	Flushed bool `json:"flushed"`
}

// RebuildGraphResponse is the body returned by POST .../rebuild-graph.
type RebuildGraphResponse struct {
	Rebuilt bool `json:"rebuilt"`
	// Chunks is the number of chunks reinserted into the rebuilt delta graph.
	Chunks int `json:"chunks"`
	// WallMS is the wall-clock rebuild time in milliseconds.
	WallMS int `json:"wall_ms"`
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

// UpsertResourceRequest is the body for PUT .../indexes/{iid}/resources/{resID}.
type UpsertResourceRequest struct {
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`

	ExternalID  string `json:"external_id,omitempty"`
	RepoID      string `json:"repo_id,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	CommitSHA   string `json:"commit_sha,omitempty"`
	SourceType  string `json:"source_type,omitempty"`
	OwnerUserID string `json:"owner_user_id,omitempty"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
}

// UpsertResourceResponse is the response from PUT .../indexes/{iid}/resources/{resID}.
type UpsertResourceResponse struct {
	ResourceID       string `json:"resource_id"`
	IndexID          string `json:"index_id,omitempty"`
	ChunksAdded      int    `json:"chunks_added"`
	ChunksTombstoned int    `json:"chunks_tombstoned"`
	// Operation is "create" on first upsert, "update" on subsequent ones.
	Operation string `json:"operation"`
}

// CleanupOrphansResponse is the body returned by POST /v1/admin/cleanup-orphans.
//
// MinAge is a Go-style duration string (e.g. "1h0m0s") echoing the
// minimum-age cutoff the server actually applied. DryRun echoes the
// dry-run flag — when true, Removed is what *would* have been removed.
type CleanupOrphansResponse struct {
	Removed    []string `json:"removed"`
	FreedBytes int64    `json:"freed_bytes"`
	MinAge     string   `json:"min_age,omitempty"`
	DryRun     bool     `json:"dry_run,omitempty"`
}

// GCResponse is the body returned by both POST .../indexes/{iid}/gc and
// POST /v1/admin/gc — they share the same shape (one or many indexes,
// the response is just the count reclaimed).
type GCResponse struct {
	IndexID      string `json:"index_id,omitempty"`
	DeletedCount int    `json:"deleted_count"`
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
	// MaxResults is an alias for K on the server.
	MaxResults int `json:"max_results,omitempty"`
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

// CreateAPIKeyRequest is the body for POST /v1/tenants/{tid}/api-keys.
// UserID is required (the server rejects an empty value). Without Role it
// must name an existing user in the tenant; with Role it may name a new
// principal. Name is the key's human-readable label.
type CreateAPIKeyRequest struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	// Role is one of the APIKeyRole constants. It is required when UserID
	// names no existing user, and must equal the user's role when it does.
	// Once a tenant has any key, only an admin key can create or revoke keys.
	Role APIKeyRole `json:"role,omitempty"`
}

// APIKeyRole is the role an API key carries.
type APIKeyRole string

const (
	APIKeyRoleAdmin  APIKeyRole = "admin"
	APIKeyRoleEditor APIKeyRole = "editor"
	APIKeyRoleViewer APIKeyRole = "viewer"
)

// CreateAPIKeyResponse is the body returned by POST /v1/tenants/{tid}/api-keys.
// Plaintext is the secret token and is returned EXACTLY ONCE — the server
// stores only an argon2id hash, so callers must persist it client-side.
type CreateAPIKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UserID    string `json:"user_id"`
	Plaintext string `json:"plaintext"`
	CreatedAt string `json:"created_at"`
}

// APIKeyListItem is one key as returned by the list endpoint. The plaintext
// and hash are never exposed here — only metadata an admin would want.
type APIKeyListItem struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

// ListAPIKeysResponse is the body returned by GET /v1/tenants/{tid}/api-keys.
type ListAPIKeysResponse struct {
	APIKeys []APIKeyListItem `json:"api_keys"`
}

// TenantQuotaResponse is the body of GET and PUT /v1/tenants/{tid}/quota.
type TenantQuotaResponse struct {
	// MaxStorageGB is the storage quota in GB; 0 means no quota.
	MaxStorageGB int `json:"max_storage_gb"`
	// UsedStorageGB is the on-disk usage in GB, refreshed periodically.
	UsedStorageGB float64 `json:"used_storage_gb"`
}

// APIKeyStatusTenantRow is one tenant in an APIKeyStatusResponse.
type APIKeyStatusTenantRow struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	// ActiveKeys counts the tenant's non-revoked API keys.
	ActiveKeys int `json:"active_keys"`
	// Locked reports that the tenant accepts only its own keys.
	Locked bool `json:"locked"`
}

// APIKeyStatusResponse is the body of GET /v1/admin/api-key-status. It
// carries counts and booleans only, never key material.
type APIKeyStatusResponse struct {
	// Tenants is scoped to the caller: all tenants for a node admin, only
	// its own for a tenant-authenticated caller.
	Tenants    []APIKeyStatusTenantRow `json:"tenants"`
	StrictAuth bool                    `json:"strict_auth"`
	// AdminTokenConfigured reports whether a node admin token is set.
	AdminTokenConfigured bool `json:"admin_token_configured"`
	MetricsPublic        bool `json:"metrics_public"`
	// TenantKeyEnforcement is false only when the operator turned the
	// per-tenant key rule off.
	TenantKeyEnforcement bool `json:"tenant_key_enforcement"`
}

// ListAllBackupsRequest filters GET /v1/admin/backups.
type ListAllBackupsRequest struct {
	// TenantID restricts results to one tenant. A tenant-authenticated
	// caller naming another tenant gets an empty list, not an error.
	TenantID string
	IndexID  string
	// Limit is the page size, 1 to 1000; zero uses the server default (200).
	Limit int
}

// UpdateTenantQuotaRequest is the body for PUT /v1/tenants/{tid}/quota.
type UpdateTenantQuotaRequest struct {
	// MaxStorageGB is the new quota in GB; 0 removes it. The field is always
	// sent, so 0 reaches the server.
	MaxStorageGB int `json:"max_storage_gb"`
}

// AdminBackupRow is one backup in the admin cross-tenant list.
type AdminBackupRow struct {
	// ID is the opaque backup id, also the storage key prefix.
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	// TenantName is empty when the owning tenant no longer exists.
	TenantName string `json:"tenant_name"`
	IndexID    string `json:"index_id"`
	// IndexName is empty when the owning index no longer exists.
	IndexName string    `json:"index_name"`
	CreatedAt time.Time `json:"created_at"`
	TotalSize int64     `json:"total_size"`
	NumChunks int       `json:"num_chunks"`
}

// AdminBackupList is the body returned by GET /v1/admin/backups.
type AdminBackupList struct {
	Backups []AdminBackupRow `json:"backups"`
	// NextCursor is empty on the last page.
	NextCursor string `json:"next_cursor"`
	// Total counts the backups matching the filters, before paging.
	Total int `json:"total"`
}

// BackupScheduleStatus is the body returned by GET /v1/admin/backups/status.
// Enabled, Keep, RunsOK and RunsFailed are always present; the remaining
// fields stay zero or nil until the scheduler records them.
type BackupScheduleStatus struct {
	// Enabled is true once a scheduler runs (--backup-dir and --backup-cron).
	Enabled bool `json:"enabled"`
	// Schedule is the cron spec the scheduler started with.
	Schedule string `json:"schedule,omitempty"`
	// Keep is the number of backups retained per index after each run.
	Keep            int        `json:"keep"`
	IntervalSeconds int64      `json:"interval_seconds,omitempty"`
	NextRun         *time.Time `json:"next_run,omitempty"`
	LastRun         *time.Time `json:"last_run,omitempty"`
	LastSuccess     *time.Time `json:"last_success,omitempty"`
	// LastRunOK is nil until a run has happened.
	LastRunOK *bool `json:"last_run_ok,omitempty"`
	// RunsOK and RunsFailed count runs since the process started.
	RunsOK     int64 `json:"runs_ok"`
	RunsFailed int64 `json:"runs_failed"`
}
