// spec: ROOT/tech_design/internal/pathvalidation@v6

// Package pathvalidation validates that a file path is safe to write within
// a project directory. This is a security-critical package — it prevents any
// write operation from escaping the intended project boundary.
//
// Threat model (from spec):
//   - Relative traversal:     ../../etc/passwd
//   - Embedded traversal:     internal/../../outside/file.go
//   - OS-specific separators: backslash on Windows (..\..\)
//   - Encoding tricks:        URL-encoded or Unicode sequences
//   - Symlinks:               a valid-looking relative path that resolves
//     outside the project via a symlink in the directory tree
//
// OWASP guidance applied here:
//   - Normalize before validating (filepath.Clean)
//   - Resolve symlinks before containment check (filepath.EvalSymlinks)
//   - Reject, never sanitize
//   - Prefer allow-list reasoning: only paths that pass every gate are accepted
package pathvalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks whether path is a safe relative path that, when joined
// with projectRoot, remains inside the project directory tree.
//
// Returns nil when the path is safe. Returns a descriptive error otherwise.
// This function never writes or creates anything on disk.
//
// Algorithm (spec §Contracts/Algorithm):
//  1. Reject empty paths.
//  2. Reject absolute paths (starts with "/" or contains ":" on Windows).
//  3. Call filepath.Clean to normalize separators and resolve "." segments.
//  4. Reject if any path component is ".." after cleaning.
//  5. Resolve the full absolute path: abs = filepath.Join(projectRoot, cleaned).
//  6. Call filepath.EvalSymlinks on abs; if the target does not exist yet,
//     evaluate the longest existing prefix instead.
//  7. Verify the resolved path starts with projectRoot. Reject if not.
func ValidatePath(path string, projectRoot string) error {
	// Step 1 — Reject empty paths.
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Step 2 — Reject absolute paths.
	// On Unix an absolute path starts with "/".
	// On Windows a drive-letter path contains ":" (e.g. "C:\...").
	// filepath.IsAbs covers both cases portably; we additionally guard
	// against the colon explicitly to match the spec wording.
	if filepath.IsAbs(path) || strings.Contains(path, ":") {
		return fmt.Errorf("path is absolute: %s", path)
	}

	// Step 3 — Normalize: resolve ".", duplicate separators, OS separators.
	// filepath.Clean handles all of these. After this call the path uses the
	// OS separator and has no redundant components.
	cleaned := filepath.Clean(path)

	// Step 4 — Reject traversal components that survived cleaning.
	// filepath.Clean collapses "a/../../b" to "../b", so a ".." component
	// after cleaning is a real escape attempt.
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if component == ".." {
			return fmt.Errorf("path contains directory traversal: %s", path)
		}
	}

	// Step 5 — Build the full absolute candidate path.
	abs := filepath.Join(projectRoot, cleaned)

	// Step 6 — Resolve symlinks before the containment check.
	// OWASP: a path that looks local may point outside the project via a
	// symlink. filepath.EvalSymlinks resolves the chain completely.
	//
	// If the final target does not exist yet (e.g. a new file being created),
	// EvalSymlinks returns an error. In that case we walk up to the longest
	// existing prefix and evaluate that, then reattach the non-existent tail.
	// This ensures symlinks in existing ancestor directories are still caught.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// The path (or part of it) does not exist yet — evaluate the longest
		// existing prefix. Walk up the directory tree one component at a time.
		resolved, err = evalExistingPrefix(abs)
		if err != nil {
			// Even the root could not be evaluated; propagate the error.
			return fmt.Errorf("path resolves outside project root: %s", path)
		}
	}

	// Ensure projectRoot itself is in canonical, symlink-resolved form so that
	// the string prefix check is reliable across platforms.
	resolvedRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		// If projectRoot cannot be resolved it is a configuration problem;
		// treat the path as unsafe.
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	// Normalize both sides to use the same separator style.
	resolved = filepath.Clean(resolved)
	resolvedRoot = filepath.Clean(resolvedRoot)

	// Step 7 — Containment check.
	// The resolved path must start with resolvedRoot followed by a separator
	// (or be exactly equal to it, though writing to the root itself is odd).
	// Using strings.HasPrefix alone would accept "/project-evil" when root is
	// "/project", so we append a separator to the root before comparing.
	if !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) &&
		resolved != resolvedRoot {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	return nil
}

// evalExistingPrefix resolves symlinks for the longest prefix of absPath that
// actually exists on disk, then reattaches the remaining (non-existent) suffix.
// This allows symlink detection in ancestor directories even when the final
// file has not been created yet.
func evalExistingPrefix(absPath string) (string, error) {
	// Split the path into components and walk from the root down, evaluating
	// each prefix until we find one that does not exist.
	dir := absPath
	var tail []string

	for {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			// This prefix exists and has been resolved. Reattach the tail.
			parts := append([]string{resolved}, tail...)
			return filepath.Join(parts...), nil
		}

		if !os.IsNotExist(err) {
			// Unexpected error (permission denied, etc.) — surface it.
			return "", err
		}

		// The current prefix does not exist. Strip one component and retry.
		parent := filepath.Dir(dir)
		if parent == dir {
			// We have reached the filesystem root without finding an existing
			// prefix. This should not happen in practice since projectRoot
			// must exist, but guard against it.
			return "", fmt.Errorf("no existing prefix found for %s", absPath)
		}
		// Prepend the stripped component to the tail we will reattach.
		tail = append([]string{filepath.Base(dir)}, tail...)
		dir = parent
	}
}
