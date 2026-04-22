// spec: TEST/tech_design/internal/pathvalidation@v9
package pathvalidation

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Happy Path ---

func TestSimpleRelativePath(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal/config/config.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestNestedPath(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("cmd/subagent-mcp/main.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSingleFilename(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("main.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestPathWithDotSegment(t *testing.T) {
	// "internal/./config/config.go" cleans to "internal/config/config.go"
	root := t.TempDir()
	err := ValidatePath("internal/./config/config.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- Edge Cases ---

func TestPathWithTrailingSlash(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath("internal/config/", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestPathWithDuplicateSeparators(t *testing.T) {
	// Duplicate separators are cleaned by filepath.Clean.
	root := t.TempDir()
	err := ValidatePath("internal//config//config.go", root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- Failure Cases ---

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

func TestAbsolutePathWithDriveLetter(t *testing.T) {
	// Windows-style drive letter path should be rejected on all platforms.
	root := t.TempDir()
	err := ValidatePath("C:\\Windows\\system32", root)
	if err == nil {
		t.Fatal("expected error for absolute path with drive letter, got nil")
	}
	if !strings.Contains(err.Error(), "path is absolute") {
		t.Fatalf("expected error containing %q, got: %v", "path is absolute", err)
	}
}

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

func TestSymlinkEscapingProjectRoot(t *testing.T) {
	// Symlink creation may require elevated privileges on Windows;
	// skip if it fails.
	root := t.TempDir()

	// Create a directory outside the project root to be the symlink target.
	outsideDir := t.TempDir()

	// Create a symlink inside the project root pointing outside.
	symlinkPath := filepath.Join(root, "escape")
	err := os.Symlink(outsideDir, symlinkPath)
	if err != nil {
		// On Windows, symlink creation may require admin privileges.
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

func TestTraversalDisguisedWithDotSegments(t *testing.T) {
	// "a/../../outside" cleans to "../outside" which contains ".."
	root := t.TempDir()
	err := ValidatePath("a/../../outside", root)
	if err == nil {
		t.Fatal("expected error for disguised traversal, got nil")
	}
	if !strings.Contains(err.Error(), "directory traversal") {
		t.Fatalf("expected error containing %q, got: %v", "directory traversal", err)
	}
}
