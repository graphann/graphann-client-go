//go:build integration

package graphann

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// Integration tests run only when GRAPHANN_BASE_URL (and optionally
// GRAPHANN_API_KEY + GRAPHANN_TENANT_ID) are set. They are skipped in
// regular `go test ./...` runs by the `integration` build tag.
//
// To run:
//
//	GRAPHANN_BASE_URL=http://localhost:38888 \
//	GRAPHANN_TENANT_ID=t_demo \
//	GRAPHANN_API_KEY=test \
//	go test -tags integration -count=1 ./...

func integrationClient(t *testing.T) *Client {
	t.Helper()
	base := os.Getenv("GRAPHANN_BASE_URL")
	if base == "" {
		t.Skip("GRAPHANN_BASE_URL not set; skipping integration test")
	}
	opts := []Option{
		WithBaseURL(base),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
			JitterFraction: 0.2,
		}),
	}
	if t1, k := os.Getenv("GRAPHANN_TENANT_ID"), os.Getenv("GRAPHANN_API_KEY"); t1 != "" && k != "" {
		opts = append(opts, WithAPIKey(t1, k))
	}
	c, err := NewClient(opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestIntegration_Health(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	hr, err := c.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if hr.Status == "" {
		t.Errorf("empty status: %+v", hr)
	}
}

func TestIntegration_TenantsRoundtrip(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	name := "sdk-int-" + time.Now().UTC().Format("20060102150405")
	created, err := c.CreateTenant(ctx, CreateTenantRequest{Name: name})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	t.Logf("created tenant: id=%s name=%s", created.ID, created.Name)

	got, err := c.GetTenant(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: %s vs %s", got.ID, created.ID)
	}

	if _, err := c.DeleteTenant(ctx, created.ID); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}

	if _, err := c.GetTenant(ctx, created.ID); err == nil {
		t.Errorf("GetTenant after delete: expected error, got nil")
	} else if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
