// spec: TEST/tech_design/internal/tools/load_chain@v2
package load_chain

// Tests for the load_chain tool handler.
//
// Spec ref: TEST/tech_design/internal/tools/load_chain § "Context"
// Each test uses t.TempDir() for isolation. The working directory is changed
// to the temp dir so that path validation and file resolution work correctly.
//
// The handler is tested via handleLoadChain directly (unexported). Tests
// cover both happy-path and failure cases as defined in the spec.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// changeDir changes the working directory for the duration of the test,
// restoring the original on cleanup.
func changeDir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("chdir restore: %v", err)
		}
	})
}

// mustWriteFile creates all parent directories and writes data to path.
func mustWriteFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirall %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("writefile %s: %v", path, err)
	}
}

// callHandler is a convenience wrapper for invoking handleLoadChain.
func callHandler(t *testing.T, logicalName string) *mcp.CallToolResult {
	t.Helper()
	args := LoadChainArgs{LogicalName: logicalName}
	result, _, err := handleLoadChain(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleLoadChain returned unexpected Go error: %v", err)
	}
	return result
}

// rootNodeFM is a minimal _node.md for ROOT with no frontmatter fields that
// cause issues — just a version so ParseFrontmatter doesn't fail.
const rootNodeFM = `---
version: 1
---

# ROOT
`

// leafNodeFM builds a _node.md for a leaf with the given implements paths.
func leafNodeFM(implements ...string) string {
	var sb strings.Builder
	sb.WriteString("---\nimplements:\n")
	for _, p := range implements {
		sb.WriteString("  - ")
		sb.WriteString(p)
		sb.WriteString("\n")
	}
	sb.WriteString("---\n\n# leaf\n")
	return sb.String()
}

// leafNodeNoImpl is a leaf _node.md with no implements field.
const leafNodeNoImpl = `---
version: 1
---

# leaf without implements
`

// ─── Happy Path ──────────────────────────────────────────────────────────────

// Spec ref: TEST § "Valid ROOT/ leaf node"
func TestHandleLoadChain_ValidRootLeaf(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	// Build spec tree: ROOT and ROOT/a
	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md", leafNodeFM("internal/a/a.go"))

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %v", textOf(result))
	}
	text := textOf(result)
	if !strings.Contains(text, "<<<FILE_") {
		t.Errorf("expected chain content with <<<FILE_ delimiter, got: %s", text)
	}
	// Chain must contain content from both ROOT and ROOT/a spec files.
	if !strings.Contains(text, "node: ROOT") {
		t.Errorf("expected chain to include ROOT node, got: %s", text)
	}
	if !strings.Contains(text, "node: ROOT/a") {
		t.Errorf("expected chain to include ROOT/a node, got: %s", text)
	}
}

// Spec ref: TEST § "Valid TEST/ node"
func TestHandleLoadChain_ValidTestNode(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	// Build spec tree: ROOT, ROOT/a (leaf), TEST/a
	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md", leafNodeFM("internal/a/a.go"))
	// TEST/a → code-from-spec/spec/a/default.test.md
	mustWriteFile(t, "code-from-spec/spec/a/default.test.md",
		"---\nparent_version: 1\nimplements:\n  - internal/a/a_test.go\n---\n\n# TEST/a\n")

	result := callHandler(t, "TEST/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %v", textOf(result))
	}
	if text := textOf(result); len(text) == 0 {
		t.Errorf("expected non-empty chain content")
	}
}

// Spec ref: TEST § "Node with dependencies"
func TestHandleLoadChain_NodeWithDependencies(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	// ROOT/a depends on EXTERNAL/db
	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md",
		"---\nimplements:\n  - internal/a/a.go\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1\n---\n\n# ROOT/a\n")

	// Create the EXTERNAL/db dependency
	mustWriteFile(t, "code-from-spec/external/db/_external.md", "---\nversion: 1\n---\n\n# EXTERNAL/db\n")
	mustWriteFile(t, "code-from-spec/external/db/schema.sql", "CREATE TABLE foo (id INT);\n")

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %v", textOf(result))
	}
	text := textOf(result)
	// Chain must include the external dependency files.
	if !strings.Contains(text, "node: EXTERNAL/db") {
		t.Errorf("expected chain to include EXTERNAL/db, got: %s", text)
	}
	if !strings.Contains(text, "schema.sql") {
		t.Errorf("expected chain to include schema.sql, got: %s", text)
	}
}

// Spec ref: TEST § "Chain content uses heredoc format"
func TestHandleLoadChain_HeredocFormat(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md", leafNodeFM("internal/a/a.go"))

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got tool error: %v", textOf(result))
	}
	text := textOf(result)

	// Spec ref: ROOT/tech_design/internal/tools/load_chain § "Chain output format"
	if !strings.Contains(text, "<<<FILE_") {
		t.Errorf("expected <<<FILE_ delimiter, got: %s", text)
	}
	if !strings.Contains(text, "<<<END_FILE_") {
		t.Errorf("expected <<<END_FILE_ delimiter, got: %s", text)
	}
	if !strings.Contains(text, "node:") {
		t.Errorf("expected node: header, got: %s", text)
	}
	if !strings.Contains(text, "path:") {
		t.Errorf("expected path: header, got: %s", text)
	}
}

// Spec ref: TEST § "Repeated calls succeed"
func TestHandleLoadChain_RepeatedCallsSucceed(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md", leafNodeFM("internal/a/a.go"))

	result1 := callHandler(t, "ROOT/a")
	result2 := callHandler(t, "ROOT/a")

	if result1.IsError {
		t.Fatalf("first call: expected success, got: %v", textOf(result1))
	}
	if result2.IsError {
		t.Fatalf("second call: expected success, got: %v", textOf(result2))
	}
	if len(textOf(result1)) == 0 {
		t.Errorf("first call returned empty content")
	}
	if len(textOf(result2)) == 0 {
		t.Errorf("second call returned empty content")
	}
	// UUIDs differ between calls, so the delimiter strings will differ.
	// We just verify both are non-empty and each starts with <<<FILE_.
	if !strings.Contains(textOf(result1), "<<<FILE_") {
		t.Errorf("first call missing <<<FILE_ delimiter")
	}
	if !strings.Contains(textOf(result2), "<<<FILE_") {
		t.Errorf("second call missing <<<FILE_ delimiter")
	}
}

// ─── Failure Cases ────────────────────────────────────────────────────────────

// Spec ref: TEST § "Invalid prefix"
func TestHandleLoadChain_InvalidPrefix(t *testing.T) {
	// No temp dir needed — validation fails before any file I/O.
	result, _, err := handleLoadChain(context.Background(), &mcp.CallToolRequest{},
		LoadChainArgs{LogicalName: "EXTERNAL/something"})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got success: %v", textOf(result))
	}
	// Spec ref: ROOT/tech_design/internal/tools/load_chain § "Algorithm" step 1
	if !strings.Contains(textOf(result), "target must be a ROOT/ or TEST/") {
		t.Errorf("expected 'target must be a ROOT/ or TEST/' in error, got: %s", textOf(result))
	}
}

// Spec ref: TEST § "Nonexistent spec file"
func TestHandleLoadChain_NonexistentSpecFile(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	// Do NOT create any spec files — ParseFrontmatter should fail.
	result := callHandler(t, "ROOT/nonexistent")

	if !result.IsError {
		t.Fatalf("expected tool error, got success")
	}
	// Error comes from ParseFrontmatter (file not found).
	if len(textOf(result)) == 0 {
		t.Errorf("expected non-empty error message")
	}
}

// Spec ref: TEST § "No implements"
func TestHandleLoadChain_NoImplements(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md", leafNodeNoImpl)

	result := callHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatalf("expected tool error, got success")
	}
	// Spec ref: ROOT/tech_design/internal/tools/load_chain § "Algorithm" step 4a
	if !strings.Contains(textOf(result), "has no implements") {
		t.Errorf("expected 'has no implements' in error, got: %s", textOf(result))
	}
}

// Spec ref: TEST § "Invalid implements path — traversal"
func TestHandleLoadChain_InvalidImplementsTraversal(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	// implements contains a directory traversal path
	mustWriteFile(t, "code-from-spec/spec/a/_node.md", leafNodeFM("../../etc/passwd"))

	result := callHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatalf("expected tool error from path validation, got success")
	}
}

// Spec ref: TEST § "Unresolvable dependency"
func TestHandleLoadChain_UnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	changeDir(t, dir)

	// ROOT/a depends on ROOT/b, but ROOT/b's spec file is not created.
	mustWriteFile(t, "code-from-spec/spec/_node.md", rootNodeFM)
	mustWriteFile(t, "code-from-spec/spec/a/_node.md",
		"---\nimplements:\n  - internal/a/a.go\ndepends_on:\n  - path: ROOT/b\n    version: 1\n---\n\n# ROOT/a\n")

	result := callHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatalf("expected tool error from chain resolution, got success")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// textOf extracts the text from the first TextContent entry in a result.
func textOf(r *mcp.CallToolResult) string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	if tc, ok := r.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
