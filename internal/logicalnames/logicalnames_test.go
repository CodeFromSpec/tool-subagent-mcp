// spec: TEST/tech_design/internal/logical_names@v13
package logicalnames

import "testing"

// ---------------------------------------------------------------------------
// PathFromLogicalName
// ---------------------------------------------------------------------------

func TestPathFromLogicalName_ROOT(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/_node.md")
	}
}

func TestPathFromLogicalName_ROOTWithPath(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/payments/processor")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/payments/processor/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/payments/processor/_node.md")
	}
}

func TestPathFromLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := PathFromLogicalName("TEST")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/default.test.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/default.test.md")
	}
}

func TestPathFromLogicalName_TESTCanonical(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/domain/config/default.test.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/domain/config/default.test.md")
	}
}

func TestPathFromLogicalName_TESTNamed(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/domain/config/edge_cases.test.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/domain/config/edge_cases.test.md")
	}
}

func TestPathFromLogicalName_EXTERNAL(t *testing.T) {
	got, ok := PathFromLogicalName("EXTERNAL/codefromspec")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/external/codefromspec/_external.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/external/codefromspec/_external.md")
	}
}

func TestPathFromLogicalName_UnrecognizedPrefix(t *testing.T) {
	got, ok := PathFromLogicalName("UNKNOWN/something")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

func TestPathFromLogicalName_EmptyString(t *testing.T) {
	got, ok := PathFromLogicalName("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

func TestPathFromLogicalName_EXTERNALWithoutName(t *testing.T) {
	got, ok := PathFromLogicalName("EXTERNAL")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

// ---------------------------------------------------------------------------
// HasParent
// ---------------------------------------------------------------------------

func TestHasParent_ROOT(t *testing.T) {
	hasParent, ok := HasParent("ROOT")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

func TestHasParent_ROOTWithPath(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

func TestHasParent_TESTWithoutPath(t *testing.T) {
	hasParent, ok := HasParent("TEST")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

func TestHasParent_TESTWithPath(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

func TestHasParent_TESTNamed(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

func TestHasParent_EXTERNAL(t *testing.T) {
	hasParent, ok := HasParent("EXTERNAL/codefromspec")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

func TestHasParent_EXTERNALWithoutName(t *testing.T) {
	hasParent, ok := HasParent("EXTERNAL")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

func TestHasParent_EmptyString(t *testing.T) {
	hasParent, ok := HasParent("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

func TestHasParent_UnrecognizedPrefix(t *testing.T) {
	hasParent, ok := HasParent("UNKNOWN/something")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

// ---------------------------------------------------------------------------
// ParentLogicalName
// ---------------------------------------------------------------------------

func TestParentLogicalName_ROOTx(t *testing.T) {
	// ROOT/x — parent is ROOT
	got, ok := ParentLogicalName("ROOT/domain")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT" {
		t.Fatalf("got %q, want %q", got, "ROOT")
	}
}

func TestParentLogicalName_ROOTxy(t *testing.T) {
	// ROOT/x/y — parent is ROOT/x
	got, ok := ParentLogicalName("ROOT/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain")
	}
}

func TestParentLogicalName_ROOTxyz(t *testing.T) {
	// ROOT/x/y/z — parent is ROOT/x/y
	got, ok := ParentLogicalName("ROOT/tech_design/logical_names")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/tech_design" {
		t.Fatalf("got %q, want %q", got, "ROOT/tech_design")
	}
}

func TestParentLogicalName_TESTWithoutPath(t *testing.T) {
	// TEST — parent is ROOT
	got, ok := ParentLogicalName("TEST")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT" {
		t.Fatalf("got %q, want %q", got, "ROOT")
	}
}

func TestParentLogicalName_TESTWithPath(t *testing.T) {
	// TEST/x — parent is ROOT/x
	got, ok := ParentLogicalName("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain/config" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain/config")
	}
}

func TestParentLogicalName_TESTNamed(t *testing.T) {
	// TEST/x(name) — parent is ROOT/x
	got, ok := ParentLogicalName("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain/config" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain/config")
	}
}

func TestParentLogicalName_ROOTHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("ROOT")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

func TestParentLogicalName_EXTERNALHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("EXTERNAL/codefromspec")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

func TestParentLogicalName_InvalidInput(t *testing.T) {
	got, ok := ParentLogicalName("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}
