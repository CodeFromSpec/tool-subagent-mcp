// spec: TEST/tech_design/internal/tools/load_chain@v10
package load_chain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// helper: creates a directory and writes a file with the given content.
// Fails the test if any operation fails.
func writeFile(t *testing.T, base string, relPath string, content string) {
	t.Helper()
	full := filepath.Join(base, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// helper: calls HandleLoadChain with the given logical name and returns
// the result. Does not fail the test — the caller inspects the result.
func callHandler(t *testing.T, logicalName string) *mcp.CallToolResult {
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
		t.Fatal("result content is not TextContent")
	}
	return tc.Text
}

// helper: changes the working directory to dir for the duration of the
// test and restores it when the test completes.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// --- Happy Path Tests ---

// TestValidROOTLeafNode verifies that a valid ROOT/ leaf node
// returns a success result containing chain content from ROOT and ROOT/a.
func TestValidROOTLeafNode(t *testing.T) {
	tmp := t.TempDir()

	// Create ROOT node
	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// Create ROOT/a leaf node with implements
	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Chain should contain files from ROOT and ROOT/a
	if !strings.Contains(text, "# ROOT") {
		t.Error("chain content missing ROOT node content")
	}
	if !strings.Contains(text, "# ROOT/a") {
		t.Error("chain content missing ROOT/a node content")
	}
}

// TestValidTESTNode verifies that a valid TEST/ node returns a
// success result.
func TestValidTESTNode(t *testing.T) {
	tmp := t.TempDir()

	// Create ROOT node
	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// Create ROOT/a leaf node
	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Create TEST/a node
	writeFile(t, tmp, "code-from-spec/spec/a/default.test.md", `---
version: 1
parent_version: 1
implements:
  - src/a_test.go
---

# TEST/a
`)

	chdir(t, tmp)

	result := callHandler(t, "TEST/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
}

// TestNodeWithDependencies verifies that the chain includes
// external dependency files.
func TestNodeWithDependencies(t *testing.T) {
	tmp := t.TempDir()

	// Create ROOT node
	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// Create ROOT/a with a dependency on EXTERNAL/db
	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
depends_on:
  - path: EXTERNAL/db
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Create the external dependency _external.md and a data file
	writeFile(t, tmp, "code-from-spec/external/db/_external.md", `---
version: 1
---

# EXTERNAL/db
`)
	writeFile(t, tmp, "code-from-spec/external/db/schema.sql", `CREATE TABLE t (id INT);`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Should contain external dependency files
	if !strings.Contains(text, "EXTERNAL/db") {
		t.Error("chain content missing EXTERNAL/db dependency")
	}
	if !strings.Contains(text, "CREATE TABLE") {
		t.Error("chain content missing schema.sql content")
	}
}

// TestChainContentUsesHeredocFormat verifies the output uses
// the <<<FILE_>>> and <<<END_FILE_>>> delimiter format with
// node: and path: headers.
func TestChainContentUsesHeredocFormat(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

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

// TestRepeatedCallsSucceed verifies that calling the handler
// twice with the same input both succeed. UUIDs may differ.
func TestRepeatedCallsSucceed(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	chdir(t, tmp)

	result1 := callHandler(t, "ROOT/a")
	result2 := callHandler(t, "ROOT/a")

	if result1.IsError {
		t.Fatalf("first call failed: %s", resultText(t, result1))
	}
	if result2.IsError {
		t.Fatalf("second call failed: %s", resultText(t, result2))
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
// files have their YAML frontmatter stripped, while the target
// preserves its frontmatter.
func TestFrontmatterStrippedFromNonTargetFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 2
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Split the output into file sections to examine individually.
	// Find the ROOT ancestor section — it should NOT contain frontmatter.
	// Find the ROOT/a target section — it SHOULD contain frontmatter.
	sections := strings.Split(text, "<<<FILE_")

	foundRootAncestor := false
	foundTarget := false

	for _, section := range sections {
		if section == "" {
			continue
		}
		if strings.Contains(section, "node: ROOT\n") {
			foundRootAncestor = true
			// The ROOT ancestor section should not contain frontmatter
			// delimiters (---) or "version"
			// Extract content after the blank line separator
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("ROOT section missing content separator")
			}
			content := parts[1]
			if strings.Contains(content, "---") {
				t.Error("ROOT ancestor section still contains frontmatter delimiters")
			}
			if strings.Contains(content, "version") {
				t.Error("ROOT ancestor section still contains version field")
			}
		}
		if strings.Contains(section, "node: ROOT/a\n") {
			foundTarget = true
			// The target section should preserve frontmatter
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("ROOT/a section missing content separator")
			}
			content := parts[1]
			if !strings.Contains(content, "---") {
				t.Error("target section missing frontmatter delimiters")
			}
			if !strings.Contains(content, "version: 2") {
				t.Error("target section missing version field")
			}
		}
	}

	if !foundRootAncestor {
		t.Error("did not find ROOT ancestor section in output")
	}
	if !foundTarget {
		t.Error("did not find ROOT/a target section in output")
	}
}

// TestFrontmatterStrippedFromDependencyFiles verifies that
// dependency files have their YAML frontmatter stripped.
func TestFrontmatterStrippedFromDependencyFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
depends_on:
  - path: EXTERNAL/db
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	writeFile(t, tmp, "code-from-spec/external/db/_external.md", `---
version: 1
---

# EXTERNAL/db

Database schema reference.
`)

	writeFile(t, tmp, "code-from-spec/external/db/data.sql", `SELECT 1;`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Find the EXTERNAL/db _external.md section
	sections := strings.Split(text, "<<<FILE_")
	foundExternal := false

	for _, section := range sections {
		if section == "" {
			continue
		}
		if strings.Contains(section, "node: EXTERNAL/db") && strings.Contains(section, "_external.md") {
			foundExternal = true
			// Extract the file content after the blank line separator
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("EXTERNAL/db section missing content separator")
			}
			content := parts[1]
			// Should NOT contain frontmatter
			if strings.Contains(content, "---") {
				t.Error("dependency _external.md section still contains frontmatter delimiters")
			}
			if strings.Contains(content, "version: 1") {
				t.Error("dependency _external.md section still contains version field")
			}
			// Should still contain the body
			if !strings.Contains(content, "Database schema reference") {
				t.Error("dependency _external.md section missing body content")
			}
		}
	}

	if !foundExternal {
		t.Error("did not find EXTERNAL/db _external.md section in output")
	}

	// The target section (ROOT/a) should still have frontmatter
	foundTarget := false
	for _, section := range sections {
		if strings.Contains(section, "node: ROOT/a\n") {
			foundTarget = true
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("ROOT/a section missing content separator")
			}
			content := parts[1]
			if !strings.Contains(content, "---") {
				t.Error("target section should preserve frontmatter")
			}
		}
	}
	if !foundTarget {
		t.Error("did not find ROOT/a target section")
	}
}

// TestExistingCodeFilesIncludedInOutput verifies that existing
// code files listed in implements are included in the output with
// path: header and no node: header.
func TestExistingCodeFilesIncludedInOutput(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Create the implemented file with known content
	writeFile(t, tmp, "src/a.go", `package a

func Hello() string { return "hello" }
`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Find the code file section — should have path: but no node:
	sections := strings.Split(text, "<<<FILE_")
	foundCode := false

	for _, section := range sections {
		if section == "" {
			continue
		}
		if strings.Contains(section, "path: src/a.go") {
			foundCode = true

			// Extract the header lines (before the blank line)
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("code section missing content separator")
			}
			header := parts[0]
			content := parts[1]

			// Code files should NOT have a node: header
			if strings.Contains(header, "node:") {
				t.Error("code file section should not have node: header")
			}

			// Content should match what was written
			if !strings.Contains(content, `func Hello() string { return "hello" }`) {
				t.Error("code file content does not match written content")
			}
		}
	}

	if !foundCode {
		t.Error("did not find src/a.go code file section in output")
	}
}

// TestNonExistingCodeFilesOmittedFromOutput verifies that code
// files that don't exist on disk are not included in the output.
func TestNonExistingCodeFilesOmittedFromOutput(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Do NOT create src/a.go

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// The output should NOT contain a section for src/a.go
	if strings.Contains(text, "path: src/a.go") {
		t.Error("output should not contain a section for non-existing code file src/a.go")
	}
}

// --- Failure Cases ---

// TestInvalidPrefix verifies that a logical name with an invalid
// prefix returns a tool error.
func TestInvalidPrefix(t *testing.T) {
	tmp := t.TempDir()
	chdir(t, tmp)

	result := callHandler(t, "EXTERNAL/something")

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
	tmp := t.TempDir()
	chdir(t, tmp)

	result := callHandler(t, "ROOT/nonexistent")

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec file")
	}
}

// TestNoImplements verifies that a node without implements
// returns a tool error.
func TestNoImplements(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// Create ROOT/a without implements
	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
---

# ROOT/a — no implements
`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for node with no implements")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "has no implements") {
		t.Errorf("unexpected error message: %s", text)
	}
}

// TestInvalidImplementsPathTraversal verifies that a path traversal
// in implements causes a tool error from path validation.
func TestInvalidImplementsPathTraversal(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - "../../etc/passwd"
---

# ROOT/a
`)

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for path traversal in implements")
	}
}

// TestUnresolvableDependency verifies that a dependency whose
// spec file does not exist causes a tool error.
func TestUnresolvableDependency(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	writeFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
depends_on:
  - path: ROOT/b
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Do NOT create ROOT/b's spec file

	chdir(t, tmp)

	result := callHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for unresolvable dependency")
	}
}
