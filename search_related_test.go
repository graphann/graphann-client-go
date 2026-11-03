package graphann

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/graphann/graphann-client-go/generated"
)

func TestSearchRequestRelatedFieldsJSON(t *testing.T) {
	b, err := json.Marshal(SearchRequest{
		RelatedTo: "doc-1", Hops: 2, Fanout: 7, NovelOnly: true, Diversity: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"related_to": "doc-1", "hops": 2.0, "fanout": 7.0, "novel_only": true, "diversity": 0.5}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
	delete(m, "filter") // struct-typed filter always marshals as {}
	if len(m) != len(want) {
		t.Errorf("unexpected extra keys: %v", m)
	}

	b, _ = json.Marshal(SearchRequest{Query: "q"})
	for _, k := range []string{"related_to", "hops", "fanout", "novel_only", "diversity"} {
		if strings.Contains(string(b), k) {
			t.Errorf("zero %s must be omitted, got %s", k, b)
		}
	}
}

func TestSearchRelatedSendsBodyAndDecodesWalk(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"results":[{"id":"d2","score":0.42,"support":3,"direct_score":0.1,
			"path":[{"id":"d9","score":0.8},{"id":"d2","score":0.525}]}],
			"total":1,"exhaustive":true,"shards_probed":2,"shards_total":4}`))
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	res, err := c.Search(context.Background(), "t", "i", SearchRequest{RelatedTo: "d1", Hops: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, hasQuery := body["query"]; hasQuery || body["related_to"] != "d1" || body["hops"] != 2.0 {
		t.Errorf("request body = %v", body)
	}
	if res.Exhaustive == nil || !*res.Exhaustive {
		t.Errorf("Exhaustive = %v, want true", res.Exhaustive)
	}
	if res.ShardsProbed == nil || *res.ShardsProbed != 2 {
		t.Errorf("ShardsProbed = %v, want 2", res.ShardsProbed)
	}
	r := res.Results[0]
	if r.Support != 3 || r.DirectScore == nil || *r.DirectScore != 0.1 {
		t.Errorf("support/direct_score = %d/%v", r.Support, r.DirectScore)
	}
	if len(r.Path) != 2 || r.Path[0].ID != "d9" || r.Path[1].Score != 0.525 {
		t.Errorf("path = %+v", r.Path)
	}
}

func TestSearchResponseWithoutWalkFieldsLeavesThemNil(t *testing.T) {
	var res SearchResponse
	if err := json.Unmarshal([]byte(`{"results":[{"id":"a","score":1}],"total":1}`), &res); err != nil {
		t.Fatal(err)
	}
	if res.Exhaustive != nil || res.ShardsProbed != nil || res.Results[0].DirectScore != nil || res.Results[0].Path != nil {
		t.Errorf("walk fields must stay unset: %+v", res)
	}
}

func TestBatchSearchPerQueryRelatedAndGrouping(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"responses":[{"results":[],"total":0,"exhaustive":false}]}`))
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	rel, group, per := "d1", generated.SearchRequestGroupByDocument, 2
	res, err := c.BatchSearch(context.Background(), "t", "i", BatchSearchRequest{
		Queries: []generated.SearchRequest{{RelatedTo: &rel, GroupBy: &group, MaxPerDoc: &per}},
	})
	if err != nil {
		t.Fatal(err)
	}
	q := body["queries"].([]any)[0].(map[string]any)
	if q["related_to"] != "d1" || q["group_by"] != "document" || q["max_per_doc"] != 2.0 {
		t.Errorf("batch query = %v", q)
	}
	if e := (*res.Responses)[0].Exhaustive; e == nil || *e {
		t.Errorf("Exhaustive = %v, want false", e)
	}
}

func TestWithAdminTokenSetsHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-GraphANN-Admin-Token")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, WithAdminToken("0123456789012345678901234"))
	if _, err := c.RunAdminGC(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got != "0123456789012345678901234" {
		t.Errorf("admin token header = %q", got)
	}

	got = "unset"
	c = newTestClient(t, srv)
	if _, err := c.RunAdminGC(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("header sent without option: %q", got)
	}
}
