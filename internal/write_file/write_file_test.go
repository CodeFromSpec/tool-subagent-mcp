// spec: TEST/tech_design/internal/tools/write_file@v1

// Package write_file provides tests for the write_file tool handler.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Context"
// Each test uses t.TempDir() as the project root/working directory.
// A spec tree is created with the necessary frontmatter containing an
// Implements list. The handler is called with WriteFileArgs.
package write_file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// buildSpecTree creates a minimal spec tree for the given logical name rooted
// at projectRoot. The frontmatter will include the provided implements list.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Context"
func buildSpecTree(t *testing.T, projectRoot, logicalName string, implements []string) {
	t.Helper()

	// Determine the spec file path from the logical name.
	// ROOT/<path> → code-from-spec/spec/<path>/_node.md
	// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName"
	var specRelPath string
	if logicalName == "ROOT" {
		specRelPath = "code-from-spec/spec/_node.md"
	} else if strings.HasPrefix(logicalName, "ROOT/") {
		rest := strings.TrimPrefix(logicalName, "ROOT/")
		specRelPath = fmt.Sprintf("code-from-spec/spec/%s/_node.md", rest)
	} else {
		t.Fatalf("buildSpecTree: unsupported logical name %q", logicalName)
	}

	specPath := filepath.Join(projectRoot, specRelPath)
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("failed to create spec directory: %v", err)
	}

	// Build the implements YAML list.
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("version: 1\n")
	if len(implements) > 0 {
		sb.WriteString("implements:\n")
		for _, impl := range implements {
			sb.WriteString(fmt.Sprintf("  - %s\n", impl))
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n", logicalName))

	if err := os.WriteFile(specPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
}

// callHandler is a helper that sets the working directory to projectRoot,
// calls handleWriteFile, then restores the original working directory.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Context"
func callHandler(t *testing.T, projectRoot string, args WriteFileArgs) *mcp.CallToolResult {
	t.Helper()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restore chdir: %v", err)
		}
	}()

	result, _, err := handleWriteFile(context.Background(), nil, args)
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	return result
}

// textOf extracts the text from the first content entry of a result.
func textOf(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// --- Happy Path ---

// TestWritesFileSuccessfully verifies the basic happy path.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Writes file successfully"
func TestWritesFileSuccessfully(t *testing.T) {
	root := t.TempDir()
	buildSpecTree(t, root, "ROOT/a", []string{"output/file.go"})

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "package main",
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", textOf(result))
	}
	if got := textOf(result); got != "wrote output/file.go" {
		t.Errorf("expected 'wrote output/file.go', got %q", got)
	}

	written, err := os.ReadFile(filepath.Join(root, "output/file.go"))
	if err != nil {
		t.Fatalf("file not found on disk: %v", err)
	}
	if string(written) != "package main" {
		t.Errorf("file content mismatch: got %q", string(written))
	}
}

// TestCreatesIntermediateDirectories verifies that missing parent dirs are created.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Creates intermediate directories"
func TestCreatesIntermediateDirectories(t *testing.T) {
	root := t.TempDir()
	buildSpecTree(t, root, "ROOT/a", []string{"deep/nested/dir/file.go"})

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "deep/nested/dir/file.go",
		Content:     "package deep",
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", textOf(result))
	}

	if _, err := os.Stat(filepath.Join(root, "deep/nested/dir/file.go")); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

// TestOverwritesExistingFile verifies that an existing file is overwritten.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Overwrites existing file"
func TestOverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	buildSpecTree(t, root, "ROOT/a", []string{"output/file.go"})

	// Pre-create the file with initial content.
	outDir := filepath.Join(root, "output")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "file.go"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("pre-write: %v", err)
	}

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Content:     "new content",
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", textOf(result))
	}

	data, err := os.ReadFile(filepath.Join(root, "output/file.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

// --- Failure Cases ---

// TestInvalidLogicalNamePrefix verifies rejection of non-ROOT/TEST prefixes.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Invalid logical name prefix"
func TestInvalidLogicalNamePrefix(t *testing.T) {
	root := t.TempDir()

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "output/file.go",
		Content:     "package main",
	})

	if !result.IsError {
		t.Fatal("expected tool error for invalid prefix, got success")
	}
}

// TestNonexistentLogicalName verifies that a missing spec file returns a tool error.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Nonexistent logical name"
func TestNonexistentLogicalName(t *testing.T) {
	root := t.TempDir()
	// Do NOT create the spec tree for ROOT/nonexistent.

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "output/file.go",
		Content:     "package main",
	})

	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec, got success")
	}
}

// TestPathNotInImplements verifies that paths outside implements are rejected.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Path not in implements"
func TestPathNotInImplements(t *testing.T) {
	root := t.TempDir()
	buildSpecTree(t, root, "ROOT/a", []string{"allowed/file.go"})

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Content:     "package main",
	})

	if !result.IsError {
		t.Fatal("expected tool error for path not in implements, got success")
	}
	msg := textOf(result)
	if !strings.Contains(msg, "path not allowed") {
		t.Errorf("expected 'path not allowed' in error message, got: %s", msg)
	}
	if !strings.Contains(msg, "allowed/file.go") {
		t.Errorf("expected allowed paths listed in error message, got: %s", msg)
	}
}

// TestPathTraversalAttempt verifies that traversal sequences in implements are rejected.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Path traversal attempt"
func TestPathTraversalAttempt(t *testing.T) {
	root := t.TempDir()
	// The traversal path is in implements — ValidatePath must catch it.
	buildSpecTree(t, root, "ROOT/a", []string{"../../etc/passwd"})

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Content:     "should not write",
	})

	if !result.IsError {
		t.Fatal("expected tool error for path traversal, got success")
	}
}

// TestEmptyPath verifies that an empty path returns a tool error.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Empty path"
func TestEmptyPath(t *testing.T) {
	root := t.TempDir()
	buildSpecTree(t, root, "ROOT/a", []string{"some/file.go"})

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Content:     "package main",
	})

	if !result.IsError {
		t.Fatal("expected tool error for empty path, got success")
	}
	msg := textOf(result)
	if !strings.Contains(msg, "path is empty") {
		t.Errorf("expected 'path is empty' in error message, got: %s", msg)
	}
}

// TestSymlinkEscapingProjectRoot verifies that symlinks pointing outside the
// project root are rejected.
// Spec ref: TEST/tech_design/internal/tools/write_file § "Symlink escaping project root"
func TestSymlinkEscapingProjectRoot(t *testing.T) {
	root := t.TempDir()
	// Create a target outside the project root.
	outside := t.TempDir()

	// Create a symlink inside root that points outside.
	symlinkName := "escape_link"
	symlinkPath := filepath.Join(root, symlinkName)
	if err := os.Symlink(outside, symlinkPath); err != nil {
		t.Skipf("cannot create symlink (may need elevated privileges): %v", err)
	}

	// The implements entry uses the symlink path.
	symlinkFile := symlinkName + "/file.go"
	buildSpecTree(t, root, "ROOT/a", []string{symlinkFile})

	result := callHandler(t, root, WriteFileArgs{
		LogicalName: "ROOT/a",
		Path:        symlinkFile,
		Content:     "package main",
	})

	if !result.IsError {
		t.Fatal("expected tool error for symlink escape, got success")
	}
	msg := textOf(result)
	if !strings.Contains(msg, "resolves outside project root") {
		t.Errorf("expected 'resolves outside project root' in error message, got: %s", msg)
	}
}
