// spec: TEST/tech_design/internal/tools/load_chain@v8
package load_chain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Helper functions ---

// writeFile creates a file at the given path (relative to dir) with the given content.
// It creates parent directories as needed.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("failed to create directories for %s: %v", relPath, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}

// chdir changes the working directory to dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// callLoadChain is a convenience wrapper that calls HandleLoadChain
// with the given logical name.
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

// resultText extracts the text content from a CallToolResult.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// createMinimalTree creates a minimal spec tree with ROOT and ROOT/a.
// ROOT/a has implements pointing to the given paths.
// Returns the temp directory path.
func createMinimalTree(t *testing.T, implements string) string {
	t.Helper()
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
Root node content.
`)

	// ROOT/a leaf node
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 2
implements:
  - `+implements+`
---

# ROOT/a
Leaf node content.
`)

	return dir
}

// --- Happy Path Tests ---

// TestValidROOTLeafNode verifies that a valid ROOT/ leaf node
// returns a success result with chain content from ROOT and ROOT/a.
func TestValidROOTLeafNode(t *testing.T) {
	dir := createMinimalTree(t, "src/a.go")
	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Chain must contain sections for ROOT and ROOT/a
	if !strings.Contains(text, "node: ROOT\n") {
		t.Error("chain content missing ROOT node section")
	}
	if !strings.Contains(text, "node: ROOT/a\n") {
		t.Error("chain content missing ROOT/a node section")
	}
}

// TestValidTESTNode verifies that a TEST/ logical name
// returns a success result.
func TestValidTESTNode(t *testing.T) {
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a leaf node
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// TEST/a test node
	writeFile(t, dir, "code-from-spec/spec/a/default.test.md", `---
version: 1
parent_version: 1
implements:
  - src/a_test.go
---

# TEST/a
`)

	chdir(t, dir)

	result := callLoadChain(t, "TEST/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
}

// TestNodeWithDependencies verifies that chain content includes
// external dependency files.
func TestNodeWithDependencies(t *testing.T) {
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a with dependency on EXTERNAL/db
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 1
depends_on:
  - path: EXTERNAL/db
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// External dependency: _external.md
	writeFile(t, dir, "code-from-spec/external/db/_external.md", `---
version: 1
---

# EXTERNAL/db
Database dependency.
`)

	// External dependency: data file
	writeFile(t, dir, "code-from-spec/external/db/schema.sql", `CREATE TABLE t (id INT);`)

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Must contain the external dependency files
	if !strings.Contains(text, "node: EXTERNAL/db") {
		t.Error("chain content missing EXTERNAL/db node section")
	}
	if !strings.Contains(text, "schema.sql") {
		t.Error("chain content missing schema.sql path reference")
	}
	if !strings.Contains(text, "CREATE TABLE") {
		t.Error("chain content missing schema.sql file content")
	}
}

// TestChainContentUsesHeredocFormat verifies that the output uses
// the <<<FILE_...>>> and <<<END_FILE_...>>> delimiter format
// with node: and path: headers.
func TestChainContentUsesHeredocFormat(t *testing.T) {
	dir := createMinimalTree(t, "src/a.go")
	chdir(t, dir)

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

// TestRepeatedCallsSucceed verifies that calling handleLoadChain
// twice with the same logical name both return success.
func TestRepeatedCallsSucceed(t *testing.T) {
	dir := createMinimalTree(t, "src/a.go")
	chdir(t, dir)

	result1 := callLoadChain(t, "ROOT/a")
	result2 := callLoadChain(t, "ROOT/a")

	if result1.IsError {
		t.Fatalf("first call returned error: %s", resultText(t, result1))
	}
	if result2.IsError {
		t.Fatalf("second call returned error: %s", resultText(t, result2))
	}

	text1 := resultText(t, result1)
	text2 := resultText(t, result2)

	if text1 == "" {
		t.Error("first call returned empty content")
	}
	if text2 == "" {
		t.Error("second call returned empty content")
	}
}

// TestFrontmatterStrippedFromNonTargetFiles verifies that ancestor
// files have their YAML frontmatter stripped, while the target node
// preserves its full frontmatter.
func TestFrontmatterStrippedFromNonTargetFiles(t *testing.T) {
	dir := t.TempDir()

	// ROOT node with version: 1 frontmatter
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
Root node content.
`)

	// ROOT/a leaf node with version: 2 frontmatter
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 2
implements:
  - src/a.go
---

# ROOT/a
Leaf node content.
`)

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Split the output into file sections to examine each one.
	// Find the ROOT section (not ROOT/a) and verify it has no frontmatter.
	sections := strings.Split(text, "<<<FILE_")
	var rootSection string
	var targetSection string
	for _, section := range sections {
		if strings.Contains(section, "node: ROOT\n") {
			rootSection = section
		}
		if strings.Contains(section, "node: ROOT/a\n") {
			targetSection = section
		}
	}

	if rootSection == "" {
		t.Fatal("could not find ROOT section in output")
	}
	if targetSection == "" {
		t.Fatal("could not find ROOT/a section in output")
	}

	// ROOT (ancestor) should NOT contain frontmatter delimiters or version
	if strings.Contains(rootSection, "---") {
		t.Error("ROOT section should have frontmatter stripped, but contains '---'")
	}
	if strings.Contains(rootSection, "version:") {
		t.Error("ROOT section should have frontmatter stripped, but contains 'version:'")
	}

	// ROOT/a (target) should preserve full frontmatter
	if !strings.Contains(targetSection, "---") {
		t.Error("ROOT/a (target) section should preserve frontmatter, but missing '---'")
	}
	if !strings.Contains(targetSection, "version: 2") {
		t.Error("ROOT/a (target) section should preserve frontmatter, but missing 'version: 2'")
	}
}

// TestFrontmatterStrippedFromDependencyFiles verifies that external
// dependency files have their YAML frontmatter stripped.
func TestFrontmatterStrippedFromDependencyFiles(t *testing.T) {
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a with dependency on EXTERNAL/db
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 1
depends_on:
  - path: EXTERNAL/db
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// External dependency with frontmatter
	writeFile(t, dir, "code-from-spec/external/db/_external.md", `---
version: 1
---

# EXTERNAL/db
Database info.
`)

	// Data file (no frontmatter to strip)
	writeFile(t, dir, "code-from-spec/external/db/data.txt", `some data`)

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Find the EXTERNAL/db section containing _external.md
	sections := strings.Split(text, "<<<FILE_")
	var externalSection string
	for _, section := range sections {
		if strings.Contains(section, "_external.md") {
			externalSection = section
			break
		}
	}

	if externalSection == "" {
		t.Fatal("could not find _external.md section in output")
	}

	// The _external.md section should have frontmatter stripped
	// Find content after the blank line (after headers)
	// The content portion should not contain frontmatter delimiters
	parts := strings.SplitN(externalSection, "\n\n", 2)
	if len(parts) < 2 {
		t.Fatal("could not split _external.md section into header and content")
	}
	content := parts[1]
	if strings.Contains(content, "---") {
		t.Error("_external.md content should have frontmatter stripped, but contains '---'")
	}
	if strings.Contains(content, "version: 1") {
		t.Error("_external.md content should have frontmatter stripped, but contains 'version: 1'")
	}
}

// TestExistingCodeFilesIncludedInOutput verifies that existing code
// files referenced by implements are included in the output with
// path: header and no node: header.
func TestExistingCodeFilesIncludedInOutput(t *testing.T) {
	dir := createMinimalTree(t, "src/a.go")

	// Create the implements file with known content
	writeFile(t, dir, "src/a.go", `package a

func Hello() string { return "hello" }
`)

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Find the code file section
	sections := strings.Split(text, "<<<FILE_")
	var codeSection string
	for _, section := range sections {
		if strings.Contains(section, "path: src/a.go") {
			codeSection = section
			break
		}
	}

	if codeSection == "" {
		t.Fatal("could not find src/a.go section in output")
	}

	// Code file sections should have path: but NOT node:
	if strings.Contains(codeSection, "node:") {
		t.Error("code file section should not have node: header")
	}

	// Content should match what was written
	if !strings.Contains(codeSection, `func Hello() string { return "hello" }`) {
		t.Error("code file section does not contain expected content")
	}
}

// TestNonExistingCodeFilesOmittedFromOutput verifies that code files
// listed in implements but not existing on disk are omitted from the output.
func TestNonExistingCodeFilesOmittedFromOutput(t *testing.T) {
	dir := createMinimalTree(t, "src/a.go")
	// Do NOT create src/a.go

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// The output should not contain a section for src/a.go
	if strings.Contains(text, "path: src/a.go") {
		t.Error("output should not contain section for non-existing src/a.go")
	}
}

// --- Failure Case Tests ---

// TestInvalidPrefix verifies that a logical name with an invalid prefix
// returns a tool error.
func TestInvalidPrefix(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	result := callLoadChain(t, "EXTERNAL/something")

	if !result.IsError {
		t.Fatal("expected tool error for invalid prefix, got success")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "target must be a ROOT/ or TEST/") {
		t.Errorf("error message does not contain expected text, got: %s", text)
	}
}

// TestNonexistentSpecFile verifies that referencing a nonexistent spec
// file returns a tool error from ParseFrontmatter.
func TestNonexistentSpecFile(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Do not create any spec files for ROOT/nonexistent
	result := callLoadChain(t, "ROOT/nonexistent")

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec file, got success")
	}
}

// TestNoImplements verifies that a node without implements
// returns a tool error containing "has no implements".
func TestNoImplements(t *testing.T) {
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a with no implements field
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 1
---

# ROOT/a
No implements here.
`)

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for missing implements, got success")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "has no implements") {
		t.Errorf("error message does not contain 'has no implements', got: %s", text)
	}
}

// TestInvalidImplementsPathTraversal verifies that a path traversal
// in implements causes a tool error from path validation.
func TestInvalidImplementsPathTraversal(t *testing.T) {
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a with traversal in implements
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 1
implements:
  - "../../etc/passwd"
---

# ROOT/a
`)

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for path traversal, got success")
	}
}

// TestUnresolvableDependency verifies that a dependency referencing
// a nonexistent node returns a tool error from chain resolution.
func TestUnresolvableDependency(t *testing.T) {
	dir := t.TempDir()

	// ROOT node
	writeFile(t, dir, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a depends on ROOT/b, which does not exist
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", `---
version: 1
depends_on:
  - path: ROOT/b
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Do NOT create ROOT/b

	chdir(t, dir)

	result := callLoadChain(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for unresolvable dependency, got success")
	}
}
