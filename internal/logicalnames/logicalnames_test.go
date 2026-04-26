// code-from-spec: TEST/tech_design/internal/logical_names@v15
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

// Qualifier is stripped before resolving — same path as without qualifier.
func TestPathFromLogicalName_ROOTWithQualifier(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/payments/processor(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/payments/processor/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/payments/processor/_node.md")
	}
}

// Verifies the qualifier is stripped (not included) in the path segment.
func TestPathFromLogicalName_ROOTWithQualifierStripsQualifier(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/x(y)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/spec/x/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/spec/x/_node.md")
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

// ROOT/<path>(<qualifier>) also has a parent.
func TestHasParent_ROOTWithQualifier(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config(interface)")
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

// Qualifier is stripped before deriving parent — ROOT/x/y(z) parent is ROOT/x.
func TestParentLogicalName_ROOTxyQualifier(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain/config(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain")
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
	// TEST/x(name) — parent is ROOT/x (qualifier stripped)
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

// ---------------------------------------------------------------------------
// HasQualifier
// ---------------------------------------------------------------------------

func TestHasQualifier_ROOTWithoutQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT/x")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

func TestHasQualifier_ROOTWithQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT/x(y)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasQualifier {
		t.Fatal("expected hasQualifier=true")
	}
}

func TestHasQualifier_ROOTNestedPathWithQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT/x/y/z(w)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasQualifier {
		t.Fatal("expected hasQualifier=true")
	}
}

func TestHasQualifier_ROOTAlone(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

func TestHasQualifier_TESTWithoutQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("TEST/x")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

func TestHasQualifier_TESTWithQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("TEST/x(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasQualifier {
		t.Fatal("expected hasQualifier=true")
	}
}

func TestHasQualifier_EXTERNAL(t *testing.T) {
	hasQualifier, ok := HasQualifier("EXTERNAL/x")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

func TestHasQualifier_EmptyString(t *testing.T) {
	hasQualifier, ok := HasQualifier("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

func TestHasQualifier_UnrecognizedPrefix(t *testing.T) {
	hasQualifier, ok := HasQualifier("UNKNOWN/x(y)")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

// ---------------------------------------------------------------------------
// QualifierName
// ---------------------------------------------------------------------------

func TestQualifierName_ROOTWithQualifier(t *testing.T) {
	got, ok := QualifierName("ROOT/x(y)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "y" {
		t.Fatalf("got %q, want %q", got, "y")
	}
}

func TestQualifierName_ROOTNestedPathWithQualifier(t *testing.T) {
	got, ok := QualifierName("ROOT/x/y(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "interface" {
		t.Fatalf("got %q, want %q", got, "interface")
	}
}

func TestQualifierName_TESTWithQualifier(t *testing.T) {
	got, ok := QualifierName("TEST/x(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "edge_cases" {
		t.Fatalf("got %q, want %q", got, "edge_cases")
	}
}

func TestQualifierName_ROOTWithoutQualifier(t *testing.T) {
	got, ok := QualifierName("ROOT/x")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

func TestQualifierName_ROOTAlone(t *testing.T) {
	got, ok := QualifierName("ROOT")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

func TestQualifierName_EmptyString(t *testing.T) {
	got, ok := QualifierName("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}
