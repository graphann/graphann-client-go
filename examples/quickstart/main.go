// Quickstart demo: create a tenant, an index, ingest 10 documents,
// search, switch the embedding model, and re-search.
//
// Configure via env:
//
//	GRAPHANN_BASE_URL  (default: http://localhost:38888)
//	GRAPHANN_TENANT_ID (optional; created on the fly when absent)
//	GRAPHANN_API_KEY   (optional)
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	graphann "github.com/graphann/graphann-client-go"
)

func main() {
	base := getenv("GRAPHANN_BASE_URL", "http://localhost:38888")

	opts := []graphann.Option{
		graphann.WithBaseURL(base),
		graphann.WithRetryPolicy(graphann.DefaultRetryPolicy()),
		graphann.WithSingleflight(100 * time.Millisecond),
		graphann.WithQueryCache(64, 30*time.Second),
	}
	if tID, key := os.Getenv("GRAPHANN_TENANT_ID"), os.Getenv("GRAPHANN_API_KEY"); tID != "" && key != "" {
		opts = append(opts, graphann.WithAPIKey(tID, key))
	}

	c, err := graphann.NewClient(opts...)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hr, hErr := c.Health(ctx)
	if hErr != nil {
		log.Fatalf("Health: %v", hErr)
	}
	fmt.Printf("server health: %s\n", hr.Status)

	t, err := c.CreateTenant(ctx, graphann.CreateTenantRequest{
		Name: "quickstart-" + time.Now().UTC().Format("20060102150405"),
	})
	if err != nil {
		log.Fatalf("CreateTenant: %v", err)
	}
	fmt.Printf("tenant: %s (%s)\n", t.ID, t.Name)

	compression := "pq"
	approx := true
	idx, err := c.CreateIndex(ctx, t.ID, graphann.CreateIndexRequest{
		Name:        "demo",
		Description: "quickstart corpus",
		Compression: &compression,
		Approximate: &approx,
	})
	if err != nil {
		log.Fatalf("CreateIndex: %v", err)
	}
	fmt.Printf("index: %s compression=%s approximate=%v\n", idx.ID, idx.Compression, idx.Approximate)

	// Demo: upsert a resource atomically (create or replace).
	upsertRes, err := c.UpsertResource(ctx, t.ID, idx.ID, "intro-doc", graphann.UpsertResourceRequest{
		Text:     "GraphANN: storage-efficient vector search via on-demand embedding recomputation.",
		Metadata: map[string]string{"source": "quickstart"},
	})
	if err != nil {
		log.Printf("UpsertResource: %v (server may not support this endpoint yet)", err)
	} else {
		fmt.Printf("upsert: resource=%s operation=%s chunks_added=%d\n",
			upsertRes.ResourceID, upsertRes.Operation, upsertRes.ChunksAdded)
	}

	corpus := []graphann.Document{
		{ID: "doc-1", Text: "Vector databases enable similarity search at scale."},
		{ID: "doc-2", Text: "GraphANN traverses an HNSW graph to find neighbours."},
		{ID: "doc-3", Text: "Product quantization compresses vectors for storage."},
		{ID: "doc-4", Text: "LEANN recomputes embeddings on-demand for storage savings."},
		{ID: "doc-5", Text: "Hybrid retrieval mixes BM25 with semantic search."},
		{ID: "doc-6", Text: "An embedding maps a text into a dense vector."},
		{ID: "doc-7", Text: "Cosine distance is a common similarity metric."},
		{ID: "doc-8", Text: "Ollama runs language models locally on a developer machine."},
		{ID: "doc-9", Text: "Reranking with a cross-encoder improves top-k quality."},
		{ID: "doc-10", Text: "Cluster mode shards indexes across nodes for HA."},
	}
	addRes, err := c.AddDocuments(ctx, t.ID, idx.ID, graphann.AddDocumentsRequest{Documents: corpus})
	if err != nil {
		log.Fatalf("AddDocuments: %v", err)
	}
	fmt.Printf("added %d documents (%d chunks)\n", addRes.Added, len(addRes.ChunkIDs))

	res, err := c.Search(ctx, t.ID, idx.ID, graphann.SearchRequest{
		Query: "what is graph-based vector search?",
		K:     3,
	})
	if err != nil {
		log.Fatalf("Search: %v", err)
	}
	fmt.Println("initial search:")
	for _, hit := range res.Results {
		fmt.Printf("  [%.3f] %s — %s\n", hit.Score, hit.ID, truncate(hit.Text, 60))
	}

	// Optional: hot model swap. Skipped when an embedding backend is not
	// configured for the demo.
	if backend := os.Getenv("GRAPHANN_NEW_BACKEND"); backend != "" {
		modelName := getenv("GRAPHANN_NEW_MODEL", "nomic-embed-text")
		dim := envInt("GRAPHANN_NEW_DIM", 768)
		swap, err := c.SwitchEmbeddingModel(ctx, t.ID, idx.ID, graphann.SwitchEmbeddingModelRequest{
			Backend:   backend,
			Model:     modelName,
			Dimension: dim,
		})
		if err != nil {
			log.Fatalf("SwitchEmbeddingModel: %v", err)
		}
		fmt.Printf("swap job: %s status=%s\n", swap.JobID, swap.Status)

		// Poll the job to completion.
		deadline := time.Now().Add(5 * time.Minute)
		for {
			j, jErr := c.GetJob(ctx, swap.JobID)
			if jErr != nil {
				log.Fatalf("GetJob: %v", jErr)
			}
			fmt.Printf("  job=%s status=%s progress=%d/%d\n",
				j.JobID, j.Status, j.Progress.ChunksDone, j.Progress.ChunksTotal)
			if j.Status == graphann.JobStatusCompleted || j.Status == graphann.JobStatusFailed {
				if j.Status == graphann.JobStatusFailed {
					log.Fatalf("reembed job failed: %s", j.Error)
				}
				break
			}
			if time.Now().After(deadline) {
				log.Fatalf("reembed job timed out")
			}
			time.Sleep(5 * time.Second)
		}

		res2, err := c.Search(ctx, t.ID, idx.ID, graphann.SearchRequest{
			Query: "what is graph-based vector search?",
			K:     3,
		})
		if err != nil {
			log.Fatalf("Search after swap: %v", err)
		}
		fmt.Println("post-swap search:")
		for _, hit := range res2.Results {
			fmt.Printf("  [%.3f] %s — %s\n", hit.Score, hit.ID, truncate(hit.Text, 60))
		}
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	if err != nil {
		return def
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
