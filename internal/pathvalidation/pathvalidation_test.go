// code-from-spec: TEST/tech_design/internal/pathvalidation@v12
package pathvalidation

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Happy Path ---

// TestSimpleRelativePath verifies that a straightforward relative path
// (no traversal, no absolute prefix) is accepted.
func TestSimpleRelativePath(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal/config/config.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestNestedPath verifies that a nested relative path is accepted.
func TestNestedPath(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("cmd/subagent-mcp/main.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestSingleFilename verifies that a bare filename (no directory component)
// is accepted.
func TestSingleFilename(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("main.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPathWithDotSegment verifies that a path containing a single-dot segment
// ("internal/./config/config.go") is accepted after being cleaned to
// "internal/config/config.go".
func TestPathWithDotSegment(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal/./config/config.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- Edge Cases ---

// TestPathWithTrailingSlash verifies that a path with a trailing slash is
// accepted (filepath.Clean removes the trailing slash).
func TestPathWithTrailingSlash(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal/config/", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPathWithDuplicateSeparators verifies that duplicate separators are
// accepted after being cleaned by filepath.Clean.
func TestPathWithDuplicateSeparators(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal//config//config.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- Failure Cases ---

// TestEmptyPath verifies that an empty string is rejected with an
// "path is empty" error.
func TestEmptyPath(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("", root)
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
	if !strings.Contains(err.Error(), "path is empty") {
		t.Fatalf("expected error containing %q, got: %v", "path is empty", err)
	}
}

// TestAbsolutePathWithLeadingSlash verifies that an absolute POSIX-style path
// is rejected with a "path is absolute" error.
func TestAbsolutePathWithLeadingSlash(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("/etc/passwd", root)
	if err == nil {
		t.Fatal("expected error for absolute path, got nil")
	}
	if !strings.Contains(err.Error(), "path is absolute") {
		t.Fatalf("expected error containing %q, got: %v", "path is absolute", err)
	}
}

// TestAbsolutePathWithDriveLetter verifies that a Windows-style drive-letter
// path (e.g., "C:\Windows\system32") is rejected on all platforms with a
// "path is absolute" error.
func TestAbsolutePathWithDriveLetter(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("C:\\Windows\\system32", root)
	if err == nil {
		t.Fatal("expected error for absolute path with drive letter, got nil")
	}
	if !strings.Contains(err.Error(), "path is absolute") {
		t.Fatalf("expected error containing %q, got: %v", "path is absolute", err)
	}
}

// TestSimpleTraversal verifies that an obvious relative traversal
// ("../../etc/passwd") is rejected with a "directory traversal" error.
func TestSimpleTraversal(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("../../etc/passwd", root)
	if err == nil {
		t.Fatal("expected error for traversal, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Fatalf("expected error containing %q, got: %v", "directory traversal", err)
	}
}

// TestEmbeddedTraversal verifies that a traversal embedded in the middle of
// a path ("internal/../../outside/file.go") is rejected with a
// "directory traversal" error.
func TestEmbeddedTraversal(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal/../../outside/file.go", root)
	if err == nil {
		t.Fatal("expected error for embedded traversal, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Fatalf("expected error containing %q, got: %v", "directory traversal", err)
	}
}

// TestSymlinkEscapingProjectRoot verifies that a path whose first component is
// a symlink pointing outside the project root is rejected with a
// "resolves outside project root" error.
//
// On Windows, symlink creation requires elevated privileges; the test is
// skipped when it cannot be performed.
func TestSymlinkEscapingProjectRoot(t *testing.T) {
	root := t.TempDir()

	// Create a directory outside the project root to act as the symlink target.
	outsideDir := t.TempDir()

	// Place a symlink inside the project root that points outside.
	symlinkPath := filepath.Join(root, "escape")
	err := os.Symlink(outsideDir, symlinkPath)
	if err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("skipping symlink test: symlink creation requires elevated privileges on Windows")
		}
		t.Fatalf("failed to create symlink: %v", err)
	}

	err = ValidatePath("escape/file.go", root)
	if err == nil {
		t.Fatal("expected error for symlink escaping project root, got nil")
	}
	if !strings.Contains(err.Error(), "resolves outside project root") {
		t.Fatalf("expected error containing %q, got: %v", "resolves outside project root", err)
	}
}

// TestTraversalDisguisedWithDotSegments verifies that a path that uses dot
// segments to disguise a traversal ("a/../../outside" cleans to "../outside")
// is rejected with a "directory traversal" error.
func TestTraversalDisguisedWithDotSegments(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("a/../../outside", root)
	if err == nil {
		t.Fatal("expected error for disguised traversal, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Fatalf("expected error containing %q, got: %v", "directory traversal", err)
	}
}
