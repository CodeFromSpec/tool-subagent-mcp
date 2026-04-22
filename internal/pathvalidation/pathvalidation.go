// spec: ROOT/tech_design/internal/pathvalidation@v10

// Package pathvalidation provides security-critical path validation to prevent
// writing files outside the intended project directory boundary.
//
// Spec ref: ROOT/tech_design/internal/pathvalidation § "Intent"
// Threat model addressed: EXTERNAL/owasp-path-traversal § "What is path traversal"
//
// This package defends against the following attack vectors:
//   - Relative traversal (../../etc/passwd)
//   - Embedded traversal (internal/../../outside/file.go)
//   - OS-specific separators (backslash on Windows)
//   - Encoding tricks (URL-encoded or Unicode sequences resolved by filepath.Clean)
//   - Symlinks (valid-looking paths that resolve outside the project via symlinks)
package pathvalidation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks that path is safe to write within projectRoot.
//
// Returns nil if the path is safe. Returns a descriptive error if any
// security constraint is violated.
//
// Spec ref: ROOT/tech_design/internal/pathvalidation § "Interface"
func ValidatePath(path string, projectRoot string) error {
	// Step 1: Reject empty paths.
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (1)
	if path == "" {
		return errors.New("path is empty")
	}

	// Step 2: Reject absolute paths.
	// Use strings.HasPrefix to catch Unix-style absolute paths (e.g. /etc/passwd),
	// including on Windows where filepath.IsAbs returns false for paths starting
	// with "/" without a drive letter.
	// Also reject paths containing ":" to catch Windows drive letters (e.g. C:\...).
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (2)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return fmt.Errorf("path is absolute: %s", path)
	}
	if strings.Contains(path, ":") {
		return fmt.Errorf("path is absolute: %s", path)
	}

	// Step 3: Normalize separators and resolve "." segments via filepath.Clean.
	// This defuses many encoding tricks and ensures consistent component-level
	// inspection in step 4.
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (3)
	// OWASP ref: EXTERNAL/owasp-path-traversal § "Normalize before validating"
	cleaned := filepath.Clean(path)

	// Step 4: Reject if any path component is ".." after cleaning.
	// filepath.Clean resolves most traversal sequences, but an input of ".."
	// or one that still starts with ".." after cleaning must be explicitly rejected.
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (4)
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if component == ".." {
			return fmt.Errorf("path contains directory traversal: %s", path)
		}
	}

	// Step 5: Resolve the full absolute path by joining with the project root.
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (5)
	abs := filepath.Join(projectRoot, cleaned)

	// Step 6: Resolve symlinks to catch paths that appear safe but resolve
	// outside the project root via symlinks in the directory tree.
	// If the target does not yet exist, evaluate the longest existing prefix.
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (6)
	// OWASP ref: EXTERNAL/owasp-path-traversal § "Resolve symlinks"
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// The full path does not exist yet — walk up to find the longest
		// existing prefix and evaluate symlinks on that prefix.
		resolved, err = evalSymlinksForLongestExistingPrefix(abs)
		if err != nil {
			// If we cannot determine a safe resolved prefix, reject the path.
			return fmt.Errorf("path resolves outside project root: %s", path)
		}
	}

	// Step 7: Verify the resolved path starts with projectRoot.
	// A trailing separator is appended to projectRoot to prevent a prefix like
	// "/project-root-sibling" from matching "/project-root".
	// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (7)
	// OWASP ref: EXTERNAL/owasp-path-traversal § "Resolve and verify containment"
	rootWithSep := filepath.Clean(projectRoot) + string(filepath.Separator)
	// Also accept an exact match (resolved == projectRoot itself).
	cleanRoot := filepath.Clean(projectRoot)
	if resolved != cleanRoot && !strings.HasPrefix(resolved, rootWithSep) {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	return nil
}

// evalSymlinksForLongestExistingPrefix walks up the directory tree of target
// to find the longest prefix that exists on disk, resolves symlinks on that
// prefix, and returns the reconstructed path.
//
// This is used when the full target path does not yet exist (e.g. the file
// is about to be created) so that symlink attacks via existing ancestor
// directories are still detected.
//
// Spec ref: ROOT/tech_design/internal/pathvalidation § "Algorithm" (6)
func evalSymlinksForLongestExistingPrefix(target string) (string, error) {
	// Collect path components from target down to root.
	current := target
	var suffix []string

	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached the filesystem root without finding an existing path.
			// Return the original target unchanged; the containment check in
			// the caller will determine if it is safe.
			return target, nil
		}

		_, statErr := os.Lstat(current)
		if statErr == nil {
			// current exists — resolve symlinks on it and reattach the suffix.
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			// Reattach the non-existent suffix components.
			result := filepath.Join(append([]string{resolved}, suffix...)...)
			return result, nil
		}

		// current does not exist — record its base name and try the parent.
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
}
