// Package logicalnames centralizes conversion between logical names and file paths.
// spec: ROOT/tech_design/internal/logical_names@v26
package logicalnames

import (
	"path/filepath"
	"strings"
)

// PathFromLogicalName resolves a logical name to a file path relative to the
// project root. Returns ("", false) if the input does not match any known pattern.
// Returned paths always use forward slashes as separators.
//
// Rules:
//   - ROOT                    → code-from-spec/spec/_node.md
//   - ROOT/<path>             → code-from-spec/spec/<path>/_node.md
//   - TEST                    → code-from-spec/spec/default.test.md
//   - TEST/<path>             → code-from-spec/spec/<path>/default.test.md
//   - TEST/<path>(<name>)     → code-from-spec/spec/<path>/<name>.test.md
//   - EXTERNAL/<name>         → code-from-spec/external/<name>/_external.md
func PathFromLogicalName(logicalName string) (string, bool) {
	if logicalName == "" {
		return "", false
	}

	// Handle ROOT namespace.
	if logicalName == "ROOT" {
		return "code-from-spec/spec/_node.md", true
	}
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		result := "code-from-spec/spec/" + rest + "/_node.md"
		return filepath.ToSlash(result), true
	}

	// Handle TEST namespace.
	if logicalName == "TEST" {
		return "code-from-spec/spec/default.test.md", true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}

		// Check for parenthesized name: TEST/<path>(<name>)
		path, name, hasName := parseTestName(rest)
		if hasName {
			result := "code-from-spec/spec/" + path + "/" + name + ".test.md"
			return filepath.ToSlash(result), true
		}

		// Default test: TEST/<path> → code-from-spec/spec/<path>/default.test.md
		result := "code-from-spec/spec/" + rest + "/default.test.md"
		return filepath.ToSlash(result), true
	}

	// Handle EXTERNAL namespace.
	if strings.HasPrefix(logicalName, "EXTERNAL/") {
		name := logicalName[len("EXTERNAL/"):]
		if name == "" {
			return "", false
		}
		// EXTERNAL only supports a single name segment (no further nesting
		// beyond what is given as <name>).
		result := "code-from-spec/external/" + name + "/_external.md"
		return filepath.ToSlash(result), true
	}

	// No known pattern matched.
	return "", false
}

// HasParent determines whether a logical name has a parent node.
// Returns (hasParent, ok) where ok indicates whether the input is a valid logical name.
//
// Rules:
//   - ROOT                → (false, true)  — root has no parent
//   - ROOT/<path>         → (true,  true)  — always has a parent
//   - TEST (any form)     → (true,  true)  — parent is always in ROOT namespace
//   - EXTERNAL/<name>     → (false, true)  — externals have no parent
//   - anything else       → (false, false) — not a valid logical name
func HasParent(logicalName string) (hasParent, ok bool) {
	if logicalName == "" {
		return false, false
	}

	// ROOT namespace.
	if logicalName == "ROOT" {
		return false, true
	}
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return false, false
		}
		return true, true
	}

	// TEST namespace — all valid TEST names have a parent.
	if logicalName == "TEST" {
		return true, true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return false, false
		}
		return true, true
	}

	// EXTERNAL namespace — must have a name after the prefix.
	if logicalName == "EXTERNAL" {
		return false, false
	}
	if strings.HasPrefix(logicalName, "EXTERNAL/") {
		name := logicalName[len("EXTERNAL/"):]
		if name == "" {
			return false, false
		}
		return false, true
	}

	// Unknown namespace.
	return false, false
}

// ParentLogicalName derives the parent's logical name from a node's logical name.
// Returns ("", false) if the node has no parent. Only call after confirming
// HasParent returns true.
//
// Rules:
//   - ROOT/<path>         → strip last segment; if one segment, parent is ROOT
//   - TEST                → ROOT
//   - TEST/<path>         → ROOT/<path>
//   - TEST/<path>(<name>) → ROOT/<path>
func ParentLogicalName(logicalName string) (string, bool) {
	// ROOT namespace.
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		// Strip the last path segment.
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash == -1 {
			// Only one segment — parent is ROOT.
			return "ROOT", true
		}
		return "ROOT/" + rest[:lastSlash], true
	}

	// TEST namespace.
	if logicalName == "TEST" {
		return "ROOT", true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}

		// Strip parenthesized name if present: TEST/<path>(<name>) → ROOT/<path>
		path, _, hasName := parseTestName(rest)
		if hasName {
			return "ROOT/" + path, true
		}

		// TEST/<path> → ROOT/<path>
		return "ROOT/" + rest, true
	}

	// ROOT itself, EXTERNAL, or anything else — no parent.
	return "", false
}

// parseTestName checks if s ends with a parenthesized name like "x/y(name)".
// Returns the path part, the name part, and whether a parenthesized name was found.
func parseTestName(s string) (path, name string, ok bool) {
	// Look for the closing paren at the end.
	if !strings.HasSuffix(s, ")") {
		return "", "", false
	}

	// Find the matching opening paren.
	openIdx := strings.LastIndex(s, "(")
	if openIdx == -1 || openIdx == 0 {
		return "", "", false
	}

	// Extract path and name.
	path = s[:openIdx]
	name = s[openIdx+1 : len(s)-1]

	// Both parts must be non-empty.
	if path == "" || name == "" {
		return "", "", false
	}

	return path, name, true
}
