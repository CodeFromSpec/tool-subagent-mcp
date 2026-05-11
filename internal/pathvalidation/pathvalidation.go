// code-from-spec: ROOT/tech_design/internal/pathvalidation@v14
//
// Package pathvalidation validates that file paths are safe to write
// within a project directory. This is a security-critical package.
//
// Threat model: prevents writing files outside the intended project
// boundary via relative traversal (../../etc/passwd), embedded traversal
// (internal/../../outside/file.go), OS-specific separators (backslash
// on Windows), encoding tricks, and symlinks. See OWASP path traversal
// guidance for details.
//
// The validation function is read-only — it never writes or creates
// anything on disk. It never sanitizes or fixes an invalid path; it
// rejects and reports, leaving the decision to the caller.
package pathvalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks that path is safe to write within projectRoot.
// Returns nil if safe, or a descriptive error if the path violates any
// security constraint.
//
// The function is read-only — it never writes or creates anything on disk.
// It never attempts to sanitize or fix an invalid path; it rejects and reports.
//
// Error messages:
//   - "path is empty"
//   - "path is absolute: <path>"
//   - "path contains directory traversal: <path>"
//   - "path resolves outside project root: <path>"
func ValidatePath(path string, projectRoot string) error {
	// Step 1: Reject empty paths.
	// An empty string cannot safely identify any file.
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Step 2: Reject absolute paths.
	//
	// Use strings.HasPrefix to catch Unix-style absolute paths that begin
	// with "/". This matters on Windows too: filepath.IsAbs returns false
	// for "/foo" (no drive letter), but the path is still absolute and
	// dangerous when passed to OS APIs on some Windows versions.
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("path is absolute: %s", path)
	}

	// Reject any path containing ":" — this catches Windows drive letters
	// such as "C:\Users\..." or "C:/Users/...". A colon has no legitimate
	// use in a safe relative file path accepted by this tool.
	if strings.Contains(path, ":") {
		return fmt.Errorf("path is absolute: %s", path)
	}

	// Step 3: Normalize the path using filepath.Clean.
	// This resolves "." components, collapses duplicate separators, and
	// converts OS-specific separators into canonical form. We perform all
	// subsequent checks on the cleaned path.
	cleaned := filepath.Clean(path)

	// Step 4: Reject if any component is ".." after cleaning.
	// filepath.Clean cannot remove ".." components that travel above the
	// current directory (e.g. "../../foo" stays as "../../foo"). We split
	// on the OS path separator to inspect every individual component.
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if component == ".." {
			return fmt.Errorf("path contains directory traversal: %s", path)
		}
	}

	// Step 5: Build the full absolute path by joining the project root
	// with the cleaned relative path. This is the candidate write target.
	abs := filepath.Join(projectRoot, cleaned)

	// Step 6: Resolve symlinks on the absolute path.
	// A relative path that appears contained within the project root may
	// still escape it if a directory component is a symlink pointing
	// outside. We use evalSymlinksLongestPrefix so that paths that do
	// not exist yet (future files) are still validated correctly.
	resolved, err := evalSymlinksLongestPrefix(abs)
	if err != nil {
		// If we cannot resolve even the existing prefix, treat it as
		// escaping the project root to be safe.
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	// Step 7: Verify containment.
	// Resolve symlinks on projectRoot itself so we compare canonical forms
	// on both sides.
	resolvedRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	// Append the OS separator before the HasPrefix check so that a project
	// root of "/project" does not accidentally accept "/projectExtra/file"
	// as a valid prefix match.
	rootWithSep := resolvedRoot + string(filepath.Separator)
	if resolved != resolvedRoot && !strings.HasPrefix(resolved+string(filepath.Separator), rootWithSep) {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	return nil
}

// evalSymlinksLongestPrefix resolves symlinks on an absolute path.
//
// filepath.EvalSymlinks requires every component of the path to exist on
// disk, which means it fails for paths that will be created in the future.
// This function handles that case by walking the path from the volume root
// downward, resolving symlinks on the longest existing prefix it can find,
// then appending the remaining non-existent components as-is.
//
// This approach guarantees that any symlink present in existing
// parent directories is still resolved and checked for containment,
// even when the final file has not been created yet.
func evalSymlinksLongestPrefix(abs string) (string, error) {
	// Fast path: the full path already exists; resolve it directly.
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}

	// The full path does not exist (yet). We must find the longest
	// existing prefix to resolve symlinks there, then reattach the
	// non-existent suffix.

	// filepath.VolumeName returns "" on Unix and "C:" (etc.) on Windows.
	vol := filepath.VolumeName(abs)

	// Strip the volume prefix to get the separator-rooted path remainder.
	rest := abs[len(vol):]

	// Split into individual path components. On Unix the leading "/"
	// produces an empty string as the first element; on Windows the rest
	// starts with "\" which similarly produces an empty first element.
	parts := strings.Split(rest, string(filepath.Separator))

	// current tracks the canonical path we have built so far.
	// We start at the volume root (e.g. "" on Unix, "C:" on Windows).
	current := vol

	for i, part := range parts {
		if part == "" {
			// Handle the leading (or any redundant) separator.
			// On the first pass this anchors current to the FS root.
			if current == vol {
				current = vol + string(filepath.Separator)
			}
			continue
		}

		// Build the candidate path for this component.
		candidate := filepath.Join(current, part)

		// Check whether this component exists without following symlinks
		// (os.Lstat so we detect the symlink itself, not its target).
		_, statErr := os.Lstat(candidate)
		if statErr != nil {
			// This component does not exist. Collect it and every
			// remaining component as the non-existent suffix.
			var tail []string
			for j := i; j < len(parts); j++ {
				if parts[j] != "" {
					tail = append(tail, parts[j])
				}
			}

			// Resolve symlinks on the existing prefix built so far.
			resolvedPrefix, evalErr := filepath.EvalSymlinks(current)
			if evalErr != nil {
				return "", evalErr
			}

			// Reattach the non-existent suffix.
			if len(tail) > 0 {
				return filepath.Join(resolvedPrefix, filepath.Join(tail...)), nil
			}
			return resolvedPrefix, nil
		}

		// The component exists — resolve symlinks through it and continue
		// building the canonical absolute path one component at a time.
		resolvedCandidate, evalErr := filepath.EvalSymlinks(candidate)
		if evalErr != nil {
			return "", evalErr
		}
		current = resolvedCandidate
	}

	// All components existed. The fast-path EvalSymlinks at the top should
	// have succeeded in this case, but we handle it gracefully anyway.
	return current, nil
}
