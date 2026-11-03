package graphann

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type specOp struct {
	method string
	path   string
}

// specOperations reads operationId -> METHOD + path template from the
// vendored spec without a YAML dependency. It relies on the spec layout:
// paths at two-space indent, methods at four, operationId at six.
func specOperations(t *testing.T) map[string]specOp {
	t.Helper()
	spec := filepath.Join("api", "openapi", "spec.yaml")
	if _, err := os.Stat(spec); os.IsNotExist(err) {
		spec = filepath.Join("..", "api", "openapi", "spec.yaml")
	}
	f, err := os.Open(spec)
	if err != nil {
		t.Fatalf("open vendored spec: %v", err)
	}
	defer f.Close()

	pathRe := regexp.MustCompile(`^  (/\S*):\s*$`)
	methodRe := regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
	opIDRe := regexp.MustCompile(`^      operationId:\s*(\S+)\s*$`)

	ops := map[string]specOp{}
	inPaths := false
	var curPath, curMethod string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, " ") && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			inPaths = strings.HasPrefix(line, "paths:")
			continue
		}
		if !inPaths {
			continue
		}
		if m := pathRe.FindStringSubmatch(line); m != nil {
			curPath, curMethod = m[1], ""
		} else if m := methodRe.FindStringSubmatch(line); m != nil {
			curMethod = strings.ToUpper(m[1])
		} else if m := opIDRe.FindStringSubmatch(line); m != nil && curMethod != "" {
			ops[m[1]] = specOp{method: curMethod, path: curPath}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan spec: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("no operations parsed from spec")
	}
	return ops
}

// TestSpecOperationCoverage calls the SDK method for every spec operation
// and checks the request line matches the spec template. A spec operation
// without a table entry fails with its operationId.
func TestSpecOperationCoverage(t *testing.T) {
	var mu sync.Mutex
	var got []specOp
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = append(got, specOp{r.Method, r.URL.Path})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	ctx := context.Background()

	const (
		tid, iid, oid, uid = "t1", "i1", "o1", "u1"
		rid, bid, kid, jid = "r1", "b1", "k1", "j1"
	)
	drain := func(it interface {
		Next(context.Context) bool
	}) error {
		it.Next(ctx)
		return nil
	}

	// Each entry: operationId -> SDK call. Path params use the constants
	// above; docID is 42, chunkID is 7 (DeleteChunks sends the 0 placeholder).
	calls := map[string]func() error{
		"healthCheck":       func() error { _, err := c.Health(ctx); return err },
		"readinessCheck":    func() error { _, err := c.Ready(ctx); return err },
		"getLicenseStatus":  func() error { _, err := c.GetLicenseStatus(ctx); return err },
		"getLicenseAudit":   func() error { _, err := c.GetLicenseAudit(ctx, 0); return err },
		"listTenants":       func() error { _, err := c.ListTenants(ctx); return err },
		"createTenant":      func() error { _, err := c.CreateTenant(ctx, CreateTenantRequest{}); return err },
		"getTenantQuota":    func() error { _, err := c.GetTenantQuota(ctx, tid); return err },
		"updateTenantQuota": func() error { _, err := c.UpdateTenantQuota(ctx, tid, UpdateTenantQuotaRequest{}); return err },
		"getTenant":         func() error { _, err := c.GetTenant(ctx, tid); return err },
		"deleteTenant":      func() error { _, err := c.DeleteTenant(ctx, tid); return err },
		"listIndexes":       func() error { _, err := c.ListIndexes(ctx, tid); return err },
		"createIndex":       func() error { _, err := c.CreateIndex(ctx, tid, CreateIndexRequest{}); return err },
		"getIndex":          func() error { _, err := c.GetIndex(ctx, tid, iid); return err },
		"deleteIndex":       func() error { return c.DeleteIndex(ctx, tid, iid) },
		"updateIndex":       func() error { _, err := c.UpdateIndex(ctx, tid, iid, UpdateIndexRequest{}); return err },
		"getIndexStatus":    func() error { _, err := c.GetIndexStatus(ctx, tid, iid); return err },
		"listDocumentsByPrefix": func() error {
			return drain(c.ListDocuments(ctx, tid, iid, ListDocumentsRequest{}))
		},
		"addDocuments":        func() error { _, err := c.AddDocuments(ctx, tid, iid, AddDocumentsRequest{}); return err },
		"bulkDeleteDocuments": func() error { _, err := c.BulkDeleteDocuments(ctx, tid, iid, BulkDeleteDocumentsRequest{}); return err },
		"bulkDeleteByExternalIDs": func() error {
			_, err := c.BulkDeleteByExternalIDs(ctx, tid, iid, BulkDeleteByExternalIDsRequest{})
			return err
		},
		"upsertResource": func() error {
			_, err := c.UpsertResource(ctx, tid, iid, rid, UpsertResourceRequest{})
			return err
		},
		"getDocument":    func() error { _, err := c.GetDocument(ctx, tid, iid, 42); return err },
		"deleteDocument": func() error { _, err := c.DeleteDocument(ctx, tid, iid, 42); return err },
		"getChunk":       func() error { _, err := c.GetChunk(ctx, tid, iid, 7); return err },
		"deleteChunk":    func() error { _, err := c.DeleteChunks(ctx, tid, iid, DeleteChunksRequest{}); return err },
		"search":         func() error { _, err := c.Search(ctx, tid, iid, SearchRequest{Query: "q"}); return err },
		"batchSearch":    func() error { _, err := c.BatchSearch(ctx, tid, iid, BatchSearchRequest{}); return err },
		"compactIndex":   func() error { _, err := c.CompactIndex(ctx, tid, iid); return err },
		"compactAllIndexes": func() error {
			_, err := c.CompactAllIndexes(ctx, tid)
			return err
		},
		"clearIndex":        func() error { _, err := c.ClearIndex(ctx, tid, iid); return err },
		"getLiveIndexStats": func() error { _, err := c.GetLiveStats(ctx, tid, iid); return err },
		"flushIndex":        func() error { _, err := c.FlushIndex(ctx, tid, iid); return err },
		"rebuildIndexGraph": func() error { _, err := c.RebuildGraph(ctx, tid, iid); return err },
		"importDocuments":   func() error { _, err := c.ImportDocuments(ctx, tid, iid, ImportDocumentsRequest{}); return err },
		"getPendingStatus":  func() error { _, err := c.GetPendingStatus(ctx, tid, iid); return err },
		"clearPending":      func() error { _, err := c.ClearPending(ctx, tid, iid); return err },
		"processPending":    func() error { _, err := c.ProcessPending(ctx, tid, iid); return err },
		"syncDocuments":     func() error { _, err := c.SyncDocuments(ctx, oid, SyncDocumentsRequest{}); return err },
		"searchMultiIndex":  func() error { _, err := c.MultiSearch(ctx, oid, uid, MultiSearchRequest{}); return err },
		"listUserIndexes":   func() error { _, err := c.ListUserIndexes(ctx, oid, uid); return err },
		"listSharedIndexes": func() error { _, err := c.ListSharedIndexes(ctx, oid); return err },
		"switchEmbeddingModel": func() error {
			_, err := c.SwitchEmbeddingModel(ctx, tid, iid, SwitchEmbeddingModelRequest{})
			return err
		},
		"listAllJobs":    func() error { return drain(c.ListJobs(ctx, ListJobsRequest{})) },
		"getJob":         func() error { _, err := c.GetJob(ctx, jid); return err },
		"listTenantJobs": func() error { return drain(c.ListJobs(ctx, ListJobsRequest{TenantID: tid})) },
		"runIndexGC":     func() error { _, err := c.RunIndexGC(ctx, tid, iid); return err },
		"runGlobalGC":    func() error { _, err := c.RunAdminGC(ctx); return err },
		"cleanupOrphans": func() error { _, err := c.CleanupOrphans(ctx, time.Hour, true); return err },
		"getEmbedSpaceAdmin": func() error {
			_, err := c.GetEmbedSpaceAdmin(ctx)
			return err
		},
		"getAPIKeyStatus": func() error { _, err := c.GetAPIKeyStatus(ctx); return err },
		"listAllBackups":  func() error { return drain(c.ListAllBackups(ctx, ListAllBackupsRequest{})) },
		"getBackupStatus": func() error { _, err := c.GetBackupStatus(ctx); return err },
		"listClusterNodes": func() error {
			_, err := c.GetClusterNodes(ctx)
			return err
		},
		"listClusterShards": func() error { _, err := c.GetClusterShards(ctx); return err },
		"clusterHealth":     func() error { _, err := c.GetClusterHealth(ctx); return err },
		"createBackup":      func() error { _, err := c.CreateBackup(ctx, tid, iid); return err },
		"listBackups":       func() error { _, err := c.ListBackups(ctx, tid); return err },
		"restoreBackup":     func() error { _, err := c.RestoreBackup(ctx, tid, bid, RestoreBackupRequest{}); return err },
		"deleteBackup":      func() error { _, err := c.DeleteBackup(ctx, tid, bid); return err },
		"createAPIKey":      func() error { _, err := c.CreateAPIKey(ctx, tid, CreateAPIKeyRequest{}); return err },
		"listAPIKeys":       func() error { _, err := c.ListAPIKeys(ctx, tid); return err },
		"revokeAPIKey":      func() error { return c.RevokeAPIKey(ctx, tid, kid) },
		"getLLMSettings":    func() error { _, err := c.GetLLMSettings(ctx, oid); return err },
		"updateLLMSettings": func() error { _, err := c.UpdateLLMSettings(ctx, oid, LLMSettings{}); return err },
		"deleteLLMSettings": func() error { _, err := c.DeleteLLMSettings(ctx, oid); return err },
	}

	params := map[string]string{
		"tenantID": tid, "indexID": iid, "orgID": oid, "userID": uid,
		"resourceID": rid, "backupID": bid, "keyID": kid, "jobID": jid,
		"docID": "42", "chunkID": "7",
	}
	paramRe := regexp.MustCompile(`\{(\w+)\}`)
	expand := func(op string, tmpl string) string {
		return paramRe.ReplaceAllStringFunc(tmpl, func(m string) string {
			name := m[1 : len(m)-1]
			if op == "deleteChunk" && name == "chunkID" {
				return "0" // DeleteChunks sends a placeholder id; the ids ride in the body.
			}
			v, ok := params[name]
			if !ok {
				t.Errorf("%s: no test value for path param %q", op, name)
			}
			return v
		})
	}

	ops := specOperations(t)
	ids := make([]string, 0, len(ops))
	for id := range ops {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		op := ops[id]
		call, ok := calls[id]
		if !ok {
			t.Errorf("spec operation %s (%s %s) has no SDK call in the coverage table", id, op.method, op.path)
			continue
		}
		mu.Lock()
		got = nil
		mu.Unlock()
		// A decode error on the empty {} body is irrelevant; only the
		// recorded request line is asserted.
		_ = call()
		mu.Lock()
		reqs := append([]specOp(nil), got...)
		mu.Unlock()
		if len(reqs) == 0 {
			t.Errorf("%s: SDK call sent no request", id)
			continue
		}
		want := specOp{op.method, expand(id, op.path)}
		if reqs[0] != want {
			t.Errorf("%s: sent %s %s, spec wants %s %s", id, reqs[0].method, reqs[0].path, want.method, want.path)
		}
	}
	for id := range calls {
		if _, ok := ops[id]; !ok {
			t.Errorf("coverage table has %s, which is not a spec operation", id)
		}
	}
}
