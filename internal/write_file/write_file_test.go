// spec: TEST/tech_design/internal/tools/write_file@v3
package write_file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// helperCreateSpecFile creates a spec node file at the path that
// PathFromLogicalName would resolve to, with the given implements list.
// For logical name "ROOT/a", the file path is
// "code-from-spec/spec/a/_node.md" relative to root.
func helperCreateSpecFile(t *testing.T, root string, implements []string) {
	t.Helper()

	// Build the frontmatter YAML for the implements list.
	specDir := filepath.Join(root, "code-from-spec", "spec", "a")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}

	var implLines string
	if len(implements) > 0 {
		implLines = "implements:\n"
		for _, p := range implements {
			implLines += fmt.Sprintf("  - %s\n", p)
		}
	}

	content := fmt.Sprintf("---\nversion: 1\n%s---\n\n# ROOT/a\n", implLines)
	filePath := filepath.Join(specDir, "_node.md")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
}

// helperCallHandler calls HandleWriteFile with the given args and returns
// the result. It changes the working directory to root for the duration
// of the call.
func helperCallHandler(t *testing.T, root string, args WriteFileArgs) *mcp.CallToolResult {
	t.Helper()

	// Save and restore the working directory.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	result, _, err := HandleWriteFile(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("HandleWriteFile returned unexpected Go error: %v", err)
	}
	return result
}

// helperAssertSuccess checks that the result is not an error and that
// the text content matches the expected string.
func helperAssertSuccess(t *testing.T, result *mcp.CallToolResult, expectedText string) {
	t.Helper()
	if result.IsError {
		text := helperResultText(t, result)
		t.Fatalf("expected success but got tool error: %s", text)
	}
	text := helperResultText(t, result)
	if text != expectedText {
		t.Errorf("expected text %q, got %q", expectedText, text)
	}
}

// helperAssertToolError checks that the result is a tool error and
// that the text content contains the expected substring.
func helperAssertToolError(t *testing.T, result *mcp.CallToolResult, substr string) {
	t.Helper()
	if !result.IsError {
		text := helperResultText(t, result)
		t.Fatalf("expected tool error but got success: %s", text)
	}
	text := helperResultText(t, result)
	if !strings.Contains(text, substr) {
		t.Errorf("expected error containing %q, got %q", substr, text)
	}
}

// helperResultText extracts the text from the first TextContent entry
// in the result.
func helperResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// --- Happy Path ---

// TestWritesFileSuccessfully verifies that a valid write_file call
// creates the file on disk with the correct content.
func TestWritesFileSuccessfully(t *testing.T) {
	root := t.TempDir()
	helperCreateSpecFile(t, root, []string{"output/file.go"})

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "package main",
	})

	helperAssertSuccess(t, result, "wrote output/file.go")

	// Verify the file exists on disk with the correct content.
	data, err := os.ReadFile(filepath.Join(root, "output", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "package main" {
		t.Errorf("expected file content %q, got %q", "package main", string(data))
	}
}

// TestCreatesIntermediateDirectories verifies that missing parent
// directories are created automatically.
func TestCreatesIntermediateDirectories(t *testing.T) {
	root := t.TempDir()
	helperCreateSpecFile(t, root, []string{"deep/nested/dir/file.go"})

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "deep/nested/dir/file.go",
		Content:     "package deep",
	})

	helperAssertSuccess(t, result, "wrote deep/nested/dir/file.go")

	// Verify the file and directories exist.
	data, err := os.ReadFile(filepath.Join(root, "deep", "nested", "dir", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "package deep" {
		t.Errorf("expected file content %q, got %q", "package deep", string(data))
	}
}

// TestOverwritesExistingFile verifies that an existing file is
// replaced with new content.
func TestOverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	helperCreateSpecFile(t, root, []string{"output/file.go"})

	// Create the initial file.
	outDir := filepath.Join(root, "output")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "file.go"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "new content",
	})

	helperAssertSuccess(t, result, "wrote output/file.go")

	// Verify the file was overwritten.
	data, err := os.ReadFile(filepath.Join(outDir, "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("expected file content %q, got %q", "new content", string(data))
	}
}

// --- Failure Cases ---

// TestInvalidLogicalNamePrefix verifies that a logical name not
// starting with ROOT/ or TEST/ is rejected.
func TestInvalidLogicalNamePrefix(t *testing.T) {
	root := t.TempDir()

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "output/file.go",
		Content:     "package main",
	})

	helperAssertToolError(t, result, "EXTERNAL/something")
}

// TestNonexistentLogicalName verifies that a logical name with no
// corresponding spec file is rejected.
func TestNonexistentLogicalName(t *testing.T) {
	root := t.TempDir()

	// Do not create any spec file — the logical name resolves to
	// a path that does not exist.
	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "output/file.go",
		Content:     "package main",
	})

	helperAssertToolError(t, result, "")
}

// TestPathNotInImplements verifies that a path not listed in the
// node's implements is rejected with an actionable error.
func TestPathNotInImplements(t *testing.T) {
	root := t.TempDir()
	helperCreateSpecFile(t, root, []string{"allowed/file.go"})

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Content:     "package main",
	})

	helperAssertToolError(t, result, "path not allowed")
	// The error should also list the allowed paths.
	text := helperResultText(t, result)
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("expected error to list allowed paths, got %q", text)
	}
}

// TestPathTraversalAttempt verifies that directory traversal in
// the implements list is caught by ValidatePath.
func TestPathTraversalAttempt(t *testing.T) {
	root := t.TempDir()
	helperCreateSpecFile(t, root, []string{"../../etc/passwd"})

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Content:     "malicious",
	})

	helperAssertToolError(t, result, "")
}

// TestEmptyPath verifies that an empty path is rejected.
func TestEmptyPath(t *testing.T) {
	root := t.TempDir()
	helperCreateSpecFile(t, root, []string{"some/file.go"})

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Content:     "package main",
	})

	helperAssertToolError(t, result, "path is empty")
}

// TestSymlinkEscapingProjectRoot verifies that a symlink pointing
// outside the project root is detected and rejected.
func TestSymlinkEscapingProjectRoot(t *testing.T) {
	// Symlink creation may require elevated privileges on Windows.
	if runtime.GOOS == "windows" {
		t.Skip("symlink test skipped on Windows (may require elevated privileges)")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Create a symlink inside root that points outside it.
	symlinkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outside, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// The implements list includes a path through the symlink.
	helperCreateSpecFile(t, root, []string{"escape/file.go"})

	result := helperCallHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "escape/file.go",
		Content:     "malicious",
	})

	helperAssertToolError(t, result, "resolves outside project root")
}
