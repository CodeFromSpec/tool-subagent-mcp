// Package logicalnames centralizes conversion between logical names and file paths.
// code-from-spec: ROOT/tech_design/internal/logical_names@v28
package logicalnames

import (
	"path/filepath"
	"strings"
)

// PathFromLogicalName resolves a logical name to a file path relative to the
// project root. Returns ("", false) if the input does not match any known pattern.
// Returned paths always use forward slashes as separators (filepath.ToSlash applied).
//
// If the logical name has a parenthetical qualifier it is stripped before resolving,
// so ROOT/x(y) resolves to the same path as ROOT/x.
//
// Rules:
//   - ROOT                      → code-from-spec/spec/_node.md
//   - ROOT/<path>               → code-from-spec/spec/<path>/_node.md
//   - ROOT/<path>(<qualifier>)  → code-from-spec/spec/<path>/_node.md
//   - TEST                      → code-from-spec/spec/default.test.md
//   - TEST/<path>               → code-from-spec/spec/<path>/default.test.md
//   - TEST/<path>(<name>)       → code-from-spec/spec/<path>/<name>.test.md
//   - EXTERNAL/<name>           → code-from-spec/external/<name>/_external.md
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
		// Strip parenthetical qualifier if present: ROOT/<path>(<qualifier>) → ROOT/<path>
		if path, _, hasQ := parseQualifier(rest); hasQ {
			rest = path
		}
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
		if path, name, hasName := parseQualifier(rest); hasName {
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
//   - ROOT                             → (false, true)  — root has no parent
//   - ROOT/<path>                      → (true,  true)  — always has a parent
//   - ROOT/<path>(<qualifier>)         → (true,  true)  — qualifier does not affect result
//   - TEST (any form)                  → (true,  true)  — parent is always in ROOT namespace
//   - EXTERNAL/<name>                  → (false, true)  — externals have no parent
//   - EXTERNAL (no name) or ""         → (false, false) — not a valid logical name
//   - anything else                    → (false, false) — not a valid logical name
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

	// EXTERNAL namespace — must have a name after the prefix; no parent.
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
// The qualifier is stripped before deriving the parent, so ROOT/x/y(z) → ROOT/x.
//
// Rules:
//   - ROOT/<path>             → strip last segment; if one segment remains, parent is ROOT
//   - ROOT/<path>(<qualifier>) → strip qualifier first, then strip last segment
//   - TEST                    → ROOT
//   - TEST/<path>             → ROOT/<path>
//   - TEST/<path>(<name>)     → ROOT/<path>
func ParentLogicalName(logicalName string) (string, bool) {
	// ROOT namespace.
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		// Strip parenthetical qualifier if present: ROOT/<path>(<qualifier>)
		if path, _, hasQ := parseQualifier(rest); hasQ {
			rest = path
		}
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
		if path, _, hasName := parseQualifier(rest); hasName {
			return "ROOT/" + path, true
		}
		// TEST/<path> → ROOT/<path>
		return "ROOT/" + rest, true
	}

	// ROOT itself, EXTERNAL, or anything else — no parent.
	return "", false
}

// HasQualifier determines whether a logical name has a parenthetical qualifier.
// Returns (hasQualifier, ok) where ok indicates whether the input is a valid logical name.
//
// Rules (valid logical names only):
//   - ROOT/x(y)     → (true, true)
//   - ROOT/x        → (false, true)
//   - ROOT          → (false, true)
//   - TEST/x(name)  → (true, true)
//   - TEST/x        → (false, true)
//   - TEST          → (false, true)
//   - EXTERNAL/x    → (false, true)
//   - ""            → (false, false)
func HasQualifier(logicalName string) (hasQualifier, ok bool) {
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
		_, _, hasQ := parseQualifier(rest)
		return hasQ, true
	}

	// TEST namespace.
	if logicalName == "TEST" {
		return false, true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return false, false
		}
		_, _, hasQ := parseQualifier(rest)
		return hasQ, true
	}

	// EXTERNAL namespace — qualifiers not supported; EXTERNAL alone is invalid.
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

// QualifierName extracts the qualifier from a logical name.
// Returns (qualifier, true) on success, ("", false) if there is no qualifier.
// Only call after confirming HasQualifier returns true.
//
// Examples:
//   - ROOT/x(y)       → "y",    true
//   - ROOT/x/y(z)     → "z",    true
//   - TEST/x(name)    → "name", true
//   - ROOT/x          → "",     false
//   - ROOT            → "",     false
func QualifierName(logicalName string) (string, bool) {
	if logicalName == "" {
		return "", false
	}

	// ROOT namespace.
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		_, qualifier, hasQ := parseQualifier(rest)
		if hasQ {
			return qualifier, true
		}
		return "", false
	}

	// TEST namespace.
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}
		_, qualifier, hasQ := parseQualifier(rest)
		if hasQ {
			return qualifier, true
		}
		return "", false
	}

	// ROOT (bare), TEST (bare), EXTERNAL, or unknown — no qualifier.
	return "", false
}

// parseQualifier checks if s ends with a parenthesized qualifier like "x/y(qualifier)".
// Returns the path part, the qualifier content, and whether a qualifier was found.
// Both path and qualifier must be non-empty for a valid match.
func parseQualifier(s string) (path, qualifier string, ok bool) {
	// Must end with ')'.
	if !strings.HasSuffix(s, ")") {
		return "", "", false
	}

	// Find the matching opening '('.
	openIdx := strings.LastIndex(s, "(")
	if openIdx <= 0 {
		// openIdx == -1 means no '(' found; openIdx == 0 means no path before '('.
		return "", "", false
	}

	path = s[:openIdx]
	qualifier = s[openIdx+1 : len(s)-1]

	// Both parts must be non-empty.
	if path == "" || qualifier == "" {
		return "", "", false
	}

	return path, qualifier, true
}
