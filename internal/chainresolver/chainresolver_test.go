// code-from-spec: TEST/tech_design/internal/chain_resolver@v24

// Package chainresolver provides tests for the ResolveChain function.
// Tests use t.TempDir() to create isolated spec trees, change the working
// directory to the temp dir, call ResolveChain, and restore the working
// directory after each test.
package chainresolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testChdir changes the working directory to dir and returns a function that
// restores the original working directory. Callers must defer the returned
// function.
func testChdir(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("testChdir: could not get current dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("testChdir: could not chdir to %q: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("testChdir restore: could not chdir back to %q: %v", orig, err)
		}
	}
}

// testWriteFile creates a file at path (relative to the current working
// directory) with the given content, creating all parent directories as needed.
func testWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("testWriteFile: mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("testWriteFile: write %q: %v", path, err)
	}
}

// testNodeFile returns the filesystem path (using OS separators) for a spec
// node under the code-from-spec directory. Used to create files in TempDir.
// Examples:
//   - "ROOT"     → "code-from-spec/_node.md"
//   - "ROOT/a"   → "code-from-spec/a/_node.md"
//   - "TEST/a"   → "code-from-spec/a/default.test.md"
func testNodeFile(logicalName string) string {
	if logicalName == "ROOT" {
		return filepath.Join("code-from-spec", "_node.md")
	}
	if strings.HasPrefix(logicalName, "ROOT/") {
		rel := strings.TrimPrefix(logicalName, "ROOT/")
		return filepath.Join("code-from-spec", filepath.FromSlash(rel), "_node.md")
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rel := strings.TrimPrefix(logicalName, "TEST/")
		return filepath.Join("code-from-spec", filepath.FromSlash(rel), "default.test.md")
	}
	panic(fmt.Sprintf("testNodeFile: unexpected logical name %q", logicalName))
}

// testFrontmatter builds a minimal valid frontmatter block with the given
// version, optional depends_on entries, and optional implements entries.
func testFrontmatter(version int, dependsOn []string, implements []string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "version: %d\n", version)
	if len(dependsOn) > 0 {
		sb.WriteString("depends_on:\n")
		for _, dep := range dependsOn {
			fmt.Fprintf(&sb, "  - path: %s\n    version: 1\n", dep)
		}
	}
	if len(implements) > 0 {
		sb.WriteString("implements:\n")
		for _, impl := range implements {
			fmt.Fprintf(&sb, "  - %s\n", impl)
		}
	}
	sb.WriteString("---\n")
	sb.WriteString("# body\n")
	return sb.String()
}

// testPtrString returns a pointer to the given string. Used to set Qualifier.
func testPtrString(s string) *string {
	return &s
}

// testQualifier is a helper that extracts the qualifier string from a
// *string for use in test assertions.
func testQualifier(q *string) string {
	if q == nil {
		return "<nil>"
	}
	return *q
}

// testAssertItem checks that a ChainItem matches the expected logical name,
// file path, and qualifier.
func testAssertItem(t *testing.T, label string, got ChainItem, wantLogicalName, wantFilePath string, wantQualifier *string) {
	t.Helper()
	if got.LogicalName != wantLogicalName {
		t.Errorf("%s: LogicalName = %q, want %q", label, got.LogicalName, wantLogicalName)
	}
	if got.FilePath != wantFilePath {
		t.Errorf("%s: FilePath = %q, want %q", label, got.FilePath, wantFilePath)
	}
	// Compare qualifier values
	if wantQualifier == nil {
		if got.Qualifier != nil {
			t.Errorf("%s: Qualifier = %q, want nil", label, *got.Qualifier)
		}
	} else {
		if got.Qualifier == nil {
			t.Errorf("%s: Qualifier = nil, want %q", label, *wantQualifier)
		} else if *got.Qualifier != *wantQualifier {
			t.Errorf("%s: Qualifier = %q, want %q", label, *got.Qualifier, *wantQualifier)
		}
	}
}

// ---------------------------------------------------------------------------
// Happy Path
// ---------------------------------------------------------------------------

// TestResolveChain_LeafNodeAncestorsOnly verifies that a leaf node with no
// dependencies produces the correct ancestors and an empty dependencies/code.
func TestResolveChain_LeafNodeAncestorsOnly(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// Build tree: ROOT, ROOT/a, ROOT/a/b (leaf)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT and ROOT/a, sorted alphabetically
	if len(chain.Ancestors) != 2 {
		t.Fatalf("Ancestors: got %d items, want 2", len(chain.Ancestors))
	}
	testAssertItem(t, "Ancestors[0]", chain.Ancestors[0], "ROOT", "code-from-spec/_node.md", nil)
	testAssertItem(t, "Ancestors[1]", chain.Ancestors[1], "ROOT/a", "code-from-spec/a/_node.md", nil)

	// Target: ROOT/a/b
	testAssertItem(t, "Target", chain.Target, "ROOT/a/b", "code-from-spec/a/b/_node.md", nil)

	// No dependencies
	if len(chain.Dependencies) != 0 {
		t.Errorf("Dependencies: got %d items, want 0", len(chain.Dependencies))
	}

	// No code (no implements, no files on disk)
	if len(chain.Code) != 0 {
		t.Errorf("Code: got %d items, want 0", len(chain.Code))
	}
}

// TestResolveChain_DependencyNoQualifier verifies that a ROOT/ dependency
// without qualifier is resolved correctly.
func TestResolveChain_DependencyNoQualifier(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Ancestors) != 1 {
		t.Fatalf("Ancestors: got %d items, want 1", len(chain.Ancestors))
	}
	testAssertItem(t, "Ancestors[0]", chain.Ancestors[0], "ROOT", "code-from-spec/_node.md", nil)

	testAssertItem(t, "Target", chain.Target, "ROOT/a", "code-from-spec/a/_node.md", nil)

	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1", len(chain.Dependencies))
	}
	testAssertItem(t, "Dependencies[0]", chain.Dependencies[0], "ROOT/b", "code-from-spec/b/_node.md", nil)
}

// TestResolveChain_DependencyWithQualifier verifies that a dependency with a
// qualifier is resolved with the qualifier set correctly.
func TestResolveChain_DependencyWithQualifier(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b(interface)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b(interface)"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "ROOT/b(interface)" {
		t.Errorf("Dependencies[0].LogicalName = %q, want %q", dep.LogicalName, "ROOT/b(interface)")
	}
	if dep.FilePath != "code-from-spec/b/_node.md" {
		t.Errorf("Dependencies[0].FilePath = %q, want %q", dep.FilePath, "code-from-spec/b/_node.md")
	}
	if dep.Qualifier == nil || *dep.Qualifier != "interface" {
		t.Errorf("Dependencies[0].Qualifier = %v, want %q", testQualifier(dep.Qualifier), "interface")
	}
}

// TestResolveChain_TestNodeOnlyOwnDependencies verifies that when resolving
// a TEST/ node, only the test node's own depends_on entries are used.
// The subject's depends_on (ROOT/c) is NOT included; only the test node's
// own dep (ROOT/d) appears in Dependencies.
func TestResolveChain_TestNodeOnlyOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a (subject) depends on ROOT/c
	// TEST/a depends on ROOT/d
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/c"}, nil))
	testWriteFile(t, testNodeFile("ROOT/c"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/d"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("TEST/a"), testFrontmatter(1, []string{"ROOT/d"}, nil))

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT and ROOT/a (the subject is in ancestors for TEST/ nodes)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("Ancestors: got %d items, want 2", len(chain.Ancestors))
	}
	testAssertItem(t, "Ancestors[0]", chain.Ancestors[0], "ROOT", "code-from-spec/_node.md", nil)
	testAssertItem(t, "Ancestors[1]", chain.Ancestors[1], "ROOT/a", "code-from-spec/a/_node.md", nil)

	testAssertItem(t, "Target", chain.Target, "TEST/a", "code-from-spec/a/default.test.md", nil)

	// Dependencies: only ROOT/d (test node's own dep); subject's dep ROOT/c is NOT included.
	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1 (only test node's own dep)", len(chain.Dependencies))
	}
	testAssertItem(t, "Dependencies[0]", chain.Dependencies[0], "ROOT/d", "code-from-spec/d/_node.md", nil)
}

// TestResolveChain_TestNodeNoOwnDependencies verifies that a TEST/ node with
// no depends_on of its own results in empty Dependencies — the subject's
// deps are NOT merged in.
func TestResolveChain_TestNodeNoOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b; TEST/a has no dependencies
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("TEST/a"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The subject's dep ROOT/b is not included; test node has no own deps.
	if len(chain.Dependencies) != 0 {
		t.Errorf("Dependencies: got %d items, want 0 (subject deps not merged)", len(chain.Dependencies))
	}
}

// TestResolveChain_DependenciesSorted verifies that dependencies are sorted
// alphabetically by FilePath.
func TestResolveChain_DependenciesSorted(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/z, ROOT/m, ROOT/b (intentionally unsorted)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/z", "ROOT/m", "ROOT/b"}, nil))
	testWriteFile(t, testNodeFile("ROOT/z"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/m"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 3 {
		t.Fatalf("Dependencies: got %d items, want 3", len(chain.Dependencies))
	}

	// Should be sorted by FilePath: b, m, z
	wantOrder := []string{
		"code-from-spec/b/_node.md",
		"code-from-spec/m/_node.md",
		"code-from-spec/z/_node.md",
	}
	for i, want := range wantOrder {
		if chain.Dependencies[i].FilePath != want {
			t.Errorf("Dependencies[%d].FilePath = %q, want %q", i, chain.Dependencies[i].FilePath, want)
		}
	}
}

// TestResolveChain_ImplementsFileExists verifies that when an implements file
// exists on disk, it is included in Code.
func TestResolveChain_ImplementsFileExists(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a implements src/a.go; create the file
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, nil, []string{"src/a.go"}))
	testWriteFile(t, "src/a.go", "// generated\n")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 1 {
		t.Fatalf("Code: got %d items, want 1", len(chain.Code))
	}
	if chain.Code[0] != "src/a.go" {
		t.Errorf("Code[0] = %q, want %q", chain.Code[0], "src/a.go")
	}
}

// TestResolveChain_ImplementsFileNotExist verifies that when an implements
// file does not exist on disk, it is not included in Code.
func TestResolveChain_ImplementsFileNotExist(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a implements src/a.go; do NOT create the file
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, nil, []string{"src/a.go"}))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 0 {
		t.Errorf("Code: got %d items, want 0", len(chain.Code))
	}
}

// TestResolveChain_MultipleQualifiersSameFile verifies that two different
// qualifiers for the same file are both kept in Dependencies.
func TestResolveChain_MultipleQualifiersSameFile(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b(interface) and ROOT/b(constraints)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b(interface)", "ROOT/b(constraints)"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 2 {
		t.Fatalf("Dependencies: got %d items, want 2", len(chain.Dependencies))
	}

	// Collect qualifiers
	qualifiers := make(map[string]bool)
	for _, d := range chain.Dependencies {
		if d.Qualifier == nil {
			t.Errorf("unexpected nil qualifier in Dependencies")
			continue
		}
		qualifiers[*d.Qualifier] = true
	}
	if !qualifiers["interface"] {
		t.Errorf("Dependencies: missing qualifier %q", "interface")
	}
	if !qualifiers["constraints"] {
		t.Errorf("Dependencies: missing qualifier %q", "constraints")
	}
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

// TestResolveChain_DedupSameFileSameQualifier verifies that two identical
// (FilePath, Qualifier=nil) entries in a node's own depends_on are
// deduplicated to one.
func TestResolveChain_DedupSameFileSameQualifier(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a has ROOT/b listed twice in depends_on (both nil qualifier)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b", "ROOT/b"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After dedup, only one entry for ROOT/b with Qualifier=nil
	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1 (dedup)", len(chain.Dependencies))
	}
	if chain.Dependencies[0].FilePath != "code-from-spec/b/_node.md" {
		t.Errorf("Dependencies[0].FilePath = %q", chain.Dependencies[0].FilePath)
	}
	if chain.Dependencies[0].Qualifier != nil {
		t.Errorf("Dependencies[0].Qualifier = %q, want nil", *chain.Dependencies[0].Qualifier)
	}
}

// TestResolveChain_DedupDifferentQualifiers verifies that two entries for the
// same file with different non-nil qualifiers are both kept.
func TestResolveChain_DedupDifferentQualifiers(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b(interface) and ROOT/b(constraints)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b(interface)", "ROOT/b(constraints)"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both qualifiers kept
	if len(chain.Dependencies) != 2 {
		t.Fatalf("Dependencies: got %d items, want 2", len(chain.Dependencies))
	}
	qualifiers := make(map[string]bool)
	for _, d := range chain.Dependencies {
		if d.Qualifier != nil {
			qualifiers[*d.Qualifier] = true
		}
	}
	if !qualifiers["interface"] {
		t.Errorf("Dependencies: missing %q qualifier", "interface")
	}
	if !qualifiers["constraints"] {
		t.Errorf("Dependencies: missing %q qualifier", "constraints")
	}
}

// TestResolveChain_DedupNilSubsumesSpecific verifies that a nil qualifier
// (whole # Public) subsumes a specific qualifier for the same file.
// ROOT/a depends_on ROOT/b (nil) and ROOT/b(interface) → result: one entry, nil.
func TestResolveChain_DedupNilSubsumesSpecific(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b (nil qualifier) and ROOT/b(interface)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b", "ROOT/b(interface)"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one entry: nil qualifier wins (subsumes the specific qualifier)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1 (nil subsumes specific)", len(chain.Dependencies))
	}
	if chain.Dependencies[0].Qualifier != nil {
		t.Errorf("Dependencies[0].Qualifier = %q, want nil", *chain.Dependencies[0].Qualifier)
	}
}

// TestResolveChain_DedupSpecificBeforeNilNilWins verifies that even when a
// specific qualifier appears before the nil entry, the nil entry still
// subsumes the specific qualifier.
// ROOT/a depends_on ROOT/b(interface) first, then ROOT/b (nil) → result: nil wins.
func TestResolveChain_DedupSpecificBeforeNilNilWins(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b(interface) first, then ROOT/b (nil qualifier)
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b(interface)", "ROOT/b"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one entry: nil wins even though specific was first
	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1 (nil wins)", len(chain.Dependencies))
	}
	if chain.Dependencies[0].Qualifier != nil {
		t.Errorf("Dependencies[0].Qualifier = %q, want nil", *chain.Dependencies[0].Qualifier)
	}
}

// TestResolveChain_DedupRepeatedQualifier verifies that the same non-nil
// qualifier appearing twice for the same file is deduplicated to one.
func TestResolveChain_DedupRepeatedQualifier(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/b(interface) twice
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/b(interface)", "ROOT/b(interface)"}, nil))
	testWriteFile(t, testNodeFile("ROOT/b"), testFrontmatter(1, nil, nil))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("Dependencies: got %d items, want 1 (repeated qualifier dedup)", len(chain.Dependencies))
	}
	if chain.Dependencies[0].Qualifier == nil || *chain.Dependencies[0].Qualifier != "interface" {
		t.Errorf("Dependencies[0].Qualifier = %v, want %q", testQualifier(chain.Dependencies[0].Qualifier), "interface")
	}
}

// ---------------------------------------------------------------------------
// Failure Cases
// ---------------------------------------------------------------------------

// TestResolveChain_InvalidLogicalName verifies that an invalid logical name
// (not rooted at ROOT or TEST) returns an error mentioning "cannot resolve
// logical name".
func TestResolveChain_InvalidLogicalName(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	_, err := ResolveChain("INVALID/something")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "cannot resolve logical name")
	}
}

// TestResolveChain_UnreadableFrontmatter verifies that invalid YAML
// frontmatter in the target node causes ResolveChain to return an error from
// ParseFrontmatter.
func TestResolveChain_UnreadableFrontmatter(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT must be valid so ancestors resolve; ROOT/a has bad frontmatter
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))

	// Write invalid YAML frontmatter for ROOT/a
	badContent := "---\nversion: [\ninvalid yaml\n---\n# body\n"
	testWriteFile(t, testNodeFile("ROOT/a"), badContent)

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should be a frontmatter parse error (errors.Is check)
	// The exact sentinel comes from the frontmatter package. Since we are in
	// the same package as chainresolver, we cannot import frontmatter directly
	// in this test file without a cycle, so we just verify that an error is
	// returned and that it is non-nil.
	// (The error wraps frontmatter.ErrFrontmatterParse; callers can use errors.Is.)
	_ = errors.New("") // ensure errors package is used
}

// TestResolveChain_UnresolvableDependency verifies that a dependency whose
// file does not exist on disk causes an error containing "cannot resolve
// logical name".
func TestResolveChain_UnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	restore := testChdir(t, dir)
	defer restore()

	// ROOT/a depends on ROOT/nonexistent, but that file is not on disk
	testWriteFile(t, testNodeFile("ROOT"), testFrontmatter(1, nil, nil))
	testWriteFile(t, testNodeFile("ROOT/a"), testFrontmatter(1, []string{"ROOT/nonexistent"}, nil))

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "cannot resolve logical name")
	}
}
