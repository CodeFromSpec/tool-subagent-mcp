// spec: TEST/tech_design/internal/logical_names@v12

package logicalnames

import "testing"

// --- PathFromLogicalName tests ---

func TestPathFromLogicalName_ROOT(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT")
	if !ok || got != "code-from-spec/spec/_node.md" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, true)", "ROOT", got, ok, "code-from-spec/spec/_node.md")
	}
}

func TestPathFromLogicalName_ROOTWithPath(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/payments/processor")
	if !ok || got != "code-from-spec/spec/payments/processor/_node.md" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, true)", "ROOT/payments/processor", got, ok, "code-from-spec/spec/payments/processor/_node.md")
	}
}

func TestPathFromLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := PathFromLogicalName("TEST")
	if !ok || got != "code-from-spec/spec/default.test.md" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, true)", "TEST", got, ok, "code-from-spec/spec/default.test.md")
	}
}

func TestPathFromLogicalName_TESTCanonical(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config")
	if !ok || got != "code-from-spec/spec/domain/config/default.test.md" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, true)", "TEST/domain/config", got, ok, "code-from-spec/spec/domain/config/default.test.md")
	}
}

func TestPathFromLogicalName_TESTNamed(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config(edge_cases)")
	if !ok || got != "code-from-spec/spec/domain/config/edge_cases.test.md" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, true)", "TEST/domain/config(edge_cases)", got, ok, "code-from-spec/spec/domain/config/edge_cases.test.md")
	}
}

func TestPathFromLogicalName_EXTERNAL(t *testing.T) {
	got, ok := PathFromLogicalName("EXTERNAL/codefromspec")
	if !ok || got != "code-from-spec/external/codefromspec/_external.md" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, true)", "EXTERNAL/codefromspec", got, ok, "code-from-spec/external/codefromspec/_external.md")
	}
}

func TestPathFromLogicalName_UnrecognizedPrefix(t *testing.T) {
	got, ok := PathFromLogicalName("UNKNOWN/something")
	if ok || got != "" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, false)", "UNKNOWN/something", got, ok, "")
	}
}

func TestPathFromLogicalName_EmptyString(t *testing.T) {
	got, ok := PathFromLogicalName("")
	if ok || got != "" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, false)", "", got, ok, "")
	}
}

func TestPathFromLogicalName_EXTERNALWithoutName(t *testing.T) {
	got, ok := PathFromLogicalName("EXTERNAL")
	if ok || got != "" {
		t.Errorf("PathFromLogicalName(%q) = (%q, %v), want (%q, false)", "EXTERNAL", got, ok, "")
	}
}

// --- HasParent tests ---

func TestHasParent_ROOT(t *testing.T) {
	hasParent, ok := HasParent("ROOT")
	if hasParent != false || ok != true {
		t.Errorf("HasParent(%q) = (%v, %v), want (false, true)", "ROOT", hasParent, ok)
	}
}

func TestHasParent_ROOTWithPath(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config")
	if hasParent != true || ok != true {
		t.Errorf("HasParent(%q) = (%v, %v), want (true, true)", "ROOT/domain/config", hasParent, ok)
	}
}

func TestHasParent_TESTWithoutPath(t *testing.T) {
	hasParent, ok := HasParent("TEST")
	if hasParent != true || ok != true {
		t.Errorf("HasParent(%q) = (%v, %v), want (true, true)", "TEST", hasParent, ok)
	}
}

func TestHasParent_TESTWithPath(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config")
	if hasParent != true || ok != true {
		t.Errorf("HasParent(%q) = (%v, %v), want (true, true)", "TEST/domain/config", hasParent, ok)
	}
}

func TestHasParent_TESTNamed(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config(edge_cases)")
	if hasParent != true || ok != true {
		t.Errorf("HasParent(%q) = (%v, %v), want (true, true)", "TEST/domain/config(edge_cases)", hasParent, ok)
	}
}

func TestHasParent_EXTERNAL(t *testing.T) {
	hasParent, ok := HasParent("EXTERNAL/codefromspec")
	if hasParent != false || ok != true {
		t.Errorf("HasParent(%q) = (%v, %v), want (false, true)", "EXTERNAL/codefromspec", hasParent, ok)
	}
}

func TestHasParent_EXTERNALWithoutName(t *testing.T) {
	hasParent, ok := HasParent("EXTERNAL")
	if hasParent != false || ok != false {
		t.Errorf("HasParent(%q) = (%v, %v), want (false, false)", "EXTERNAL", hasParent, ok)
	}
}

func TestHasParent_EmptyString(t *testing.T) {
	hasParent, ok := HasParent("")
	if hasParent != false || ok != false {
		t.Errorf("HasParent(%q) = (%v, %v), want (false, false)", "", hasParent, ok)
	}
}

func TestHasParent_UnrecognizedPrefix(t *testing.T) {
	hasParent, ok := HasParent("UNKNOWN/something")
	if hasParent != false || ok != false {
		t.Errorf("HasParent(%q) = (%v, %v), want (false, false)", "UNKNOWN/something", hasParent, ok)
	}
}

// --- ParentLogicalName tests ---

func TestParentLogicalName_ROOTx_ParentIsROOT(t *testing.T) {
	// ROOT/x -> parent is ROOT
	got, ok := ParentLogicalName("ROOT/domain")
	if !ok || got != "ROOT" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, true)", "ROOT/domain", got, ok, "ROOT")
	}
}

func TestParentLogicalName_ROOTxy_ParentIsROOTx(t *testing.T) {
	// ROOT/x/y -> parent is ROOT/x
	got, ok := ParentLogicalName("ROOT/domain/config")
	if !ok || got != "ROOT/domain" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, true)", "ROOT/domain/config", got, ok, "ROOT/domain")
	}
}

func TestParentLogicalName_ROOTxyz_ParentIsROOTxy(t *testing.T) {
	// ROOT/x/y/z -> parent is ROOT/x/y
	got, ok := ParentLogicalName("ROOT/tech_design/logical_names")
	if !ok || got != "ROOT/tech_design" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, true)", "ROOT/tech_design/logical_names", got, ok, "ROOT/tech_design")
	}
}

func TestParentLogicalName_TESTWithoutPath_ParentIsROOT(t *testing.T) {
	// TEST -> parent is ROOT
	got, ok := ParentLogicalName("TEST")
	if !ok || got != "ROOT" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, true)", "TEST", got, ok, "ROOT")
	}
}

func TestParentLogicalName_TESTx_ParentIsROOTx(t *testing.T) {
	// TEST/x -> parent is ROOT/x
	got, ok := ParentLogicalName("TEST/domain/config")
	if !ok || got != "ROOT/domain/config" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, true)", "TEST/domain/config", got, ok, "ROOT/domain/config")
	}
}

func TestParentLogicalName_TESTxName_ParentIsROOTx(t *testing.T) {
	// TEST/x(name) -> parent is ROOT/x
	got, ok := ParentLogicalName("TEST/domain/config(edge_cases)")
	if !ok || got != "ROOT/domain/config" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, true)", "TEST/domain/config(edge_cases)", got, ok, "ROOT/domain/config")
	}
}

func TestParentLogicalName_ROOTHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("ROOT")
	if ok || got != "" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, false)", "ROOT", got, ok, "")
	}
}

func TestParentLogicalName_EXTERNALHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("EXTERNAL/codefromspec")
	if ok || got != "" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, false)", "EXTERNAL/codefromspec", got, ok, "")
	}
}

func TestParentLogicalName_InvalidInput(t *testing.T) {
	got, ok := ParentLogicalName("")
	if ok || got != "" {
		t.Errorf("ParentLogicalName(%q) = (%q, %v), want (%q, false)", "", got, ok, "")
	}
}
