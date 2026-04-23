// spec: TEST/tech_design/internal/tools/write_file@v4
package write_file

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupSpecTree creates a minimal spec tree inside tmpDir so that
// PathFromLogicalName("ROOT/a") resolves to a file with the given
// implements list. Returns the original working directory so the
// caller can restore it in a cleanup function.
func setupSpecTree(t *testing.T, tmpDir string, implements []string) string {
	t.Helper()

	// Save original working directory and change to tmpDir.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	// Build the frontmatter YAML for the spec file.
	specDir := filepath.Join(tmpDir, "code-from-spec", "spec", "a")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("---\nversion: 1\nimplements:\n")
	for _, imp := range implements {
		sb.WriteString("  - " + imp + "\n")
	}
	sb.WriteString("---\n# Node\n")

	specFile := filepath.Join(specDir, "_node.md")
	if err := os.WriteFile(specFile, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	return origDir
}

// callHandler is a convenience wrapper that invokes HandleWriteFile
// with the given arguments.
func callHandler(t *testing.T, logicalName, path, content string) *mcp.CallToolResult {
	t.Helper()
	result, _, err := HandleWriteFile(
		context.Background(),
		&mcp.CallToolRequest{},
		WriteFileArgs{
			LogicalName: logicalName,
			Path:        path,
			Content:     content,
		},
	)
	if err != nil {
		t.Fatalf("HandleWriteFile returned unexpected Go error: %v", err)
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
		t.Fatal("result content is not TextContent")
	}
	return tc.Text
}

// --- Happy Path ---

// TestWritesFileSuccessfully verifies that the handler writes a file
// to disk and returns a success message when given valid inputs.
func TestWritesFileSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"output/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/a", "output/file.go", "package main")

	// Verify success (not a tool error).
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	// Verify the result text.
	text := resultText(t, result)
	if text != "wrote output/file.go" {
		t.Errorf("unexpected result text: %q", text)
	}

	// Verify the file exists on disk with correct content.
	written, err := os.ReadFile(filepath.Join(tmpDir, "output", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(written) != "package main" {
		t.Errorf("unexpected file content: %q", string(written))
	}
}

// TestCreatesIntermediateDirectories verifies that missing parent
// directories are created automatically.
func TestCreatesIntermediateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"deep/nested/dir/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/a", "deep/nested/dir/file.go", "package deep")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	// Verify the file was created.
	if _, err := os.Stat(filepath.Join(tmpDir, "deep", "nested", "dir", "file.go")); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

// TestOverwritesExistingFile verifies that calling the handler on
// an existing file replaces its content.
func TestOverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"output/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	// Write initial file.
	outDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "file.go"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	result := callHandler(t, "ROOT/a", "output/file.go", "new content")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	written, err := os.ReadFile(filepath.Join(outDir, "file.go"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(written) != "new content" {
		t.Errorf("expected 'new content', got %q", string(written))
	}
}

// TestPathWithBackslashesIsNormalized verifies that backslash paths
// are normalized to forward slashes and matched against implements.
func TestPathWithBackslashesIsNormalized(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"output/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	// Pass a path with backslashes — should be normalized to forward slashes.
	result := callHandler(t, "ROOT/a", "output\\file.go", "package main")

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	text := resultText(t, result)
	if text != "wrote output/file.go" {
		t.Errorf("unexpected result text: %q", text)
	}
}

// --- Failure Cases ---

// TestInvalidLogicalNamePrefix verifies that a logical name not
// starting with ROOT/ or TEST/ is rejected.
func TestInvalidLogicalNamePrefix(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"some/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "EXTERNAL/something", "some/file.go", "content")

	if !result.IsError {
		t.Fatal("expected tool error for invalid prefix, got success")
	}
}

// TestNonexistentLogicalName verifies that a logical name with no
// corresponding spec file produces a tool error.
func TestNonexistentLogicalName(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir but do NOT create a spec tree for ROOT/nonexistent.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/nonexistent", "some/file.go", "content")

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent logical name, got success")
	}
}

// TestPathNotInImplements verifies that a path not listed in the
// node's implements field is rejected with an actionable error.
func TestPathNotInImplements(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"allowed/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/a", "other/file.go", "content")

	if !result.IsError {
		t.Fatal("expected tool error for path not in implements, got success")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("expected error to contain 'path not allowed', got: %s", text)
	}
	// The error should list the allowed paths.
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("expected error to list allowed paths, got: %s", text)
	}
}

// TestPathTraversalAttempt verifies that a path containing directory
// traversal is rejected by ValidatePath.
func TestPathTraversalAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"../../etc/passwd"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/a", "../../etc/passwd", "malicious")

	if !result.IsError {
		t.Fatal("expected tool error for path traversal, got success")
	}
}

// TestEmptyPath verifies that an empty path is rejected.
func TestEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	origDir := setupSpecTree(t, tmpDir, []string{"some/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/a", "", "content")

	if !result.IsError {
		t.Fatal("expected tool error for empty path, got success")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "path is empty") {
		t.Errorf("expected error to contain 'path is empty', got: %s", text)
	}
}

// TestSymlinkEscapingProjectRoot verifies that a symlink pointing
// outside the project root is detected and rejected.
func TestSymlinkEscapingProjectRoot(t *testing.T) {
	// Symlink creation on Windows requires elevated privileges,
	// so skip if we cannot create one.
	if runtime.GOOS == "windows" {
		// Attempt to create a symlink; skip if it fails (no privilege).
		testLink := filepath.Join(t.TempDir(), "testlink")
		if err := os.Symlink(os.TempDir(), testLink); err != nil {
			t.Skip("skipping symlink test: insufficient privileges on Windows")
		}
	}

	tmpDir := t.TempDir()

	// Create a directory outside the project root to be the symlink target.
	outsideDir := t.TempDir()

	// Create the symlink inside tmpDir pointing outside.
	symlinkPath := filepath.Join(tmpDir, "escape")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Set up spec tree with the symlink-based path in implements.
	origDir := setupSpecTree(t, tmpDir, []string{"escape/file.go"})
	t.Cleanup(func() { os.Chdir(origDir) })

	result := callHandler(t, "ROOT/a", "escape/file.go", "content")

	if !result.IsError {
		t.Fatal("expected tool error for symlink escaping project root, got success")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("expected error to contain 'resolves outside project root', got: %s", text)
	}
}
