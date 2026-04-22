// spec: ROOT/tech_design/internal/logical_names@v23

// Package logicalnames centralizes conversion between logical names and file
// paths, as well as parent derivation for the Code from Spec spec tree.
//
// Spec ref: ROOT/tech_design/internal/logical_names § "Intent"
package logicalnames

import (
	"strings"
)

// PathFromLogicalName resolves a logical name to a file path relative to the
// project root. Returns (path, true) on success or ("", false) if the input
// does not match any known pattern.
//
// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName"
//
// Resolution rules:
//   - ROOT                     → code-from-spec/spec/_node.md
//   - ROOT/<path>              → code-from-spec/spec/<path>/_node.md
//   - TEST                     → code-from-spec/spec/default.test.md
//   - TEST/<path>              → code-from-spec/spec/<path>/default.test.md
//   - TEST/<path>(<name>)      → code-from-spec/spec/<path>/<name>.test.md
//   - EXTERNAL/<name>          → code-from-spec/external/<name>/_external.md
func PathFromLogicalName(logicalName string) (string, bool) {
	switch {
	case logicalName == "ROOT":
		// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName" table row ROOT
		return "code-from-spec/spec/_node.md", true

	case strings.HasPrefix(logicalName, "ROOT/"):
		// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName" rule ROOT/<path>
		path := strings.TrimPrefix(logicalName, "ROOT/")
		if path == "" {
			return "", false
		}
		return "code-from-spec/spec/" + path + "/_node.md", true

	case logicalName == "TEST":
		// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName" table row TEST
		return "code-from-spec/spec/default.test.md", true

	case strings.HasPrefix(logicalName, "TEST/"):
		// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName" rules TEST/<path> and TEST/<path>(<name>)
		rest := strings.TrimPrefix(logicalName, "TEST/")
		if rest == "" {
			return "", false
		}
		// Check for named test variant: TEST/<path>(<name>)
		// The name is enclosed in parentheses at the end of the logical name.
		if parenStart := strings.LastIndex(rest, "("); parenStart != -1 {
			if strings.HasSuffix(rest, ")") {
				path := rest[:parenStart]
				name := rest[parenStart+1 : len(rest)-1]
				if path == "" || name == "" {
					return "", false
				}
				// Spec ref: TEST/<path>(<name>) → code-from-spec/spec/<path>/<name>.test.md
				return "code-from-spec/spec/" + path + "/" + name + ".test.md", true
			}
			// Malformed: has '(' but not a closing ')'
			return "", false
		}
		// Spec ref: TEST/<path> → code-from-spec/spec/<path>/default.test.md
		return "code-from-spec/spec/" + rest + "/default.test.md", true

	case strings.HasPrefix(logicalName, "EXTERNAL/"):
		// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName" rule EXTERNAL/<name>
		name := strings.TrimPrefix(logicalName, "EXTERNAL/")
		if name == "" || strings.Contains(name, "/") {
			// EXTERNAL only supports a single-level name
			return "", false
		}
		return "code-from-spec/external/" + name + "/_external.md", true

	default:
		return "", false
	}
}

// HasParent determines whether a logical name has a parent node.
// Returns (hasParent, ok) where ok indicates whether the input is a valid
// logical name.
//
// Spec ref: ROOT/tech_design/internal/logical_names § "HasParent"
//
// Rules:
//   - ROOT                         → (false, true)   — no parent
//   - ROOT/<path>                  → (true,  true)   — has parent
//   - TEST                         → (true,  true)   — parent is ROOT
//   - TEST/<path>                  → (true,  true)   — parent is ROOT/<path>
//   - TEST/<path>(<name>)          → (true,  true)   — parent is ROOT/<path>
//   - EXTERNAL/<name>              → (false, true)   — no parent
//   - EXTERNAL (bare)              → (false, false)  — invalid
//   - "" or anything else          → (false, false)  — invalid
func HasParent(logicalName string) (hasParent, ok bool) {
	switch {
	case logicalName == "ROOT":
		return false, true

	case strings.HasPrefix(logicalName, "ROOT/"):
		path := strings.TrimPrefix(logicalName, "ROOT/")
		if path == "" {
			return false, false
		}
		return true, true

	case logicalName == "TEST":
		// Spec ref: HasParent table — TEST → hasParent=true, ok=true
		return true, true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := strings.TrimPrefix(logicalName, "TEST/")
		if rest == "" {
			return false, false
		}
		// Validate: if parentheses are present they must be well-formed at the end.
		if parenStart := strings.LastIndex(rest, "("); parenStart != -1 {
			if !strings.HasSuffix(rest, ")") {
				return false, false
			}
			name := rest[parenStart+1 : len(rest)-1]
			path := rest[:parenStart]
			if path == "" || name == "" {
				return false, false
			}
		}
		return true, true

	case logicalName == "EXTERNAL":
		// Spec ref: HasParent table — EXTERNAL (bare) → ok=false
		return false, false

	case strings.HasPrefix(logicalName, "EXTERNAL/"):
		name := strings.TrimPrefix(logicalName, "EXTERNAL/")
		if name == "" || strings.Contains(name, "/") {
			return false, false
		}
		// Spec ref: HasParent table — EXTERNAL/<name> → hasParent=false, ok=true
		return false, true

	default:
		return false, false
	}
}

// ParentLogicalName derives the parent's logical name from a node's logical
// name. Returns (parent, true) on success, or ("", false) if the node has no
// parent. Only call after confirming HasParent returns true.
//
// Spec ref: ROOT/tech_design/internal/logical_names § "ParentLogicalName"
//
// Rules:
//   - ROOT/<path>         → strip last segment; if one segment remains, parent is ROOT
//   - TEST                → ROOT
//   - TEST/<path>         → ROOT/<path>
//   - TEST/<path>(<name>) → ROOT/<path>
func ParentLogicalName(logicalName string) (string, bool) {
	switch {
	case strings.HasPrefix(logicalName, "ROOT/"):
		// Spec ref: ParentLogicalName rule ROOT/<path>
		path := strings.TrimPrefix(logicalName, "ROOT/")
		if path == "" {
			return "", false
		}
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash == -1 {
			// Only one segment: parent is ROOT
			return "ROOT", true
		}
		return "ROOT/" + path[:lastSlash], true

	case logicalName == "TEST":
		// Spec ref: ParentLogicalName table — TEST → ROOT
		return "ROOT", true

	case strings.HasPrefix(logicalName, "TEST/"):
		// Spec ref: ParentLogicalName rules TEST/<path> and TEST/<path>(<name>)
		rest := strings.TrimPrefix(logicalName, "TEST/")
		if rest == "" {
			return "", false
		}
		// Strip named variant suffix (<name>) if present.
		if parenStart := strings.LastIndex(rest, "("); parenStart != -1 {
			if strings.HasSuffix(rest, ")") {
				rest = rest[:parenStart]
			} else {
				return "", false
			}
		}
		if rest == "" {
			return "", false
		}
		// TEST/<path> → ROOT/<path>
		return "ROOT/" + rest, true

	default:
		// No parent (ROOT, EXTERNAL/*, or invalid).
		return "", false
	}
}
