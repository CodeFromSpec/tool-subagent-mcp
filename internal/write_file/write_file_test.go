// code-from-spec: TEST/tech_design/internal/tools/write_file@v12
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

// testSetupSpecTree creates a minimal spec tree inside tmpDir with a single
// node whose frontmatter contains the given implements list. The logical
// name of the created node is "ROOT/a".
//
// ROOT/a maps to code-from-spec/a/_node.md per PathFromLogicalName.
func testSetupSpecTree(t *testing.T, tmpDir string, implements []string) {
	t.Helper()

	// Build the implements YAML list.
	var implLines string
	for _, p := range implements {
		implLines += "  - " + p + "\n"
	}

	content := "---\nversion: 1\nimplements:\n" + implLines + "---\n\n# ROOT/a\n"

	nodeDir := filepath.Join(tmpDir, "code-from-spec", "a")
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "_node.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
}

// testCallHandler changes cwd to tmpDir, calls HandleWriteFile with the
// given args, then restores the original working directory. The handler
// resolves all paths against the process working directory (project root).
func testCallHandler(t *testing.T, tmpDir string, args WriteFileArgs) *mcp.CallToolResult {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// The returned Go error is reserved for catastrophic server failures;
	// all expected errors must appear as IsError: true on the result.
	result, _, err := HandleWriteFile(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("HandleWriteFile returned unexpected Go error: %v", err)
	}
	return result
}

// testResultText extracts the text from the first TextContent entry of a
// CallToolResult.
func testResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// ---------------------------------------------------------------------------
// Happy Path
// ---------------------------------------------------------------------------

// TestWritesFileSuccessfully verifies the basic write: the handler writes
// the provided content to the specified path and returns a success message.
func TestWritesFileSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"output/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "package main",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	// The success message must identify the path that was written.
	text := testResultText(t, result)
	if text != "wrote output/file.go" {
		t.Errorf("expected %q, got %q", "wrote output/file.go", text)
	}

	// Verify the file exists on disk with the exact content provided.
	data, err := os.ReadFile(filepath.Join(tmpDir, "output", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "package main" {
		t.Errorf("expected file content %q, got %q", "package main", string(data))
	}
}

// TestCreatesIntermediateDirectories verifies that the handler creates any
// missing intermediate directories rather than failing.
func TestCreatesIntermediateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"deep/nested/dir/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "deep/nested/dir/file.go",
		Content:     "package nested",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	// Verify the full directory chain and file were created.
	data, err := os.ReadFile(filepath.Join(tmpDir, "deep", "nested", "dir", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "package nested" {
		t.Errorf("expected content %q, got %q", "package nested", string(data))
	}
}

// TestOverwritesExistingFile verifies that the handler replaces the content
// of an already-existing file rather than failing or appending.
func TestOverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"output/file.go"})

	// Pre-create the file with old content.
	outDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "file.go"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	// Overwrite with new content.
	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "new content",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	// The file must contain only the new content.
	data, err := os.ReadFile(filepath.Join(outDir, "file.go"))
	if err != nil {
		t.Fatalf("failed to read file after overwrite: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("expected %q, got %q", "new content", string(data))
	}
}

// TestBackslashPathNormalized verifies that on Windows, backslash separators
// in the path argument are normalized to forward slashes before comparison
// against the implements list and before writing.
//
// This test is skipped on non-Windows platforms because on Linux/macOS,
// a backslash is a valid filename character (not a separator), so
// normalization would be incorrect there.
func TestBackslashPathNormalized(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("backslash path normalization only applies on Windows")
	}

	tmpDir := t.TempDir()
	// The implements list uses forward slashes (canonical form).
	testSetupSpecTree(t, tmpDir, []string{"output/file.go"})

	// The agent passes a Windows-style path with backslashes.
	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        `output\file.go`,
		Content:     "package main",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	// The success text must use the normalized (forward-slash) path.
	text := testResultText(t, result)
	if text != "wrote output/file.go" {
		t.Errorf("expected %q, got %q", "wrote output/file.go", text)
	}
}

// ---------------------------------------------------------------------------
// Failure Cases
// ---------------------------------------------------------------------------

// TestInvalidLogicalNamePrefix verifies that logical names that do not start
// with a recognized prefix (ROOT/ or TEST/) are rejected with a tool error.
func TestInvalidLogicalNamePrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// "EXTERNAL/something" does not start with ROOT/ or TEST/, so it must
	// be rejected before any spec file lookup.
	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "some/file.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for invalid logical name prefix, got success")
	}
}

// TestNonexistentLogicalName verifies that a valid-prefix logical name that
// has no corresponding spec file on disk is rejected with a tool error.
func TestNonexistentLogicalName(t *testing.T) {
	tmpDir := t.TempDir()
	// Deliberately do NOT create any spec file for ROOT/nonexistent.

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "some/file.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent logical name, got success")
	}
}

// TestPathNotInImplements verifies that writing to a path that is not listed
// in the node's implements list is rejected, and that the error message
// identifies the disallowed path and lists the permitted paths.
func TestPathNotInImplements(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"allowed/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for path not in implements, got success")
	}

	text := testResultText(t, result)
	// The error must be actionable: it should say what went wrong.
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("expected error to contain %q, got %q", "path not allowed", text)
	}
	// It should list the allowed paths so the agent knows what to use.
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("expected error to list allowed paths, got %q", text)
	}
}

// TestPathTraversalAttempt verifies that a path containing ".." components
// that escape the project root is rejected by ValidatePath.
func TestPathTraversalAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	// The traversal path is placed in implements so the allows check passes,
	// but ValidatePath must still catch the escape attempt.
	testSetupSpecTree(t, tmpDir, []string{"../../etc/passwd"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for path traversal attempt, got success")
	}
}

// TestEmptyPath verifies that an empty path string is rejected before any
// spec lookup or file I/O, with an error message that names the problem.
func TestEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"some/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for empty path, got success")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path is empty") {
		t.Errorf("expected error to contain %q, got %q", "path is empty", text)
	}
}

// TestSymlinkEscapingProjectRoot verifies that a path which resolves, via a
// symlink, to a location outside the project root is rejected by ValidatePath.
//
// On Windows, symlink creation may require elevated privileges; the test is
// skipped if os.Symlink fails.
func TestSymlinkEscapingProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory outside the project root to be the symlink target.
	outsideDir := t.TempDir()

	// Create a symlink inside the project root pointing to the outside dir.
	symlinkPath := filepath.Join(tmpDir, "escape")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("cannot create symlink (may require elevated privileges): %v", err)
	}

	// Place the symlink-relative path in implements so the allows check passes;
	// ValidatePath must still detect that the resolved path escapes the root.
	testSetupSpecTree(t, tmpDir, []string{"escape/evil.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "escape/evil.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for symlink escaping project root, got success")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("expected error to contain %q, got %q", "resolves outside project root", text)
	}
}
