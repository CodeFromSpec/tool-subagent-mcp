// code-from-spec: TEST/tech_design/internal/chain_resolver@v20
package chainresolver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
)

// --- Test helpers ---

// writeFile creates a file at the given path (relative to dir) with the given content.
// It creates all necessary parent directories.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("failed to create directories for %s: %v", relPath, err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}

// specNode creates a _node.md file for a ROOT logical name with the given frontmatter body.
// The fm parameter should be the YAML content between the --- delimiters.
func specNode(t *testing.T, dir, logicalName, fm string) {
	t.Helper()
	// ROOT      -> code-from-spec/spec/_node.md
	// ROOT/a    -> code-from-spec/spec/a/_node.md
	// ROOT/a/b  -> code-from-spec/spec/a/b/_node.md
	var relPath string
	if logicalName == "ROOT" {
		relPath = "code-from-spec/spec/_node.md"
	} else {
		suffix := strings.TrimPrefix(logicalName, "ROOT/")
		relPath = "code-from-spec/spec/" + suffix + "/_node.md"
	}
	content := "---\n" + fm + "\n---\n\n# " + logicalName + "\n"
	writeFile(t, dir, relPath, content)
}

// testNode creates a default.test.md file for a TEST logical name with the given frontmatter body.
func testNode(t *testing.T, dir, logicalName, fm string) {
	t.Helper()
	// TEST/a -> code-from-spec/spec/a/default.test.md
	suffix := strings.TrimPrefix(logicalName, "TEST/")
	relPath := "code-from-spec/spec/" + suffix + "/default.test.md"
	content := "---\n" + fm + "\n---\n\n# " + logicalName + "\n"
	writeFile(t, dir, relPath, content)
}

// withWorkDir changes the working directory to dir for the duration of the test,
// restoring it on cleanup.
func withWorkDir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// testStrPtr returns a pointer to the given string. Used to build expected Qualifier values.
func testStrPtr(s string) *string {
	return &s
}

// testAssertQualifier checks that a ChainItem's Qualifier matches the expected value.
// Pass nil for expected when Qualifier should be nil.
func testAssertQualifier(t *testing.T, label string, item ChainItem, expected *string) {
	t.Helper()
	if expected == nil && item.Qualifier != nil {
		t.Errorf("%s: expected Qualifier = nil, got %q", label, *item.Qualifier)
		return
	}
	if expected != nil && item.Qualifier == nil {
		t.Errorf("%s: expected Qualifier = %q, got nil", label, *expected)
		return
	}
	if expected != nil && item.Qualifier != nil && *expected != *item.Qualifier {
		t.Errorf("%s: expected Qualifier = %q, got %q", label, *expected, *item.Qualifier)
	}
}

// --- Happy Path Tests ---

// TestLeafNode_AncestorsOnly_NoDependencies verifies that a leaf node with no
// dependencies yields the correct ancestors, target, empty dependencies, and
// empty code list.
func TestLeafNode_AncestorsOnly_NoDependencies(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// Create spec tree: ROOT, ROOT/a, ROOT/a/b (leaf)
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1")
	specNode(t, dir, "ROOT/a/b", "version: 1\nparent_version: 1")

	chain, err := ResolveChain("ROOT/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (sorted alphabetically)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestor[0] expected ROOT, got %s", chain.Ancestors[0].LogicalName)
	}
	testAssertQualifier(t, "ancestor ROOT", chain.Ancestors[0], nil)
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestor[1] expected ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}
	testAssertQualifier(t, "ancestor ROOT/a", chain.Ancestors[1], nil)

	// Target: ROOT/a/b with Qualifier = nil
	if chain.Target.LogicalName != "ROOT/a/b" {
		t.Errorf("target expected ROOT/a/b, got %s", chain.Target.LogicalName)
	}
	testAssertQualifier(t, "target", chain.Target, nil)

	// Dependencies: empty
	if len(chain.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(chain.Dependencies))
	}

	// Code: empty
	if len(chain.Code) != 0 {
		t.Errorf("expected 0 code files, got %d", len(chain.Code))
	}
}

// TestLeafNode_WithRootDependency_NoQualifier verifies that a ROOT/ dependency
// with no qualifier sets Qualifier to nil in the resulting ChainItem.
func TestLeafNode_WithRootDependency_NoQualifier(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on ROOT/b (no qualifier)
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT
	if len(chain.Ancestors) != 1 {
		t.Fatalf("expected 1 ancestor, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestor[0] expected ROOT, got %s", chain.Ancestors[0].LogicalName)
	}

	// Target: ROOT/a
	if chain.Target.LogicalName != "ROOT/a" {
		t.Errorf("target expected ROOT/a, got %s", chain.Target.LogicalName)
	}

	// Dependencies: ROOT/b with Qualifier = nil
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dependency[0] expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
	testAssertQualifier(t, "ROOT/b dependency", chain.Dependencies[0], nil)
}

// TestLeafNode_WithRootDependency_WithQualifier verifies that a ROOT/ dependency
// expressed with a parenthetical qualifier sets the ChainItem's Qualifier field.
func TestLeafNode_WithRootDependency_WithQualifier(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on ROOT/b(interface)
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dependencies: one item for ROOT/b with Qualifier = "interface"
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "ROOT/b(interface)" {
		t.Errorf("dependency LogicalName expected ROOT/b(interface), got %s", dep.LogicalName)
	}
	// FilePath should point to ROOT/b's _node.md (qualifier stripped from path resolution)
	expectedFilePath := "code-from-spec/spec/b/_node.md"
	if dep.FilePath != expectedFilePath {
		t.Errorf("dependency FilePath expected %q, got %q", expectedFilePath, dep.FilePath)
	}
	testAssertQualifier(t, "ROOT/b(interface) dependency", dep, testStrPtr("interface"))
}

// TestTestNode_IncludesSubjectDependencies verifies that for a TEST/ node,
// the subject's depends_on entries are collected alongside the test node's own.
func TestTestNode_IncludesSubjectDependencies(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on ROOT/c; TEST/a depends on ROOT/d
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/c\n    version: 1")
	specNode(t, dir, "ROOT/c", "version: 1\nparent_version: 1")
	specNode(t, dir, "ROOT/d", "version: 1\nparent_version: 1")
	testNode(t, dir, "TEST/a", "version: 1\ndepends_on:\n  - path: ROOT/d\n    version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (subject in ancestors, sorted alphabetically)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestor[0] expected ROOT, got %s", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestor[1] expected ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}

	// Target: TEST/a
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("target expected TEST/a, got %s", chain.Target.LogicalName)
	}

	// Dependencies: items from both ROOT/c (from subject) and ROOT/d (from test node)
	if len(chain.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies (ROOT/c and ROOT/d), got %d", len(chain.Dependencies))
	}
	// Sorted alphabetically by FilePath: ROOT/c < ROOT/d
	if chain.Dependencies[0].LogicalName != "ROOT/c" {
		t.Errorf("dep[0] expected ROOT/c, got %s", chain.Dependencies[0].LogicalName)
	}
	if chain.Dependencies[1].LogicalName != "ROOT/d" {
		t.Errorf("dep[1] expected ROOT/d, got %s", chain.Dependencies[1].LogicalName)
	}
}

// TestTestNode_NoOwnDependencies verifies that when a TEST/ node has no
// depends_on of its own, it still picks up the subject node's dependencies.
func TestTestNode_NoOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	// TEST/a has no depends_on
	testNode(t, dir, "TEST/a", "version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestor[0] expected ROOT, got %s", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestor[1] expected ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}

	// Target: TEST/a
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("target expected TEST/a, got %s", chain.Target.LogicalName)
	}

	// Dependencies: ROOT/b (from subject)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep[0] expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
}

// TestDependencies_SortedByFilePath verifies that dependencies are returned
// sorted alphabetically by FilePath.
func TestDependencies_SortedByFilePath(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on ROOT/z, ROOT/m, ROOT/b (declared out of order)
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/z\n    version: 1\n  - path: ROOT/m\n    version: 1\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/z", "version: 1\nparent_version: 1")
	specNode(t, dir, "ROOT/m", "version: 1\nparent_version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(chain.Dependencies))
	}
	// Sorted alphabetically by FilePath: ROOT/b < ROOT/m < ROOT/z
	expected := []string{"ROOT/b", "ROOT/m", "ROOT/z"}
	for i, e := range expected {
		if chain.Dependencies[i].LogicalName != e {
			t.Errorf("dep[%d] expected %s, got %s", i, e, chain.Dependencies[i].LogicalName)
		}
	}
}

// TestLeafNode_ImplementsFileExists verifies that an implements file that
// exists on disk is included in Code.
func TestLeafNode_ImplementsFileExists(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\nimplements:\n  - src/a.go")
	// Create the implements file on disk
	writeFile(t, dir, "src/a.go", "package src")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 1 {
		t.Fatalf("expected 1 code file, got %d", len(chain.Code))
	}
	if chain.Code[0] != "src/a.go" {
		t.Errorf("code[0] expected src/a.go, got %s", chain.Code[0])
	}
}

// TestLeafNode_ImplementsFileDoesNotExist verifies that an implements file
// that does not exist on disk is skipped (Code is empty).
func TestLeafNode_ImplementsFileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\nimplements:\n  - src/a.go")
	// Do NOT create src/a.go

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 0 {
		t.Errorf("expected 0 code files, got %d", len(chain.Code))
	}
}

// TestMultipleQualifiers_SameFile verifies that two qualified entries for the
// same file are both kept as separate ChainItems when the qualifiers differ.
func TestMultipleQualifiers_SameFile(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on ROOT/b(interface) and ROOT/b(constraints)
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1\n  - path: ROOT/b(constraints)\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both entries must be present; they point to the same file but have different qualifiers.
	if len(chain.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(chain.Dependencies))
	}

	// Both should point to ROOT/b's _node.md
	expectedFilePath := "code-from-spec/spec/b/_node.md"
	for i, dep := range chain.Dependencies {
		if dep.FilePath != expectedFilePath {
			t.Errorf("dep[%d] FilePath expected %q, got %q", i, expectedFilePath, dep.FilePath)
		}
	}

	// Collect qualifier values
	qualifiers := map[string]bool{}
	for _, dep := range chain.Dependencies {
		if dep.Qualifier == nil {
			t.Error("expected non-nil Qualifier for both dependencies")
			continue
		}
		qualifiers[*dep.Qualifier] = true
	}
	if !qualifiers["interface"] {
		t.Error("expected Qualifier \"interface\" to be present")
	}
	if !qualifiers["constraints"] {
		t.Error("expected Qualifier \"constraints\" to be present")
	}
}

// --- Edge Cases ---

// TestDedup_SameFileSameQualifier verifies that when both the subject and the
// test node depend on the same logical name (same file, nil qualifier), only
// one entry is kept.
func TestDedup_SameFileSameQualifier(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	testNode(t, dir, "TEST/a", "version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ROOT/b should appear only once (not twice)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d (dedup failed)", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
	testAssertQualifier(t, "ROOT/b", chain.Dependencies[0], nil)
}

// TestDedup_SameFileDifferentQualifiers_BothKept verifies that when subject and
// test node each reference the same file but with different qualifiers, both
// entries are kept.
func TestDedup_SameFileDifferentQualifiers_BothKept(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	testNode(t, dir, "TEST/a", "version: 1\ndepends_on:\n  - path: ROOT/b(constraints)\n    version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both entries should be kept
	if len(chain.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(chain.Dependencies))
	}
	qualifiers := map[string]bool{}
	for _, dep := range chain.Dependencies {
		if dep.Qualifier == nil {
			t.Error("expected non-nil Qualifier")
			continue
		}
		qualifiers[*dep.Qualifier] = true
	}
	if !qualifiers["interface"] {
		t.Error("expected Qualifier \"interface\" present")
	}
	if !qualifiers["constraints"] {
		t.Error("expected Qualifier \"constraints\" present")
	}
}

// TestDedup_NilQualifierSubsumesSpecific verifies that a nil-qualifier entry
// (the whole # Public section) subsumes any specific-qualifier entry for the
// same file — the specific entry should be removed.
//
// Setup: ROOT/a depends on ROOT/b (nil); TEST/a depends on ROOT/b(interface).
// Expected result: one entry with Qualifier = nil.
func TestDedup_NilQualifierSubsumesSpecific(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	testNode(t, dir, "TEST/a", "version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one entry: ROOT/b with Qualifier = nil
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	testAssertQualifier(t, "ROOT/b", chain.Dependencies[0], nil)
}

// TestDedup_SpecificBeforeNil_NilWins verifies that even when a specific
// qualifier entry appears before the nil-qualifier entry, the nil wins.
//
// Setup: ROOT/a depends on ROOT/b(interface); TEST/a depends on ROOT/b (nil).
// Expected result: one entry with Qualifier = nil.
func TestDedup_SpecificBeforeNil_NilWins(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	// Subject has the specific-qualifier entry (appears first in combined list)
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	// Test node has the nil-qualifier entry (appears second)
	testNode(t, dir, "TEST/a", "version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Even though the specific-qualifier entry appeared first, nil subsumes it.
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	testAssertQualifier(t, "ROOT/b", chain.Dependencies[0], nil)
}

// TestDedup_RepeatedQualifier_OnlyOneKept verifies that duplicate entries with
// the same file and the same non-nil qualifier are deduplicated to one.
func TestDedup_RepeatedQualifier_OnlyOneKept(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	testNode(t, dir, "TEST/a", "version: 1\ndepends_on:\n  - path: ROOT/b(interface)\n    version: 1")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be deduplicated to one item with Qualifier = "interface"
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	testAssertQualifier(t, "ROOT/b(interface)", chain.Dependencies[0], testStrPtr("interface"))
}

// --- Failure Cases ---

// TestInvalidLogicalName verifies that an unrecognized logical name prefix
// returns an error containing "cannot resolve logical name".
func TestInvalidLogicalName(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	_, err := ResolveChain("INVALID/something")
	if err == nil {
		t.Fatal("expected error for invalid logical name")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error should contain 'cannot resolve logical name', got: %s", err.Error())
	}
}

// TestUnreadableFrontmatter verifies that invalid YAML frontmatter causes
// ResolveChain to return an error (wrapping ErrFrontmatterParse).
func TestUnreadableFrontmatter(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	// Write invalid YAML frontmatter for ROOT/a
	writeFile(t, dir, "code-from-spec/spec/a/_node.md", "---\n: invalid: yaml: [[\n---\n")

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for unreadable frontmatter")
	}
	// The error must wrap ErrFrontmatterParse so callers can use errors.Is().
	if !errors.Is(err, frontmatter.ErrFrontmatterParse) {
		t.Errorf("expected errors.Is(err, frontmatter.ErrFrontmatterParse), got: %v", err)
	}
}

// TestUnresolvableDependency verifies that a depends_on entry whose file does
// not exist on disk returns an error containing "cannot resolve logical name".
func TestUnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/nonexistent\n    version: 1")
	// Do NOT create ROOT/nonexistent

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for unresolvable dependency")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error should contain 'cannot resolve logical name', got: %s", err.Error())
	}
}
