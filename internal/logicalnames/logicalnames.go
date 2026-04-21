// spec: ROOT/tech_design/internal/logical_names@v21

// Package logicalnames centralizes conversion between logical names and
// file paths, and provides parent-derivation utilities for the Code from
// Spec node hierarchy.
//
// Logical name prefixes:
//   - ROOT      — spec nodes  (code-from-spec/spec/**/_node.md)
//   - TEST      — test nodes  (code-from-spec/spec/**/*.test.md)
//   - EXTERNAL  — external dependencies (code-from-spec/external/**/_external.md)
package logicalnames

import (
	"strings"
)

// PathFromLogicalName resolves a logical name to a file path relative to
// the project root.
//
// Resolution rules (ref: ROOT/tech_design/internal/logical_names §PathFromLogicalName):
//
//	ROOT              → code-from-spec/spec/_node.md
//	ROOT/<path>       → code-from-spec/spec/<path>/_node.md
//	TEST              → code-from-spec/spec/default.test.md
//	TEST/<path>       → code-from-spec/spec/<path>/default.test.md
//	TEST/<path>(name) → code-from-spec/spec/<path>/<name>.test.md
//	EXTERNAL/<name>   → code-from-spec/external/<name>/_external.md
//
// Returns ("", false) if the input does not match any known pattern.
func PathFromLogicalName(logicalName string) (string, bool) {
	switch {
	case logicalName == "ROOT":
		// The root spec node.
		return "code-from-spec/spec/_node.md", true

	case strings.HasPrefix(logicalName, "ROOT/"):
		// e.g. ROOT/x/y → code-from-spec/spec/x/y/_node.md
		rest := strings.TrimPrefix(logicalName, "ROOT/")
		if rest == "" {
			return "", false
		}
		return "code-from-spec/spec/" + rest + "/_node.md", true

	case logicalName == "TEST":
		// Canonical test node at the spec root (edge case — unlikely in practice
		// but consistent with the table in the spec).
		return "code-from-spec/spec/default.test.md", true

	case strings.HasPrefix(logicalName, "TEST/"):
		// e.g. TEST/x/y          → code-from-spec/spec/x/y/default.test.md
		//      TEST/x/y(name)    → code-from-spec/spec/x/y/name.test.md
		rest := strings.TrimPrefix(logicalName, "TEST/")
		if rest == "" {
			return "", false
		}
		return resolveTestPath(rest)

	case strings.HasPrefix(logicalName, "EXTERNAL/"):
		// e.g. EXTERNAL/celcoin-api → code-from-spec/external/celcoin-api/_external.md
		name := strings.TrimPrefix(logicalName, "EXTERNAL/")
		if name == "" || strings.Contains(name, "/") {
			// EXTERNAL only supports a single-level name (no further slashes).
			return "", false
		}
		return "code-from-spec/external/" + name + "/_external.md", true

	default:
		return "", false
	}
}

// resolveTestPath converts the portion of a TEST logical name that follows
// the "TEST/" prefix into a filesystem path.
//
// Input forms:
//
//	"x/y"        → "code-from-spec/spec/x/y/default.test.md"
//	"x/y(name)"  → "code-from-spec/spec/x/y/name.test.md"
func resolveTestPath(rest string) (string, bool) {
	// Check for an explicit test name in parentheses at the end.
	// e.g. "x/y(edge_cases)" — split on the last '('.
	if idx := strings.LastIndex(rest, "("); idx != -1 {
		// Must end with ')'.
		if !strings.HasSuffix(rest, ")") {
			return "", false
		}
		path := rest[:idx]        // everything before '('
		name := rest[idx+1 : len(rest)-1] // content between '(' and ')'
		if path == "" || name == "" {
			return "", false
		}
		return "code-from-spec/spec/" + path + "/" + name + ".test.md", true
	}

	// No explicit name — use the default test file name.
	return "code-from-spec/spec/" + rest + "/default.test.md", true
}

// HasParent determines whether a logical name has a parent node.
//
// Returns (hasParent, ok) where ok indicates whether the input is a
// valid logical name at all.
//
// Rules (ref: ROOT/tech_design/internal/logical_names §HasParent):
//
//	ROOT              → (false, true)   — root has no parent
//	ROOT/<path>       → (true,  true)
//	TEST              → (true,  true)   — parent is ROOT
//	TEST/<path>       → (true,  true)
//	TEST/<path>(name) → (true,  true)
//	EXTERNAL/<name>   → (false, true)   — externals have no parent
//	EXTERNAL          → (false, false)  — bare EXTERNAL is not valid
//	""                → (false, false)
func HasParent(logicalName string) (hasParent, ok bool) {
	switch {
	case logicalName == "":
		return false, false

	case logicalName == "ROOT":
		return false, true

	case strings.HasPrefix(logicalName, "ROOT/"):
		rest := strings.TrimPrefix(logicalName, "ROOT/")
		if rest == "" {
			return false, false
		}
		return true, true

	case logicalName == "TEST":
		// TEST without a path — its parent is ROOT.
		return true, true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := strings.TrimPrefix(logicalName, "TEST/")
		if rest == "" {
			return false, false
		}
		return true, true

	case logicalName == "EXTERNAL":
		// Bare "EXTERNAL" is not a valid logical name.
		return false, false

	case strings.HasPrefix(logicalName, "EXTERNAL/"):
		name := strings.TrimPrefix(logicalName, "EXTERNAL/")
		if name == "" || strings.Contains(name, "/") {
			return false, false
		}
		// External dependencies have no parent in the spec tree.
		return false, true

	default:
		return false, false
	}
}

// ParentLogicalName derives the parent's logical name from a node's
// logical name.
//
// Returns (parent, true) on success, ("", false) if the node has no
// parent. The caller should confirm HasParent returns (true, true)
// before calling this function.
//
// Rules (ref: ROOT/tech_design/internal/logical_names §ParentLogicalName):
//
//	ROOT/x          → ROOT
//	ROOT/x/y        → ROOT/x
//	TEST            → ROOT
//	TEST/x          → ROOT/x
//	TEST/x(name)    → ROOT/x
func ParentLogicalName(logicalName string) (string, bool) {
	switch {
	case logicalName == "TEST":
		// TEST with no path — parent is always the spec root.
		return "ROOT", true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := strings.TrimPrefix(logicalName, "TEST/")
		if rest == "" {
			return "", false
		}
		// Strip any trailing (name) suffix — the parent is always a ROOT node.
		if idx := strings.LastIndex(rest, "("); idx != -1 {
			rest = rest[:idx]
		}
		if rest == "" {
			return "", false
		}
		return "ROOT/" + rest, true

	case strings.HasPrefix(logicalName, "ROOT/"):
		rest := strings.TrimPrefix(logicalName, "ROOT/")
		if rest == "" {
			return "", false
		}
		// Strip the last path segment to find the parent.
		if idx := strings.LastIndex(rest, "/"); idx != -1 {
			// There is at least one more segment — parent is ROOT/<rest up to last />.
			return "ROOT/" + rest[:idx], true
		}
		// Only one segment left — parent is ROOT itself.
		return "ROOT", true

	default:
		// ROOT, EXTERNAL/*, and anything invalid have no parent.
		return "", false
	}
}
