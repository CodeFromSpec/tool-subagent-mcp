// spec: TEST/tech_design/internal/tools/load_chain@v5
package load_chain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// helper: creates a spec file at the given path relative to dir,
// writing the provided content. Creates parent directories as needed.
func createFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating dirs for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", relPath, err)
	}
}

// helper: calls HandleLoadChain with the given logical name.
func callLoadChain(t *testing.T, logicalName string) *mcp.CallToolResult {
	t.Helper()
	result, _, err := HandleLoadChain(
		context.Background(),
		&mcp.CallToolRequest{},
		LoadChainArgs{LogicalName: logicalName},
	)
	if err != nil {
		t.Fatalf("HandleLoadChain returned unexpected Go error: %v", err)
	}
	return result
}

// helper: extracts the text content from a CallToolResult.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("result content is not TextContent: %T", result.Content[0])
	}
	return tc.Text
}

// helper: changes the working directory to dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// rootNode returns minimal frontmatter for a ROOT node (no implements, no deps).
func rootNode() string {
	return "---\nversion: 1\n---\n# ROOT\n"
}

// leafNode returns frontmatter for a leaf node with implements.
func leafNode() string {
	return "---\nversion: 1\nimplements:\n  - internal/foo/foo.go\n---\n# Leaf\n"
}

// leafNodeNoDeps returns frontmatter for a leaf node without implements.
func leafNodeNoImplements() string {
	return "---\nversion: 1\n---\n# Leaf without implements\n"
}

// --- Happy Path ---

// TestValidRootLeafNode tests that a valid ROOT/ leaf node returns
// a success result containing chain content from ROOT and ROOT/a.
func TestValidRootLeafNode(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Create ROOT node
	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	// Create ROOT/a leaf node with implements
	createFile(t, dir, "code-from-spec/spec/a/_node.md", leafNode())

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Chain should contain both ROOT and ROOT/a
	if !strings.Contains(text, "node: ROOT\n") {
		t.Error("chain content missing ROOT node")
	}
	if !strings.Contains(text, "node: ROOT/a\n") {
		t.Error("chain content missing ROOT/a node")
	}
}

// TestValidTestNode tests that a valid TEST/ node returns a success result.
func TestValidTestNode(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Create ROOT node
	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	// Create ROOT/a leaf node with implements
	createFile(t, dir, "code-from-spec/spec/a/_node.md", leafNode())
	// Create TEST/a node with implements
	createFile(t, dir, "code-from-spec/spec/a/default.test.md",
		"---\nversion: 1\nimplements:\n  - internal/foo/foo_test.go\n---\n# Test\n")

	result := callLoadChain(t, "TEST/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
}

// TestNodeWithDependencies tests that a node with external dependencies
// returns chain content including the dependency files.
func TestNodeWithDependencies(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Create ROOT node
	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())

	// Create ROOT/a leaf with depends_on referencing EXTERNAL/db
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		"---\nversion: 1\nimplements:\n  - internal/foo/foo.go\ndepends_on:\n  - path: EXTERNAL/db\n---\n# Leaf with dep\n")

	// Create EXTERNAL/db with _external.md and a data file
	createFile(t, dir, "code-from-spec/external/db/_external.md",
		"---\nversion: 1\n---\n# EXTERNAL/db\n")
	createFile(t, dir, "code-from-spec/external/db/schema.sql",
		"CREATE TABLE foo (id INT);")

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Chain should contain the external dependency files
	if !strings.Contains(text, "node: EXTERNAL/db") {
		t.Error("chain content missing EXTERNAL/db node")
	}
	if !strings.Contains(text, "schema.sql") {
		t.Error("chain content missing schema.sql data file path")
	}
}

// TestChainContentUsesHeredocFormat verifies the output uses
// <<<FILE_<uuid>>> and <<<END_FILE_<uuid>>> delimiters with
// node: and path: headers.
func TestChainContentUsesHeredocFormat(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	createFile(t, dir, "code-from-spec/spec/a/_node.md", leafNode())

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	if !strings.Contains(text, "<<<FILE_") {
		t.Error("chain content missing <<<FILE_ delimiter")
	}
	if !strings.Contains(text, "<<<END_FILE_") {
		t.Error("chain content missing <<<END_FILE_ delimiter")
	}
	if !strings.Contains(text, "node:") {
		t.Error("chain content missing node: header")
	}
	if !strings.Contains(text, "path:") {
		t.Error("chain content missing path: header")
	}
}

// TestRepeatedCallsSucceed verifies that calling HandleLoadChain
// twice with the same logical name both succeed. UUIDs may differ.
func TestRepeatedCallsSucceed(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	createFile(t, dir, "code-from-spec/spec/a/_node.md", leafNode())

	result1 := callLoadChain(t, "ROOT/a")
	result2 := callLoadChain(t, "ROOT/a")

	if result1.IsError {
		t.Fatalf("first call failed: %s", resultText(t, result1))
	}
	if result2.IsError {
		t.Fatalf("second call failed: %s", resultText(t, result2))
	}

	text1 := resultText(t, result1)
	text2 := resultText(t, result2)

	if len(text1) == 0 {
		t.Error("first call returned empty content")
	}
	if len(text2) == 0 {
		t.Error("second call returned empty content")
	}
}

// --- Failure Cases ---

// TestInvalidPrefix verifies that a logical name not starting with
// ROOT/ or TEST/ returns a tool error.
func TestInvalidPrefix(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	result := callLoadChain(t, "EXTERNAL/something")

	if !result.IsError {
		t.Fatal("expected tool error for EXTERNAL/ prefix")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "target must be a ROOT/ or TEST/") {
		t.Errorf("unexpected error message: %s", text)
	}
}

// TestNonexistentSpecFile verifies that a logical name pointing
// to a nonexistent spec file returns a tool error.
func TestNonexistentSpecFile(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Do NOT create ROOT/nonexistent spec file.
	result := callLoadChain(t, "ROOT/nonexistent")

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec file")
	}
}

// TestNoImplements verifies that a node without implements
// returns a tool error containing "has no implements".
func TestNoImplements(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	createFile(t, dir, "code-from-spec/spec/a/_node.md", leafNodeNoImplements())

	result := callLoadChain(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for node with no implements")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "has no implements") {
		t.Errorf("unexpected error message: %s", text)
	}
}

// TestInvalidImplementsPathTraversal verifies that a node with
// a path-traversal implements entry returns a tool error from
// path validation.
func TestInvalidImplementsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		"---\nversion: 1\nimplements:\n  - \"../../etc/passwd\"\n---\n# Leaf with bad path\n")

	result := callLoadChain(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for path traversal in implements")
	}
}

// TestUnresolvableDependency verifies that a node with a dependency
// pointing to a nonexistent node returns a tool error from chain
// resolution.
func TestUnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	createFile(t, dir, "code-from-spec/spec/_node.md", rootNode())
	// ROOT/a depends on ROOT/b which does not exist
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		"---\nversion: 1\nimplements:\n  - internal/foo/foo.go\ndepends_on:\n  - path: ROOT/b\n---\n# Leaf with missing dep\n")

	result := callLoadChain(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for unresolvable dependency")
	}
}
