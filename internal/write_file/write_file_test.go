// code-from-spec: TEST/tech_design/internal/tools/write_file@v8
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
func testSetupSpecTree(t *testing.T, tmpDir string, implements []string) {
	t.Helper()

	// Build the implements YAML list.
	var implLines string
	for _, p := range implements {
		implLines += "  - " + p + "\n"
	}

	content := "---\nversion: 1\nimplements:\n" + implLines + "---\n\n# ROOT/a\n"

	// ROOT/a maps to code-from-spec/a/_node.md per PathFromLogicalName.
	nodeDir := filepath.Join(tmpDir, "code-from-spec", "a")
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "_node.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
}

// testCallHandler changes to tmpDir, calls HandleWriteFile, and restores
// the original working directory. The handler resolves paths against the
// process working directory (project root).
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

	result, _, err := HandleWriteFile(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("HandleWriteFile returned unexpected Go error: %v", err)
	}
	return result
}

// testResultText extracts the text content from a CallToolResult.
func testResultText(t *testing.T, result *mcp.CallToolResult) string {
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

func TestWritesFileSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"output/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "package main",
	})

	// Expect success.
	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if text != "wrote output/file.go" {
		t.Errorf("expected %q, got %q", "wrote output/file.go", text)
	}

	// Verify file exists on disk with correct content.
	data, err := os.ReadFile(filepath.Join(tmpDir, "output", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "package main" {
		t.Errorf("expected content %q, got %q", "package main", string(data))
	}
}

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

	// Verify directories were created and file exists.
	data, err := os.ReadFile(filepath.Join(tmpDir, "deep", "nested", "dir", "file.go"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != "package nested" {
		t.Errorf("expected content %q, got %q", "package nested", string(data))
	}
}

func TestOverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"output/file.go"})

	// Write an initial file at the target path.
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

	// Verify the file was overwritten.
	data, err := os.ReadFile(filepath.Join(outDir, "file.go"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("expected %q, got %q", "new content", string(data))
	}
}

func TestBackslashPathNormalized(t *testing.T) {
	// Skip on non-Windows — backslash is a valid filename character
	// on Linux/macOS, not a path separator.
	if runtime.GOOS != "windows" {
		t.Skip("backslash normalization only applies on Windows")
	}

	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"output/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output\\file.go",
		Content:     "package main",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if text != "wrote output/file.go" {
		t.Errorf("expected %q, got %q", "wrote output/file.go", text)
	}
}

// --- Failure Cases ---

func TestInvalidLogicalNamePrefix(t *testing.T) {
	tmpDir := t.TempDir()

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "some/file.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for invalid logical name prefix")
	}
}

func TestNonexistentLogicalName(t *testing.T) {
	tmpDir := t.TempDir()
	// Do not create any spec file for ROOT/nonexistent.

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "some/file.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent logical name")
	}
}

func TestPathNotInImplements(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"allowed/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for path not in implements")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("expected error to contain %q, got %q", "path not allowed", text)
	}
	// Should list the allowed paths.
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("expected error to list allowed paths, got %q", text)
	}
}

func TestPathTraversalAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"../../etc/passwd"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for path traversal attempt")
	}
}

func TestEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	testSetupSpecTree(t, tmpDir, []string{"some/file.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for empty path")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path is empty") {
		t.Errorf("expected error to contain %q, got %q", "path is empty", text)
	}
}

func TestSymlinkEscapingProjectRoot(t *testing.T) {
	// Symlinks may require elevated privileges on Windows.
	// Skip if we cannot create one.
	tmpDir := t.TempDir()

	// Create a directory outside the temp dir to be the symlink target.
	outsideDir := t.TempDir()

	// Create a symlink inside tmpDir pointing outside.
	symlinkPath := filepath.Join(tmpDir, "escape")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("cannot create symlink (may need elevated privileges): %v", err)
	}

	// Set up spec tree with the symlink-based path in implements.
	testSetupSpecTree(t, tmpDir, []string{"escape/evil.go"})

	result := testCallHandler(t, tmpDir, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "escape/evil.go",
		Content:     "content",
	})

	if !result.IsError {
		t.Fatal("expected tool error for symlink escaping project root")
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("expected error to contain %q, got %q", "resolves outside project root", text)
	}
}
