// spec: TEST/tech_design/internal/logical_names@v8

// Package logicalnames_test exercises the three pure functions
// exported by package logicalnames:
//   - PathFromLogicalName
//   - HasParent
//   - ParentLogicalName
//
// All tests are table-driven and rely solely on the standard
// "testing" package (no external test framework).
// No filesystem access is required — these are pure string
// transformations.
package logicalnames_test

import "testing"

// ---------------------------------------------------------------------------
// PathFromLogicalName
// ---------------------------------------------------------------------------

// TestPathFromLogicalName covers every rule from the spec:
//
//	ROOT                    → code-from-spec/spec/_node.md
//	ROOT/<path>             → code-from-spec/spec/<path>/_node.md
//	TEST                    → code-from-spec/spec/default.test.md
//	TEST/<path>             → code-from-spec/spec/<path>/default.test.md
//	TEST/<path>(<name>)     → code-from-spec/spec/<path>/<name>.test.md
//	EXTERNAL/<name>         → code-from-spec/external/<name>/_external.md
//	anything else           → ("", false)
func TestPathFromLogicalName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		wantPath  string
		wantOK    bool
	}{
		// --- ROOT ---
		{
			name:     "ROOT resolves to spec root node",
			input:    "ROOT",
			wantPath: "code-from-spec/spec/_node.md",
			wantOK:   true,
		},
		{
			name:     "ROOT with path resolves to nested node",
			input:    "ROOT/domain/modes",
			wantPath: "code-from-spec/spec/domain/modes/_node.md",
			wantOK:   true,
		},

		// --- TEST ---
		{
			name:     "TEST without path resolves to canonical default.test.md at spec root",
			input:    "TEST",
			wantPath: "code-from-spec/spec/default.test.md",
			wantOK:   true,
		},
		{
			name:     "TEST with path resolves to canonical default.test.md",
			input:    "TEST/domain/config",
			wantPath: "code-from-spec/spec/domain/config/default.test.md",
			wantOK:   true,
		},
		{
			name:     "TEST named variant resolves to named test file",
			input:    "TEST/domain/config(edge_cases)",
			wantPath: "code-from-spec/spec/domain/config/edge_cases.test.md",
			wantOK:   true,
		},

		// --- EXTERNAL ---
		{
			name:     "EXTERNAL with name resolves to _external.md",
			input:    "EXTERNAL/codefromspec",
			wantPath: "code-from-spec/external/codefromspec/_external.md",
			wantOK:   true,
		},

		// --- failure cases ---
		{
			name:    "unrecognized prefix returns false",
			input:   "UNKNOWN/something",
			wantOK:  false,
		},
		{
			name:    "empty string returns false",
			input:   "",
			wantOK:  false,
		},
		{
			name:    "EXTERNAL without name returns false",
			input:   "EXTERNAL",
			wantOK:  false,
		},
	}

	for _, tc := range cases {
		tc := tc // capture range variable for parallel sub-tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotPath, gotOK := PathFromLogicalName(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("PathFromLogicalName(%q) ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotPath != tc.wantPath {
				t.Errorf("PathFromLogicalName(%q) path = %q, want %q", tc.input, gotPath, tc.wantPath)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HasParent
// ---------------------------------------------------------------------------

// TestHasParent covers the full truth-table from the spec.
//
// hasParent semantics:
//   ROOT              → false, true   (valid, no parent)
//   ROOT/<path>       → true,  true   (valid, has parent)
//   TEST              → true,  true   (always has a ROOT parent)
//   TEST/<path>       → true,  true
//   TEST/<path>(name) → true,  true
//   EXTERNAL/<name>   → false, true   (valid, no parent)
//   EXTERNAL          → false, false  (invalid — must have a name)
//   ""                → false, false
//   other prefix      → false, false
func TestHasParent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		input         string
		wantHasParent bool
		wantOK        bool
	}{
		// --- ROOT ---
		{
			name:          "ROOT has no parent",
			input:         "ROOT",
			wantHasParent: false,
			wantOK:        true,
		},
		{
			name:          "ROOT with path has a parent",
			input:         "ROOT/domain/config",
			wantHasParent: true,
			wantOK:        true,
		},

		// --- TEST ---
		{
			name:          "TEST without path has parent (ROOT)",
			input:         "TEST",
			wantHasParent: true,
			wantOK:        true,
		},
		{
			name:          "TEST with path has parent",
			input:         "TEST/domain/config",
			wantHasParent: true,
			wantOK:        true,
		},
		{
			name:          "TEST named has parent",
			input:         "TEST/domain/config(edge_cases)",
			wantHasParent: true,
			wantOK:        true,
		},

		// --- EXTERNAL ---
		{
			name:          "EXTERNAL with name has no parent",
			input:         "EXTERNAL/codefromspec",
			wantHasParent: false,
			wantOK:        true,
		},

		// --- failure cases ---
		{
			name:          "EXTERNAL without name is invalid",
			input:         "EXTERNAL",
			wantHasParent: false,
			wantOK:        false,
		},
		{
			name:          "empty string is invalid",
			input:         "",
			wantHasParent: false,
			wantOK:        false,
		},
		{
			name:          "unrecognized prefix is invalid",
			input:         "UNKNOWN/something",
			wantHasParent: false,
			wantOK:        false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotHasParent, gotOK := HasParent(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("HasParent(%q) ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotHasParent != tc.wantHasParent {
				t.Errorf("HasParent(%q) hasParent = %v, want %v", tc.input, gotHasParent, tc.wantHasParent)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParentLogicalName
// ---------------------------------------------------------------------------

// TestParentLogicalName covers derivation rules from the spec.
//
// Derivation rules:
//   ROOT/<seg>         → ROOT  (single segment stripped → just ROOT)
//   ROOT/<x>/<y>...   → ROOT/<x>/<y-1>...  (strip last segment)
//   TEST               → ROOT
//   TEST/<path>        → ROOT/<path>
//   TEST/<path>(name)  → ROOT/<path>  (strip parenthesised name first)
//   ROOT               → ("", false)
//   EXTERNAL/<name>    → ("", false)
//   ""                 → ("", false)
func TestParentLogicalName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantParent string
		wantOK     bool
	}{
		// --- ROOT/<path> ---
		{
			name:       "ROOT/x parent is ROOT",
			input:      "ROOT/domain",
			wantParent: "ROOT",
			wantOK:     true,
		},
		{
			name:       "ROOT/x/y parent is ROOT/x",
			input:      "ROOT/domain/config",
			wantParent: "ROOT/domain",
			wantOK:     true,
		},
		{
			name:       "ROOT/x/y/z parent is ROOT/x/y",
			input:      "ROOT/tech_design/logical_names",
			wantParent: "ROOT/tech_design",
			wantOK:     true,
		},

		// --- TEST ---
		{
			name:       "TEST parent is ROOT",
			input:      "TEST",
			wantParent: "ROOT",
			wantOK:     true,
		},
		{
			name:       "TEST/x parent is ROOT/x",
			input:      "TEST/domain/config",
			wantParent: "ROOT/domain/config",
			wantOK:     true,
		},
		{
			name:       "TEST/x(name) parent is ROOT/x (name stripped)",
			input:      "TEST/domain/config(edge_cases)",
			wantParent: "ROOT/domain/config",
			wantOK:     true,
		},

		// --- no-parent cases ---
		{
			name:    "ROOT has no parent",
			input:   "ROOT",
			wantOK:  false,
		},
		{
			name:    "EXTERNAL has no parent",
			input:   "EXTERNAL/codefromspec",
			wantOK:  false,
		},
		{
			name:    "empty string has no parent",
			input:   "",
			wantOK:  false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotParent, gotOK := ParentLogicalName(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("ParentLogicalName(%q) ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotParent != tc.wantParent {
				t.Errorf("ParentLogicalName(%q) parent = %q, want %q", tc.input, gotParent, tc.wantParent)
			}
		})
	}
}
