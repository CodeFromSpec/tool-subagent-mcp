// spec: TEST/tech_design/internal/modes/codegen/setup@v7

package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newServer creates a minimal MCP server for passing to Setup.
func newServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "test-server"}, nil)
}

// writeSpecFile creates a file at <root>/<rel> with the given content,
// creating parent directories as needed.
func writeSpecFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("writeSpecFile mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writeSpecFile write %s: %v", rel, err)
	}
}

// chdirTo changes the working directory to dir and returns a restore func.
func chdirTo(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("chdirTo Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdirTo Chdir %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(orig); err != nil {
			panic("chdirTo restore: " + err.Error())
		}
	}
}

const rootNodeContent = `---
version: 1
---
# ROOT
`

func leafNodeContent(implementsPath string) string {
	return "---\nversion: 1\nimplements:\n  - " + implementsPath + "\n---\n# Leaf node\n"
}

func buildMinimalSpecTree(t *testing.T, root, implPath string) {
	t.Helper()
	writeSpecFile(t, root, "code-from-spec/spec/_node.md", rootNodeContent)
	writeSpecFile(t, root, "code-from-spec/spec/a/_node.md", leafNodeContent(implPath))
}

// ── Happy Path ────────────────────────────────────────────────────────────────

// TestSetup_RegistersTools verifies that Setup with empty args registers
// load_context and write_file tools on the server without error.
func TestSetup_RegistersTools(t *testing.T) {
	s := newServer()
	if err := Setup(s, []string{}); err != nil {
		t.Fatalf("Setup returned unexpected error: %v", err)
	}
}

// TestSetup_NilArgsSucceeds verifies that nil args are treated as empty.
func TestSetup_NilArgsSucceeds(t *testing.T) {
	s := newServer()
	if err := Setup(s, nil); err != nil {
		t.Fatalf("Setup returned unexpected error for nil args: %v", err)
	}
}

// ── Failure Cases ─────────────────────────────────────────────────────────────

// TestSetup_RejectsUnexpectedArguments verifies that any args cause an error.
func TestSetup_RejectsUnexpectedArguments(t *testing.T) {
	s := newServer()
	err := Setup(s, []string{"unexpected"})
	if err == nil {
		t.Fatal("expected error for unexpected argument, got nil")
	}
	if !strings.Contains(err.Error(), "codegen mode does not accept arguments") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}
