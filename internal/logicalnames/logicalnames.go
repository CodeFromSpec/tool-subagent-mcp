// Package logicalnames centralizes conversion between logical names and file paths.
// spec: ROOT/tech_design/internal/logical_names@v24
package logicalnames

import "strings"

// PathFromLogicalName resolves a logical name to a file path relative to the project root.
// Returns ("", false) if the input does not match any known pattern.
//
// Supported patterns:
//   ROOT            → code-from-spec/spec/_node.md
//   ROOT/<path>     → code-from-spec/spec/<path>/_node.md
//   TEST            → code-from-spec/spec/default.test.md
//   TEST/<path>     → code-from-spec/spec/<path>/default.test.md
//   TEST/<path>(<name>) → code-from-spec/spec/<path>/<name>.test.md
//   EXTERNAL/<name> → code-from-spec/external/<name>/_external.md
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
		return "code-from-spec/spec/" + rest + "/_node.md", true
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
			return "code-from-spec/spec/" + path + "/" + name + ".test.md", true
		}
		return "code-from-spec/spec/" + rest + "/default.test.md", true
	}

	// Handle EXTERNAL namespace.
	if strings.HasPrefix(logicalName, "EXTERNAL/") {
		name := logicalName[len("EXTERNAL/"):]
		if name == "" {
			return "", false
		}
		// EXTERNAL only supports a single name segment (no further slashes
		// are shown in the spec, but the spec only defines EXTERNAL/<name>).
		return "code-from-spec/external/" + name + "/_external.md", true
	}

	// No known pattern matched.
	return "", false
}

// HasParent determines whether a logical name has a parent node.
// Returns (hasParent, ok) where ok indicates whether the input is a valid logical name.
//
//   ROOT              → (false, true)   — root has no parent
//   ROOT/<path>       → (true,  true)
//   TEST              → (true,  true)   — parent is ROOT
//   TEST/<path>       → (true,  true)
//   TEST/<path>(<n>)  → (true,  true)
//   EXTERNAL/<name>   → (false, true)   — externals have no parent
//   anything else     → (false, false)  — not a valid logical name
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

	// TEST namespace — always has a parent (in the ROOT namespace).
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

	// EXTERNAL namespace.
	if strings.HasPrefix(logicalName, "EXTERNAL/") {
		name := logicalName[len("EXTERNAL/"):]
		if name == "" {
			return false, false
		}
		return false, true
	}

	// Bare "EXTERNAL" or anything else is not a valid logical name.
	return false, false
}

// ParentLogicalName derives the parent's logical name from a node's logical name.
// Returns (parent, true) on success, ("", false) if the node has no parent.
// Only call after confirming HasParent returns true.
//
//   ROOT/<single>     → ROOT
//   ROOT/<a>/<b>/...  → ROOT/<a>/<b> (strip last segment)
//   TEST              → ROOT
//   TEST/<path>       → ROOT/<path>
//   TEST/<path>(<n>)  → ROOT/<path>
func ParentLogicalName(logicalName string) (string, bool) {
	// ROOT namespace children.
	if strings.HasPrefix(logicalName, "ROOT/") {
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		// Strip the last segment.
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash == -1 {
			// Only one segment: ROOT/<x> → ROOT.
			return "ROOT", true
		}
		return "ROOT/" + rest[:lastSlash], true
	}

	// TEST namespace — parent is always in the ROOT namespace.
	if logicalName == "TEST" {
		return "ROOT", true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}
		// Strip parenthesized name if present: TEST/<path>(<name>) → ROOT/<path>.
		path, _, hasName := parseTestName(rest)
		if hasName {
			return "ROOT/" + path, true
		}
		// TEST/<path> → ROOT/<path>.
		return "ROOT/" + rest, true
	}

	// ROOT itself, EXTERNAL/*, and invalid names have no parent.
	return "", false
}

// parseTestName checks whether s (the part after "TEST/") ends with a
// parenthesized name like "x/y(name)". If so, it returns the path ("x/y"),
// the name ("name"), and true. Otherwise it returns ("", "", false).
func parseTestName(s string) (path, name string, ok bool) {
	// The parenthesized name must be at the very end: ...(<name>)
	if !strings.HasSuffix(s, ")") {
		return "", "", false
	}
	openParen := strings.LastIndex(s, "(")
	if openParen == -1 {
		return "", "", false
	}
	// Extract path and name.
	path = s[:openParen]
	name = s[openParen+1 : len(s)-1]
	if path == "" || name == "" {
		return "", "", false
	}
	return path, name, true
}
