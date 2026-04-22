// spec: TEST/tech_design/internal/logical_names@v12
package logicalnames_test

import (
	"testing"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
)

// ---------------------------------------------------------------------------
// PathFromLogicalName tests
// Spec ref: ROOT/tech_design/internal/logical_names § "PathFromLogicalName"
// and TEST/tech_design/internal/logical_names § "PathFromLogicalName"
// ---------------------------------------------------------------------------

func TestPathFromLogicalName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPath    string
		wantOK      bool
	}{
		// ROOT → code-from-spec/spec/_node.md
		{
			name:     "ROOT",
			input:    "ROOT",
			wantPath: "code-from-spec/spec/_node.md",
			wantOK:   true,
		},
		// ROOT with path → code-from-spec/spec/<path>/_node.md
		{
			name:     "ROOT with path",
			input:    "ROOT/domain/modes",
			wantPath: "code-from-spec/spec/domain/modes/_node.md",
			wantOK:   true,
		},
		// TEST without path → code-from-spec/spec/default.test.md
		{
			name:     "TEST without path",
			input:    "TEST",
			wantPath: "code-from-spec/spec/default.test.md",
			wantOK:   true,
		},
		// TEST canonical → code-from-spec/spec/<path>/default.test.md
		{
			name:     "TEST canonical",
			input:    "TEST/domain/config",
			wantPath: "code-from-spec/spec/domain/config/default.test.md",
			wantOK:   true,
		},
		// TEST named → code-from-spec/spec/<path>/<name>.test.md
		{
			name:     "TEST named",
			input:    "TEST/domain/config(edge_cases)",
			wantPath: "code-from-spec/spec/domain/config/edge_cases.test.md",
			wantOK:   true,
		},
		// EXTERNAL/<name> → code-from-spec/external/<name>/_external.md
		{
			name:     "EXTERNAL",
			input:    "EXTERNAL/codefromspec",
			wantPath: "code-from-spec/external/codefromspec/_external.md",
			wantOK:   true,
		},
		// Unrecognized prefix → ("", false)
		{
			name:     "Unrecognized prefix",
			input:    "UNKNOWN/something",
			wantPath: "",
			wantOK:   false,
		},
		// Empty string → ("", false)
		{
			name:     "Empty string",
			input:    "",
			wantPath: "",
			wantOK:   false,
		},
		// EXTERNAL without name → ("", false)
		{
			name:     "EXTERNAL without name",
			input:    "EXTERNAL",
			wantPath: "",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := logicalnames.PathFromLogicalName(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("PathFromLogicalName(%q): ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotPath != tc.wantPath {
				t.Errorf("PathFromLogicalName(%q): path = %q, want %q", tc.input, gotPath, tc.wantPath)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HasParent tests
// Spec ref: ROOT/tech_design/internal/logical_names § "HasParent"
// and TEST/tech_design/internal/logical_names § "HasParent"
// ---------------------------------------------------------------------------

func TestHasParent(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantHasParent bool
		wantOK        bool
	}{
		// ROOT → no parent, valid
		{
			name:          "ROOT",
			input:         "ROOT",
			wantHasParent: false,
			wantOK:        true,
		},
		// ROOT with path → has parent, valid
		{
			name:          "ROOT with path",
			input:         "ROOT/domain/config",
			wantHasParent: true,
			wantOK:        true,
		},
		// TEST without path → has parent (parent is ROOT), valid
		{
			name:          "TEST without path",
			input:         "TEST",
			wantHasParent: true,
			wantOK:        true,
		},
		// TEST with path → has parent, valid
		{
			name:          "TEST with path",
			input:         "TEST/domain/config",
			wantHasParent: true,
			wantOK:        true,
		},
		// TEST named → has parent, valid
		{
			name:          "TEST named",
			input:         "TEST/domain/config(edge_cases)",
			wantHasParent: true,
			wantOK:        true,
		},
		// EXTERNAL/<name> → no parent, valid
		{
			name:          "EXTERNAL",
			input:         "EXTERNAL/codefromspec",
			wantHasParent: false,
			wantOK:        true,
		},
		// EXTERNAL without name → not valid
		{
			name:          "EXTERNAL without name",
			input:         "EXTERNAL",
			wantHasParent: false,
			wantOK:        false,
		},
		// Empty string → not valid
		{
			name:          "Empty string",
			input:         "",
			wantHasParent: false,
			wantOK:        false,
		},
		// Unrecognized prefix → not valid
		{
			name:          "Unrecognized prefix",
			input:         "UNKNOWN/something",
			wantHasParent: false,
			wantOK:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotHasParent, gotOK := logicalnames.HasParent(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("HasParent(%q): ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotHasParent != tc.wantHasParent {
				t.Errorf("HasParent(%q): hasParent = %v, want %v", tc.input, gotHasParent, tc.wantHasParent)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParentLogicalName tests
// Spec ref: ROOT/tech_design/internal/logical_names § "ParentLogicalName"
// and TEST/tech_design/internal/logical_names § "ParentLogicalName"
// ---------------------------------------------------------------------------

func TestParentLogicalName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantParent string
		wantOK     bool
	}{
		// ROOT/x → ROOT
		{
			name:       "ROOT/x — parent is ROOT",
			input:      "ROOT/domain",
			wantParent: "ROOT",
			wantOK:     true,
		},
		// ROOT/x/y → ROOT/x
		{
			name:       "ROOT/x/y — parent is ROOT/x",
			input:      "ROOT/domain/config",
			wantParent: "ROOT/domain",
			wantOK:     true,
		},
		// ROOT/x/y/z → ROOT/x/y
		{
			name:       "ROOT/x/y/z — parent is ROOT/x/y",
			input:      "ROOT/tech_design/logical_names",
			wantParent: "ROOT/tech_design",
			wantOK:     true,
		},
		// TEST → ROOT
		{
			name:       "TEST without path — parent is ROOT",
			input:      "TEST",
			wantParent: "ROOT",
			wantOK:     true,
		},
		// TEST/x → ROOT/x
		{
			name:       "TEST/x — parent is ROOT/x",
			input:      "TEST/domain/config",
			wantParent: "ROOT/domain/config",
			wantOK:     true,
		},
		// TEST/x(name) → ROOT/x
		{
			name:       "TEST/x(name) — parent is ROOT/x",
			input:      "TEST/domain/config(edge_cases)",
			wantParent: "ROOT/domain/config",
			wantOK:     true,
		},
		// ROOT has no parent → ("", false)
		{
			name:       "ROOT has no parent",
			input:      "ROOT",
			wantParent: "",
			wantOK:     false,
		},
		// EXTERNAL has no parent → ("", false)
		{
			name:       "EXTERNAL has no parent",
			input:      "EXTERNAL/codefromspec",
			wantParent: "",
			wantOK:     false,
		},
		// Invalid input → ("", false)
		{
			name:       "Invalid input",
			input:      "",
			wantParent: "",
			wantOK:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotParent, gotOK := logicalnames.ParentLogicalName(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("ParentLogicalName(%q): ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotParent != tc.wantParent {
				t.Errorf("ParentLogicalName(%q): parent = %q, want %q", tc.input, gotParent, tc.wantParent)
			}
		})
	}
}
