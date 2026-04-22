// spec: TEST/tech_design/internal/frontmatter@v8

// Package frontmatter — tests for ParseFrontmatter.
//
// Each test uses t.TempDir() to create an isolated temporary directory.
// Test files are written with controlled frontmatter content.
// ParseFrontmatter is called with the path to each test file and the
// result is verified against the spec scenarios defined in default.test.md.
//
// No test framework beyond the standard "testing" package is used, per the
// constraint in ROOT/tech_design.
package frontmatter

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a helper that creates a file inside dir with the given content.
// It returns the full path to the created file.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Happy Path
// ---------------------------------------------------------------------------

// TestParseFrontmatter_CompleteFields covers the happy path where all
// recognised fields (depends_on with and without filter, implements) are
// present in the frontmatter.
//
// Spec ref: "Parses complete frontmatter"
func TestParseFrontmatter_CompleteFields(t *testing.T) {
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
# body is ignored
`
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify depends_on — two entries expected.
	if len(fm.DependsOn) != 2 {
		t.Fatalf("DependsOn: want 2 entries, got %d", len(fm.DependsOn))
	}

	// First entry: ROOT/other, no filter.
	if fm.DependsOn[0].LogicalName != "ROOT/other" {
		t.Errorf("DependsOn[0].LogicalName: want %q, got %q", "ROOT/other", fm.DependsOn[0].LogicalName)
	}
	if fm.DependsOn[0].Filter != nil {
		t.Errorf("DependsOn[0].Filter: want nil, got %v", fm.DependsOn[0].Filter)
	}

	// Second entry: EXTERNAL/database, with one filter glob.
	if fm.DependsOn[1].LogicalName != "EXTERNAL/database" {
		t.Errorf("DependsOn[1].LogicalName: want %q, got %q", "EXTERNAL/database", fm.DependsOn[1].LogicalName)
	}
	if len(fm.DependsOn[1].Filter) != 1 || fm.DependsOn[1].Filter[0] != "schema/*.sql" {
		t.Errorf("DependsOn[1].Filter: want [schema/*.sql], got %v", fm.DependsOn[1].Filter)
	}

	// Verify implements — two file paths expected.
	wantImpl := []string{
		"internal/config/config.go",
		"internal/config/config_test.go",
	}
	if len(fm.Implements) != len(wantImpl) {
		t.Fatalf("Implements: want %d entries, got %d", len(wantImpl), len(fm.Implements))
	}
	for i, want := range wantImpl {
		if fm.Implements[i] != want {
			t.Errorf("Implements[%d]: want %q, got %q", i, want, fm.Implements[i])
		}
	}
}

// TestParseFrontmatter_OnlyImplements verifies that a file containing only
// the implements field (no depends_on) is parsed correctly, with DependsOn
// being nil.
//
// Spec ref: "Parses frontmatter with only implements"
func TestParseFrontmatter_OnlyImplements(t *testing.T) {
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

	// DependsOn must be nil — it was not declared in the frontmatter.
	if fm.DependsOn != nil {
		t.Errorf("DependsOn: want nil, got %v", fm.DependsOn)
	}

	// Implements must have exactly one entry.
	if len(fm.Implements) != 1 || fm.Implements[0] != "internal/config/config.go" {
		t.Errorf("Implements: want [internal/config/config.go], got %v", fm.Implements)
	}
}

// TestParseFrontmatter_NoRelevantFields verifies that a frontmatter block
// containing only untracked fields (e.g. version) returns zero-value slices
// with no error.
//
// Spec ref: "Parses frontmatter with no relevant fields"
func TestParseFrontmatter_NoRelevantFields(t *testing.T) {
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
		t.Errorf("DependsOn: want nil, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements: want nil, got %v", fm.Implements)
	}
}

// TestParseFrontmatter_UnknownFieldsIgnored verifies that extra, unrecognised
// YAML keys in the frontmatter are silently ignored and do not cause an error.
//
// Spec ref: "Ignores unknown frontmatter fields"
func TestParseFrontmatter_UnknownFieldsIgnored(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
parent_version: 1
some_future_field: hello
another: 42
---
`
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error for unknown fields: %v", err)
	}

	// Known fields that were absent must still be nil/zero.
	if fm.DependsOn != nil {
		t.Errorf("DependsOn: want nil, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements: want nil, got %v", fm.Implements)
	}
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

// TestParseFrontmatter_EmptyFrontmatter verifies that a file whose frontmatter
// block contains no content at all (consecutive --- delimiters) returns a
// zero-value Frontmatter with no error.
//
// Spec ref: "Empty frontmatter"
func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	// Two consecutive delimiter lines — the frontmatter block is empty.
	content := "---\n---\n"
	path := writeFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error for empty frontmatter: %v", err)
	}

	if fm.DependsOn != nil {
		t.Errorf("DependsOn: want nil, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements: want nil, got %v", fm.Implements)
	}
}

// TestParseFrontmatter_OnlyFrontmatterNoBody verifies that a file whose
// content ends immediately after the closing delimiter (no body) is parsed
// without error. The body is never read per the efficiency contract.
//
// Spec ref: "File with only frontmatter, nothing after"
func TestParseFrontmatter_OnlyFrontmatterNoBody(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\n---"
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error for file with no body: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Failure Cases
// ---------------------------------------------------------------------------

// TestParseFrontmatter_FileNotExist verifies that calling ParseFrontmatter
// with a path that does not exist returns an error that contains the path.
//
// Spec ref: "File does not exist"
func TestParseFrontmatter_FileNotExist(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist.md")

	_, err := ParseFrontmatter(nonExistent)
	if err == nil {
		t.Fatal("expected an error for non-existent file, got nil")
	}

	// The error message must mention the file path so callers can diagnose
	// which file caused the failure.
	if !containsString(err.Error(), nonExistent) {
		t.Errorf("error %q does not contain path %q", err.Error(), nonExistent)
	}
}

// TestParseFrontmatter_NoDelimiters verifies that a file with no "---"
// delimiters at all returns an error indicating that frontmatter was not found.
//
// Spec ref: "No frontmatter delimiters"
func TestParseFrontmatter_NoDelimiters(t *testing.T) {
	dir := t.TempDir()
	content := "Just some text.\n"
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected an error for file with no frontmatter delimiters, got nil")
	}

	// The error must signal that frontmatter was not found.
	if !containsString(err.Error(), "frontmatter not found") {
		t.Errorf("error %q does not indicate frontmatter not found", err.Error())
	}
}

// TestParseFrontmatter_MalformedYAML verifies that invalid YAML between the
// frontmatter delimiters causes a parse error.
//
// Spec ref: "Malformed YAML in frontmatter"
func TestParseFrontmatter_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	// "version: [invalid" is not valid YAML — the list is never closed.
	content := "---\nversion: [invalid\n---\n"
	path := writeFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected a parse error for malformed YAML, got nil")
	}

	// The error must mention parsing failure (not a read error).
	if !containsString(err.Error(), "error parsing frontmatter") {
		t.Errorf("error %q does not indicate parse failure", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// containsString reports whether substr appears anywhere in s.
// Using a simple contains check avoids importing "strings" at the test level
// while keeping error assertions readable.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

// findSubstr performs a linear scan to locate substr within s.
func findSubstr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
