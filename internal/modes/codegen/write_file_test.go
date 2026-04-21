// spec: TEST/tech_design/internal/modes/codegen/tools/write_file@v4

// Package codegen — tests for the write_file tool handler.
//
// Spec: ROOT/tech_design/internal/modes/codegen/tools/write_file
//
// Each test uses t.TempDir() as the project root and working directory. A
// Target is created with a known Frontmatter.Implements list. The handler
// closure returned by handleWriteFile is invoked with WriteFileArgs.
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

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
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

// makeWriteFileTarget builds a minimal *Target whose Frontmatter contains only
// the given implements paths. LogicalName and FilePath are placeholder values
// because the write_file handler does not inspect them.
func makeWriteFileTarget(implements []string) *Target {
	return &Target{
		LogicalName: "TEST/write_file_test",
		FilePath:    "placeholder",
		Frontmatter: &frontmatter.Frontmatter{
			Implements: implements,
		},
	}
}

// invokeWriteFile is a thin wrapper that obtains the handler closure from
// handleWriteFile and invokes it, returning the *mcp.CallToolResult directly.
// It fails the test immediately if the returned Go error is non-nil (that path
// is reserved for catastrophic server failures, which are never expected here).
func invokeWriteFile(t *testing.T, target *Target, args WriteFileArgs) *mcp.CallToolResult {
	t.Helper()
	handler := handleWriteFile(target)
	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleWriteFile returned unexpected Go error: %v", err)
	}
	if result == nil {
		t.Fatal("handleWriteFile returned nil result")
	}
	return result
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
// a file path that appears in Implements is written to disk and the result text
// equals "wrote <path>".
//
// Spec §Happy Path / "Writes file successfully":
//   - Create a Target with Implements: ["output/file.go"].
//   - Call handler with Path:"output/file.go", Content:"package main".
//   - Expect: success result with text "wrote output/file.go".
//   - Verify the file exists on disk with the correct content.
func TestWriteFile_WritesFileSuccessfully(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	target := makeWriteFileTarget([]string{"output/file.go"})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    "output/file.go",
		Content: "package main",
	})

	// Expect success (not an MCP tool error).
	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", resultText(t, result))
	}

	// Result text must be "wrote output/file.go".
	got := resultText(t, result)
	if got != "wrote output/file.go" {
		t.Errorf("result text: got %q, want %q", got, "wrote output/file.go")
	}

	// The file must exist on disk with the exact content that was passed in.
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
//
// Spec §Happy Path / "Creates intermediate directories":
//   - Create a Target with Implements: ["deep/nested/dir/file.go"].
//   - Call handler with Path:"deep/nested/dir/file.go".
//   - Expect: success. Directories created automatically.
func TestWriteFile_CreatesIntermediateDirectories(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	target := makeWriteFileTarget([]string{"deep/nested/dir/file.go"})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    "deep/nested/dir/file.go",
		Content: "package deep",
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", resultText(t, result))
	}

	// Confirm the file exists — which implies all intermediate directories were
	// created by the handler (they did not exist before the call).
	if _, err := os.Stat(filepath.Join(root, "deep/nested/dir/file.go")); err != nil {
		t.Errorf("expected file to exist after write: %v", err)
	}
}

// TestWriteFile_OverwritesExistingFile verifies that calling the handler on a
// path that already holds content replaces that content completely.
//
// Spec §Happy Path / "Overwrites existing file":
//   - Create a Target with Implements: ["output/file.go"].
//   - Write an initial file at that path.
//   - Call handler with new content.
//   - Expect: success. File content replaced.
func TestWriteFile_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	// Pre-create the directory and file so there is existing content to overwrite.
	if err := os.MkdirAll(filepath.Join(root, "output"), 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "output/file.go"), []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	target := makeWriteFileTarget([]string{"output/file.go"})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    "output/file.go",
		Content: "new content",
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", resultText(t, result))
	}

	// The file must now contain only the new content — old content fully replaced.
	written, err := os.ReadFile(filepath.Join(root, "output/file.go"))
	if err != nil {
		t.Fatalf("file not found after overwrite: %v", err)
	}
	if string(written) != "new content" {
		t.Errorf("file content after overwrite: got %q, want %q", string(written), "new content")
	}
}

// ── Failure Cases ─────────────────────────────────────────────────────────────

// TestWriteFile_PathNotInImplements verifies that a path absent from the
// target's Implements list is rejected with a descriptive tool error.
//
// Spec §Failure Cases / "Path not in implements":
//   - Create a Target with Implements: ["allowed/file.go"].
//   - Call handler with Path:"other/file.go".
//   - Expect: tool error containing "path not allowed" and listing allowed paths.
func TestWriteFile_PathNotInImplements(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	target := makeWriteFileTarget([]string{"allowed/file.go"})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    "other/file.go",
		Content: "package other",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success with text: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// The error message must identify the violation.
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("error text must contain %q, got: %s", "path not allowed", text)
	}
	// The error message must list the allowed paths so the agent can act on it.
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("error text must list allowed path %q, got: %s", "allowed/file.go", text)
	}
}

// TestWriteFile_PathTraversalAttempt verifies that a path containing directory
// traversal sequences is caught by ValidatePath and returned as a tool error.
//
// Spec §Failure Cases / "Path traversal attempt":
//   - Create a Target with Implements: ["../../etc/passwd"].
//   - Call handler with Path:"../../etc/passwd".
//   - Expect: tool error from ValidatePath.
//
// Note: we place the traversal path in Implements so the implements check does
// not fire before ValidatePath — per the handler algorithm, ValidatePath runs
// first (step 1), before the implements check (step 2). Putting it in Implements
// isolates which guard triggers the error.
func TestWriteFile_PathTraversalAttempt(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	// Both the target implements list and the call use the traversal path.
	// ValidatePath must fire at step 1 and reject the write.
	target := makeWriteFileTarget([]string{"../../etc/passwd"})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    "../../etc/passwd",
		Content: "malicious",
	})

	// The handler must return an MCP tool error — no panic, no write.
	if !result.IsError {
		t.Fatalf("expected tool error for traversal path, got success: %s", resultText(t, result))
	}
	// No file must have been written outside the temp directory.
	// We do not assert the exact message wording — only IsError:true is
	// mandated by the spec for this case (the error originates from ValidatePath).
}

// TestWriteFile_EmptyPath verifies that an empty path string is rejected
// immediately with a "path is empty" tool error (before any other check).
//
// Spec §Failure Cases / "Empty path":
//   - Call handler with Path:"".
//   - Expect: tool error containing "path is empty".
func TestWriteFile_EmptyPath(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	target := makeWriteFileTarget([]string{"some/file.go"})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    "",
		Content: "whatever",
	})

	if !result.IsError {
		t.Fatalf("expected tool error for empty path, got success: %s", resultText(t, result))
	}

	text := resultText(t, result)
	// ValidatePath returns "path is empty" for an empty string (spec §pathvalidation).
	// The write_file handler wraps this message in the tool error text.
	if !strings.Contains(text, "path is empty") {
		t.Errorf("error text must contain %q, got: %s", "path is empty", text)
	}
}

// TestWriteFile_SymlinkEscapingProjectRoot verifies that a symlink inside the
// temp dir that resolves outside the project root is detected and rejected.
//
// Spec §Failure Cases / "Symlink escaping project root":
//   - Create a symlink inside the temp dir pointing outside it.
//   - Add the symlink path to Implements.
//   - Call handler with that path.
//   - Expect: tool error containing "resolves outside project root".
//
// The test is skipped when symlink creation requires elevated privileges (common
// on Windows without Developer Mode or SeCreateSymbolicLinkPrivilege).
func TestWriteFile_SymlinkEscapingProjectRoot(t *testing.T) {
	root := t.TempDir()
	chdirTemp(t, root)

	// Point the symlink at the OS temp dir, which is always outside root
	// (root is itself a child of os.TempDir()).
	symlinkName := "escape_link"
	if err := os.Symlink(os.TempDir(), filepath.Join(root, symlinkName)); err != nil {
		t.Skipf("cannot create symlink (may need elevated privileges): %v", err)
	}

	// The symlink name is added to Implements so the handler reaches ValidatePath
	// (step 1) and does not stop early at the implements check (step 2).
	target := makeWriteFileTarget([]string{symlinkName})
	result := invokeWriteFile(t, target, WriteFileArgs{
		Path:    symlinkName,
		Content: "malicious",
	})

	if !result.IsError {
		t.Fatalf("expected tool error for symlink escaping project root, got success: %s", resultText(t, result))
	}

	text := resultText(t, result)
	// ValidatePath returns "path resolves outside project root: <path>" for this
	// case. The write_file handler includes this in the tool error text.
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("error text must contain %q, got: %s", "resolves outside project root", text)
	}
}
