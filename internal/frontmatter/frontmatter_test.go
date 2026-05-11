// code-from-spec: TEST/tech_design/internal/frontmatter@v15

// Package frontmatter provides tests for the frontmatter package.
// Tests use t.TempDir() for isolation and validate ParseFrontmatter
// behavior across happy-path, edge-case, and failure scenarios.
package frontmatter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// testIntPtr returns a pointer to the given int value.
// Used to construct expected *int values in assertions.
func testIntPtr(v int) *int {
	return &v
}

// testWriteFile writes content to a file at filePath, failing the
// test immediately if the write fails.
func testWriteFile(t *testing.T, filePath string, content string) {
	t.Helper()
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

// --- Happy Path ---

// TestParsesCompleteFrontmatter verifies that all fields are correctly
// parsed from a file containing version, parent_version, depends_on,
// and implements.
func TestParsesCompleteFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 3
parent_version: 2
depends_on:
  - path: ROOT/other
    version: 1
  - path: ROOT/architecture/backend
    version: 5
implements:
  - internal/config/config.go
  - internal/config/config_test.go
---
`
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Version
	if fm.Version != 3 {
		t.Errorf("expected Version 3, got %d", fm.Version)
	}

	// ParentVersion
	if fm.ParentVersion == nil {
		t.Fatal("expected non-nil ParentVersion")
	}
	if *fm.ParentVersion != 2 {
		t.Errorf("expected ParentVersion 2, got %d", *fm.ParentVersion)
	}

	// SubjectVersion should be nil
	if fm.SubjectVersion != nil {
		t.Errorf("expected nil SubjectVersion, got %v", fm.SubjectVersion)
	}

	// DependsOn: two entries
	if len(fm.DependsOn) != 2 {
		t.Fatalf("expected 2 DependsOn entries, got %d", len(fm.DependsOn))
	}
	if fm.DependsOn[0].LogicalName != "ROOT/other" {
		t.Errorf("expected DependsOn[0].LogicalName %q, got %q", "ROOT/other", fm.DependsOn[0].LogicalName)
	}
	if fm.DependsOn[0].Version != 1 {
		t.Errorf("expected DependsOn[0].Version 1, got %d", fm.DependsOn[0].Version)
	}
	if fm.DependsOn[1].LogicalName != "ROOT/architecture/backend" {
		t.Errorf("expected DependsOn[1].LogicalName %q, got %q", "ROOT/architecture/backend", fm.DependsOn[1].LogicalName)
	}
	if fm.DependsOn[1].Version != 5 {
		t.Errorf("expected DependsOn[1].Version 5, got %d", fm.DependsOn[1].Version)
	}

	// Implements: two entries
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

// TestParsesTestNodeFrontmatter verifies that subject_version is
// correctly parsed, and that parent_version is nil when absent.
func TestParsesTestNodeFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 2
subject_version: 5
implements:
  - internal/config/config_test.go
---
`
	filePath := filepath.Join(dir, "default.test.md")
	testWriteFile(t, filePath, content)

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Version
	if fm.Version != 2 {
		t.Errorf("expected Version 2, got %d", fm.Version)
	}

	// ParentVersion should be nil
	if fm.ParentVersion != nil {
		t.Errorf("expected nil ParentVersion, got %v", fm.ParentVersion)
	}

	// SubjectVersion
	if fm.SubjectVersion == nil {
		t.Fatal("expected non-nil SubjectVersion")
	}
	if *fm.SubjectVersion != 5 {
		t.Errorf("expected SubjectVersion 5, got %d", *fm.SubjectVersion)
	}

	// DependsOn should be nil
	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}

	// Implements
	if len(fm.Implements) != 1 || fm.Implements[0] != "internal/config/config_test.go" {
		t.Errorf("expected Implements [\"internal/config/config_test.go\"], got %v", fm.Implements)
	}
}

// TestParsesOnlyVersion verifies that a frontmatter containing only
// the version field is parsed correctly with all pointer/slice fields nil.
func TestParsesOnlyVersion(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 5
---
`
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version != 5 {
		t.Errorf("expected Version 5, got %d", fm.Version)
	}
	if fm.ParentVersion != nil {
		t.Errorf("expected nil ParentVersion, got %v", fm.ParentVersion)
	}
	if fm.SubjectVersion != nil {
		t.Errorf("expected nil SubjectVersion, got %v", fm.SubjectVersion)
	}
	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected nil Implements, got %v", fm.Implements)
	}
}

// TestParsesOnlyImplements verifies that a frontmatter with version
// and implements (but no depends_on) has nil DependsOn.
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
	testWriteFile(t, filePath, content)

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

// TestIgnoresUnknownFields verifies that unknown YAML keys do not
// cause errors, and that known fields are still parsed correctly.
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
	testWriteFile(t, filePath, content)

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version != 1 {
		t.Errorf("expected Version 1, got %d", fm.Version)
	}
	if fm.ParentVersion == nil || *fm.ParentVersion != 1 {
		t.Errorf("expected ParentVersion 1, got %v", fm.ParentVersion)
	}
	// Unknown fields are ignored; DependsOn and Implements should be nil.
	if fm.DependsOn != nil {
		t.Errorf("expected nil DependsOn, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected nil Implements, got %v", fm.Implements)
	}
}

// --- Edge Cases ---

// TestEmptyFrontmatter verifies that a file with an empty frontmatter
// block (no fields at all) returns ErrMissingVersion.
func TestEmptyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
---
`
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	_, err := ParseFrontmatter(filePath)
	if !errors.Is(err, ErrMissingVersion) {
		t.Errorf("expected errors.Is(err, ErrMissingVersion), got: %v", err)
	}
}

// TestFileWithOnlyFrontmatter verifies that a file that ends
// immediately after the closing delimiter is parsed without error.
// The body is not read by ParseFrontmatter, so an empty body is fine.
func TestFileWithOnlyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: 1
---
`
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	fm, err := ParseFrontmatter(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version != 1 {
		t.Errorf("expected Version 1, got %d", fm.Version)
	}
}

// --- Failure Cases ---

// TestFileDoesNotExist verifies that calling ParseFrontmatter with a
// non-existent path returns an error wrapping ErrRead.
func TestFileDoesNotExist(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist.md")

	_, err := ParseFrontmatter(nonExistent)
	if !errors.Is(err, ErrRead) {
		t.Errorf("expected errors.Is(err, ErrRead), got: %v", err)
	}
}

// TestNoFrontmatterDelimiters verifies that a file with no --- lines
// returns an error wrapping ErrFrontmatterMissing.
func TestNoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	content := "Just some text.\n"
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	_, err := ParseFrontmatter(filePath)
	if !errors.Is(err, ErrFrontmatterMissing) {
		t.Errorf("expected errors.Is(err, ErrFrontmatterMissing), got: %v", err)
	}
}

// TestMalformedYAML verifies that invalid YAML in the frontmatter
// block returns an error wrapping ErrFrontmatterParse.
func TestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := `---
version: [invalid
---
`
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	_, err := ParseFrontmatter(filePath)
	if !errors.Is(err, ErrFrontmatterParse) {
		t.Errorf("expected errors.Is(err, ErrFrontmatterParse), got: %v", err)
	}
}

// TestMissingVersionField verifies that a frontmatter block that
// omits the version field returns an error wrapping ErrMissingVersion.
func TestMissingVersionField(t *testing.T) {
	dir := t.TempDir()
	content := `---
parent_version: 1
implements:
  - internal/config/config.go
---
`
	filePath := filepath.Join(dir, "_node.md")
	testWriteFile(t, filePath, content)

	_, err := ParseFrontmatter(filePath)
	if !errors.Is(err, ErrMissingVersion) {
		t.Errorf("expected errors.Is(err, ErrMissingVersion), got: %v", err)
	}
}
