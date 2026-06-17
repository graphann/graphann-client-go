# AGENTS.md — graphann-go for coding agents

LLM-usage guide for the Go client SDK. Every snippet below uses real method
names and signatures from this package (`package graphann`). Import path:
`github.com/graphann/graphann-client-go`. Package name in code: `graphann`.

All methods take a `context.Context` first and return `(*T, error)` (or just
`error` for `DeleteIndex` / `RevokeAPIKey`).

## Install

```bash
go get github.com/graphann/graphann-client-go@v0.8.0
```

## Client construction + auth

`NewClient(opts ...Option)` requires at least `WithBaseURL`. `WithAPIKey`
sets the tenant ID and API key sent as `X-Tenant-ID` / `X-API-Key` on every
request. Call `Close()` when done.

```go
import (
    "context"
    graphann "github.com/graphann/graphann-client-go"
)

c, err := graphann.NewClient(
    graphann.WithBaseURL("https://graphann.example.com"),
    graphann.WithAPIKey("t_demo", "gak_secret"),
)
if err != nil {
    return err
}
defer c.Close()

ctx := context.Background()
```

Other options: `WithHTTPClient`, `WithTLSConfig`, `WithRetryPolicy`,
`WithUserAgent`, `WithLogger`, `WithSingleflight`, `WithQueryCache`,
`WithMetricsHook`. Base URL and creds can also come from `GRAPHANN_BASE_URL`,
`GRAPHANN_TENANT_ID`, `GRAPHANN_API_KEY` (see README).

## Create tenant

```go
tnt, err := c.CreateTenant(ctx, graphann.CreateTenantRequest{Name: "demo"})
// tnt.ID is the tenant ID. CreateTenant is idempotent when you pass ID.
```

## Create index

`Compression` and `Approximate` are pointers — leave nil to take the server
default. Compression values: `"none"`, `"scalar"`, `"binary"`, `"pq"`,
`"recompute"`.

```go
approx := true
compression := "scalar"
idx, err := c.CreateIndex(ctx, tnt.ID, graphann.CreateIndexRequest{
    Name:        "docs",
    Compression: &compression,
    Approximate: &approx,
})
// idx.ID is the index ID used by the calls below.
```

## Ingest text documents

```go
res, err := c.AddDocuments(ctx, tnt.ID, idx.ID, graphann.AddDocumentsRequest{
    Documents: []graphann.Document{
        {ID: "doc-1", Text: "the quick brown fox", Metadata: map[string]any{"lang": "en"}},
        {ID: "doc-2", Text: "lorem ipsum dolor sit amet"},
    },
})
// res.Added, res.ChunkIDs ([]string), res.ExternalIDs (sharded ingest only).
```

Set `Document.Upsert = true` to make ingest idempotent by external ID (deletes
prior chunks with the same ID first).

## Ingest precomputed vectors

Set `Document.Vector` on EVERY document in the batch (mixed batches are
rejected with 400). Vector length must match the index dimension; the first
ingest into a fresh index fixes it. On this path the per-document `Upsert`
flag is ignored (precomputed inserts are upsert-by-external-ID already).

```go
res, err := c.AddDocuments(ctx, tnt.ID, idx.ID, graphann.AddDocumentsRequest{
    Documents: []graphann.Document{
        {ID: "v-1", Vector: []float32{0.1, 0.2, 0.3 /* ... */}},
        {ID: "v-2", Vector: []float32{0.4, 0.5, 0.6 /* ... */}},
    },
})
```

## Search (text, vector, rerank, ef_search)

Set either `Query` (text) or `Vector` (precomputed embedding). `Rerank` only
applies to the text `Query` path; `CandidateK` / `RerankK` are effective only
when `Rerank` is true. `EfSearch` overrides the HNSW beam width per query.

```go
sr, err := c.Search(ctx, tnt.ID, idx.ID, graphann.SearchRequest{
    Query:      "brown fox",
    K:          10,
    EfSearch:   128,
    Rerank:     true,
    CandidateK: 50,
    RerankK:    10,
    Filter: graphann.SearchFilter{
        RepoIDs: []string{"repo-a"},
        Equals:  map[string]string{"lang": "en"},
    },
})
for _, hit := range sr.Results {
    // hit.Score is always first-stage cosine. hit.RerankScore is non-nil
    // only when the server actually reranked this result.
    _ = hit.ID
    _ = hit.Score
    _ = hit.RerankScore
}
```

## Bulk ingest with defer_save / bulk + FlushIndex

`DeferSave` skips the per-batch save (data stays in memory, still searchable).
`Bulk` implies `DeferSave` and additionally defers the HNSW insert so the
delta graph is built once at flush — bulk data is not searchable until then,
except via build-on-read (the first search against a pending build triggers
it). Persist with `FlushIndex` after the batch sequence.

```go
for _, batch := range batches {
    if _, err := c.AddDocuments(ctx, tnt.ID, idx.ID, graphann.AddDocumentsRequest{
        Documents: batch,
        Bulk:      true, // implies DeferSave
    }); err != nil {
        return err
    }
}
fr, err := c.FlushIndex(ctx, tnt.ID, idx.ID) // builds the deferred graph + persists
_ = fr // fr.Flushed
```

`RebuildGraph` rebuilds the HNSW graph in place; `CompactIndex` runs
compaction (409 → `ErrCompactInProgress`, which also matches `ErrConflict`).

## API keys (create / list / revoke)

The create response carries the secret in `Plaintext` exactly ONCE — the
server stores only an argon2id hash, so persist it client-side immediately.
Admin-only endpoints.

```go
created, err := c.CreateAPIKey(ctx, tnt.ID, graphann.CreateAPIKeyRequest{
    UserID: "u_1", // optional; "" allowed
    Name:   "ci-runner",
})
// created.ID, created.Name, created.UserID, created.Plaintext, created.CreatedAt
secret := created.Plaintext // store NOW — not recoverable later

list, err := c.ListAPIKeys(ctx, tnt.ID)
for _, k := range list.APIKeys { // wrapper field is APIKeys
    _ = k.ID
    _ = k.UserID
    _ = k.Name
    _ = k.CreatedAt  // string
    _ = k.LastUsedAt // string, omitempty
}

err = c.RevokeAPIKey(ctx, tnt.ID, created.ID) // returns only error
```

## Error handling idioms

Errors wrap package sentinels — match with `errors.Is`, or pull structured
detail with `errors.As(&graphann.APIError{})`.

```go
import "errors"

_, err := c.GetIndex(ctx, tnt.ID, "missing")
if errors.Is(err, graphann.ErrNotFound) {
    // 404
}

var apiErr *graphann.APIError
if errors.As(err, &apiErr) {
    _ = apiErr.Status     // HTTP status
    _ = apiErr.Code       // server error code, e.g. "not_found"
    _ = apiErr.Message
    _ = apiErr.RequestID  // X-Request-ID when present
    _ = apiErr.RetryAfter // parsed Retry-After on 429/503
}
```

Sentinels: `ErrUnauthorized` (401), `ErrNotFound` (404), `ErrConflict` (409),
`ErrCompactInProgress` (409 on compact, also matches `ErrConflict`),
`ErrRateLimited` (429), `ErrBadRequest` (400), `ErrValidation`.

## Key gotchas

- `CreateAPIKey` returns the plaintext token ONCE in `Plaintext`. Lose it and
  the only recovery is to rotate (revoke + create).
- Request body cap is 16 MB server-side. For precomputed-vector ingest this
  limits a batch to roughly 1700 documents — split larger loads.
- `DeleteChunks` always posts to `.../chunks/0`; the path ID is a placeholder
  and the `DeleteChunksRequest.ChunkIDs` body is the source of truth.
- Bulk-ingested data (`Bulk: true`) is not searchable until `FlushIndex` or a
  build-on-read search. Call `FlushIndex` when you need durability and full
  searchability after a bulk load.
- `Search.Rerank` is a no-op on the text path when the server has no reranker
  configured, and is ignored entirely on vector-only requests and on the
  sharded scatter-gather path.
- `CreateIndexRequest.Compression` / `Approximate` are pointers: nil means
  "server default", not "off".
