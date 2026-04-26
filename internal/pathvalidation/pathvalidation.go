// code-from-spec: ROOT/tech_design/internal/pathvalidation@v11
//
// Package pathvalidation validates that file paths are safe to write
// within a project directory. This is a security-critical package.
//
// Threat model: prevents writing files outside the intended project
// boundary via relative traversal (../../etc/passwd), embedded traversal
// (internal/../../outside/file.go), OS-specific separators, encoding
// tricks, and symlinks. See OWASP path traversal guidance for details.

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
//
// Error messages:
//   - "path is empty"
//   - "path is absolute: <path>"
//   - "path contains directory traversal: <path>"
//   - "path resolves outside project root: <path>"
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
	// filepath.Clean handles duplicate separators, "." components, and
	// OS-specific quirks, giving us a canonical relative path to inspect.
	cleaned := filepath.Clean(path)

	// Step 4: Reject if any component is ".." after cleaning.
	// After filepath.Clean, ".." components appear literally if they
	// cannot be resolved (e.g. "../foo" cleans to "../foo").
	// Splitting on the OS separator covers both "/" and "\" on Windows.
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if component == ".." {
			return fmt.Errorf("path contains directory traversal: %s", path)
		}
	}

	// Step 5: Resolve the full absolute path by joining with projectRoot.
	abs := filepath.Join(projectRoot, cleaned)

	// Step 6: Resolve symlinks. If the target does not exist yet,
	// evaluate the longest existing prefix of the path.
	// This catches symlinks that redirect a path outside the project root.
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

	// Append a separator before comparing so that "/project" does not
	// accidentally match "/projectX/file" as a prefix.
	rootPrefix := resolvedRoot + string(filepath.Separator)
	if !strings.HasPrefix(resolved+string(filepath.Separator), rootPrefix) && resolved != resolvedRoot {
		return fmt.Errorf("path resolves outside project root: %s", path)
	}

	return nil
}

// evalSymlinksLongestPrefix resolves symlinks on the given absolute path.
// If the full path does not exist, it walks from the root of the path
// downward, resolving symlinks on the longest existing prefix, then
// appends the remaining unresolved (non-existent) components.
//
// This is necessary because filepath.EvalSymlinks fails if any component
// of the path does not exist, but we must still validate paths that will
// be created by the caller.
func evalSymlinksLongestPrefix(abs string) (string, error) {
	// Fast path: try the full path first (target already exists).
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}

	// The full path doesn't exist. Walk from the root to find the
	// longest existing prefix, then append the non-existent suffix.

	// On Windows, filepath.VolumeName gives us "C:" etc.
	vol := filepath.VolumeName(abs)
	// Strip the volume name to get the separator-rooted remainder.
	rest := abs[len(vol):]

	// Split into path components; leading separator produces an empty
	// first element which we use to anchor the root.
	parts := strings.Split(rest, string(filepath.Separator))

	// Start from the volume root.
	current := vol

	for i, part := range parts {
		if part == "" {
			// Leading separator or trailing empty element — attach the
			// separator to anchor the root on the first pass.
			if current == vol {
				current += string(filepath.Separator)
			}
			continue
		}

		candidate := current + string(filepath.Separator) + part

		// Check whether this component exists on disk.
		_, statErr := os.Lstat(candidate)
		if statErr != nil {
			// This component doesn't exist. Collect all remaining parts
			// (including this one) as the non-existent suffix.
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

			// Append the non-existent suffix to the resolved prefix.
			if len(tail) > 0 {
				return filepath.Join(resolvedPrefix, filepath.Join(tail...)), nil
			}
			return resolvedPrefix, nil
		}

		// This component exists — resolve symlinks at this level and
		// continue building the canonical path.
		resolved, evalErr := filepath.EvalSymlinks(candidate)
		if evalErr != nil {
			return "", evalErr
		}
		current = resolved
	}

	// All components exist (EvalSymlinks on the full path should have
	// succeeded above, but handle gracefully).
	return current, nil
}
