// code-from-spec: TEST/tech_design/internal/tools/find_replace@v2

package find_replace

// This file contains tests for the find_replace tool handler.
// It exercises HandleFindReplace against a temporary filesystem
// to validate both happy-path behaviour and all documented
// failure modes described in the spec.
//
// Conventions (ROOT/tech_design constraint):
//   - All helper functions/types are prefixed with "test".
//   - No third-party test framework — only the standard "testing" package.

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testWriteFile creates all necessary parent directories and writes content
// to the given absolute path. It fails the test on any error.
func testWriteFile(t *testing.T, absPath, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("testWriteFile: MkdirAll: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("testWriteFile: WriteFile: %v", err)
	}
}

// testReadFile reads and returns the content of an absolute path.
// It fails the test on any error.
func testReadFile(t *testing.T, absPath string) string {
	t.Helper()
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("testReadFile: %v", err)
	}
	return string(data)
}

// testSpecContent returns a minimal YAML front-matter spec that declares
// the given relative paths in its implements list.
//
// The spec format expected by ValidatePath / the chain resolver is:
//
//	---
//	implements:
//	  - some/path.go
//	---
func testSpecContent(paths ...string) string {
	var sb strings.Builder
	sb.WriteString("---\nversion: 1\nimplements:\n")
	for _, p := range paths {
		fmt.Fprintf(&sb, "  - %s\n", p)
	}
	sb.WriteString("---\n")
	return sb.String()
}

// testSpecPath returns the absolute path of the spec file for a given
// logical name inside a temporary project root directory.
//
// Logical name format:  ROOT/a/b/c  or  TEST/a/b/c
// Spec file convention: code-from-spec/<segment after ROOT or TEST>/_node.md
//
// Example:
//
//	ROOT/tech_design/internal/tools/find_replace
//	→ <root>/code-from-spec/tech_design/internal/tools/find_replace/_node.md
func testSpecPath(root, logicalName string) string {
	// Strip the leading qualifier (ROOT or TEST) and the first slash.
	idx := strings.Index(logicalName, "/")
	if idx < 0 {
		// Malformed — return something that will simply not exist.
		return filepath.Join(root, "code-from-spec", logicalName, "_node.md")
	}
	rel := logicalName[idx+1:]
	return filepath.Join(root, "code-from-spec", filepath.FromSlash(rel), "_node.md")
}

// testMakeRequest builds a *mcp.CallToolRequest with the given tool name.
// (The handler does not inspect the request fields directly; they are passed
// through the typed args struct.)
func testMakeRequest(toolName string) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{}
}

// testCallHandler invokes HandleFindReplace after temporarily switching the
// process working directory to the given project root.
//
// Many internal helpers (e.g. ValidatePath) resolve paths relative to the
// working directory. Temporarily changing cwd lets the tests use an isolated
// temp directory instead of the real project root.
func testCallHandler(t *testing.T, root string, args FindReplaceArgs) (*mcp.CallToolResult, error) {
	t.Helper()

	// Save and restore the original working directory.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("testCallHandler: Getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("testCallHandler: Chdir(%q): %v", root, err)
	}
	t.Cleanup(func() {
		// Restore unconditionally — test failures must not pollute other tests.
		_ = os.Chdir(orig)
	})

	result, _, goErr := HandleFindReplace(context.Background(), testMakeRequest("find_replace"), args)
	return result, goErr
}

// testAssertSuccess verifies that the result is not an error result and that
// its text content contains the expected substring.
func testAssertSuccess(t *testing.T, result *mcp.CallToolResult, wantSubstr string) {
	t.Helper()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Fatalf("expected success result, got error: %s", testResultText(result))
	}
	got := testResultText(result)
	if !strings.Contains(got, wantSubstr) {
		t.Errorf("result text %q does not contain %q", got, wantSubstr)
	}
}

// testAssertToolError verifies that the result is an error result whose text
// contains all of the provided substrings.
func testAssertToolError(t *testing.T, result *mcp.CallToolResult, wantSubstrs ...string) {
	t.Helper()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsError {
		t.Fatalf("expected error result, got success: %s", testResultText(result))
	}
	got := testResultText(result)
	for _, want := range wantSubstrs {
		if !strings.Contains(got, want) {
			t.Errorf("error text %q does not contain %q", got, want)
		}
	}
}

// testResultText extracts the text from the first TextContent entry in a
// CallToolResult, or returns an empty string if no such entry exists.
func testResultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Happy-path tests
// ---------------------------------------------------------------------------

// TestFindReplace_ReplacesStringSuccessfully verifies the basic single-line
// replace scenario described in the spec.
func TestFindReplace_ReplacesStringSuccessfully(t *testing.T) {
	root := t.TempDir()

	// Build a spec file whose implements list contains our target file path.
	targetRelPath := "internal/find_replace/test.go"
	logicalName := "ROOT/find_replace/test_node"
	specFile := testSpecPath(root, logicalName)
	testWriteFile(t, specFile, testSpecContent(targetRelPath))

	// Create the target file.
	targetAbsPath := filepath.Join(root, filepath.FromSlash(targetRelPath))
	testWriteFile(t, targetAbsPath, "hello world")

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        targetRelPath,
		OldString:   "hello",
		NewString:   "goodbye",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertSuccess(t, result, "edited "+targetRelPath)

	// Verify the file was actually modified.
	got := testReadFile(t, targetAbsPath)
	if got != "goodbye world" {
		t.Errorf("file content = %q; want %q", got, "goodbye world")
	}
}

// TestFindReplace_ReplacesMultiLineOldString verifies that the handler can
// match and replace a block of text that spans multiple lines.
func TestFindReplace_ReplacesMultiLineOldString(t *testing.T) {
	root := t.TempDir()

	targetRelPath := "internal/find_replace/test.go"
	logicalName := "ROOT/find_replace/multiline_node"
	specFile := testSpecPath(root, logicalName)
	testWriteFile(t, specFile, testSpecContent(targetRelPath))

	original := "line one\nline two\nline three\n"
	targetAbsPath := filepath.Join(root, filepath.FromSlash(targetRelPath))
	testWriteFile(t, targetAbsPath, original)

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        targetRelPath,
		OldString:   "line one\nline two",
		NewString:   "replaced",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertSuccess(t, result, "edited "+targetRelPath)

	got := testReadFile(t, targetAbsPath)
	want := "replaced\nline three\n"
	if got != want {
		t.Errorf("file content = %q; want %q", got, want)
	}
}

// TestFindReplace_ReplacesWithEmptyNewString verifies that an empty NewString
// effectively deletes the matched text.
func TestFindReplace_ReplacesWithEmptyNewString(t *testing.T) {
	root := t.TempDir()

	targetRelPath := "internal/find_replace/test.go"
	logicalName := "ROOT/find_replace/delete_node"
	specFile := testSpecPath(root, logicalName)
	testWriteFile(t, specFile, testSpecContent(targetRelPath))

	targetAbsPath := filepath.Join(root, filepath.FromSlash(targetRelPath))
	testWriteFile(t, targetAbsPath, "remove this part and keep the rest")

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        targetRelPath,
		OldString:   "remove this part and ",
		NewString:   "",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertSuccess(t, result, "edited "+targetRelPath)

	got := testReadFile(t, targetAbsPath)
	want := "keep the rest"
	if got != want {
		t.Errorf("file content = %q; want %q", got, want)
	}
}

// TestFindReplace_BackslashPathNormalized verifies that Windows-style backslash
// separators in the Path argument are treated the same as forward slashes.
// This test is skipped on non-Windows platforms.
func TestFindReplace_BackslashPathNormalized(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("backslash normalization test is Windows-only")
	}

	root := t.TempDir()

	targetRelPath := "internal/find_replace/test.go"
	logicalName := "ROOT/find_replace/backslash_node"
	specFile := testSpecPath(root, logicalName)
	testWriteFile(t, specFile, testSpecContent(targetRelPath))

	targetAbsPath := filepath.Join(root, filepath.FromSlash(targetRelPath))
	testWriteFile(t, targetAbsPath, "original content")

	// Use backslash path as input — the handler must normalize it.
	backslashPath := `internal\find_replace\test.go`

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        backslashPath,
		OldString:   "original",
		NewString:   "replaced",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	// The success message should contain the normalized (forward-slash) path.
	testAssertSuccess(t, result, "edited "+targetRelPath)

	got := testReadFile(t, targetAbsPath)
	if !strings.Contains(got, "replaced") {
		t.Errorf("file content %q should contain %q after replace", got, "replaced")
	}
}

// ---------------------------------------------------------------------------
// Failure-case tests
// ---------------------------------------------------------------------------

// TestFindReplace_InvalidLogicalNamePrefix verifies that a LogicalName that
// does not begin with ROOT/ or TEST/ yields a tool error mentioning the name.
func TestFindReplace_InvalidLogicalNamePrefix(t *testing.T) {
	root := t.TempDir()

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: "INVALID/something",
		Path:        "some/file.go",
		OldString:   "x",
		NewString:   "y",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "INVALID/something")
}

// TestFindReplace_NonexistentLogicalName verifies that a logical name that
// does not resolve to an existing spec file yields a tool error.
func TestFindReplace_NonexistentLogicalName(t *testing.T) {
	root := t.TempDir()

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: "ROOT/does/not/exist",
		Path:        "some/file.go",
		OldString:   "x",
		NewString:   "y",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result)
}

// TestFindReplace_PathNotInImplements verifies that a path not declared in
// the node's implements list is rejected with an actionable error.
func TestFindReplace_PathNotInImplements(t *testing.T) {
	root := t.TempDir()

	logicalName := "ROOT/find_replace/allowed_node"
	specFile := testSpecPath(root, logicalName)
	// The spec only allows "internal/allowed/allowed.go".
	testWriteFile(t, specFile, testSpecContent("internal/allowed/allowed.go"))

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        "internal/not/allowed.go",
		OldString:   "x",
		NewString:   "y",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "path not allowed", "internal/allowed/allowed.go")
}

// TestFindReplace_PathTraversalAttempt verifies that a path traversal through
// the implements list is caught by ValidatePath.
func TestFindReplace_PathTraversalAttempt(t *testing.T) {
	root := t.TempDir()

	logicalName := "ROOT/find_replace/traversal_node"
	specFile := testSpecPath(root, logicalName)
	// Declare a traversal path in implements.
	testWriteFile(t, specFile, testSpecContent("../../etc/passwd"))

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        "../../etc/passwd",
		OldString:   "x",
		NewString:   "y",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result)
}

// TestFindReplace_EmptyPath verifies that an empty Path argument yields a
// tool error containing "path is empty".
func TestFindReplace_EmptyPath(t *testing.T) {
	root := t.TempDir()

	specPath := testSpecPath(root, "ROOT/find_replace/empty_path_node")
	testWriteFile(t, specPath, testSpecContent("internal/find_replace/test.go"))

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: "ROOT/find_replace/empty_path_node",
		Path:        "",
		OldString:   "x",
		NewString:   "y",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "path is empty")
}

// TestFindReplace_SymlinkEscapingProjectRoot verifies that a symlink inside
// the project directory that resolves to a path outside the project root is
// rejected with "resolves outside project root".
//
// The test is skipped if symlink creation fails (e.g. on Windows without the
// required privilege).
func TestFindReplace_SymlinkEscapingProjectRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir() // a directory outside the project root

	// Create a file outside the project to be the symlink target.
	outsideFile := filepath.Join(outside, "secret.txt")
	testWriteFile(t, outsideFile, "sensitive data")

	// Create the symlink inside the project root.
	symlinkDir := filepath.Join(root, "internal", "symlinked")
	if err := os.MkdirAll(symlinkDir, 0o755); err != nil {
		t.Skipf("cannot create symlink directory: %v", err)
	}
	symlinkPath := filepath.Join(symlinkDir, "escape.txt")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skipf("cannot create symlink (may require elevated privilege): %v", err)
	}

	logicalName := "ROOT/find_replace/symlink_node"
	specFile := testSpecPath(root, logicalName)
	// The implements list uses the in-project relative path of the symlink.
	symlinkRel := "internal/symlinked/escape.txt"
	testWriteFile(t, specFile, testSpecContent(symlinkRel))

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        symlinkRel,
		OldString:   "sensitive",
		NewString:   "replaced",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "resolves outside project root")
}

// TestFindReplace_FileDoesNotExist verifies that when the path is declared in
// implements but the file does not exist on disk, the handler returns a tool
// error containing "file does not exist".
func TestFindReplace_FileDoesNotExist(t *testing.T) {
	root := t.TempDir()

	logicalName := "ROOT/find_replace/no_file_node"
	specFile := testSpecPath(root, logicalName)
	missingRelPath := "internal/find_replace/missing.go"
	testWriteFile(t, specFile, testSpecContent(missingRelPath))

	// Do NOT create the target file — it must be absent.

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        missingRelPath,
		OldString:   "x",
		NewString:   "y",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "file does not exist")
}

// TestFindReplace_OldStringNotFound verifies that when OldString does not
// appear in the file the handler returns a tool error containing
// "old_string not found".
func TestFindReplace_OldStringNotFound(t *testing.T) {
	root := t.TempDir()

	logicalName := "ROOT/find_replace/not_found_node"
	specFile := testSpecPath(root, logicalName)
	targetRelPath := "internal/find_replace/test.go"
	testWriteFile(t, specFile, testSpecContent(targetRelPath))

	targetAbsPath := filepath.Join(root, filepath.FromSlash(targetRelPath))
	testWriteFile(t, targetAbsPath, "some content here")

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        targetRelPath,
		OldString:   "this string is absent",
		NewString:   "replacement",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "old_string not found")
}

// TestFindReplace_OldStringMatchesMultipleLocations verifies that when
// OldString appears more than once the handler returns a tool error
// containing "old_string matches multiple locations".
func TestFindReplace_OldStringMatchesMultipleLocations(t *testing.T) {
	root := t.TempDir()

	logicalName := "ROOT/find_replace/multi_match_node"
	specFile := testSpecPath(root, logicalName)
	targetRelPath := "internal/find_replace/test.go"
	testWriteFile(t, specFile, testSpecContent(targetRelPath))

	// "duplicate" appears twice in this file.
	targetAbsPath := filepath.Join(root, filepath.FromSlash(targetRelPath))
	testWriteFile(t, targetAbsPath, "duplicate content and another duplicate here")

	result, goErr := testCallHandler(t, root, FindReplaceArgs{
		LogicalName: logicalName,
		Path:        targetRelPath,
		OldString:   "duplicate",
		NewString:   "unique",
	})
	if goErr != nil {
		t.Fatalf("unexpected Go error: %v", goErr)
	}

	testAssertToolError(t, result, "old_string matches multiple locations")
}
