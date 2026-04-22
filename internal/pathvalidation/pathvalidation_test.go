// spec: TEST/tech_design/internal/pathvalidation@v8

// Package pathvalidation_test contains tests for the pathvalidation package.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Context"
// Each test uses t.TempDir() as the project root.
package pathvalidation_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

// --- Happy Path Tests ---
// Spec ref: TEST/tech_design/internal/pathvalidation § "Happy Path"

// TestSimpleRelativePath verifies that a normal relative path is accepted.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Simple relative path"
func TestSimpleRelativePath(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("internal/config/config.go", root)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestNestedPath verifies that a deeper nested relative path is accepted.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Nested path"
func TestNestedPath(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("cmd/subagent-mcp/main.go", root)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestSingleFilename verifies that a simple filename with no directory is accepted.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Single filename"
func TestSingleFilename(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("main.go", root)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestPathWithDotSegment verifies that a path with a dot segment is accepted
// after cleaning resolves it to a normal path.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Path with dot segment"
func TestPathWithDotSegment(t *testing.T) {
	root := t.TempDir()
	// "internal/./config/config.go" cleans to "internal/config/config.go"
	err := pathvalidation.ValidatePath("internal/./config/config.go", root)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// --- Edge Case Tests ---
// Spec ref: TEST/tech_design/internal/pathvalidation § "Edge Cases"

// TestPathWithTrailingSlash verifies that a path ending with a slash is accepted.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Path with trailing slash"
func TestPathWithTrailingSlash(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("internal/config/", root)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestPathWithDuplicateSeparators verifies that duplicate separators are handled
// gracefully by filepath.Clean.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Path with duplicate separators"
func TestPathWithDuplicateSeparators(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("internal//config//config.go", root)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// --- Failure Case Tests ---
// Spec ref: TEST/tech_design/internal/pathvalidation § "Failure Cases"

// TestEmptyPath verifies that an empty path is rejected with the correct message.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Empty path"
func TestEmptyPath(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("", root)
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
	if !strings.Contains(err.Error(), "path is empty") {
		t.Errorf("expected error containing %q, got: %v", "path is empty", err)
	}
}

// TestAbsolutePathWithLeadingSlash verifies that a Unix-style absolute path is rejected.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Absolute path with leading slash"
func TestAbsolutePathWithLeadingSlash(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("/etc/passwd", root)
	if err == nil {
		t.Fatal("expected error for absolute path, got nil")
	}
	if !strings.Contains(err.Error(), "path is absolute") {
		t.Errorf("expected error containing %q, got: %v", "path is absolute", err)
	}
}

// TestAbsolutePathWithDriveLetter verifies that a Windows-style drive path is rejected.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Absolute path with drive letter (Windows-style)"
func TestAbsolutePathWithDriveLetter(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath(`C:\Windows\system32`, root)
	if err == nil {
		t.Fatal("expected error for drive-letter path, got nil")
	}
	if !strings.Contains(err.Error(), "path is absolute") {
		t.Errorf("expected error containing %q, got: %v", "path is absolute", err)
	}
}

// TestSimpleTraversal verifies that a leading traversal sequence is rejected.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Simple traversal"
func TestSimpleTraversal(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("../../etc/passwd", root)
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Errorf("expected error containing %q, got: %v", "directory traversal", err)
	}
}

// TestEmbeddedTraversal verifies that a traversal sequence embedded in a path is rejected.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Embedded traversal"
func TestEmbeddedTraversal(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("internal/../../outside/file.go", root)
	if err == nil {
		t.Fatal("expected error for embedded traversal path, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Errorf("expected error containing %q, got: %v", "directory traversal", err)
	}
}

// TestSymlinkEscapingProjectRoot verifies that a symlink pointing outside the
// project root is rejected.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Symlink escaping project root"
func TestSymlinkEscapingProjectRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Symlink creation on Windows typically requires elevated privileges;
		// skip rather than fail to keep CI reliable on standard Windows runners.
		t.Skip("skipping symlink test on Windows")
	}

	root := t.TempDir()
	// outsideDir is outside the project root — use the OS temp dir's parent or
	// another temp directory entirely.
	outsideDir := t.TempDir()

	// Create a symlink inside root that points to outsideDir.
	symlinkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Attempt to validate a path through the symlink.
	err := pathvalidation.ValidatePath("escape/file.go", root)
	if err == nil {
		t.Fatal("expected error for symlink escaping project root, got nil")
	}
	if !strings.Contains(err.Error(), "resolves outside project root") {
		t.Errorf("expected error containing %q, got: %v", "resolves outside project root", err)
	}
}

// TestTraversalDisguisedWithDotSegments verifies that a traversal hidden behind
// dot segments is rejected after cleaning.
// Spec ref: TEST/tech_design/internal/pathvalidation § "Traversal disguised with dot segments"
func TestTraversalDisguisedWithDotSegments(t *testing.T) {
	root := t.TempDir()
	err := pathvalidation.ValidatePath("a/../../outside", root)
	if err == nil {
		t.Fatal("expected error for disguised traversal path, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Errorf("expected error containing %q, got: %v", "directory traversal", err)
	}
}
