# Changelog

All notable changes to the `graphann` Go SDK are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
