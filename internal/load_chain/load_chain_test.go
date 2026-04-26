// code-from-spec: TEST/tech_design/internal/tools/load_chain@v14
package load_chain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// testWriteFile creates a directory and writes a file with the given content.
// Fails the test if any operation fails.
func testWriteFile(t *testing.T, base string, relPath string, content string) {
	t.Helper()
	full := filepath.Join(base, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// testCallHandler calls HandleLoadChain with the given logical name and returns
// the result. Does not fail the test — the caller inspects the result.
func testCallHandler(t *testing.T, logicalName string) *mcp.CallToolResult {
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

// testResultText extracts the text content from a CallToolResult.
func testResultText(t *testing.T, result *mcp.CallToolResult) string {
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

// testChdir changes the working directory to dir for the duration of the
// test and restores it when the test completes.
func testChdir(t *testing.T, dir string) {
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
// Both nodes have # Public sections.
func TestValidROOTLeafNode(t *testing.T) {
	tmp := t.TempDir()

	// ROOT node with a # Public section
	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT

# Public

## Overview

Root overview.
`)

	// ROOT/a leaf node with # Public and implements
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a

# Public

## Interface

The interface for a.
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// Chain should contain sections for ROOT and ROOT/a
	if !strings.Contains(text, "node: ROOT") {
		t.Error("chain content missing ROOT node section")
	}
	if !strings.Contains(text, "node: ROOT/a") {
		t.Error("chain content missing ROOT/a node section")
	}
}

// TestValidTESTNode verifies that a valid TEST/ node returns a success result.
// ROOT and ROOT/a contain only their # Public sections; TEST/a contains
// reduced frontmatter and full body.
func TestValidTESTNode(t *testing.T) {
	tmp := t.TempDir()

	// ROOT node with # Public
	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT

# Public

## Overview

Root public content.
`)

	// ROOT/a with # Public
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a

# Public

## Interface

The interface for a.
`)

	// TEST/a node
	testWriteFile(t, tmp, "code-from-spec/spec/a/default.test.md", `---
version: 2
parent_version: 1
implements:
  - src/a_test.go
---

# TEST/a

## Happy Path

Test happy path cases.
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "TEST/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// ROOT and ROOT/a should be present (as ancestors)
	if !strings.Contains(text, "node: ROOT") {
		t.Error("chain content missing ROOT ancestor section")
	}
	if !strings.Contains(text, "node: ROOT/a") {
		t.Error("chain content missing ROOT/a ancestor section")
	}

	// TEST/a should be the target with full body
	if !strings.Contains(text, "node: TEST/a") {
		t.Error("chain content missing TEST/a target section")
	}

	// Verify TEST/a has its body content (Happy Path section)
	if !strings.Contains(text, "Happy Path") {
		t.Error("TEST/a target section missing body content")
	}
}

// TestNodeWithDependencyNoQualifier verifies that a node with a ROOT/
// dependency (no qualifier) includes the full # Public section of the dependency.
func TestNodeWithDependencyNoQualifier(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a depends on ROOT/b (no qualifier)
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
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

	// ROOT/b has # Public with two subsections
	testWriteFile(t, tmp, "code-from-spec/spec/b/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/b.go
---

# ROOT/b

# Public

## Interface

The interface for b.

## Constraints

The constraints for b.
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// Dependency ROOT/b section should contain both subsections (full # Public)
	if !strings.Contains(text, "The interface for b.") {
		t.Error("dependency ROOT/b section missing ## Interface subsection content")
	}
	if !strings.Contains(text, "The constraints for b.") {
		t.Error("dependency ROOT/b section missing ## Constraints subsection content")
	}
}

// TestNodeWithDependencyWithQualifier verifies that a node with a ROOT/
// dependency using a qualifier includes only the specified subsection.
func TestNodeWithDependencyWithQualifier(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a depends on ROOT/b with qualifier "interface"
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
depends_on:
  - path: ROOT/b(interface)
    version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// ROOT/b has # Public with ## Interface and ## Constraints
	testWriteFile(t, tmp, "code-from-spec/spec/b/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/b.go
---

# ROOT/b

# Public

## Interface

The interface for b.

## Constraints

The constraints for b.
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// Only ## Interface should be present, not ## Constraints
	if !strings.Contains(text, "The interface for b.") {
		t.Error("dependency ROOT/b section missing qualified ## Interface subsection content")
	}
	if strings.Contains(text, "The constraints for b.") {
		t.Error("dependency ROOT/b section should not include ## Constraints when qualified with 'interface'")
	}
}

// TestChainContentUsesHeredocFormat verifies the output uses
// <<<FILE_>>> and <<<END_FILE_>>> delimiters with node: and path: headers.
func TestChainContentUsesHeredocFormat(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

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

// TestAncestorsContainOnlyPublic verifies that ancestor sections contain
// only the # Public section content, not private sections or the node name section.
func TestAncestorsContainOnlyPublic(t *testing.T) {
	tmp := t.TempDir()

	// ROOT with # Public and a private section
	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT

# Public

## Overview

Root public content.

# Private Section

Root private content.
`)

	// ROOT/a with # Public and a private section
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a/b.go
---

# ROOT/a

# Public

## Overview

a public content.

# Private Details

a private content.
`)

	// ROOT/a/b leaf with implements
	testWriteFile(t, tmp, "code-from-spec/spec/a/b/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a/b.go
---

# ROOT/a/b

## Happy Path

Target content.
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a/b")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// ROOT ancestor section should contain only # Public content
	// Private sections must not appear in the ROOT ancestor section
	sections := strings.Split(text, "<<<FILE_")

	for _, section := range sections {
		if section == "" {
			continue
		}
		// Check ROOT ancestor (not ROOT/a or ROOT/a/b)
		if strings.Contains(section, "node: ROOT\n") {
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("ROOT section missing content separator")
			}
			content := parts[1]
			if strings.Contains(content, "Root private content") {
				t.Error("ROOT ancestor section contains private content")
			}
			if strings.Contains(content, "# Private Section") {
				t.Error("ROOT ancestor section contains private section heading")
			}
		}

		// Check ROOT/a ancestor
		if strings.Contains(section, "node: ROOT/a\n") {
			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("ROOT/a section missing content separator")
			}
			content := parts[1]
			if strings.Contains(content, "a private content") {
				t.Error("ROOT/a ancestor section contains private content")
			}
			if strings.Contains(content, "# Private Details") {
				t.Error("ROOT/a ancestor section contains private section heading")
			}
		}
	}
}

// TestTargetHasReducedFrontmatter verifies that the target node's section
// contains only version and implements in the frontmatter; other fields
// (parent_version, depends_on) are stripped.
func TestTargetHasReducedFrontmatter(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/b is a dependency
	testWriteFile(t, tmp, "code-from-spec/spec/b/_node.md", `---
version: 2
parent_version: 1
implements:
  - src/b.go
---

# ROOT/b
`)

	// ROOT/a with version, parent_version, depends_on, and implements
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 5
parent_version: 1
depends_on:
  - path: ROOT/b
    version: 2
implements:
  - src/a.go
---

# ROOT/a
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// Find the ROOT/a target section and examine its frontmatter
	sections := strings.Split(text, "<<<FILE_")
	foundTarget := false

	for _, section := range sections {
		if section == "" {
			continue
		}
		if strings.Contains(section, "node: ROOT/a\n") {
			foundTarget = true

			parts := strings.SplitN(section, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("ROOT/a section missing content separator")
			}
			content := parts[1]

			// Must contain version: 5
			if !strings.Contains(content, "version: 5") {
				t.Error("target section missing version field")
			}
			// Must contain implements
			if !strings.Contains(content, "implements:") {
				t.Error("target section missing implements field")
			}
			if !strings.Contains(content, "src/a.go") {
				t.Error("target section missing implements path")
			}
			// Must NOT contain parent_version
			if strings.Contains(content, "parent_version") {
				t.Error("target section should not contain parent_version field")
			}
			// Must NOT contain depends_on
			if strings.Contains(content, "depends_on") {
				t.Error("target section should not contain depends_on field")
			}
		}
	}

	if !foundTarget {
		t.Error("did not find ROOT/a target section in output")
	}
}

// TestExistingCodeFilesIncludedInOutput verifies that existing code files
// listed in implements are included in the output with a path: header
// and no node: header.
func TestExistingCodeFilesIncludedInOutput(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Create the implemented file with known content
	testWriteFile(t, tmp, "src/a.go", `package a

func Hello() string { return "hello" }
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

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

// TestNonExistingCodeFilesOmittedFromOutput verifies that code files
// that don't exist on disk are not included in the output.
func TestNonExistingCodeFilesOmittedFromOutput(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	// Do NOT create src/a.go

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// The output should NOT contain a section for src/a.go
	if strings.Contains(text, "path: src/a.go") {
		t.Error("output should not contain a section for non-existing code file src/a.go")
	}
}

// TestAncestorWithNoPublicSection verifies that when an ancestor has no
// # Public section, its section is included in the output but with empty content.
func TestAncestorWithNoPublicSection(t *testing.T) {
	tmp := t.TempDir()

	// ROOT has no # Public section — only the node name and a private section
	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT

## Intent

Root intent (private).
`)

	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - src/a.go
---

# ROOT/a
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)

	// ROOT section should still be included (even though it has no # Public)
	if !strings.Contains(text, "node: ROOT") {
		t.Error("output should include ROOT ancestor section even when it has no # Public")
	}

	// Verify ROOT section exists but has empty/minimal content
	sections := strings.Split(text, "<<<FILE_")
	foundRoot := false

	for _, section := range sections {
		if section == "" {
			continue
		}
		if strings.Contains(section, "node: ROOT\n") {
			foundRoot = true
			// Private content should not be included
			if strings.Contains(section, "Root intent (private)") {
				t.Error("ROOT ancestor section should not contain private content")
			}
		}
	}

	if !foundRoot {
		t.Error("did not find ROOT ancestor section")
	}
}

// --- Failure Cases ---

// TestInvalidPrefix verifies that a logical name with an invalid
// prefix returns a tool error containing the expected message.
func TestInvalidPrefix(t *testing.T) {
	tmp := t.TempDir()
	testChdir(t, tmp)

	result := testCallHandler(t, "INVALID/something")

	if !result.IsError {
		t.Fatal("expected tool error for invalid prefix")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "target must be a ROOT/ or TEST/") {
		t.Errorf("unexpected error message: %s", text)
	}
}

// TestNonexistentSpecFile verifies that a logical name pointing
// to a nonexistent spec file returns a tool error.
func TestNonexistentSpecFile(t *testing.T) {
	tmp := t.TempDir()
	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/nonexistent")

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec file")
	}
}

// TestNoImplements verifies that a node without implements
// returns a tool error containing "has no implements".
func TestNoImplements(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	// ROOT/a without implements
	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
---

# ROOT/a
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for node with no implements")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "has no implements") {
		t.Errorf("unexpected error message: %s", text)
	}
}

// TestInvalidImplementsPathTraversal verifies that a path traversal
// in implements causes a tool error from path validation.
func TestInvalidImplementsPathTraversal(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
version: 1
parent_version: 1
implements:
  - "../../etc/passwd"
---

# ROOT/a
`)

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for path traversal in implements")
	}
}

// TestUnresolvableDependency verifies that a dependency whose spec file
// does not exist causes a tool error from chain resolution.
func TestUnresolvableDependency(t *testing.T) {
	tmp := t.TempDir()

	testWriteFile(t, tmp, "code-from-spec/spec/_node.md", `---
version: 1
---

# ROOT
`)

	testWriteFile(t, tmp, "code-from-spec/spec/a/_node.md", `---
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

	testChdir(t, tmp)

	result := testCallHandler(t, "ROOT/a")

	if !result.IsError {
		t.Fatal("expected tool error for unresolvable dependency")
	}
}
