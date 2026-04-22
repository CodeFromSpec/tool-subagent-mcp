// spec: TEST/tech_design/internal/frontmatter@v9

// Package frontmatter_test exercises ParseFrontmatter against all scenarios
// described in TEST/tech_design/internal/frontmatter (default.test.md).
//
// Each test creates an isolated temp directory via t.TempDir(), writes a
// controlled file, and asserts on the returned *Frontmatter or error.
// No external test framework is used — only the standard "testing" package.
// (Spec ref: ROOT/tech_design § "Constraints")
package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- helpers ----------------------------------------------------------------

// writeFile creates a file with the given content inside dir and returns the
// full path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

// ---- Happy Path -------------------------------------------------------------

// TestParsesCompleteFrontmatter verifies that all supported fields are parsed
// correctly when present.
// Spec ref: TEST/tech_design/internal/frontmatter § "Parses complete frontmatter"
func TestParsesCompleteFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 3
parent_version: 2
depends_on:
  - path: ROOT/other
    version: 1
  - path: EXTERNAL/database
    version: 5
    filter:
      - "schema/*.sql"
implements:
  - internal/config/config.go
  - internal/config/config_test.go
---
Some body text that must not be read.
`
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spec ref: TEST § "Parses complete frontmatter" — DependsOn has two entries.
	if got := len(fm.DependsOn); got != 2 {
		t.Fatalf("DependsOn length: got %d, want 2", got)
	}

	// First entry: LogicalName = "ROOT/other", Filter = nil
	if fm.DependsOn[0].LogicalName != "ROOT/other" {
		t.Errorf("DependsOn[0].LogicalName = %q, want %q", fm.DependsOn[0].LogicalName, "ROOT/other")
	}
	if fm.DependsOn[0].Filter != nil {
		t.Errorf("DependsOn[0].Filter = %v, want nil", fm.DependsOn[0].Filter)
	}

	// Second entry: LogicalName = "EXTERNAL/database", Filter = ["schema/*.sql"]
	if fm.DependsOn[1].LogicalName != "EXTERNAL/database" {
		t.Errorf("DependsOn[1].LogicalName = %q, want %q", fm.DependsOn[1].LogicalName, "EXTERNAL/database")
	}
	if len(fm.DependsOn[1].Filter) != 1 || fm.DependsOn[1].Filter[0] != "schema/*.sql" {
		t.Errorf("DependsOn[1].Filter = %v, want [schema/*.sql]", fm.DependsOn[1].Filter)
	}

	// Implements = ["internal/config/config.go", "internal/config/config_test.go"]
	wantImpl := []string{"internal/config/config.go", "internal/config/config_test.go"}
	if len(fm.Implements) != len(wantImpl) {
		t.Fatalf("Implements length: got %d, want %d", len(fm.Implements), len(wantImpl))
	}
	for i, want := range wantImpl {
		if fm.Implements[i] != want {
			t.Errorf("Implements[%d] = %q, want %q", i, fm.Implements[i], want)
		}
	}
}

// TestParsesOnlyImplements verifies that a file with only implements is parsed
// correctly and DependsOn is nil.
// Spec ref: TEST § "Parses frontmatter with only implements"
func TestParsesOnlyImplements(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
parent_version: 1
implements:
  - internal/config/config.go
---
`
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil", fm.DependsOn)
	}
	if len(fm.Implements) != 1 || fm.Implements[0] != "internal/config/config.go" {
		t.Errorf("Implements = %v, want [internal/config/config.go]", fm.Implements)
	}
}

// TestParsesNoRelevantFields verifies that a file with only version produces
// nil DependsOn and nil Implements without error.
// Spec ref: TEST § "Parses frontmatter with no relevant fields"
func TestParsesNoRelevantFields(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 5
---
`
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements = %v, want nil", fm.Implements)
	}
}

// TestIgnoresUnknownFields verifies that extra/future YAML fields do not cause
// an error and that known fields are still parsed correctly.
// Spec ref: TEST § "Ignores unknown frontmatter fields"
func TestIgnoresUnknownFields(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
parent_version: 1
some_future_field: hello
another: 42
---
`
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error for unknown fields: %v", err)
	}
}

// ---- Edge Cases -------------------------------------------------------------

// TestEmptyFrontmatter verifies that a file with an empty YAML block (--- ---)
// produces all-nil fields and no error.
// Spec ref: TEST § "Empty frontmatter"
func TestEmptyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\n---\n"
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements = %v, want nil", fm.Implements)
	}
}

// TestOnlyFrontmatterNoBody verifies that a file which ends immediately after
// the closing delimiter parses without error.
// Spec ref: TEST § "File with only frontmatter, nothing after"
func TestOnlyFrontmatterNoBody(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\n---\n"
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- Failure Cases ----------------------------------------------------------

// TestFileDoesNotExist verifies that a non-existent path returns an error that
// contains the file path.
// Spec ref: TEST § "File does not exist"
// Spec ref: ROOT/tech_design/internal/frontmatter § "Error handling"
//   — "error reading <path>: <underlying error>"
func TestFileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	nonexistent := filepath.Join(dir, "does_not_exist.md")

	_, err := ParseFrontmatter(nonexistent)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), nonexistent) {
		t.Errorf("error %q does not contain path %q", err.Error(), nonexistent)
	}
}

// TestNoFrontmatterDelimiters verifies that a file with no "---" lines returns
// an error indicating frontmatter was not found.
// Spec ref: TEST § "No frontmatter delimiters"
// Spec ref: ROOT/tech_design/internal/frontmatter § "Error handling"
//   — "frontmatter not found in <path>"
func TestNoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	content := "Just some text.\n"
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected error for missing frontmatter, got nil")
	}
	// Error must indicate frontmatter not found and include the path.
	if !strings.Contains(err.Error(), "frontmatter not found") {
		t.Errorf("error %q does not contain 'frontmatter not found'", err.Error())
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

// TestMalformedYAML verifies that invalid YAML inside the frontmatter block
// returns an error indicating a parse failure.
// Spec ref: TEST § "Malformed YAML in frontmatter"
// Spec ref: ROOT/tech_design/internal/frontmatter § "Error handling"
//   — "error parsing frontmatter in <path>: <underlying error>"
func TestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: [invalid\n---\n"
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
	// Error must reference the path and indicate a parsing problem.
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}
