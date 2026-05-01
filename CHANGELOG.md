# Changelog

All notable changes to the `graphann` Go SDK are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
