// code-from-spec: ROOT/tech_design/internal/logical_names@v33

// Package logicalnames centralizes conversion between logical names and file paths.
//
// A logical name identifies a spec node in the tree. There are two namespaces:
//   - ROOT/... — spec nodes (_node.md files)
//   - TEST/... — test nodes (*.test.md files)
//
// A logical name may optionally contain a parenthetical qualifier at the end,
// e.g. ROOT/x/y(z). Qualifiers affect path resolution for TEST nodes (they
// select the named test file) but are stripped for ROOT nodes (they select a
// subsection of the same file).
package logicalnames

import (
	"path/filepath"
	"strings"
)

// PathFromLogicalName resolves a logical name to a file path relative to the
// project root. Returned paths always use forward slashes as separators,
// regardless of the operating system (filepath.ToSlash is applied).
//
// If the logical name has a parenthetical qualifier it is stripped before
// resolving for ROOT names, so ROOT/x(y) resolves to the same path as ROOT/x.
// For TEST names, the qualifier selects the named test file.
//
// Resolution rules:
//
//	ROOT                      → code-from-spec/_node.md
//	ROOT/<path>               → code-from-spec/<path>/_node.md
//	ROOT/<path>(<qualifier>)  → code-from-spec/<path>/_node.md
//	TEST                      → code-from-spec/default.test.md
//	TEST/<path>               → code-from-spec/<path>/default.test.md
//	TEST/<path>(<name>)       → code-from-spec/<path>/<name>.test.md
//
// Returns ("", false) if the input does not match any known pattern.
func PathFromLogicalName(logicalName string) (string, bool) {
	if logicalName == "" {
		return "", false
	}

	// ── ROOT namespace ────────────────────────────────────────────────────────

	// Bare ROOT → the root spec node.
	if logicalName == "ROOT" {
		return "code-from-spec/_node.md", true
	}

	if strings.HasPrefix(logicalName, "ROOT/") {
		// Everything after "ROOT/"
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			// "ROOT/" with nothing after is not a valid logical name.
			return "", false
		}

		// Strip parenthetical qualifier if present.
		// ROOT/x/y(z) → path="x/y", qualifier="z" → resolved as ROOT/x/y
		if path, _, hasQ := parseQualifier(rest); hasQ {
			rest = path
		}
		if rest == "" {
			return "", false
		}

		result := "code-from-spec/" + rest + "/_node.md"
		return filepath.ToSlash(result), true
	}

	// ── TEST namespace ────────────────────────────────────────────────────────

	// Bare TEST → the root-level default test file.
	if logicalName == "TEST" {
		return "code-from-spec/default.test.md", true
	}

	if strings.HasPrefix(logicalName, "TEST/") {
		// Everything after "TEST/"
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			// "TEST/" with nothing after is not a valid logical name.
			return "", false
		}

		// If there is a parenthesized name, it selects the named test file.
		// TEST/x/y(name) → code-from-spec/x/y/name.test.md
		if path, name, hasName := parseQualifier(rest); hasName {
			result := "code-from-spec/" + path + "/" + name + ".test.md"
			return filepath.ToSlash(result), true
		}

		// No qualifier → canonical (default) test file for that path.
		// TEST/x/y → code-from-spec/x/y/default.test.md
		result := "code-from-spec/" + rest + "/default.test.md"
		return filepath.ToSlash(result), true
	}

	// Unknown namespace — not a valid logical name.
	return "", false
}

// HasParent determines whether a logical name has a parent node.
// Returns (hasParent, ok) where ok indicates whether the input is a valid
// logical name at all.
//
// Rules:
//
//	ROOT                             → (false, true)  — root of the tree, no parent
//	ROOT/<path>                      → (true,  true)
//	ROOT/<path>(<qualifier>)         → (true,  true)  — qualifier does not affect this
//	TEST (any form)                  → (true,  true)  — parent is always in ROOT namespace
//	""                               → (false, false) — invalid input
//	anything else                    → (false, false) — unknown namespace
func HasParent(logicalName string) (hasParent, ok bool) {
	if logicalName == "" {
		return false, false
	}

	// ROOT namespace.
	if logicalName == "ROOT" {
		// Root of the tree — no parent.
		return false, true
	}
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			// "ROOT/" alone is malformed.
			return false, false
		}
		return true, true
	}

	// TEST namespace — every TEST node has a parent in the ROOT namespace.
	if logicalName == "TEST" {
		return true, true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			// "TEST/" alone is malformed.
			return false, false
		}
		return true, true
	}

	// Unknown namespace — not a valid logical name.
	return false, false
}

// ParentLogicalName derives the parent's logical name from a node's logical name.
// Returns ("", false) if the node has no parent (i.e. for ROOT itself).
// Only call after confirming HasParent returns true.
//
// The qualifier is stripped before deriving the parent, so ROOT/x/y(z) → ROOT/x.
//
// Rules:
//
//	ROOT/<seg>                → ROOT            (single segment → root)
//	ROOT/<seg1>/…/<segN>      → ROOT/<seg1>/…/<segN-1>
//	ROOT/<path>(<qualifier>)  → strip qualifier, then apply above rules
//	TEST                      → ROOT
//	TEST/<path>               → ROOT/<path>
//	TEST/<path>(<name>)       → ROOT/<path>     (name is stripped)
func ParentLogicalName(logicalName string) (string, bool) {
	// ROOT namespace.
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}

		// Strip parenthetical qualifier if present.
		if path, _, hasQ := parseQualifier(rest); hasQ {
			rest = path
		}
		if rest == "" {
			return "", false
		}

		// Strip the last path segment to get the parent path.
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash == -1 {
			// Only one segment — parent is the ROOT node itself.
			return "ROOT", true
		}
		return "ROOT/" + rest[:lastSlash], true
	}

	// TEST namespace — parent is always in the ROOT namespace.
	if logicalName == "TEST" {
		// TEST's parent is ROOT.
		return "ROOT", true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}

		// Strip parenthesized name if present.
		// TEST/x/y(name) → ROOT/x/y
		if path, _, hasName := parseQualifier(rest); hasName {
			return "ROOT/" + path, true
		}

		// TEST/<path> → ROOT/<path>
		return "ROOT/" + rest, true
	}

	// ROOT (bare) or unknown namespace — no parent can be derived.
	return "", false
}

// HasQualifier determines whether a logical name has a parenthetical qualifier.
// Returns (hasQualifier, ok) where ok indicates whether the input is a valid
// logical name.
//
// Examples:
//
//	ROOT        → (false, true)
//	ROOT/x      → (false, true)
//	ROOT/x(y)   → (true,  true)
//	TEST        → (false, true)
//	TEST/x      → (false, true)
//	TEST/x(n)   → (true,  true)
//	""          → (false, false)
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
			// "ROOT/" alone is malformed.
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
			// "TEST/" alone is malformed.
			return false, false
		}
		_, _, hasQ := parseQualifier(rest)
		return hasQ, true
	}

	// Unknown namespace — not a valid logical name.
	return false, false
}

// QualifierName extracts the qualifier from a logical name.
// Returns (qualifier, true) on success, ("", false) if there is no qualifier.
// Only call after confirming HasQualifier returns true.
//
// Examples:
//
//	ROOT/x(y)       → "y",    true
//	ROOT/x/y(z)     → "z",    true
//	TEST/x(name)    → "name", true
//	ROOT/x          → "",     false
//	ROOT            → "",     false
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

	// ROOT (bare), TEST (bare), or unknown namespace — no qualifier.
	return "", false
}

// ── internal helpers ──────────────────────────────────────────────────────────

// parseQualifier checks whether s ends with a parenthesized qualifier, e.g.
// "x/y(qualifier)". It returns the path part (before '('), the qualifier
// content (between '(' and ')'), and whether a valid qualifier was found.
//
// Both the path and the qualifier content must be non-empty for a valid match.
// The function looks for the LAST '(' so that paths with multiple segments are
// handled correctly: "a/b/c(q)" → path="a/b/c", qualifier="q".
func parseQualifier(s string) (path, qualifier string, ok bool) {
	// A qualifier always ends with ')'.
	if !strings.HasSuffix(s, ")") {
		return "", "", false
	}

	// Find the matching opening '(' — use the last occurrence so that
	// intermediate parentheses (not expected but guarded against) do not
	// confuse the parser.
	openIdx := strings.LastIndex(s, "(")
	if openIdx <= 0 {
		// openIdx == -1: no '(' found at all.
		// openIdx ==  0: '(' is at the very start — no path before it.
		return "", "", false
	}

	path = s[:openIdx]
	qualifier = s[openIdx+1 : len(s)-1] // content between '(' and ')'

	// Both parts must be non-empty to be considered a valid qualifier.
	if path == "" || qualifier == "" {
		return "", "", false
	}

	return path, qualifier, true
}
