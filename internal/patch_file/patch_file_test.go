// code-from-spec: TEST/tech_design/internal/tools/patch_file@v2

// Package patch_file provides tests for the patch_file tool handler.
// Each test creates a fresh temp directory as its project root and
// working directory, builds a minimal spec tree with frontmatter,
// writes any required source files, then calls HandlePatchFile directly.
package patch_file

import (
	"context"
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

// testMakeSpecTree creates a minimal spec node at
// code-from-spec/spec/<subPath>/_node.md with YAML frontmatter that lists
// the given implements paths. The file is created inside root.
func testMakeSpecTree(t *testing.T, root, logicalSubPath string, implements []string) {
	t.Helper()

	// Build the YAML frontmatter.
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("version: 1\n")
	if len(implements) > 0 {
		sb.WriteString("implements:\n")
		for _, p := range implements {
			sb.WriteString("  - " + p + "\n")
		}
	}
	sb.WriteString("---\n\n# node\n")

	// Determine the _node.md path inside root.
	// logicalSubPath is the part after "ROOT/" (or empty for ROOT itself).
	var nodeMdPath string
	if logicalSubPath == "" {
		nodeMdPath = filepath.Join(root, "code-from-spec", "spec", "_node.md")
	} else {
		// Convert forward-slash logical path to OS path segments.
		rel := filepath.FromSlash(logicalSubPath)
		nodeMdPath = filepath.Join(root, "code-from-spec", "spec", rel, "_node.md")
	}

	if err := os.MkdirAll(filepath.Dir(nodeMdPath), 0o755); err != nil {
		t.Fatalf("testMakeSpecTree: MkdirAll: %v", err)
	}
	if err := os.WriteFile(nodeMdPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("testMakeSpecTree: WriteFile: %v", err)
	}
}

// testWriteFile writes content to root/<relPath>, creating directories as
// needed. relPath uses forward slashes.
func testWriteFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("testWriteFile: MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("testWriteFile: WriteFile: %v", err)
	}
}

// testReadFile reads root/<relPath> and returns its content as a string.
func testReadFile(t *testing.T, root, relPath string) string {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("testReadFile: %v", err)
	}
	return string(data)
}

// testCallHandler calls HandlePatchFile with the given args after changing
// the working directory to root. It restores the working directory on cleanup.
func testCallHandler(t *testing.T, root string, args PatchFileArgs) *mcp.CallToolResult {
	t.Helper()

	// Save and restore the working directory.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("testCallHandler: Getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("testCallHandler cleanup: Chdir: %v", err)
		}
	})
	if err := os.Chdir(root); err != nil {
		t.Fatalf("testCallHandler: Chdir to %s: %v", root, err)
	}

	result, _, err := HandlePatchFile(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("testCallHandler: unexpected Go error (should always be nil): %v", err)
	}
	return result
}

// testIsError returns true when the CallToolResult has IsError set.
func testIsError(result *mcp.CallToolResult) bool {
	return result != nil && result.IsError
}

// testResultText extracts the text from the first TextContent in the result.
func testResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatal("testResultText: result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("testResultText: first content is not *mcp.TextContent")
	}
	return tc.Text
}

// ---------------------------------------------------------------------------
// Happy path tests
// ---------------------------------------------------------------------------

// TestHandlePatchFile_AppliesSimpleDiff verifies that a basic single-hunk
// diff is applied correctly and the file on disk is updated.
func TestHandlePatchFile_AppliesSimpleDiff(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})

	initial := "package main\n\nfunc hello() string {\n\treturn \"hello\"\n}\n"
	testWriteFile(t, root, "output/file.go", initial)

	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -3,3 +3,3 @@\n func hello() string {\n-\treturn \"hello\"\n+\treturn \"world\"\n }\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if testIsError(result) {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	got := testResultText(t, result)
	if got != "patched output/file.go" {
		t.Errorf("unexpected result text: %q", got)
	}

	content := testReadFile(t, root, "output/file.go")
	if !strings.Contains(content, `"world"`) {
		t.Errorf("expected 'world' in patched file, got:\n%s", content)
	}
	if strings.Contains(content, `"hello"`) {
		t.Errorf("expected 'hello' to be replaced in patched file, got:\n%s", content)
	}
}

// TestHandlePatchFile_AppliesMultiHunkDiff verifies that a diff with two
// separate hunks is applied and both modifications are reflected on disk.
func TestHandlePatchFile_AppliesMultiHunkDiff(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})

	initial := strings.Join([]string{
		"package main",
		"",
		"func foo() string {",
		"\treturn \"foo\"",
		"}",
		"",
		"func bar() string {",
		"\treturn \"bar\"",
		"}",
		"",
	}, "\n")
	testWriteFile(t, root, "output/file.go", initial)

	// Two-hunk diff: change "foo" → "FOO" and "bar" → "BAR".
	diff := "--- a/output/file.go\n+++ b/output/file.go\n" +
		"@@ -3,3 +3,3 @@\n func foo() string {\n-\treturn \"foo\"\n+\treturn \"FOO\"\n }\n" +
		"@@ -7,3 +7,3 @@\n func bar() string {\n-\treturn \"bar\"\n+\treturn \"BAR\"\n }\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if testIsError(result) {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	content := testReadFile(t, root, "output/file.go")
	if !strings.Contains(content, `"FOO"`) {
		t.Errorf("expected 'FOO' in patched file, got:\n%s", content)
	}
	if !strings.Contains(content, `"BAR"`) {
		t.Errorf("expected 'BAR' in patched file, got:\n%s", content)
	}
}

// TestHandlePatchFile_AppliesDiffThatAddsLines verifies that a diff adding
// new lines (no removals) is applied correctly.
func TestHandlePatchFile_AppliesDiffThatAddsLines(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})

	initial := "package main\n\nfunc hello() {}\n"
	testWriteFile(t, root, "output/file.go", initial)

	// Add a new function after the existing one.
	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -3,1 +3,4 @@\n func hello() {}\n+\n+func world() {\n+\t// added\n+}\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if testIsError(result) {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	content := testReadFile(t, root, "output/file.go")
	if !strings.Contains(content, "func world()") {
		t.Errorf("expected 'func world()' in patched file, got:\n%s", content)
	}
}

// TestHandlePatchFile_AppliesDiffThatRemovesLines verifies that a diff
// removing lines (no additions) is applied correctly.
func TestHandlePatchFile_AppliesDiffThatRemovesLines(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})

	initial := "package main\n\n// TODO: remove this\nfunc hello() {}\n"
	testWriteFile(t, root, "output/file.go", initial)

	// Remove the TODO comment line.
	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -2,3 +2,2 @@\n \n-// TODO: remove this\n func hello() {}\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if testIsError(result) {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	content := testReadFile(t, root, "output/file.go")
	if strings.Contains(content, "TODO") {
		t.Errorf("expected TODO line to be removed, got:\n%s", content)
	}
}

// TestHandlePatchFile_PathWithBackslashesNormalized verifies that a path
// supplied with Windows-style backslashes is normalized to forward slashes
// before validation and matching. Only runs on Windows.
func TestHandlePatchFile_PathWithBackslashesNormalized(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("backslash normalization test only runs on Windows")
	}

	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})

	initial := "package main\n\nfunc hello() string {\n\treturn \"hello\"\n}\n"
	testWriteFile(t, root, "output/file.go", initial)

	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -3,3 +3,3 @@\n func hello() string {\n-\treturn \"hello\"\n+\treturn \"world\"\n }\n"

	// Supply path with Windows backslashes.
	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        `output\file.go`,
		Diff:        diff,
	})

	if testIsError(result) {
		t.Fatalf("expected success, got error: %s", testResultText(t, result))
	}

	got := testResultText(t, result)
	// The result message should use forward slashes.
	if got != "patched output/file.go" {
		t.Errorf("unexpected result text: %q", got)
	}
}

// ---------------------------------------------------------------------------
// Failure cases
// ---------------------------------------------------------------------------

// TestHandlePatchFile_InvalidLogicalNamePrefix verifies that a logical name
// not starting with ROOT/, TEST/, ROOT, or TEST produces a tool error.
func TestHandlePatchFile_InvalidLogicalNamePrefix(t *testing.T) {
	root := t.TempDir()

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "some/file.go",
		Diff:        "",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for invalid logical name prefix, got success: %s", testResultText(t, result))
	}
}

// TestHandlePatchFile_NonexistentLogicalName verifies that a logical name
// that maps to a missing spec file returns a tool error.
func TestHandlePatchFile_NonexistentLogicalName(t *testing.T) {
	root := t.TempDir()
	// Do NOT create the spec file for ROOT/nonexistent.

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "some/file.go",
		Diff:        "",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for nonexistent logical name, got success: %s", testResultText(t, result))
	}
}

// TestHandlePatchFile_PathNotInImplements verifies that supplying a path not
// listed in implements returns a tool error mentioning "path not allowed" and
// the allowed paths.
func TestHandlePatchFile_PathNotInImplements(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"allowed/file.go"})

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Diff:        "",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for path not in implements, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("expected 'path not allowed' in error message, got: %s", text)
	}
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("expected allowed path listed in error message, got: %s", text)
	}
}

// TestHandlePatchFile_PathTraversalAttempt verifies that a path containing
// directory traversal sequences listed in implements is rejected by
// ValidatePath.
func TestHandlePatchFile_PathTraversalAttempt(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"../../etc/passwd"})

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Diff:        "",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for path traversal, got success: %s", testResultText(t, result))
	}
}

// TestHandlePatchFile_EmptyPath verifies that an empty path returns a tool
// error mentioning "path is empty".
func TestHandlePatchFile_EmptyPath(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"some/file.go"})

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Diff:        "",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for empty path, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path is empty") {
		t.Errorf("expected 'path is empty' in error message, got: %s", text)
	}
}

// TestHandlePatchFile_SymlinkEscapingProjectRoot verifies that a symlink
// pointing outside the project root is rejected by ValidatePath.
func TestHandlePatchFile_SymlinkEscapingProjectRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Symlink creation on Windows requires elevated privileges or
		// Developer Mode enabled — skip to avoid test infrastructure issues.
		t.Skip("symlink test skipped on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir() // a directory outside root

	// Create a symlink inside root pointing outside.
	symlinkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outside, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Register the symlink-based path in implements.
	testMakeSpecTree(t, root, "a", []string{"escape/secret.go"})

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "escape/secret.go",
		Diff:        "",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for symlink escaping project root, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("expected 'resolves outside project root' in error message, got: %s", text)
	}
}

// TestHandlePatchFile_FileDoesNotExist verifies that patching a non-existent
// file returns a tool error with the appropriate message.
func TestHandlePatchFile_FileDoesNotExist(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})
	// Do NOT create output/file.go.

	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -1,1 +1,1 @@\n-old\n+new\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for non-existent file, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "file does not exist: output/file.go") {
		t.Errorf("expected 'file does not exist: output/file.go' in error, got: %s", text)
	}
}

// TestHandlePatchFile_MalformedDiff verifies that an unparseable diff string
// returns a tool error mentioning "failed to parse diff".
func TestHandlePatchFile_MalformedDiff(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})
	testWriteFile(t, root, "output/file.go", "package main\n")

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        "this is not a valid diff",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for malformed diff, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "failed to parse diff") {
		t.Errorf("expected 'failed to parse diff' in error, got: %s", text)
	}
}

// TestHandlePatchFile_DiffWithZeroFileEntries verifies that an empty or
// whitespace-only diff returns a tool error mentioning
// "diff must contain exactly one file".
func TestHandlePatchFile_DiffWithZeroFileEntries(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})
	testWriteFile(t, root, "output/file.go", "package main\n")

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        "   \n",
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for zero file entries, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "diff must contain exactly one file") {
		t.Errorf("expected 'diff must contain exactly one file' in error, got: %s", text)
	}
}

// TestHandlePatchFile_DiffWithMultipleFileEntries verifies that a diff
// touching more than one file returns a tool error mentioning
// "diff must contain exactly one file".
func TestHandlePatchFile_DiffWithMultipleFileEntries(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})
	testWriteFile(t, root, "output/file.go", "package main\n")

	// A diff that modifies two different files.
	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -1,1 +1,1 @@\n-package main\n+package x\n" +
		"--- a/other/file.go\n+++ b/other/file.go\n@@ -1,1 +1,1 @@\n-package main\n+package y\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for multiple file entries, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "diff must contain exactly one file") {
		t.Errorf("expected 'diff must contain exactly one file' in error, got: %s", text)
	}
}

// TestHandlePatchFile_DiffContextDoesNotMatchFile verifies that a diff whose
// context lines do not match the file's actual content returns a tool error
// mentioning "failed to apply diff to output/file.go".
func TestHandlePatchFile_DiffContextDoesNotMatchFile(t *testing.T) {
	root := t.TempDir()
	testMakeSpecTree(t, root, "a", []string{"output/file.go"})
	testWriteFile(t, root, "output/file.go", "package main\n\nfunc actual() {}\n")

	// The context line refers to "func different()" which does not exist.
	diff := "--- a/output/file.go\n+++ b/output/file.go\n@@ -3,1 +3,1 @@\n func different() {}\n-func removed() {}\n+func added() {}\n"

	result := testCallHandler(t, root, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if !testIsError(result) {
		t.Fatalf("expected tool error for mismatched diff context, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "failed to apply diff to output/file.go") {
		t.Errorf("expected 'failed to apply diff to output/file.go' in error, got: %s", text)
	}
}
