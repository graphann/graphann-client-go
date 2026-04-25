// Package graphann is the official Go client SDK for the GraphANN vector
// database. It speaks the v1 HTTP API and exposes idiomatic Go types and
// methods for tenant, index, document, search, job, cluster, and settings
// operations.
//
// # Quickstart
//
//	c, err := graphann.NewClient(
//	    graphann.WithBaseURL("http://localhost:38888"),
//	    graphann.WithAPIKey("t_my-tenant", "my-api-key"),
//	)
//	if err != nil { panic(err) }
//	defer c.Close()
//
//	ctx := context.Background()
//	_, _ = c.Health(ctx)
//
//	idx, _ := c.CreateIndex(ctx, "t_my-tenant", graphann.CreateIndexRequest{Name: "demo"})
//	_, _ = c.AddDocuments(ctx, "t_my-tenant", idx.ID, graphann.AddDocumentsRequest{
//	    Documents: []graphann.Document{{Text: "hello world"}},
//	})
//	res, _ := c.Search(ctx, "t_my-tenant", idx.ID, graphann.SearchRequest{Query: "hello", K: 5})
//	fmt.Printf("got %d hits\n", res.Total)
//
// # Authentication
//
// GraphANN supports two operational modes:
//
//   - Authenticated mode: pass WithAPIKey(tenantID, apiKey). Headers
//     X-Tenant-ID and X-API-Key will be sent on every request.
//   - Unauthenticated mode: omit WithAPIKey. The default tenant configured
//     on the server is used.
//
// # Retries and rate limiting
//
// The SDK retries on 429 (rate limited) and 503 (service unavailable)
// using exponential backoff with jitter, honouring Retry-After when the
// server provides it. Configure with WithRetryPolicy.
//
// # Concurrent query coalescing
//
// When WithSingleflight is configured, concurrent identical search
// requests within a window are coalesced into a single network round-trip.
//
// # Result caching
//
// When WithQueryCache is configured, search responses are cached locally
// (LRU + TTL) keyed on (tenantID, indexID, request fingerprint). Cache
// is invalidated implicitly by TTL only.
package graphann
