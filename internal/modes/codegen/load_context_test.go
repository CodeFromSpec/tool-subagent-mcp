// spec: TEST/tech_design/internal/modes/codegen/tools/load_context@v8

package codegen

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// invokeLoadContext calls handleLoadContext with the given logical name.
func invokeLoadContext(t *testing.T, logicalName string) *mcp.CallToolResult {
	t.Helper()
	result, _, err := handleLoadContext(context.Background(), &mcp.CallToolRequest{}, LoadContextArgs{LogicalName: logicalName})
	if err != nil {
		t.Fatalf("handleLoadContext returned unexpected Go error: %v", err)
	}
	if result == nil {
		t.Fatal("handleLoadContext returned nil result")
	}
	return result
}

func loadContextText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content entries")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// ── Happy Path ────────────────────────────────────────────────────────────────

// TestHandleLoadContext_ValidRootLeafNode verifies the basic happy path:
// a valid ROOT/ leaf node returns chain content.
func TestHandleLoadContext_ValidRootLeafNode(t *testing.T) {
	root := t.TempDir()
	buildMinimalSpecTree(t, root, "internal/foo/foo.go")
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", loadContextText(t, result))
	}
	text := loadContextText(t, result)
	if text == "" {
		t.Error("expected non-empty chain content")
	}
}

// TestHandleLoadContext_ValidTestNode verifies that a TEST/ node resolves correctly.
func TestHandleLoadContext_ValidTestNode(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "code-from-spec/spec/_node.md", rootNodeContent)
	writeSpecFile(t, root, "code-from-spec/spec/a/_node.md", "---\nversion: 1\n---\n# ROOT/a\n")
	writeSpecFile(t, root, "code-from-spec/spec/a/default.test.md",
		"---\nversion: 1\nimplements:\n  - internal/a/a_test.go\n---\n# TEST/a\n")
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "TEST/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", loadContextText(t, result))
	}
}

// TestHandleLoadContext_NodeWithDependencies verifies that chain includes
// external dependency files.
func TestHandleLoadContext_NodeWithDependencies(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "code-from-spec/spec/_node.md", rootNodeContent)
	writeSpecFile(t, root, "code-from-spec/spec/a/_node.md", `---
version: 1
depends_on:
  - path: EXTERNAL/db
    version: 1
implements:
  - internal/baz/baz.go
---
# ROOT/a
`)
	writeSpecFile(t, root, "code-from-spec/external/db/_external.md", "---\nversion: 1\n---\n# EXTERNAL/db\n")
	writeSpecFile(t, root, "code-from-spec/external/db/schema.sql", "CREATE TABLE t (id INT);")
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", loadContextText(t, result))
	}
	text := loadContextText(t, result)
	if !strings.Contains(text, "schema.sql") {
		t.Error("chain content should include external dependency file schema.sql")
	}
}

// TestHandleLoadContext_ChainContentUsesHeredocFormat verifies the delimiter format.
func TestHandleLoadContext_ChainContentUsesHeredocFormat(t *testing.T) {
	root := t.TempDir()
	buildMinimalSpecTree(t, root, "internal/foo/foo.go")
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", loadContextText(t, result))
	}
	text := loadContextText(t, result)
	if !strings.Contains(text, "<<<FILE_") {
		t.Error("chain content must contain <<<FILE_ delimiter")
	}
	if !strings.Contains(text, "<<<END_FILE_") {
		t.Error("chain content must contain <<<END_FILE_ delimiter")
	}
	if !strings.Contains(text, "node:") {
		t.Error("chain content must contain node: header")
	}
	if !strings.Contains(text, "path:") {
		t.Error("chain content must contain path: header")
	}
}

// TestHandleLoadContext_RepeatedCallsSucceed verifies that the stateless handler
// can be called multiple times with the same logical name, succeeding each time.
// Per spec: "Content may differ between calls because a new UUID is generated each time."
func TestHandleLoadContext_RepeatedCallsSucceed(t *testing.T) {
	root := t.TempDir()
	buildMinimalSpecTree(t, root, "internal/foo/foo.go")
	restore := chdirTo(t, root)
	defer restore()

	// First call — expect success with non-empty chain content.
	result1 := invokeLoadContext(t, "ROOT/a")
	if result1.IsError {
		t.Fatalf("first call: expected success, got tool error: %s", loadContextText(t, result1))
	}
	text1 := loadContextText(t, result1)
	if text1 == "" {
		t.Error("first call: expected non-empty chain content")
	}

	// Second call — expect success with non-empty chain content.
	// Content may differ because a new UUID is generated each time.
	result2 := invokeLoadContext(t, "ROOT/a")
	if result2.IsError {
		t.Fatalf("second call: expected success, got tool error: %s", loadContextText(t, result2))
	}
	text2 := loadContextText(t, result2)
	if text2 == "" {
		t.Error("second call: expected non-empty chain content")
	}
}

// ── Failure Cases ─────────────────────────────────────────────────────────────

// TestHandleLoadContext_InvalidPrefix verifies that EXTERNAL/ prefix is rejected.
func TestHandleLoadContext_InvalidPrefix(t *testing.T) {
	result, _, err := handleLoadContext(context.Background(), &mcp.CallToolRequest{}, LoadContextArgs{LogicalName: "EXTERNAL/something"})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for EXTERNAL/ prefix")
	}
	text := loadContextText(t, result)
	if !strings.Contains(text, "ROOT/ or TEST/") {
		t.Errorf("error must mention 'ROOT/ or TEST/', got: %s", text)
	}
}

func TestHandleLoadContext_NonexistentSpecFile(t *testing.T) {
	root := t.TempDir()
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/nonexistent")

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec file")
	}
}

// TestHandleLoadContext_NoImplements verifies that a node without implements
// returns a tool error.
func TestHandleLoadContext_NoImplements(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "code-from-spec/spec/_node.md", rootNodeContent)
	writeSpecFile(t, root, "code-from-spec/spec/a/_node.md", "---\nversion: 1\n---\n# ROOT/a\n")
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for node with no implements")
	}
	if !strings.Contains(loadContextText(t, result), "has no implements") {
		t.Errorf("error must mention 'has no implements', got: %s", loadContextText(t, result))
	}
}

// TestHandleLoadContext_InvalidImplementsPath verifies that a traversal path
// in implements returns a tool error.
func TestHandleLoadContext_InvalidImplementsPath(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "code-from-spec/spec/_node.md", rootNodeContent)
	writeSpecFile(t, root, "code-from-spec/spec/a/_node.md",
		"---\nversion: 1\nimplements:\n  - ../../etc/passwd\n---\n# ROOT/a\n")
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for traversal implements path")
	}
}

// TestHandleLoadContext_UnresolvableDependency verifies that a missing
// dependency returns a tool error.
func TestHandleLoadContext_UnresolvableDependency(t *testing.T) {
	root := t.TempDir()
	writeSpecFile(t, root, "code-from-spec/spec/_node.md", rootNodeContent)
	writeSpecFile(t, root, "code-from-spec/spec/a/_node.md", `---
version: 1
depends_on:
  - path: ROOT/b
    version: 1
implements:
  - internal/foo/foo.go
---
# ROOT/a
`)
	restore := chdirTo(t, root)
	defer restore()

	result := invokeLoadContext(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for unresolvable dependency")
	}
}
