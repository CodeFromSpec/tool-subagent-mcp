// code-from-spec: TEST/tech_design/internal/logical_names@v17
package logicalnames

import "testing"

// ---------------------------------------------------------------------------
// PathFromLogicalName
// ---------------------------------------------------------------------------

// ROOT — bare root resolves to the root spec file.
func TestPathFromLogicalName_ROOT(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/_node.md")
	}
}

// ROOT with path — each segment becomes a directory.
func TestPathFromLogicalName_ROOTWithPath(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/payments/processor")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/payments/processor/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/payments/processor/_node.md")
	}
}

// ROOT with qualifier — qualifier is stripped, same path as without.
func TestPathFromLogicalName_ROOTWithQualifier(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/payments/processor(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/payments/processor/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/payments/processor/_node.md")
	}
}

// ROOT with qualifier — verifies qualifier text is not included in path.
func TestPathFromLogicalName_ROOTWithQualifierStripsQualifier(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/x(y)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/x/_node.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/x/_node.md")
	}
}

// TEST without path — resolves to default test file at root.
func TestPathFromLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := PathFromLogicalName("TEST")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/default.test.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/default.test.md")
	}
}

// TEST canonical — path segments become directories, file is default.test.md.
func TestPathFromLogicalName_TESTCanonical(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/domain/config/default.test.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/domain/config/default.test.md")
	}
}

// TEST named — qualifier becomes the test file name.
func TestPathFromLogicalName_TESTNamed(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "code-from-spec/domain/config/edge_cases.test.md" {
		t.Fatalf("got %q, want %q", got, "code-from-spec/domain/config/edge_cases.test.md")
	}
}

// Unrecognized prefix — not ROOT or TEST, returns false.
func TestPathFromLogicalName_UnrecognizedPrefix(t *testing.T) {
	got, ok := PathFromLogicalName("UNKNOWN/something")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

// Empty string — invalid input, returns false.
func TestPathFromLogicalName_EmptyString(t *testing.T) {
	got, ok := PathFromLogicalName("")
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

// ROOT alone has no parent.
func TestHasParent_ROOT(t *testing.T) {
	hasParent, ok := HasParent("ROOT")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

// ROOT with path has a parent.
func TestHasParent_ROOTWithPath(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

// ROOT with qualifier has a parent.
func TestHasParent_ROOTWithQualifier(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

// TEST without path — has parent (parent is ROOT).
func TestHasParent_TESTWithoutPath(t *testing.T) {
	hasParent, ok := HasParent("TEST")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

// TEST with path — has parent.
func TestHasParent_TESTWithPath(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

// TEST named — has parent.
func TestHasParent_TESTNamed(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasParent {
		t.Fatal("expected hasParent=true")
	}
}

// Empty string — invalid, ok=false.
func TestHasParent_EmptyString(t *testing.T) {
	hasParent, ok := HasParent("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasParent {
		t.Fatal("expected hasParent=false")
	}
}

// Unrecognized prefix — invalid, ok=false.
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

// ROOT/x — parent is ROOT.
func TestParentLogicalName_ROOTx(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT" {
		t.Fatalf("got %q, want %q", got, "ROOT")
	}
}

// ROOT/x/y — parent is ROOT/x.
func TestParentLogicalName_ROOTxy(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain")
	}
}

// ROOT/x/y/z — parent is ROOT/x/y.
func TestParentLogicalName_ROOTxyz(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/tech_design/logical_names")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/tech_design" {
		t.Fatalf("got %q, want %q", got, "ROOT/tech_design")
	}
}

// ROOT/x/y(z) — qualifier stripped, parent is ROOT/x.
func TestParentLogicalName_ROOTxyQualifier(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain/config(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain")
	}
}

// TEST without path — parent is ROOT.
func TestParentLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := ParentLogicalName("TEST")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT" {
		t.Fatalf("got %q, want %q", got, "ROOT")
	}
}

// TEST/x — parent is ROOT/x.
func TestParentLogicalName_TESTWithPath(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain/config" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain/config")
	}
}

// TEST/x(name) — qualifier stripped, parent is ROOT/x.
func TestParentLogicalName_TESTNamed(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "ROOT/domain/config" {
		t.Fatalf("got %q, want %q", got, "ROOT/domain/config")
	}
}

// ROOT has no parent — returns false.
func TestParentLogicalName_ROOTHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("ROOT")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

// Invalid input — returns false.
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

// ROOT without qualifier.
func TestHasQualifier_ROOTWithoutQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT/x")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

// ROOT with qualifier.
func TestHasQualifier_ROOTWithQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT/x(y)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasQualifier {
		t.Fatal("expected hasQualifier=true")
	}
}

// ROOT with nested path and qualifier.
func TestHasQualifier_ROOTNestedPathWithQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT/x/y/z(w)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasQualifier {
		t.Fatal("expected hasQualifier=true")
	}
}

// ROOT alone — no qualifier.
func TestHasQualifier_ROOTAlone(t *testing.T) {
	hasQualifier, ok := HasQualifier("ROOT")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

// TEST without qualifier.
func TestHasQualifier_TESTWithoutQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("TEST/x")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

// TEST with qualifier.
func TestHasQualifier_TESTWithQualifier(t *testing.T) {
	hasQualifier, ok := HasQualifier("TEST/x(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !hasQualifier {
		t.Fatal("expected hasQualifier=true")
	}
}

// Empty string — invalid, ok=false.
func TestHasQualifier_EmptyString(t *testing.T) {
	hasQualifier, ok := HasQualifier("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if hasQualifier {
		t.Fatal("expected hasQualifier=false")
	}
}

// Unrecognized prefix — invalid, ok=false.
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

// ROOT with qualifier — extracts qualifier.
func TestQualifierName_ROOTWithQualifier(t *testing.T) {
	got, ok := QualifierName("ROOT/x(y)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "y" {
		t.Fatalf("got %q, want %q", got, "y")
	}
}

// ROOT with nested path and qualifier.
func TestQualifierName_ROOTNestedPathWithQualifier(t *testing.T) {
	got, ok := QualifierName("ROOT/x/y(interface)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "interface" {
		t.Fatalf("got %q, want %q", got, "interface")
	}
}

// TEST with qualifier.
func TestQualifierName_TESTWithQualifier(t *testing.T) {
	got, ok := QualifierName("TEST/x(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got != "edge_cases" {
		t.Fatalf("got %q, want %q", got, "edge_cases")
	}
}

// ROOT without qualifier — no qualifier to extract.
func TestQualifierName_ROOTWithoutQualifier(t *testing.T) {
	got, ok := QualifierName("ROOT/x")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

// ROOT alone — no qualifier.
func TestQualifierName_ROOTAlone(t *testing.T) {
	got, ok := QualifierName("ROOT")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}

// Empty string — invalid input.
func TestQualifierName_EmptyString(t *testing.T) {
	got, ok := QualifierName("")
	if ok {
		t.Fatal("expected ok=false")
	}
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}
