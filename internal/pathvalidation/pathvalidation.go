// Package pathvalidation validates that file paths are safe to write
// within a project directory. This is a security-critical package.
//
// spec: ROOT/tech_design/internal/pathvalidation@v10

package pathvalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks that path is safe to write within projectRoot.
// Returns nil if safe, or an error describing the violation.
//
// The function is read-only — it never writes or creates anything on disk.
// It never attempts to sanitize or fix an invalid path; it rejects and reports.
func ValidatePath(path string, projectRoot string) error {
	// Step 1: Reject empty paths.
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Step 2: Reject absolute paths.
	// Use strings.HasPrefix for "/" to catch Unix-style absolute paths,
	// including on Windows where filepath.IsAbs returns false for paths
	// starting with "/" without a drive letter.
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("path is absolute: %s", path)
	}
	// Reject paths containing ":" which indicates a Windows drive letter (e.g. C:\...).
	if strings.Contains(path, ":") {
		return fmt.Errorf("path is absolute: %s", path)
	}

	// Step 3: Normalize the path — resolve separators and "." segments.
	cleaned := filepath.Clean(path)

	// Step 4: Reject if any component is ".." after cleaning.
	// After filepath.Clean, ".." components appear literally if they
	// cannot be resolved (e.g. "../foo" cleans to "../foo").
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if component == ".." {
			return fmt.Errorf("path contains directory traversal: %s", path)
		}
	}

	// Step 5: Resolve the full absolute path by joining with projectRoot.
	abs := filepath.Join(projectRoot, cleaned)

	// Step 6: Resolve symlinks. If the target does not exist yet,
	// evaluate the longest existing prefix of the path.
	resolved, err := evalSymlinksLongestPrefix(abs)
	if err != nil {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	// Step 7: Verify containment — the resolved path must start with projectRoot.
	// Resolve symlinks on projectRoot itself so both sides are canonical.
	resolvedRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	// Ensure the root ends with a separator so that "/project" does not
	// match "/projectX/file".
	rootPrefix := resolvedRoot + string(filepath.Separator)
	if !strings.HasPrefix(resolved+string(filepath.Separator), rootPrefix) && resolved != resolvedRoot {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	return nil
}

// evalSymlinksLongestPrefix resolves symlinks on the given absolute path.
// If the full path does not exist, it walks from the root of the path
// downward, resolving symlinks on the longest existing prefix, then
// appends the remaining unresolved components.
func evalSymlinksLongestPrefix(abs string) (string, error) {
	// Try the full path first — fast path when the target exists.
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}

	// The full path doesn't exist. Walk from the root to find the
	// longest existing prefix.
	//
	// Split the path into components and progressively extend,
	// resolving symlinks at each existing level.

	// On Windows, filepath.VolumeName gives us "C:" etc.
	vol := filepath.VolumeName(abs)
	// Get the path without the volume name.
	rest := abs[len(vol):]

	// Split into components (skip empty strings from leading separator).
	parts := strings.Split(rest, string(filepath.Separator))

	// Build up the resolved prefix.
	current := vol
	remaining := 0

	for i, part := range parts {
		if part == "" {
			// Leading or trailing separator — just add separator.
			if current == vol {
				current += string(filepath.Separator)
			}
			continue
		}

		candidate := current + string(filepath.Separator) + part

		// Check if this level exists.
		_, statErr := os.Lstat(candidate)
		if statErr != nil {
			// This component doesn't exist. Everything from here onward
			// is the non-existent suffix.
			remaining = i
			// Collect remaining parts.
			var tail []string
			for j := i; j < len(parts); j++ {
				if parts[j] != "" {
					tail = append(tail, parts[j])
				}
			}
			// Resolve symlinks on the existing prefix (current).
			resolvedPrefix, evalErr := filepath.EvalSymlinks(current)
			if evalErr != nil {
				return "", evalErr
			}
			// Append the non-existent suffix to the resolved prefix.
			if len(tail) > 0 {
				return filepath.Join(resolvedPrefix, filepath.Join(tail...)), nil
			}
			return resolvedPrefix, nil
		}

		// Component exists — resolve symlinks at this level and continue.
		resolved, evalErr := filepath.EvalSymlinks(candidate)
		if evalErr != nil {
			return "", evalErr
		}
		current = resolved
	}

	_ = remaining
	// If we got here, all components exist (shouldn't happen since
	// the full EvalSymlinks failed above, but handle gracefully).
	return current, nil
}
