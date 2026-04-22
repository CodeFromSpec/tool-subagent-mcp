// spec: TEST/tech_design/internal/frontmatter@v10

package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsesCompleteFrontmatter verifies that all fields are correctly
// parsed from a file containing depends_on (with and without filter)
// and implements.
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
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify DependsOn has two entries.
	if len(fm.DependsOn) != 2 {
		t.Fatalf("expected 2 DependsOn entries, got %d", len(fm.DependsOn))
	}

	// First entry: ROOT/other with no filter.
	if fm.DependsOn[0].LogicalName != "ROOT/other" {
		t.Errorf("expected LogicalName %q, got %q", "ROOT/other", fm.DependsOn[0].LogicalName)
	}
	if fm.DependsOn[0].Filter != nil {
		t.Errorf("expected nil Filter for first entry, got %v", fm.DependsOn[0].Filter)
	}

	// Second entry: EXTERNAL/database with filter.
	if fm.DependsOn[1].LogicalName != "EXTERNAL/database" {
		t.Errorf("expected LogicalName %q, got %q", "EXTERNAL/database", fm.DependsOn[1].LogicalName)
	}
	if len(fm.DependsOn[1].Filter) != 1 || fm.DependsOn[1].Filter[0] != "schema/*.sql" {
		t.Errorf("expected Filter [\"schema/*.sql\"], got %v", fm.DependsOn[1].Filter)
	}

	// Verify Implements.
	if len(fm.Implements) != 2 {
		t.Fatalf("expected 2 Implements entries, got %d", len(fm.Implements))
	}
	if fm.Implements[0] != "internal/config/config.go" {
		t.Errorf("expected Implements[0] %q, got %q", "internal/config/config.go", fm.Implements[0])
	}
	if fm.Implements[1] != "internal/config/config_test.go" {
		t.Errorf("expected Implements[1] %q, got %q", "internal/config/config_test.go", fm.Implements[1])
	}
}

// TestParsesOnlyImplements verifies parsing when only the implements
// field is present. DependsOn should be nil.
func TestParsesOnlyImplements(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
parent_version: 1
implements:
  - internal/config/config.go
---
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if len(fm.Implements) != 1 || fm.Implements[0] != "internal/config/config.go" {
		t.Errorf("expected Implements [\"internal/config/config.go\"], got %v", fm.Implements)
	}
}

// TestParsesNoRelevantFields verifies that a file with only version
// (no depends_on, no implements) parses without error and returns
// nil for both fields.
func TestParsesNoRelevantFields(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 5
---
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected nil Implements, got %v", fm.Implements)
	}
}

// TestIgnoresUnknownFields verifies that unknown YAML fields in the
// frontmatter do not cause errors and known fields are still parsed.
func TestIgnoresUnknownFields(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
parent_version: 1
some_future_field: hello
another: 42
---
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No relevant fields set, so both should be nil.
	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected nil Implements, got %v", fm.Implements)
	}
}

// TestEmptyFrontmatter verifies that an empty frontmatter block
// (just two --- delimiters) returns zero/nil fields without error.
func TestEmptyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
---
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected nil Implements, got %v", fm.Implements)
	}
}

// TestFileWithOnlyFrontmatter verifies that a file ending right
// after the closing --- delimiter is handled correctly.
func TestFileWithOnlyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
---
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// version is not extracted by the parser, so all fields nil.
	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected nil Implements, got %v", fm.Implements)
	}
}

// TestFileDoesNotExist verifies that parsing a non-existent file
// returns an error containing the file path.
func TestFileDoesNotExist(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist.md")

	_, err := ParseFrontmatter(nonExistent)
	if err == nil {
		t.Fatal("expected an error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), nonExistent) {
		t.Errorf("expected error to contain path %q, got: %v", nonExistent, err)
	}
}

// TestNoFrontmatterDelimiters verifies that a file without any ---
// delimiters returns an error indicating frontmatter was not found.
func TestNoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	content := `Just some text.
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := ParseFrontmatter(filePath)
	if err == nil {
		t.Fatal("expected an error for missing frontmatter, got nil")
	}
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("expected error to contain path %q, got: %v", filePath, err)
	}
}

// TestMalformedYAML verifies that invalid YAML between the
// frontmatter delimiters produces a parse error.
func TestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: [invalid
---
`
	filePath := filepath.Join(dir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := ParseFrontmatter(filePath)
	if err == nil {
		t.Fatal("expected an error for malformed YAML, got nil")
	}
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("expected error to contain path %q, got: %v", filePath, err)
	}
}
