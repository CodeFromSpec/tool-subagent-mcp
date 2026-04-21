// spec: TEST/tech_design/internal/pathvalidation@v3

// Package pathvalidation_test exercises ValidatePath against the cases
// described in the spec leaf node. Each sub-test uses t.TempDir() as the
// project root so tests are isolated and leave no artifacts on disk.
//
// Threat model (from spec parent node):
//   - relative traversal:   ../../etc/passwd
//   - embedded traversal:   internal/../../outside/file.go
//   - OS-specific separators (backslash on Windows)
//   - symlinks that escape the project root
//
// Prevention approach follows OWASP guidance:
//   normalize first (filepath.Clean), then check for ".." components,
//   then resolve symlinks and verify containment.
package pathvalidation_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gustavo-neto/tool-subagent-mcp/internal/pathvalidation"
)

// run is a small helper that calls ValidatePath and checks whether the
// returned error matches expectations. It keeps each test case concise.
//
//   - wantErr == false  → expect nil (happy path)
//   - wantErr == true   → expect a non-nil error whose message contains errFragment
func run(t *testing.T, path, root string, wantErr bool, errFragment string) {
	t.Helper()

	err := pathvalidation.ValidatePath(path, root)

	if !wantErr {
		// Happy path: no error expected.
		if err != nil {
			t.Errorf("ValidatePath(%q, root) returned unexpected error: %v", path, err)
		}
		return
	}

	// Failure path: an error is required.
	if err == nil {
		t.Errorf("ValidatePath(%q, root) expected error containing %q, got nil", path, errFragment)
		return
	}
	if !strings.Contains(err.Error(), errFragment) {
		t.Errorf("ValidatePath(%q, root) error = %q; want substring %q", path, err.Error(), errFragment)
	}
}

// ---------------------------------------------------------------------------
// Happy Path
// ---------------------------------------------------------------------------

// TestHappyPath_SimpleRelative validates a typical relative source path.
// Spec step: "Simple relative path — expect no error."
func TestHappyPath_SimpleRelative(t *testing.T) {
	root := t.TempDir()
	run(t, "internal/config/config.go", root, false, "")
}

// TestHappyPath_NestedPath validates a deeper relative path with two segments
// before the filename.
// Spec step: "Nested path — expect no error."
func TestHappyPath_NestedPath(t *testing.T) {
	root := t.TempDir()
	run(t, "cmd/subagent-mcp/main.go", root, false, "")
}

// TestHappyPath_SingleFilename validates a bare filename with no directory
// component.
// Spec step: "Single filename — expect no error."
func TestHappyPath_SingleFilename(t *testing.T) {
	root := t.TempDir()
	run(t, "main.go", root, false, "")
}

// TestHappyPath_DotSegment validates that a "." component in the middle of a
// path is accepted — filepath.Clean collapses it to a normal path.
// Spec step: "Path with dot segment — cleaned to internal/config/config.go."
func TestHappyPath_DotSegment(t *testing.T) {
	root := t.TempDir()
	run(t, "internal/./config/config.go", root, false, "")
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

// TestEdge_TrailingSlash validates that a path ending with "/" is accepted.
// filepath.Clean strips trailing slashes, leaving a valid directory path.
// Spec step: "Path with trailing slash — expect no error."
func TestEdge_TrailingSlash(t *testing.T) {
	root := t.TempDir()
	run(t, "internal/config/", root, false, "")
}

// TestEdge_DuplicateSeparators validates that consecutive slashes are
// collapsed by filepath.Clean and the result is accepted.
// Spec step: "Path with duplicate separators — expect no error."
func TestEdge_DuplicateSeparators(t *testing.T) {
	root := t.TempDir()
	run(t, "internal//config//config.go", root, false, "")
}

// ---------------------------------------------------------------------------
// Failure Cases
// ---------------------------------------------------------------------------

// TestFail_EmptyPath validates that an empty string is rejected immediately.
// Spec step: "Empty path — expect error containing 'path is empty'."
func TestFail_EmptyPath(t *testing.T) {
	root := t.TempDir()
	run(t, "", root, true, "path is empty")
}

// TestFail_AbsoluteLeadingSlash validates that a Unix absolute path is
// rejected as absolute.
// Spec step: "Absolute path with leading slash — expect error containing
// 'path is absolute'."
func TestFail_AbsoluteLeadingSlash(t *testing.T) {
	root := t.TempDir()
	run(t, "/etc/passwd", root, true, "path is absolute")
}

// TestFail_AbsoluteDriveLetter validates that a Windows-style drive-letter
// path is rejected as absolute even on non-Windows hosts (the colon check
// must be OS-agnostic per the spec algorithm step 2).
// Spec step: "Absolute path with drive letter — expect error containing
// 'path is absolute'."
func TestFail_AbsoluteDriveLetter(t *testing.T) {
	root := t.TempDir()
	// The raw string contains a backslash-separated Windows path.
	// We use a raw literal to avoid Go escape confusion.
	run(t, `C:\Windows\system32`, root, true, "path is absolute")
}

// TestFail_SimpleTraversal validates the canonical `../../` attack.
// Spec step: "Simple traversal — expect error containing 'directory traversal'."
func TestFail_SimpleTraversal(t *testing.T) {
	root := t.TempDir()
	run(t, "../../etc/passwd", root, true, "directory traversal")
}

// TestFail_EmbeddedTraversal validates traversal that is hidden inside an
// otherwise plausible-looking path.
// Spec step: "Embedded traversal — expect error containing 'directory traversal'."
func TestFail_EmbeddedTraversal(t *testing.T) {
	root := t.TempDir()
	run(t, "internal/../../outside/file.go", root, true, "directory traversal")
}

// TestFail_SymlinkEscape validates that a symlink inside the project root
// that resolves to a directory outside it is rejected.
//
// Setup: create a real directory outside the temp dir, then create a symlink
// inside the temp dir pointing to that external directory. The tested path is
// "<symlink>/file.go".
//
// Spec step: "Symlink escaping project root — expect error containing
// 'resolves outside project root'."
//
// Skipped on platforms where os.Symlink is unavailable (e.g. Windows without
// the SeCreateSymbolicLink privilege).
func TestFail_SymlinkEscape(t *testing.T) {
	// Create the external target outside the project root.
	// We use a sibling temp dir so cleanup is automatic.
	externalDir := t.TempDir()

	// The project root is a separate temp dir.
	root := t.TempDir()

	// Place a symlink named "escape" inside root pointing at externalDir.
	symlinkPath := filepath.Join(root, "escape")
	err := os.Symlink(externalDir, symlinkPath)
	if err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("skipping symlink test: os.Symlink requires elevated privileges on Windows")
		}
		t.Fatalf("os.Symlink: %v", err)
	}

	// The path "escape/file.go" looks relative and clean, but its symlink
	// resolves to a location outside root.
	run(t, "escape/file.go", root, true, "resolves outside project root")
}

// TestFail_TraversalWithDotSegments validates that a disguised traversal using
// embedded "." segments is still caught after filepath.Clean normalization.
// Spec step: "Traversal disguised with dot segments — expect error containing
// 'directory traversal'."
func TestFail_TraversalWithDotSegments(t *testing.T) {
	root := t.TempDir()
	// After filepath.Clean, "internal/config/./../../outside" becomes
	// "outside" — no ".." survives, BUT the path escapes root.
	// The spec expects "directory traversal" for this case.
	// The path "internal/config/./../../outside" cleans to "../outside"
	// which contains a ".." component.
	run(t, "internal/config/./../../outside", root, true, "directory traversal")
}
