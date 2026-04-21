// spec: TEST/tech_design/internal/chain_resolver@v7

// Package chainresolver_test exercises ResolveChain against a real (temp)
// filesystem. Every test constructs the minimal directory tree it needs using
// t.TempDir(), then calls ResolveChain and asserts on the returned Chain.
//
// The tests rely on the fact that ResolveChain reads files relative to the
// process working directory. We change the working directory to a temp dir at
// the start of each test and restore it afterward so tests remain isolated.
//
// No external test framework is used — only the standard library "testing"
// package, as required by ROOT/tech_design.
package chainresolver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
)

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

// chdir temporarily changes the working directory and returns a restore
// function. Call it with defer: defer chdir(t, dir)().
func chdir(t *testing.T, dir string) func() {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore chdir %s: %v", prev, err)
		}
	}
}

// writeFile creates parent directories as needed and writes content to path
// (relative to the current working directory when the helper is called,
// which callers set to a temp dir via chdir).
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirAll %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}

// nodeFile returns a valid minimal _node.md for a spec node.
// dependsOn is raw YAML for the depends_on list (may be empty string).
func nodeFile(dependsOn string) string {
	if dependsOn == "" {
		return "---\nversion: 1\n---\n# node\n"
	}
	return "---\nversion: 1\ndepends_on:\n" + dependsOn + "\n---\n# node\n"
}

// testNodeFile returns a minimal default.test.md.
// dependsOn is raw YAML for the depends_on list (may be empty string).
func testNodeFile(dependsOn string) string {
	if dependsOn == "" {
		return "---\nversion: 1\nimplements:\n  - internal/x/x_test.go\n---\n# test\n"
	}
	return "---\nversion: 1\ndepends_on:\n" + dependsOn + "\nimplements:\n  - internal/x/x_test.go\n---\n# test\n"
}

// externalMd returns a valid _external.md.
func externalMd() string {
	return "---\nversion: 1\n---\n# EXTERNAL/x\n"
}

// specPath converts a ROOT logical name to the filesystem path used by
// PathFromLogicalName — i.e., code-from-spec/spec/<segments>/_node.md.
func specPath(logicalName string) string {
	// "ROOT" → "code-from-spec/spec/_node.md"
	// "ROOT/a/b" → "code-from-spec/spec/a/b/_node.md"
	const prefix = "ROOT"
	if logicalName == prefix {
		return filepath.Join("code-from-spec", "spec", "_node.md")
	}
	rel := strings.TrimPrefix(logicalName, prefix+"/")
	return filepath.Join("code-from-spec", "spec", filepath.FromSlash(rel), "_node.md")
}

// testPath converts a TEST logical name to the filesystem path.
// Handles "TEST/x" → "code-from-spec/spec/x/default.test.md".
func testPath(logicalName string) string {
	// "TEST/x" → "code-from-spec/spec/x/default.test.md"
	rel := strings.TrimPrefix(logicalName, "TEST/")
	return filepath.Join("code-from-spec", "spec", filepath.FromSlash(rel), "default.test.md")
}

// externalPath converts an EXTERNAL logical name to its _external.md path.
func externalPath(logicalName string) string {
	// "EXTERNAL/db" → "code-from-spec/external/db/_external.md"
	name := strings.TrimPrefix(logicalName, "EXTERNAL/")
	return filepath.Join("code-from-spec", "external", name, "_external.md")
}

// externalDir returns the directory portion of an external dependency.
func externalDir(logicalName string) string {
	name := strings.TrimPrefix(logicalName, "EXTERNAL/")
	return filepath.Join("code-from-spec", "external", name)
}

// chainItemPaths extracts the FilePaths slice from the ChainItem with the
// given LogicalName. Returns nil if not found.
func findItem(items []chainresolver.ChainItem, logicalName string) *chainresolver.ChainItem {
	for i := range items {
		if items[i].LogicalName == logicalName {
			return &items[i]
		}
	}
	return nil
}

// logicalNames extracts all LogicalName values from a slice of ChainItems.
func logicalNames(items []chainresolver.ChainItem) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.LogicalName
	}
	return names
}

// --------------------------------------------------------------------------
// Happy path tests
// --------------------------------------------------------------------------

// TestResolveChain_AncestorsOnly verifies the basic case: a leaf node with
// no dependencies. Ancestors must be sorted alphabetically and the target
// must be the deepest node.
//
// Spec reference: Happy Path — "Leaf node — ancestors only, no dependencies"
func TestResolveChain_AncestorsOnly(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// Build spec tree: ROOT, ROOT/a, ROOT/a/b (leaf)
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile(""))
	writeFile(t, specPath("ROOT/a/b"), nodeFile(""))

	chain, err := chainresolver.ResolveChain("ROOT/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Target must be the deepest node
	if chain.Target.LogicalName != "ROOT/a/b" {
		t.Errorf("target: got %q, want %q", chain.Target.LogicalName, "ROOT/a/b")
	}
	if len(chain.Target.FilePaths) != 1 {
		t.Errorf("target file paths: got %d, want 1", len(chain.Target.FilePaths))
	}

	// Ancestors must contain ROOT and ROOT/a, sorted alphabetically
	wantAncestors := []string{"ROOT", "ROOT/a"}
	gotAncestors := logicalNames(chain.Ancestors)
	if len(gotAncestors) != len(wantAncestors) {
		t.Errorf("ancestor count: got %d, want %d", len(gotAncestors), len(wantAncestors))
	} else {
		for i, want := range wantAncestors {
			if gotAncestors[i] != want {
				t.Errorf("ancestors[%d]: got %q, want %q", i, gotAncestors[i], want)
			}
		}
	}

	// No dependencies expected
	if len(chain.Dependencies) != 0 {
		t.Errorf("dependencies: got %d, want 0", len(chain.Dependencies))
	}
}

// TestResolveChain_RootDependency verifies that a ROOT/ depends_on entry
// is resolved and added to Dependencies.
//
// Spec reference: Happy Path — "Leaf node — with ROOT/ dependency"
func TestResolveChain_RootDependency(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a depends on ROOT/b
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile("  - path: ROOT/b\n    version: 1"))
	writeFile(t, specPath("ROOT/b"), nodeFile(""))

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chain.Target.LogicalName != "ROOT/a" {
		t.Errorf("target: got %q, want %q", chain.Target.LogicalName, "ROOT/a")
	}

	wantAncestors := []string{"ROOT"}
	gotAncestors := logicalNames(chain.Ancestors)
	if len(gotAncestors) != len(wantAncestors) || gotAncestors[0] != wantAncestors[0] {
		t.Errorf("ancestors: got %v, want %v", gotAncestors, wantAncestors)
	}

	// One dependency: ROOT/b with a single file path
	if len(chain.Dependencies) != 1 {
		t.Fatalf("dependencies count: got %d, want 1", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "ROOT/b" {
		t.Errorf("dependency logical name: got %q, want %q", dep.LogicalName, "ROOT/b")
	}
	if len(dep.FilePaths) != 1 {
		t.Errorf("dependency file paths: got %d, want 1", len(dep.FilePaths))
	}
}

// TestResolveChain_ExternalNoFilter verifies that an EXTERNAL/ depends_on
// entry without a filter includes _external.md and all files in the folder.
//
// Spec reference: Happy Path — "Leaf node — with EXTERNAL/ dependency, no filter"
func TestResolveChain_ExternalNoFilter(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a depends on EXTERNAL/db (no filter)
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile("  - path: EXTERNAL/db\n    version: 1"))

	// External dependency: _external.md + schema.sql
	writeFile(t, externalPath("EXTERNAL/db"), externalMd())
	writeFile(t, filepath.Join(externalDir("EXTERNAL/db"), "schema.sql"), "-- schema\n")

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("dependencies count: got %d, want 1", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/db" {
		t.Errorf("dependency logical name: got %q, want %q", dep.LogicalName, "EXTERNAL/db")
	}

	// Both _external.md and schema.sql must be present (sorted)
	if len(dep.FilePaths) != 2 {
		t.Fatalf("EXTERNAL/db file paths: got %d, want 2 — paths: %v", len(dep.FilePaths), dep.FilePaths)
	}
	// Paths are sorted; _external.md sorts before schema.sql lexicographically
	// (both use forward slashes per spec: "relative to project root")
	for _, fp := range dep.FilePaths {
		if !strings.Contains(fp, "db") {
			t.Errorf("unexpected file path in EXTERNAL/db: %q", fp)
		}
	}
}

// TestResolveChain_ExternalWithFilter verifies that a filter restricts which
// files are included from an external dependency folder.
//
// Spec reference: Happy Path — "Leaf node — with EXTERNAL/ dependency, with filter"
func TestResolveChain_ExternalWithFilter(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a depends on EXTERNAL/api with filter "endpoints/*.md"
	dependsOnYAML := "  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"endpoints/*.md\""
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile(dependsOnYAML))

	// External dependency files
	writeFile(t, externalPath("EXTERNAL/api"), externalMd())
	writeFile(t, filepath.Join(externalDir("EXTERNAL/api"), "endpoints", "create.md"), "# create\n")
	writeFile(t, filepath.Join(externalDir("EXTERNAL/api"), "endpoints", "delete.md"), "# delete\n")
	writeFile(t, filepath.Join(externalDir("EXTERNAL/api"), "types.md"), "# types\n") // excluded

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("dependencies count: got %d, want 1", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/api" {
		t.Errorf("dependency logical name: got %q, want %q", dep.LogicalName, "EXTERNAL/api")
	}

	// Expected: _external.md, endpoints/create.md, endpoints/delete.md (sorted)
	// types.md is excluded by the filter
	if len(dep.FilePaths) != 3 {
		t.Fatalf("EXTERNAL/api file paths: got %d, want 3 — paths: %v", len(dep.FilePaths), dep.FilePaths)
	}

	// Verify types.md is not included
	for _, fp := range dep.FilePaths {
		if strings.HasSuffix(fp, "types.md") {
			t.Errorf("types.md should be excluded by filter but was found: %q", fp)
		}
	}

	// Verify _external.md is always included regardless of filter
	hasExternal := false
	for _, fp := range dep.FilePaths {
		if strings.HasSuffix(fp, "_external.md") {
			hasExternal = true
			break
		}
	}
	if !hasExternal {
		t.Error("_external.md must always be included regardless of filter")
	}
}

// TestResolveChain_TestNode_IncludesParentDeps verifies that when resolving a
// TEST/ node, the parent leaf node's dependencies are merged in alongside the
// test node's own dependencies.
//
// Spec reference: Happy Path — "Test node — includes parent leaf's dependencies"
func TestResolveChain_TestNode_IncludesParentDeps(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a (leaf) depends on EXTERNAL/db
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile("  - path: EXTERNAL/db\n    version: 1"))
	writeFile(t, externalPath("EXTERNAL/db"), externalMd())

	// TEST/a depends on EXTERNAL/fixtures
	writeFile(t, testPath("TEST/a"), testNodeFile("  - path: EXTERNAL/fixtures\n    version: 1"))
	writeFile(t, externalPath("EXTERNAL/fixtures"), externalMd())

	chain, err := chainresolver.ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Target must be the TEST node
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("target: got %q, want %q", chain.Target.LogicalName, "TEST/a")
	}

	// Ancestors must include ROOT and ROOT/a (the parent leaf is an ancestor)
	wantAncestors := []string{"ROOT", "ROOT/a"}
	gotAncestors := logicalNames(chain.Ancestors)
	if len(gotAncestors) != len(wantAncestors) {
		t.Errorf("ancestor count: got %d, want %d — got: %v", len(gotAncestors), len(wantAncestors), gotAncestors)
	} else {
		for i, want := range wantAncestors {
			if gotAncestors[i] != want {
				t.Errorf("ancestors[%d]: got %q, want %q", i, gotAncestors[i], want)
			}
		}
	}

	// Dependencies: EXTERNAL/db (from parent) + EXTERNAL/fixtures (from test node)
	// sorted alphabetically
	if len(chain.Dependencies) != 2 {
		t.Fatalf("dependencies count: got %d, want 2 — got: %v", len(chain.Dependencies), logicalNames(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("dep[0]: got %q, want %q", chain.Dependencies[0].LogicalName, "EXTERNAL/db")
	}
	if chain.Dependencies[1].LogicalName != "EXTERNAL/fixtures" {
		t.Errorf("dep[1]: got %q, want %q", chain.Dependencies[1].LogicalName, "EXTERNAL/fixtures")
	}
}

// TestResolveChain_TestNode_NoDeps verifies that a TEST/ node with no own
// dependencies still inherits the parent leaf's dependencies.
//
// Spec reference: Happy Path — "Test node — no own dependencies"
func TestResolveChain_TestNode_NoDeps(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a (leaf) depends on ROOT/b
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile("  - path: ROOT/b\n    version: 1"))
	writeFile(t, specPath("ROOT/b"), nodeFile(""))

	// TEST/a has no depends_on of its own
	writeFile(t, testPath("TEST/a"), testNodeFile(""))

	chain, err := chainresolver.ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("target: got %q, want %q", chain.Target.LogicalName, "TEST/a")
	}

	// Ancestors: ROOT, ROOT/a
	wantAncestors := []string{"ROOT", "ROOT/a"}
	gotAncestors := logicalNames(chain.Ancestors)
	if len(gotAncestors) != len(wantAncestors) {
		t.Fatalf("ancestor count: got %d, want %d", len(gotAncestors), len(wantAncestors))
	}
	for i, want := range wantAncestors {
		if gotAncestors[i] != want {
			t.Errorf("ancestors[%d]: got %q, want %q", i, gotAncestors[i], want)
		}
	}

	// Dependencies: ROOT/b inherited from parent leaf
	if len(chain.Dependencies) != 1 {
		t.Fatalf("dependencies count: got %d, want 1", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep[0]: got %q, want %q", chain.Dependencies[0].LogicalName, "ROOT/b")
	}
}

// TestResolveChain_DependenciesSorted verifies that Dependencies are returned
// sorted alphabetically by logical name regardless of declaration order.
//
// Spec reference: Happy Path — "Dependencies sorted alphabetically"
func TestResolveChain_DependenciesSorted(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a depends on ROOT/z, ROOT/m, ROOT/b (deliberately out of order)
	dependsOnYAML := "  - path: ROOT/z\n    version: 1\n  - path: ROOT/m\n    version: 1\n  - path: ROOT/b\n    version: 1"
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile(dependsOnYAML))
	writeFile(t, specPath("ROOT/z"), nodeFile(""))
	writeFile(t, specPath("ROOT/m"), nodeFile(""))
	writeFile(t, specPath("ROOT/b"), nodeFile(""))

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantOrder := []string{"ROOT/b", "ROOT/m", "ROOT/z"}
	gotNames := logicalNames(chain.Dependencies)
	if len(gotNames) != len(wantOrder) {
		t.Fatalf("dependencies count: got %d, want %d", len(gotNames), len(wantOrder))
	}
	for i, want := range wantOrder {
		if gotNames[i] != want {
			t.Errorf("dep[%d]: got %q, want %q", i, gotNames[i], want)
		}
	}
}

// --------------------------------------------------------------------------
// Failure case tests
// --------------------------------------------------------------------------

// TestResolveChain_InvalidLogicalName verifies that an unrecognized logical
// name prefix causes an error containing "cannot resolve logical name".
//
// Spec reference: Failure Cases — "Invalid logical name"
func TestResolveChain_InvalidLogicalName(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	_, err := chainresolver.ResolveChain("INVALID/something")
	if err == nil {
		t.Fatal("expected error for invalid logical name, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error %q does not contain %q", err.Error(), "cannot resolve logical name")
	}
}

// TestResolveChain_UnreadableFrontmatter verifies that a malformed frontmatter
// block causes an error (wrapping the ParseFrontmatter error).
//
// Spec reference: Failure Cases — "Unreadable frontmatter"
func TestResolveChain_UnreadableFrontmatter(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT is valid; ROOT/a has corrupted YAML frontmatter
	writeFile(t, specPath("ROOT"), nodeFile(""))
	// Deliberately invalid YAML — mismatched types / syntax error
	writeFile(t, specPath("ROOT/a"), "---\nversion: [not: valid: yaml\n---\n# node\n")

	_, err := chainresolver.ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for unreadable frontmatter, got nil")
	}
	// The error originates from ParseFrontmatter — any non-nil error is acceptable
}

// TestResolveChain_UnresolvableDependency verifies that a depends_on entry
// pointing to a non-existent logical name causes an error.
//
// Spec reference: Failure Cases — "Unresolvable dependency"
func TestResolveChain_UnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a depends on ROOT/nonexistent — the file is never created
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile("  - path: ROOT/nonexistent\n    version: 1"))

	_, err := chainresolver.ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for unresolvable dependency, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error %q does not contain %q", err.Error(), "cannot resolve logical name")
	}
}

// TestResolveChain_InvalidGlobFilter verifies that a syntactically invalid
// glob pattern in an EXTERNAL/ filter produces an error containing
// "error evaluating filter".
//
// Spec reference: Failure Cases — "Invalid glob pattern in filter"
func TestResolveChain_InvalidGlobFilter(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	// ROOT/a depends on EXTERNAL/api with a broken glob pattern
	dependsOnYAML := "  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"[invalid\""
	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/a"), nodeFile(dependsOnYAML))

	// Create the external dependency directory so the error is about the glob
	writeFile(t, externalPath("EXTERNAL/api"), externalMd())

	_, err := chainresolver.ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for invalid glob filter, got nil")
	}
	if !strings.Contains(err.Error(), "error evaluating filter") {
		t.Errorf("error %q does not contain %q", err.Error(), "error evaluating filter")
	}
}

// --------------------------------------------------------------------------
// Additional coverage
// --------------------------------------------------------------------------

// TestResolveChain_RootNode verifies that resolving the root node itself
// returns no ancestors, the root as target, and no dependencies.
func TestResolveChain_RootNode(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	writeFile(t, specPath("ROOT"), nodeFile(""))

	chain, err := chainresolver.ResolveChain("ROOT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chain.Target.LogicalName != "ROOT" {
		t.Errorf("target: got %q, want %q", chain.Target.LogicalName, "ROOT")
	}
	if len(chain.Ancestors) != 0 {
		t.Errorf("ancestors: got %d, want 0", len(chain.Ancestors))
	}
	if len(chain.Dependencies) != 0 {
		t.Errorf("dependencies: got %d, want 0", len(chain.Dependencies))
	}
}

// TestResolveChain_AncestorsSortedAlphabetically verifies the "sorted
// alphabetically" guarantee for Ancestors (mirrors the Dependencies sort
// guarantee but for the ancestor chain). Because parent walk is deterministic
// by tree position, this test uses a three-level tree and checks order.
func TestResolveChain_AncestorsSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	writeFile(t, specPath("ROOT"), nodeFile(""))
	writeFile(t, specPath("ROOT/alpha"), nodeFile(""))
	writeFile(t, specPath("ROOT/alpha/beta"), nodeFile(""))
	writeFile(t, specPath("ROOT/alpha/beta/gamma"), nodeFile(""))

	chain, err := chainresolver.ResolveChain("ROOT/alpha/beta/gamma")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Alphabetical order: ROOT < ROOT/alpha < ROOT/alpha/beta
	wantAncestors := []string{"ROOT", "ROOT/alpha", "ROOT/alpha/beta"}
	gotAncestors := logicalNames(chain.Ancestors)
	if len(gotAncestors) != len(wantAncestors) {
		t.Fatalf("ancestor count: got %d, want %d", len(gotAncestors), len(wantAncestors))
	}
	for i, want := range wantAncestors {
		if gotAncestors[i] != want {
			t.Errorf("ancestors[%d]: got %q, want %q", i, gotAncestors[i], want)
		}
	}
	if chain.Target.LogicalName != "ROOT/alpha/beta/gamma" {
		t.Errorf("target: got %q, want %q", chain.Target.LogicalName, "ROOT/alpha/beta/gamma")
	}
}

// Ensure findItem helper compiles — it is used in callers that may be added
// later; suppress the unused-variable lint by referencing it here.
var _ = findItem
