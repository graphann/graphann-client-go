# Changelog

All notable changes to the `graphann` Go SDK are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.10.1] - 2026-09-30

### Added

- `ListAllBackupsPage` fetches one page of `GET /v1/admin/backups` and
  returns `AdminBackupList`, including `Total` and `NextCursor`. The
  `ListAllBackups` iterator now calls it.
- A test checks that the `ErrorCode*` constants match the spec's
  `APIError.code` enum exactly.

### Changed

- **Breaking:** removed the `IndexStatusEmpty` constant. The server never
  emits `"empty"`; the spec status enum is `pending`, `building`, `ready`,
  `error` and `deleted`. Use `IndexStatusPending` for a new index.
- Generated types are regenerated from the refreshed `spec.yaml`:
  `APIError.code` gains `bad_gateway` and `unprocessable_entity`, and
  `TenantQuotaResponse`, `APIKeyStatusTenantRow` and `APIKeyStatusResponse`
  mark their fields required.
- The README documents `GetEmbedSpaceAdmin`, `CompactAllIndexes` and
  `ListAllBackupsPage`. AGENTS.md now describes `CreateAPIKeyRequest.UserID`
  as required, and lists the 403, 422 and 507 sentinels.

## [0.10.0] - 2026-09-30

### Added

- Client methods for five new server operations: `GetTenantQuota` and
  `UpdateTenantQuota` (`/v1/tenants/{tid}/quota`), `GetAPIKeyStatus`
  (`GET /v1/admin/api-key-status`), `ListAllBackups` (`GET /v1/admin/backups`,
  a keyset-paginated iterator) and `GetBackupStatus`
  (`GET /v1/admin/backups/status`).
- New types: `TenantQuotaResponse`, `UpdateTenantQuotaRequest`,
  `APIKeyStatusResponse`, `APIKeyStatusTenantRow`, `AdminBackupRow`,
  `AdminBackupList`, `BackupScheduleStatus` and `ListAllBackupsRequest`.
- `CreateIndexRequest` and `Index` carry `ChunkSize` and `ChunkOverlap`.
- `AddDocumentsResponse.Warnings`.
- `SearchRequest.GroupBy` and `MaxPerDoc`, with `SearchGroupByDocument`.
- `CreateAPIKeyRequest.Role` with the `APIKeyRole*` constants.
- `RestoreBackupRequest.Name`, and `EmbedSpaceIndexRow.NumChunks`.
- `IndexStatusResponse` exposes the embed-space fields the server already
  returned, with `EmbedSpaceState` constants that include `EmbedSpaceEmpty`.
  `EmbedSpaceAdminResponse.Counts` now holds seven state keys.
- `ErrInsufficientStorage` (HTTP 507) and `ErrUnprocessable` (HTTP 422)
  sentinels, and `ErrorCode*` constants for `APIError.Code`.
- Fields the server already returned or accepted but the SDK dropped:
  `Index` (`Error`, `LastCompactedAt`, `CreatedBy`, `Metadata`,
  `VectorsExternal`, `Embedding*`), `LiveStatsResponse` size and rebuild
  fields, `UpsertResourceRequest` metadata fields, `MultiSearchRequest.MaxResults`,
  and the `IndexStatusPending` and `IndexStatusDeleted` constants.

### Changed

- Generated types are regenerated from the refreshed `spec.yaml`.
- `APIError.Message` for `GET /ready` 503 responses now carries the server's
  `reason` (for example "shutting down") instead of the generic status text.
- The `CreateAPIKeyRequest.UserID` comment now says the field is required,
  which matches the server.

## [0.9.1] - 2026-09-07

### Fixed

- Regeneration and the staleness check support standalone GitHub checkouts
  with `api/openapi/spec.yaml`, while retaining the shared monorepo schema
  fallback and `GRAPHANN_SPEC` override.
- The runtime User-Agent reports 0.9.1. Installation instructions use the
  GitHub `v0.9.1` tag instead of a local module replacement.

### Distribution

- This SDK is distributed through tagged GitHub source releases. See
  [installation](README.md#install).

## [0.9.0] - 2026-08-11

### Added

- The nine endpoints that no SDK, in any language, previously implemented. All
  nine were absent because `api/openapi/spec.yaml` omitted them, not because any
  one client lagged: backup lifecycle (list, create, delete, restore),
  `POST .../search/batch`, `POST .../indexes/compact-all`, `GET /v1/license/status`,
  `GET /v1/license/audit`, and `GET /v1/admin/embed-space`.
- Six request fields that were likewise unreachable:
  - `hybrid` — fuses dense results with BM25 lexical ranking via RRF. Text
    queries only, default off. Previously undocumented anywhere in the SDKs.
  - `vector_b64` — query vector as base64 little-endian float32, mutually
    exclusive with `vector`. Exists because JSON float parsing measured 17.5% of
    server CPU under load.
  - `omit_text` (on the search filter) — drops chunk text from results. Text is
    ~75% of the response body at k=10, and omitting it also skips a zstd
    decompression server-side.
  - `embedding_endpoint`, `embedding_dimension`, `embedding_api_key_env` — the
    per-index embedder override, which is in production use.

### Changed

- Generated wire types now supplement the handwritten API using the server's
  OpenAPI spec. The spec is vendored at `api/openapi/spec.yaml` in this
  repository, so generation is reproducible from a fresh clone.
- A staleness check regenerates the types and byte-compares them against what is
  committed, failing if they differ. It runs as part of the normal test task.

### Fixed

- Request bodies remain plain JSON above 64 KiB; gzip response decoding is
  retained. The server does not support gzip request bodies.
- Successful mutations invalidate all cached searches. Generation-scoped
  singleflight prevents new reads from joining pre-mutation requests, and
  delayed responses cannot repopulate the cache after invalidation.
- Resource IDs are path-escaped, including slashes, query delimiters, and `%`.
- `SearchResponse.RerankApplied *bool` preserves omitted, false, and true.
  Sharded text searches support coordinator-side reranking.
- The runtime User-Agent was updated to 0.9.0. Installation instructions at
  that time used a local checkout.
- The search documentation described "hybrid search" as supplying either a text
  query or a vector. That is dense search with two input modes; it is not what
  hybrid means. The wording came from the spec, which titled the plain search
  operation "Hybrid search" and never documented the real `hybrid` flag. Corrected
  here and at the source.

### Notes

Generated wire types supplement the handwritten API; they do not replace
every handwritten request or response type. For the current GitHub release,
see [installation](README.md#install).

## [0.8.0] - 2026-06-17

### Fixed

- API-key wire contract corrected to match the server
  (`internal/server/apikey_handlers.go`). The create response now decodes
  the one-time secret from the `plaintext` JSON field (was `key`), the list
  response wrapper key is `api_keys` (was `keys`), and the request/response
  structs carry only the fields the server actually emits. Invented fields
  that the server never sends were removed: `prefix`, `expires_at`,
  `tenant_id`, and the list `total`.
  - `CreateAPIKey` now returns `*CreateAPIKeyResponse`
    (`{id, name, user_id, plaintext, created_at}`) instead of `*APIKey`.
  - `ListAPIKeysResponse` now holds `APIKeys []APIKeyListItem`
    (`{id, user_id, name, created_at, last_used_at}`) instead of
    `Keys []APIKey` + `Total`.
  - The `APIKey` struct was removed.

  This is mildly breaking: code that read `resp.Plaintext` on the old
  `*APIKey` create result, or `resp.Keys` / `resp.Total` on the list
  result, must move to `CreateAPIKeyResponse.Plaintext` and
  `ListAPIKeysResponse.APIKeys`. The method names are unchanged.

### Added

- `AGENTS.md` — an LLM-usage guide for coding agents, grounded in the
  current SDK surface.

## [0.7.0] - 2026-06-10

### Added

- `Document.Vector []float32` — precomputed-vector ingest. When every
  document in an `AddDocuments` batch carries a non-empty vector, the
  server skips embedding and ingests the vectors directly. Mixed batches
  (some with, some without) are rejected with 400 `validation_error`.
  Vector length must match the index dimension once set; the first
  ingest into a fresh index fixes the dimension. Precomputed inserts are
  idempotent by external ID, so the per-document `Upsert` flag is
  ignored on this path. Note the server's 16 MB request-body cap limits
  precomputed batches to roughly 1700 documents.
- `AddDocumentsRequest.DeferSave` / `AddDocumentsRequest.Bulk` — bulk-load
  ingest options. `DeferSave` skips the per-batch save (data stays in
  memory, still searchable); `Bulk` additionally defers the per-node HNSW
  insert so the delta graph is built once at flush. Bulk data is not
  searchable until the graph is built — except via build-on-read: the
  first search against a pending deferred build triggers it server-side.
  Defaults (both false) preserve the existing behaviour.
- `Client.FlushIndex` — `POST .../indexes/{iid}/flush`. Persists the live
  index's in-memory delta and builds any pending deferred bulk graph in
  the same flush. Pairs with `DeferSave`/`Bulk`; safe on a clean index.
- `Client.RebuildGraph` — `POST .../indexes/{iid}/rebuild-graph`. In-place
  delta-HNSW rebuild for indexes ingested before the 2026-06
  neighbor-selection fix. Returns 409 (`ErrConflict`) while a compaction
  is running.
- `AddDocumentsResponse.ExternalIDs []string` — present only when the
  server minted at least one ExternalID (sharded ingest of ID-less
  documents). Positionally aligned with the request array, mixing minted
  and client-supplied IDs; persist these as the durable document IDs.
  Absent (nil) on unsharded deployments.
- `SearchRequest.EfSearch` — per-query HNSW beam-width override. Zero
  uses the server default (`--search-ef`, default 64); the server clamps
  negative values to the default and caps at 2000. Binary/PQ flat scans
  ignore ef entirely.
- `SearchResponse.Partial *bool`, `ShardsTotal *int`, `ShardsOK *int`,
  `DegradedShards []string` — optional sharded scatter-gather fields,
  populated only when a clustered server fans the query out across more
  than one shard (`Partial != nil` detects that path). All other
  deployments keep the exact pre-sharding `{results, total}` shape.
  Results are deduplicated by external ID keeping the highest score.
  Sharded text searches now support coordinator-side reranking.

### Changed

- `Client.CompactIndex` now maps any HTTP 409 from the compact endpoint
  to an error matching **both** `ErrCompactInProgress` and `ErrConflict`.
  The server signals an in-flight compaction with the generic `conflict`
  code, which previously only matched `ErrConflict`. Existing
  `errors.Is(err, ErrConflict)` checks keep working.
- `Version` constant bumped to `0.7.0` (was stale at `0.4.0`).

## [0.6.0] - 2026-05-01

### Added

- `SearchRequest.Rerank`, `SearchRequest.CandidateK`, `SearchRequest.RerankK`
  fields wire the optional cross-encoder reranker. When the server has a
  reranker configured (via `--reranker-url`), set `Rerank: true` to rescore
  the top-`CandidateK` HNSW candidates with the reranker and return the
  top-`RerankK` (or top-`K`). Defaults: `CandidateK = max(4*K, 50)`,
  `RerankK = K`. No-op against non-rerank-aware servers — safe to roll
  out unconditionally.
- `SearchResult.RerankScore *float32` — populated only when the server
  actually applied the reranker. Carries the cross-encoder's native
  relevance score (different scale from cosine, typically -10..10 for
  bge-reranker-v2-m3). When non-nil, it also reflects the result
  ordering. When nil, the response is ordered by `Score` (cosine).

### Unchanged

- `SearchResult.Score` is still always the first-stage cosine
  similarity, regardless of rerank state. Existing client code that
  only reads `Score` keeps working — even when accidentally hitting a
  rerank-enabled endpoint.

## [0.5.0] - 2026-04-30

### Changed (BREAKING)

- `Client.CleanupOrphans` signature is now
  `CleanupOrphans(ctx, minAge time.Duration, dryRun bool)`. Pass `0, false`
  to preserve the previous behaviour (server defaults: 1h min-age, real
  sweep). Server enforces a 5-minute floor on positive `minAge` values.
- `CleanupOrphansResponse` gains `MinAge` (Go duration string echoed by
  the server) and `DryRun` fields. Both are present on every response.

## [0.3.0] - 2026-04-28

### Removed (BREAKING)

- `Client.SearchText` — endpoint `POST .../search/text` removed server-side
  (strict subset of `/search`). Use `Client.Search` with `req.Query` set.
- `Client.SearchVector` — endpoint `POST .../search/vector` removed server-side
  (strict subset of `/search`). Use `Client.Search` with `req.Vector` set.
- `Client.BuildIndex` — endpoint `POST .../build` was a no-op stub; removed.
  `BuildIndexResponse` type also removed.

### Added

- `Client.UpsertResource` — `PUT .../indexes/{iid}/resources/{resID}`. Atomic
  upsert: creates or replaces all chunks for a resource by external ID.
- `UpsertResourceRequest` / `UpsertResourceResponse` types for the new endpoint.
- `ErrCompactInProgress` sentinel — returned when `CompactIndex` receives HTTP
  409 with code `compact_in_progress`. Retryable.
- `CreateIndexRequest.Compression` / `CreateIndexRequest.Approximate` fields.
- `UpdateIndexRequest.Compression` / `UpdateIndexRequest.Approximate` fields
  (PATCH endpoint is now fully functional server-side).
- `Index.Compression` / `Index.Approximate` response fields.
- `SearchFilter.Equals` — generic metadata pre-filter (`map[string]string`).

## [0.2.0] - 2026-04-25

### Changed (BREAKING)

- Renamed `Client.GetPending` to `Client.GetPendingStatus` for naming
  parity with the Python (`get_pending_status`) and TypeScript
  (`getPendingStatus`) SDKs. Same signature, same return type, same
  behavior — only the symbol name changed.

  Migration: `s/GetPending/GetPendingStatus/` at every call site.

## [0.1.1] - 2026-04-25

### Added

- `Client.BuildIndex` — `POST /v1/tenants/{tenantID}/indexes/{indexID}/build`.
  Kicks off an asynchronous index build; poll `GetIndexStatus` for completion.
- `Client.CleanupOrphans` — `POST /v1/admin/cleanup-orphans`. Admin-only sweep
  of stale compaction artifacts (1h minimum-age guard server-side).
- `BuildIndexResponse` and `CleanupOrphansResponse` types covering the new
  endpoints.

### Changed

- `Client.GetLLMSettings`, `Client.UpdateLLMSettings`, `Client.DeleteLLMSettings`
  now hit the canonical org-scoped routes that the server actually registers:
  - `GET /v1/orgs/{orgID}/llm-settings`
  - `PATCH /v1/orgs/{orgID}/llm-settings` (was `PUT /v1/orgs/{orgID}/settings/llm`).
    The server applies partial-merge semantics — supplied fields are merged
    onto current settings.
  - `DELETE /v1/orgs/{orgID}/llm-settings`

  Function signatures are unchanged. The previous path/method combination
  was never wired into the server router and would have returned 404 in
  production.

## [0.1.0] - 2026-04-25

### Added

- Initial public release of the synchronous `Client`.
- Full surface coverage for the v1 HTTP API: tenants, indexes, documents,
  search (hybrid / text / vector), batch import, index maintenance
  (compact / clear), live stats, hot model switching, async jobs, cluster
  introspection, org-level multi-source sync, LLM settings, API-key
  administration.
- Hardened HTTP transport: tunable timeouts, connection pooling, automatic
  gzip on request bodies larger than 64 KiB, exponential backoff with
  full jitter, and `Retry-After` header handling on 429/503.
- Typed exception hierarchy backed by sentinel errors and an `APIError`
  wrapper (`ErrConfig`, `ErrNetwork`, `ErrServer`, `ErrNotFound`,
  `ErrConflict`, `ErrPayloadTooLarge`, `ErrRateLimited`, `ErrValidation`,
  `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`).
- Cursor pagination via the generic `Iter[T]` helper for `ListDocuments`
  and `ListJobs`.
- Singleflight coalescing of duplicate concurrent search calls plus an
  opt-in LRU + TTL response cache.
- Optional `MetricsHook` for Prometheus / OpenTelemetry integration.
