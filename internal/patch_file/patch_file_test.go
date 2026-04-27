// spec: TEST/tech_design/internal/tools/patch_file@v7
// code-from-spec: TEST/tech_design/internal/tools/patch_file@v7

package patch_file

// Tests for HandlePatchFile.
//
// Each test uses t.TempDir() as the project root and working directory.
// A spec tree is created with the necessary frontmatter containing an
// Implements list. The handler is called with PatchFileArgs including
// the LogicalName of the node.
//
// Spec files are created at paths matching logicalnames.PathFromLogicalName:
//   - ROOT   → <tmpdir>/code-from-spec/_node.md
//   - ROOT/a → <tmpdir>/code-from-spec/a/_node.md

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --------------------------------------------------------------------------
// Test helpers
// --------------------------------------------------------------------------

// testMakeSpecFile creates a spec file at the path corresponding to the
// given logical name inside rootDir, writing the provided YAML frontmatter.
//
// For ROOT/a the file path is: <rootDir>/code-from-spec/a/_node.md
// For ROOT   the file path is: <rootDir>/code-from-spec/_node.md
func testMakeSpecFile(t *testing.T, rootDir, logicalName string, implements []string) {
	t.Helper()

	// Build the relative path under code-from-spec.
	// ROOT      → code-from-spec/_node.md
	// ROOT/<x>  → code-from-spec/<x>/_node.md
	var relPath string
	switch logicalName {
	case "ROOT":
		relPath = filepath.Join("code-from-spec", "_node.md")
	default:
		// Strip "ROOT/" prefix and build <segments>/_node.md
		suffix := strings.TrimPrefix(logicalName, "ROOT/")
		segments := filepath.FromSlash(suffix)
		relPath = filepath.Join("code-from-spec", segments, "_node.md")
	}

	fullPath := filepath.Join(rootDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("testMakeSpecFile: mkdir %s: %v", filepath.Dir(fullPath), err)
	}

	// Build frontmatter YAML.
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("version: 1\n")
	if len(implements) > 0 {
		sb.WriteString("implements:\n")
		for _, p := range implements {
			sb.WriteString("  - " + p + "\n")
		}
	}
	sb.WriteString("---\n\n# Node\n")

	if err := os.WriteFile(fullPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("testMakeSpecFile: write %s: %v", fullPath, err)
	}
}

// testWriteFile creates a file (and parent directories) at
// filepath.Join(rootDir, relPath) with the given content.
func testWriteFile(t *testing.T, rootDir, relPath, content string) {
	t.Helper()
	full := filepath.Join(rootDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("testWriteFile: mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("testWriteFile: write: %v", err)
	}
}

// testReadFile reads and returns the content of filepath.Join(rootDir, relPath).
func testReadFile(t *testing.T, rootDir, relPath string) string {
	t.Helper()
	full := filepath.Join(rootDir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("testReadFile: %v", err)
	}
	return string(data)
}

// testResultText extracts the text from the first content entry of a
// *mcp.CallToolResult.
func testResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("testResultText: result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("testResultText: first content is not *mcp.TextContent")
	}
	return tc.Text
}

// testCall invokes HandlePatchFile, changing os working directory to rootDir
// for the duration of the call and restoring it afterwards.
func testCall(t *testing.T, rootDir string, args PatchFileArgs) *mcp.CallToolResult {
	t.Helper()

	// Save and restore working directory so each test is isolated.
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("testCall: getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("testCall: chdir to %s: %v", rootDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWD); err != nil {
			t.Errorf("testCall: restore wd: %v", err)
		}
	})

	result, _, err := HandlePatchFile(context.Background(), nil, args)
	if err != nil {
		t.Fatalf("testCall: unexpected Go error: %v", err)
	}
	return result
}

// --------------------------------------------------------------------------
// Happy path tests
// --------------------------------------------------------------------------

// TestHandlePatchFile_AppliesSimpleDiff verifies that a diff changing one
// string in the file is applied correctly.
func TestHandlePatchFile_AppliesSimpleDiff(t *testing.T) {
	rootDir := t.TempDir()

	// Create spec node with implements.
	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})

	// Write the initial file.
	initial := `package main

func hello() string {
	return "hello"
}
`
	testWriteFile(t, rootDir, "output/file.go", initial)

	// Diff changes "hello" to "world".
	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -3,3 +3,3 @@
 func hello() string {
-	return "hello"
+	return "world"
 }
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if text != "patched output/file.go" {
		t.Errorf("unexpected success message: %q", text)
	}

	// Verify the file on disk.
	got := testReadFile(t, rootDir, "output/file.go")
	if strings.Contains(got, `"hello"`) {
		t.Errorf("file still contains old string %q:\n%s", `"hello"`, got)
	}
	if !strings.Contains(got, `"world"`) {
		t.Errorf("file does not contain new string %q:\n%s", `"world"`, got)
	}
}

// TestHandlePatchFile_AppliesMultiHunkDiff verifies that a diff with two
// separate hunks is applied correctly.
func TestHandlePatchFile_AppliesMultiHunkDiff(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})

	initial := `package main

func greet() string {
	return "hello"
}

func farewell() string {
	return "goodbye"
}
`
	testWriteFile(t, rootDir, "output/file.go", initial)

	// Two hunks: change "hello" → "hi" and "goodbye" → "bye".
	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -3,3 +3,3 @@
 func greet() string {
-	return "hello"
+	return "hi"
 }
@@ -7,3 +7,3 @@
 func farewell() string {
-	return "goodbye"
+	return "bye"
 }
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", testResultText(t, result))
	}

	got := testReadFile(t, rootDir, "output/file.go")
	if !strings.Contains(got, `"hi"`) {
		t.Errorf("first hunk not applied; missing %q:\n%s", `"hi"`, got)
	}
	if !strings.Contains(got, `"bye"`) {
		t.Errorf("second hunk not applied; missing %q:\n%s", `"bye"`, got)
	}
}

// TestHandlePatchFile_AppliesDiffThatAddsLines verifies that a diff that
// only adds lines is applied correctly.
func TestHandlePatchFile_AppliesDiffThatAddsLines(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})

	initial := `package main

func foo() {}
`
	testWriteFile(t, rootDir, "output/file.go", initial)

	// Diff adds a new function after foo.
	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -3,1 +3,4 @@
 func foo() {}
+
+func bar() {}
+
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", testResultText(t, result))
	}

	got := testReadFile(t, rootDir, "output/file.go")
	if !strings.Contains(got, "func bar()") {
		t.Errorf("added lines not present in file:\n%s", got)
	}
}

// TestHandlePatchFile_AppliesDiffThatRemovesLines verifies that a diff that
// only removes lines is applied correctly.
func TestHandlePatchFile_AppliesDiffThatRemovesLines(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})

	initial := `package main

func foo() {}

func bar() {}
`
	testWriteFile(t, rootDir, "output/file.go", initial)

	// Diff removes bar().
	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -4,2 +4,0 @@
-
-func bar() {}
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", testResultText(t, result))
	}

	got := testReadFile(t, rootDir, "output/file.go")
	if strings.Contains(got, "func bar()") {
		t.Errorf("removed lines still present in file:\n%s", got)
	}
}

// TestHandlePatchFile_BackslashPathNormalized verifies that a path with
// backslashes (Windows-style) is normalized to forward slashes before
// validation. This test is only meaningful on Windows.
func TestHandlePatchFile_BackslashPathNormalized(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})

	initial := `package main

func hello() string {
	return "hello"
}
`
	testWriteFile(t, rootDir, "output/file.go", initial)

	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -3,3 +3,3 @@
 func hello() string {
-	return "hello"
+	return "world"
 }
`

	// Use backslash path — the handler must normalize it.
	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        `output\file.go`,
		Diff:        diff,
	})

	if result.IsError {
		t.Fatalf("expected success, got tool error: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if text != "patched output/file.go" {
		t.Errorf("unexpected success message: %q", text)
	}
}

// --------------------------------------------------------------------------
// Failure case tests
// --------------------------------------------------------------------------

// TestHandlePatchFile_InvalidLogicalNamePrefix verifies that a logical name
// not starting with ROOT or TEST is rejected.
func TestHandlePatchFile_InvalidLogicalNamePrefix(t *testing.T) {
	rootDir := t.TempDir()

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "EXTERNAL/something",
		Path:        "output/file.go",
		Diff:        "--- a\n+++ b\n",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}
}

// TestHandlePatchFile_NonexistentLogicalName verifies that a logical name
// that resolves to a nonexistent spec file returns a tool error.
func TestHandlePatchFile_NonexistentLogicalName(t *testing.T) {
	rootDir := t.TempDir()
	// Deliberately do NOT create the spec file for ROOT/nonexistent.

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/nonexistent",
		Path:        "output/file.go",
		Diff:        "--- a\n+++ b\n",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}
}

// TestHandlePatchFile_PathNotInImplements verifies that a path not listed in
// the node's implements returns a tool error mentioning "path not allowed".
func TestHandlePatchFile_PathNotInImplements(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"allowed/file.go"})

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "other/file.go",
		Diff:        "--- a\n+++ b\n",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path not allowed") {
		t.Errorf("expected error to contain %q, got: %s", "path not allowed", text)
	}
	if !strings.Contains(text, "allowed/file.go") {
		t.Errorf("expected error to list allowed paths, got: %s", text)
	}
}

// TestHandlePatchFile_PathTraversalAttempt verifies that a traversal path
// listed in implements is caught by ValidatePath.
func TestHandlePatchFile_PathTraversalAttempt(t *testing.T) {
	rootDir := t.TempDir()

	// The spec lists a traversal path in implements — ValidatePath must block it.
	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"../../etc/passwd"})

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "../../etc/passwd",
		Diff:        "--- a\n+++ b\n",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}
}

// TestHandlePatchFile_EmptyPath verifies that an empty path is rejected with
// a message containing "path is empty".
func TestHandlePatchFile_EmptyPath(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"some/file.go"})

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "",
		Diff:        "--- a\n+++ b\n",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "path is empty") {
		t.Errorf("expected error to contain %q, got: %s", "path is empty", text)
	}
}

// TestHandlePatchFile_SymlinkEscapingProjectRoot verifies that a symlink
// inside the temp dir pointing outside it is rejected.
func TestHandlePatchFile_SymlinkEscapingProjectRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated privileges on Windows")
	}

	rootDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a symlink inside rootDir that points outside rootDir.
	symlinkPath := filepath.Join(rootDir, "escape_link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	// Use a path through the symlink as the implements entry.
	escapePath := "escape_link/evil.go"

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{escapePath})

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        escapePath,
		Diff:        "--- a\n+++ b\n",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "resolves outside project root") {
		t.Errorf("expected error to contain %q, got: %s", "resolves outside project root", text)
	}
}

// TestHandlePatchFile_FileDoesNotExist verifies that patching a file that has
// not been created returns a tool error mentioning "file does not exist".
func TestHandlePatchFile_FileDoesNotExist(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})
	// Deliberately do NOT create output/file.go.

	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -1,1 +1,1 @@
-old
+new
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "file does not exist: output/file.go") {
		t.Errorf("expected error to contain %q, got: %s", "file does not exist: output/file.go", text)
	}
}

// TestHandlePatchFile_MalformedDiff verifies that completely malformed diff
// input (not parseable as any diff) is rejected.
// gitdiff.Parse does not error on completely malformed input — it returns
// zero file entries, which is caught by the "exactly one file" check.
func TestHandlePatchFile_MalformedDiff(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})
	testWriteFile(t, rootDir, "output/file.go", "package main\n")

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        "this is not a valid diff",
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "diff must contain exactly one file") {
		t.Errorf("expected error to contain %q, got: %s", "diff must contain exactly one file", text)
	}
}

// TestHandlePatchFile_DiffWithZeroFileEntries verifies that an empty or
// whitespace-only diff is rejected.
func TestHandlePatchFile_DiffWithZeroFileEntries(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})
	testWriteFile(t, rootDir, "output/file.go", "package main\n")

	for _, emptyDiff := range []string{"", "   ", "\n\n"} {
		result := testCall(t, rootDir, PatchFileArgs{
			LogicalName: "ROOT/a",
			Path:        "output/file.go",
			Diff:        emptyDiff,
		})

		if !result.IsError {
			t.Fatalf("expected tool error for empty diff %q, got success: %s", emptyDiff, testResultText(t, result))
		}

		text := testResultText(t, result)
		if !strings.Contains(text, "diff must contain exactly one file") {
			t.Errorf("expected error to contain %q for diff %q, got: %s", "diff must contain exactly one file", emptyDiff, text)
		}
	}
}

// TestHandlePatchFile_DiffWithMultipleFileEntries verifies that a diff
// spanning multiple files is rejected.
func TestHandlePatchFile_DiffWithMultipleFileEntries(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})
	testWriteFile(t, rootDir, "output/file.go", "package main\n")

	// Diff modifies two different files.
	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -1,1 +1,1 @@
-package main
+package mainx
--- a/output/other.go
+++ b/output/other.go
@@ -1,1 +1,1 @@
-old
+new
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	if !strings.Contains(text, "diff must contain exactly one file") {
		t.Errorf("expected error to contain %q, got: %s", "diff must contain exactly one file", text)
	}
}

// TestHandlePatchFile_DiffContextDoesNotMatchFile verifies that a diff whose
// context lines do not match the file's actual content is rejected.
// Per the spec, gitdiff.Parse rejects such diffs as semantically invalid
// (e.g. "fragment contains no changes"), so the error is caught at parse
// time (step 9) and the message contains "failed to parse diff".
func TestHandlePatchFile_DiffContextDoesNotMatchFile(t *testing.T) {
	rootDir := t.TempDir()

	testMakeSpecFile(t, rootDir, "ROOT/a", []string{"output/file.go"})
	// Write a file whose content does NOT match the diff's context lines.
	testWriteFile(t, rootDir, "output/file.go", "package main\n\nfunc foo() {}\n")

	// Diff references context lines that do not exist in the file.
	diff := `--- a/output/file.go
+++ b/output/file.go
@@ -1,3 +1,3 @@
 this line does not exist in the file
-old content that is not there
+new content
 another missing line
`

	result := testCall(t, rootDir, PatchFileArgs{
		LogicalName: "ROOT/a",
		Path:        "output/file.go",
		Diff:        diff,
	})

	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}

	text := testResultText(t, result)
	// Per spec note: gitdiff.Parse catches context mismatches as parse errors.
	if !strings.Contains(text, "failed to parse diff") && !strings.Contains(text, "failed to apply diff") {
		t.Errorf("expected error to contain %q or %q, got: %s",
			"failed to parse diff", "failed to apply diff", text)
	}
}
