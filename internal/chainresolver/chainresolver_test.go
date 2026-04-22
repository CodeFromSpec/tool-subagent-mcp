// spec: TEST/tech_design/internal/chain_resolver@v9

// Package chainresolver_test contains tests for the ResolveChain function.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Context"
// Tests use t.TempDir() to create isolated project structures and call ResolveChain.
// The working directory is temporarily changed to the temp dir for each test.
package chainresolver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
)

// setupProjectRoot changes the working directory to dir for the duration of
// the test and restores it afterwards. ResolveChain resolves paths relative
// to the process working directory (project root).
func setupProjectRoot(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("chdir restore: %v", err)
		}
	})
}

// writeFile writes content to path (relative to cwd), creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirall %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writefile %s: %v", path, err)
	}
}

// nodeFile returns a minimal _node.md frontmatter with optional depends_on entries.
// Spec ref: EXTERNAL/codefromspec § "Frontmatter"
func nodeFile(version int, dependsOn string) string {
	fm := "---\nversion: " + itoa(version) + "\n"
	if dependsOn != "" {
		fm += dependsOn
	}
	fm += "---\n\n# node\n"
	return fm
}

// testNodeFile returns a minimal test node frontmatter.
func testNodeFile(version int, dependsOn string) string {
	fm := "---\nversion: " + itoa(version) + "\nparent_version: 1\n"
	if dependsOn != "" {
		fm += dependsOn
	}
	fm += "---\n\n# test\n"
	return fm
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// logicalNamesMatch checks that the ChainItems' logical names match expected
// in the given order.
func logicalNamesMatch(t *testing.T, label string, items []chainresolver.ChainItem, expected []string) {
	t.Helper()
	if len(items) != len(expected) {
		t.Errorf("%s: got %d items, want %d; got logical names: %v", label, len(items), len(expected), logicalNames(items))
		return
	}
	for i, item := range items {
		if item.LogicalName != expected[i] {
			t.Errorf("%s[%d]: got LogicalName=%q, want %q", label, i, item.LogicalName, expected[i])
		}
	}
}

func logicalNames(items []chainresolver.ChainItem) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.LogicalName
	}
	return names
}

// filePathsMatch checks that a ChainItem's FilePaths match expected (sorted).
func filePathsMatch(t *testing.T, label string, item chainresolver.ChainItem, expected []string) {
	t.Helper()
	if len(item.FilePaths) != len(expected) {
		t.Errorf("%s FilePaths: got %v, want %v", label, item.FilePaths, expected)
		return
	}
	for i, fp := range item.FilePaths {
		if fp != expected[i] {
			t.Errorf("%s FilePaths[%d]: got %q, want %q", label, i, fp, expected[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Happy Path Tests
// Spec ref: TEST/tech_design/internal/chain_resolver § "Happy Path"
// ---------------------------------------------------------------------------

// TestLeafNode_AncestorsOnly verifies that a leaf node with no dependencies
// produces the correct ancestors and target, with no dependencies.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Leaf node — ancestors only, no dependencies"
func TestLeafNode_AncestorsOnly(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	// Create spec tree: ROOT, ROOT/a, ROOT/a/b
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"))
	writeFile(t, "code-from-spec/spec/a/b/_node.md", nodeFile(1, "parent_version: 1\n"))

	chain, err := chainresolver.ResolveChain("ROOT/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spec ref: § "Leaf node — ancestors only, no dependencies" — Ancestors: ROOT, ROOT/a (sorted)
	logicalNamesMatch(t, "Ancestors", chain.Ancestors, []string{"ROOT", "ROOT/a"})
	// Target: ROOT/a/b
	if chain.Target.LogicalName != "ROOT/a/b" {
		t.Errorf("Target.LogicalName: got %q, want %q", chain.Target.LogicalName, "ROOT/a/b")
	}
	// Dependencies: empty
	if len(chain.Dependencies) != 0 {
		t.Errorf("Dependencies: got %d, want 0", len(chain.Dependencies))
	}
}

// TestLeafNode_WithROOTDependency verifies that a leaf node with a ROOT/ dependency
// has that dependency resolved in Dependencies.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Leaf node — with ROOT/ dependency"
func TestLeafNode_WithROOTDependency(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	// ROOT/a depends_on ROOT/b
	dependsOn := "depends_on:\n  - path: ROOT/b\n    version: 1\n"
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+dependsOn))
	writeFile(t, "code-from-spec/spec/b/_node.md", nodeFile(1, "parent_version: 1\n"))

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logicalNamesMatch(t, "Ancestors", chain.Ancestors, []string{"ROOT"})
	if chain.Target.LogicalName != "ROOT/a" {
		t.Errorf("Target.LogicalName: got %q, want %q", chain.Target.LogicalName, "ROOT/a")
	}
	// Spec ref: § "Leaf node — with ROOT/ dependency" — one item ROOT/b with single file path
	logicalNamesMatch(t, "Dependencies", chain.Dependencies, []string{"ROOT/b"})
	if len(chain.Dependencies[0].FilePaths) != 1 {
		t.Errorf("Dependencies[0].FilePaths: got %d, want 1", len(chain.Dependencies[0].FilePaths))
	}
}

// TestLeafNode_WithEXTERNALDependency_NoFilter verifies that an EXTERNAL/ dependency
// without a filter includes _external.md and all files in the dependency folder.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Leaf node — with EXTERNAL/ dependency, no filter"
func TestLeafNode_WithEXTERNALDependency_NoFilter(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	dependsOn := "depends_on:\n  - path: EXTERNAL/db\n    version: 1\n"
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+dependsOn))

	// Create EXTERNAL/db with _external.md and schema.sql
	writeFile(t, "code-from-spec/external/db/_external.md", "---\nversion: 1\n---\n# EXTERNAL/db\n")
	writeFile(t, "code-from-spec/external/db/schema.sql", "-- schema\n")

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logicalNamesMatch(t, "Dependencies", chain.Dependencies, []string{"EXTERNAL/db"})

	dep := chain.Dependencies[0]
	// Spec ref: § "no filter" — FilePaths contains _external.md and schema.sql (sorted)
	filePathsMatch(t, "EXTERNAL/db", dep, []string{
		"code-from-spec/external/db/_external.md",
		"code-from-spec/external/db/schema.sql",
	})
}

// TestLeafNode_WithEXTERNALDependency_WithFilter verifies that a filter restricts
// files to those matching the glob patterns (plus _external.md always).
// Spec ref: TEST/tech_design/internal/chain_resolver § "Leaf node — with EXTERNAL/ dependency, with filter"
func TestLeafNode_WithEXTERNALDependency_WithFilter(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	dependsOn := "depends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"endpoints/*.md\"\n"
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+dependsOn))

	// Create EXTERNAL/api with various files
	writeFile(t, "code-from-spec/external/api/_external.md", "---\nversion: 1\n---\n# EXTERNAL/api\n")
	writeFile(t, "code-from-spec/external/api/endpoints/create.md", "# create\n")
	writeFile(t, "code-from-spec/external/api/endpoints/delete.md", "# delete\n")
	writeFile(t, "code-from-spec/external/api/types.md", "# types\n")

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logicalNamesMatch(t, "Dependencies", chain.Dependencies, []string{"EXTERNAL/api"})

	dep := chain.Dependencies[0]
	// Spec ref: § "with filter" — _external.md, endpoints/create.md, endpoints/delete.md (sorted). types.md excluded.
	filePathsMatch(t, "EXTERNAL/api", dep, []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/endpoints/create.md",
		"code-from-spec/external/api/endpoints/delete.md",
	})
}

// TestTestNode_IncludesParentLeafDependencies verifies that a TEST/ node merges
// its own depends_on with the parent leaf node's depends_on.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Test node — includes parent leaf's dependencies"
func TestTestNode_IncludesParentLeafDependencies(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	// ROOT/a depends_on EXTERNAL/db; TEST/a depends_on EXTERNAL/fixtures
	leafDependsOn := "depends_on:\n  - path: EXTERNAL/db\n    version: 1\n"
	testDependsOn := "depends_on:\n  - path: EXTERNAL/fixtures\n    version: 1\n"

	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+leafDependsOn))
	writeFile(t, "code-from-spec/spec/a/default.test.md", testNodeFile(1, testDependsOn))

	writeFile(t, "code-from-spec/external/db/_external.md", "---\nversion: 1\n---\n# EXTERNAL/db\n")
	writeFile(t, "code-from-spec/external/fixtures/_external.md", "---\nversion: 1\n---\n# EXTERNAL/fixtures\n")

	chain, err := chainresolver.ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spec ref: § "Test node" — Ancestors: ROOT, ROOT/a (parent leaf in ancestors)
	logicalNamesMatch(t, "Ancestors", chain.Ancestors, []string{"ROOT", "ROOT/a"})
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("Target.LogicalName: got %q, want %q", chain.Target.LogicalName, "TEST/a")
	}
	// Dependencies: EXTERNAL/db and EXTERNAL/fixtures (sorted alphabetically)
	logicalNamesMatch(t, "Dependencies", chain.Dependencies, []string{"EXTERNAL/db", "EXTERNAL/fixtures"})
}

// TestTestNode_NoOwnDependencies verifies that a TEST/ node with no own depends_on
// still inherits the parent leaf's dependencies.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Test node — no own dependencies"
func TestTestNode_NoOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	leafDependsOn := "depends_on:\n  - path: ROOT/b\n    version: 1\n"

	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+leafDependsOn))
	writeFile(t, "code-from-spec/spec/b/_node.md", nodeFile(1, "parent_version: 1\n"))
	writeFile(t, "code-from-spec/spec/a/default.test.md", testNodeFile(1, "")) // no own depends_on

	chain, err := chainresolver.ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logicalNamesMatch(t, "Ancestors", chain.Ancestors, []string{"ROOT", "ROOT/a"})
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("Target.LogicalName: got %q, want %q", chain.Target.LogicalName, "TEST/a")
	}
	// Spec ref: § "no own dependencies" — one item ROOT/b (from parent leaf)
	logicalNamesMatch(t, "Dependencies", chain.Dependencies, []string{"ROOT/b"})
}

// TestDependencies_SortedAlphabetically verifies that multiple dependencies are
// returned sorted by logical name.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Dependencies sorted alphabetically"
func TestDependencies_SortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	dependsOn := "depends_on:\n  - path: ROOT/z\n    version: 1\n  - path: ROOT/m\n    version: 1\n  - path: ROOT/b\n    version: 1\n"
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+dependsOn))
	writeFile(t, "code-from-spec/spec/z/_node.md", nodeFile(1, "parent_version: 1\n"))
	writeFile(t, "code-from-spec/spec/m/_node.md", nodeFile(1, "parent_version: 1\n"))
	writeFile(t, "code-from-spec/spec/b/_node.md", nodeFile(1, "parent_version: 1\n"))

	chain, err := chainresolver.ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spec ref: § "Dependencies sorted alphabetically" — ROOT/b, ROOT/m, ROOT/z
	logicalNamesMatch(t, "Dependencies", chain.Dependencies, []string{"ROOT/b", "ROOT/m", "ROOT/z"})
}

// ---------------------------------------------------------------------------
// Failure Case Tests
// Spec ref: TEST/tech_design/internal/chain_resolver § "Failure Cases"
// ---------------------------------------------------------------------------

// TestInvalidLogicalName verifies that an unrecognized prefix returns an error
// containing "cannot resolve logical name".
// Spec ref: TEST/tech_design/internal/chain_resolver § "Invalid logical name"
func TestInvalidLogicalName(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	_, err := chainresolver.ResolveChain("INVALID/something")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error %q does not contain %q", err.Error(), "cannot resolve logical name")
	}
}

// TestUnreadableFrontmatter verifies that invalid YAML frontmatter produces
// an error from ParseFrontmatter.
// Spec ref: TEST/tech_design/internal/chain_resolver § "Unreadable frontmatter"
func TestUnreadableFrontmatter(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	// Write invalid YAML in ROOT/a's frontmatter
	writeFile(t, "code-from-spec/spec/a/_node.md", "---\nversion: [\ninvalid yaml here\n---\n")

	_, err := chainresolver.ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Expect error from ParseFrontmatter — any non-nil error is acceptable here
}

// TestUnresolvableDependency verifies that a depends_on referencing a non-existent
// ROOT/ node returns an error containing "cannot resolve logical name".
// Spec ref: TEST/tech_design/internal/chain_resolver § "Unresolvable dependency"
func TestUnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	dependsOn := "depends_on:\n  - path: ROOT/nonexistent\n    version: 1\n"
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+dependsOn))
	// NOTE: ROOT/nonexistent/_node.md is intentionally NOT created

	_, err := chainresolver.ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error %q does not contain %q", err.Error(), "cannot resolve logical name")
	}
}

// TestInvalidGlobPatternInFilter verifies that an invalid glob pattern in a filter
// returns an error containing "error evaluating filter".
// Spec ref: TEST/tech_design/internal/chain_resolver § "Invalid glob pattern in filter"
func TestInvalidGlobPatternInFilter(t *testing.T) {
	dir := t.TempDir()
	setupProjectRoot(t, dir)

	dependsOn := "depends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"[invalid\"\n"
	writeFile(t, "code-from-spec/spec/_node.md", nodeFile(1, ""))
	writeFile(t, "code-from-spec/spec/a/_node.md", nodeFile(1, "parent_version: 1\n"+dependsOn))
	writeFile(t, "code-from-spec/external/api/_external.md", "---\nversion: 1\n---\n# EXTERNAL/api\n")

	_, err := chainresolver.ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "error evaluating filter") {
		t.Errorf("error %q does not contain %q", err.Error(), "error evaluating filter")
	}
}

// contains is a simple helper to check substring presence.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
