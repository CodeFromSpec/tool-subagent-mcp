// spec: TEST/tech_design/internal/modes/codegen/tools/write_file@v6

// Package codegen — tests for the write_file tool handler.
//
// Spec: ROOT/tech_design/internal/modes/codegen/tools/write_file
//
// The handler is stateless: write_file receives logical_name as a parameter and
// resolves frontmatter independently. There is no currentTarget, no Target type,
// no makeWriteFileTarget helper.
//
// Each test creates a spec tree under t.TempDir() with frontmatter containing
// implements. The handler is called with WriteFileArgs including LogicalName.
//
// Because handleWriteFile calls os.Getwd() to derive the project root, each
// test changes the process working directory to its temp dir and restores it
// on cleanup. Tests are intentionally sequential for this reason (parallel
// Chdir is unsafe).
//
// Constraints inherited from ancestor nodes:
//   - No test framework beyond the standard "testing" package.
//   - All expected error conditions return IsError:true — no panics or Go errors.
package codegen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// chdirTemp changes the process working directory to dir and registers a
// cleanup function that restores the original directory after the test.
// The test is fatally aborted if either Getwd or Chdir fails.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			// Log but do not fail — the test itself may have already failed.
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	})
}

// resultText extracts the text from the first content entry of result.
// It fails the test if the content is missing or is not *mcp.TextContent.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content entries")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// ── Happy Path ────────────────────────────────────────────────────────────────

// TestWriteFile_WritesFileSuccessfully verifies the baseline success case:
// a spec tree with ROOT/a having implements: ["output/file.go"], the handler is
// called with LogicalName: "ROOT/a", Path: "output/file.go", Content: "package main".
// Expect success "wrote output/file.go", verify file on disk.
func TestWriteFile_WritesFileSuccessfully(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	buildMinimalSpecTree(t, root, "output/file.go")

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "package main",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", resultText(t, result))
	}

	got := resultText(t, result)
	if got != "wrote output/file.go" {
		t.Errorf("result text: got %q, want %q", got, "wrote output/file.go")
	}

	written, err := os.ReadFile(filepath.Join(root, "output/file.go"))
	if err != nil {
		t.Fatalf("file not found after write: %v", err)
	}
	if string(written) != "package main" {
		t.Errorf("file content: got %q, want %q", string(written), "package main")
	}
}

// TestWriteFile_CreatesIntermediateDirectories verifies that the handler
// creates any missing parent directories before writing the target file.
func TestWriteFile_CreatesIntermediateDirectories(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	buildMinimalSpecTree(t, root, "deep/nested/dir/file.go")

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "deep/nested/dir/file.go",
		Content:     "package deep",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", resultText(t, result))
	}

	if _, err := os.Stat(filepath.Join(root, "deep/nested/dir/file.go")); err != nil {
		t.Errorf("expected file to exist after write: %v", err)
	}
}

// TestWriteFile_OverwritesExistingFile verifies that calling the handler on a
// path that already holds content replaces that content completely.
func TestWriteFile_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	buildMinimalSpecTree(t, root, "output/file.go")

	// Pre-create the directory and file so there is existing content to overwrite.
	if err := os.MkdirAll(filepath.Join(root, "output"), 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "output/file.go"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "new content",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", resultText(t, result))
	}

	written, err := os.ReadFile(filepath.Join(root, "output/file.go"))
	if err != nil {
		t.Fatalf("file not found after overwrite: %v", err)
	}
	if string(written) != "new content" {
		t.Errorf("file content after overwrite: got %q, want %q", string(written), "new content")
	}
}

// ── Failure Cases ─────────────────────────────────────────────────────────────

// TestWriteFile_InvalidLogicalNamePrefix verifies that a logical name with an
// invalid prefix (e.g. "EXTERNAL/something") is rejected with a tool error.
func TestWriteFile_InvalidLogicalNamePrefix(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "any/file.go",
		Content:     "whatever",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error for invalid prefix, got success: %s", resultText(t, result))
	}
}

// TestWriteFile_NonexistentLogicalName verifies that a logical name that does
// not correspond to any spec file on disk is rejected with a tool error.
func TestWriteFile_NonexistentLogicalName(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	// No spec tree created — ROOT/nonexistent has no backing file.
	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "any/file.go",
		Content:     "whatever",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error for nonexistent logical name, got success: %s", resultText(t, result))
	}
}

// TestWriteFile_PathNotInImplements verifies that a path absent from the
// node's implements list is rejected with a descriptive tool error.
func TestWriteFile_PathNotInImplements(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	buildMinimalSpecTree(t, root, "allowed/file.go")

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Content:     "package other",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got success with text: %s", resultText(t, result))
	}

	text := resultText(t, result)
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("error text must contain %q, got: %s", "path not allowed", text)
	}
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("error text must list allowed path %q, got: %s", "allowed/file.go", text)
	}
}

// TestWriteFile_PathTraversalAttempt verifies that a path containing directory
// traversal sequences is caught by ValidatePath and returned as a tool error.
func TestWriteFile_PathTraversalAttempt(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	buildMinimalSpecTree(t, root, "../../etc/passwd")

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Content:     "malicious",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error for traversal path, got success: %s", resultText(t, result))
	}
}

// TestWriteFile_EmptyPath verifies that an empty path string is rejected
// immediately with a "path is empty" tool error.
func TestWriteFile_EmptyPath(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	buildMinimalSpecTree(t, root, "some/file.go")

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Content:     "whatever",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error for empty path, got success: %s", resultText(t, result))
	}

	text := resultText(t, result)
	if !strings.Contains(text, "path is empty") {
		t.Errorf("error text must contain %q, got: %s", "path is empty", text)
	}
}

// TestWriteFile_SymlinkEscapingProjectRoot verifies that a symlink inside the
// temp dir that resolves outside the project root is detected and rejected.
//
// The test is skipped when symlink creation requires elevated privileges (common
// on Windows without Developer Mode or SeCreateSymbolicLinkPrivilege).
func TestWriteFile_SymlinkEscapingProjectRoot(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	symlinkName := "escape_link"
	if err := os.Symlink(os.TempDir(), filepath.Join(root, symlinkName)); err != nil {
		t.Skipf("cannot create symlink (may need elevated privileges): %v", err)
	}

	buildMinimalSpecTree(t, root, symlinkName)

	result, _, err := handleWriteFile(context.Background(), &mcp.CallToolRequest{}, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        symlinkName,
		Content:     "malicious",
	})
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error for symlink escaping project root, got success: %s", resultText(t, result))
	}

	text := resultText(t, result)
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("error text must contain %q, got: %s", "resolves outside project root", text)
	}
}
