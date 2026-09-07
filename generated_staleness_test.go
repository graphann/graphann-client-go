package graphann

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGeneratedTypesUpToDate re-runs the oapi-codegen generator against the
// spec and fails if generated/types.go would come out any different. This
// is the staleness gate scripts/generate-types.sh exists to satisfy: a spec
// change with no matching `go generate` (or scripts/generate-types.sh run)
// must fail CI, not silently ship a stale SDK.
//
// Skips when the generator binary is unavailable. A missing vendored spec
// fails the check once the generator is available.
func TestGeneratedTypesUpToDate(t *testing.T) {
	oapiCodegen, err := exec.LookPath("oapi-codegen")
	if err != nil {
		out, gerr := exec.Command("go", "env", "GOPATH").Output()
		if gerr != nil {
			t.Skip("oapi-codegen not found on PATH and `go env GOPATH` failed; skipping staleness check")
		}
		candidate := filepath.Join(trimNewline(out), "bin", "oapi-codegen")
		if _, statErr := os.Stat(candidate); statErr != nil {
			t.Skip("oapi-codegen not found on PATH or in $(go env GOPATH)/bin; skipping staleness check")
		}
		oapiCodegen = candidate
	}

	// Prefer the standalone repository's spec, then the shared monorepo spec.
	spec := os.Getenv("GRAPHANN_SPEC")
	if spec == "" {
		spec = filepath.Join("api", "openapi", "spec.yaml")
		if _, statErr := os.Stat(spec); os.IsNotExist(statErr) {
			spec = filepath.Join("..", "api", "openapi", "spec.yaml")
		}
	}
	if _, statErr := os.Stat(spec); statErr != nil {
		// The vendored spec is committed, so absence means the checkout is broken
		// rather than that this machine lacks something. Fail rather than skip.
		t.Fatalf("vendored spec not found at %s: %v", spec, statErr)
	}

	committed, err := os.ReadFile(filepath.Join("generated", "types.go"))
	if err != nil {
		t.Fatalf("reading committed generated/types.go: %v", err)
	}

	tmpOut := filepath.Join(t.TempDir(), "types.go")
	cmd := exec.Command(oapiCodegen, "-generate", "types", "-package", "generated", "-o", tmpOut, spec)
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("oapi-codegen failed: %v\n%s", runErr, out)
	}

	fresh, err := os.ReadFile(tmpOut)
	if err != nil {
		t.Fatalf("reading freshly generated types.go: %v", err)
	}

	if !bytes.Equal(committed, fresh) {
		t.Fatalf("generated/types.go is stale: re-running oapi-codegen against %s produces a different file.\n"+
			"Run scripts/generate-types.sh and commit the result.", spec)
	}
}

func trimNewline(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
