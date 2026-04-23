// spec: TEST/tech_design/internal/chain_resolver@v15
package chainresolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Helpers ---

// chdir changes the working directory to dir for the duration of the test,
// restoring the original directory when the test completes.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

// writeFile creates a file at the given path (relative to the current directory)
// with the given content. It creates parent directories as needed.
func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeNode creates a spec node file at the standard path for the given
// logical name segments. For example, writeNode(t, "---\nversion: 1\n---\n# ROOT\n")
// creates code-from-spec/spec/_node.md.
func writeNode(t *testing.T, logicalName string, frontmatter string, body string) {
	t.Helper()
	path := logicalNameToPath(logicalName)
	content := "---\n" + frontmatter + "\n---\n" + body
	writeFile(t, path, content)
}

// logicalNameToPath converts a logical name to a file path, mirroring the
// logicalnames package behavior.
func logicalNameToPath(name string) string {
	switch {
	case name == "ROOT":
		return "code-from-spec/spec/_node.md"
	case strings.HasPrefix(name, "ROOT/"):
		return "code-from-spec/spec/" + name[5:] + "/_node.md"
	case name == "TEST":
		return "code-from-spec/spec/default.test.md"
	case strings.HasPrefix(name, "TEST/"):
		rest := name[5:]
		// Check for parenthesized test name
		if idx := strings.Index(rest, "("); idx >= 0 {
			path := rest[:idx]
			testName := rest[idx+1 : len(rest)-1]
			return "code-from-spec/spec/" + path + "/" + testName + ".test.md"
		}
		return "code-from-spec/spec/" + rest + "/default.test.md"
	case strings.HasPrefix(name, "EXTERNAL/"):
		folder := name[9:]
		return "code-from-spec/external/" + folder + "/_external.md"
	}
	return ""
}

// writeExternal creates an _external.md file for an external dependency.
func writeExternal(t *testing.T, name string, version int) {
	t.Helper()
	path := "code-from-spec/external/" + name + "/_external.md"
	content := "---\nversion: " + itoa(version) + "\n---\n# EXTERNAL/" + name + "\n"
	writeFile(t, path, content)
}

// writeExternalFile creates a supporting file inside an external dependency folder.
func writeExternalFile(t *testing.T, depName string, relPath string, content string) {
	t.Helper()
	path := "code-from-spec/external/" + depName + "/" + relPath
	writeFile(t, path, content)
}

// itoa is a trivial int-to-string for small positive ints used in frontmatter.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// --- Happy Path Tests ---

// TestLeafNodeAncestorsOnly verifies that a leaf node with no dependencies
// produces only ancestors and a target, with empty Dependencies and Code.
func TestLeafNodeAncestorsOnly(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	// Create spec tree: ROOT, ROOT/a, ROOT/a/b (leaf)
	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a", "version: 1\nparent_version: 1", "# ROOT/a\n")
	writeNode(t, "ROOT/a/b", "version: 1\nparent_version: 1", "# ROOT/a/b\n")

	chain, err := ResolveChain("ROOT/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (sorted alphabetically)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("expected first ancestor ROOT, got %s", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("expected second ancestor ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}

	// Target: ROOT/a/b
	if chain.Target.LogicalName != "ROOT/a/b" {
		t.Errorf("expected target ROOT/a/b, got %s", chain.Target.LogicalName)
	}

	// Dependencies: empty
	if len(chain.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(chain.Dependencies))
	}

	// Code: empty
	if len(chain.Code) != 0 {
		t.Errorf("expected 0 code files, got %d", len(chain.Code))
	}
}

// TestLeafNodeWithROOTDependency verifies that ROOT/ dependencies are resolved.
func TestLeafNodeWithROOTDependency(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1",
		"# ROOT/a\n")
	writeNode(t, "ROOT/b", "version: 1\nparent_version: 1", "# ROOT/b\n")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Ancestors) != 1 {
		t.Fatalf("expected 1 ancestor, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("expected ancestor ROOT, got %s", chain.Ancestors[0].LogicalName)
	}

	if chain.Target.LogicalName != "ROOT/a" {
		t.Errorf("expected target ROOT/a, got %s", chain.Target.LogicalName)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("expected dependency ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
	if len(chain.Dependencies[0].FilePaths) != 1 {
		t.Fatalf("expected 1 file path for ROOT/b, got %d", len(chain.Dependencies[0].FilePaths))
	}
	expectedPath := "code-from-spec/spec/b/_node.md"
	if chain.Dependencies[0].FilePaths[0] != expectedPath {
		t.Errorf("expected file path %s, got %s", expectedPath, chain.Dependencies[0].FilePaths[0])
	}
}

// TestLeafNodeWithExternalDependencyNoFilter verifies EXTERNAL/ deps with no filter
// include _external.md and all files in the folder.
func TestLeafNodeWithExternalDependencyNoFilter(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1",
		"# ROOT/a\n")
	writeExternal(t, "db", 1)
	writeExternalFile(t, "db", "schema.sql", "CREATE TABLE test;")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/db" {
		t.Errorf("expected EXTERNAL/db, got %s", dep.LogicalName)
	}

	// FilePaths should contain _external.md and schema.sql, sorted
	if len(dep.FilePaths) != 2 {
		t.Fatalf("expected 2 file paths, got %d: %v", len(dep.FilePaths), dep.FilePaths)
	}
	// Sorted: _external.md comes before schema.sql
	expected0 := "code-from-spec/external/db/_external.md"
	expected1 := "code-from-spec/external/db/schema.sql"
	if dep.FilePaths[0] != expected0 {
		t.Errorf("expected FilePaths[0]=%s, got %s", expected0, dep.FilePaths[0])
	}
	if dep.FilePaths[1] != expected1 {
		t.Errorf("expected FilePaths[1]=%s, got %s", expected1, dep.FilePaths[1])
	}
}

// TestLeafNodeWithExternalDependencyWithFilter verifies that filter glob patterns
// limit which files are included alongside _external.md.
func TestLeafNodeWithExternalDependencyWithFilter(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"endpoints/*.md\"",
		"# ROOT/a\n")
	writeExternal(t, "api", 1)
	writeExternalFile(t, "api", "endpoints/create.md", "create endpoint")
	writeExternalFile(t, "api", "endpoints/delete.md", "delete endpoint")
	writeExternalFile(t, "api", "types.md", "types")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/api" {
		t.Errorf("expected EXTERNAL/api, got %s", dep.LogicalName)
	}

	// Should include _external.md, endpoints/create.md, endpoints/delete.md (sorted)
	// types.md is excluded by filter
	if len(dep.FilePaths) != 3 {
		t.Fatalf("expected 3 file paths, got %d: %v", len(dep.FilePaths), dep.FilePaths)
	}
	expected := []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/endpoints/create.md",
		"code-from-spec/external/api/endpoints/delete.md",
	}
	for i, e := range expected {
		if dep.FilePaths[i] != e {
			t.Errorf("expected FilePaths[%d]=%s, got %s", i, e, dep.FilePaths[i])
		}
	}
}

// TestTestNodeIncludesParentLeafDependencies verifies that a TEST/ node
// includes the parent leaf's dependencies alongside its own.
func TestTestNodeIncludesParentLeafDependencies(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1",
		"# ROOT/a\n")
	writeExternal(t, "db", 1)

	// Test node with its own dependency on EXTERNAL/fixtures
	writeFile(t, logicalNameToPath("TEST/a"),
		"---\nversion: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/fixtures\n    version: 1\nimplements:\n  - internal/a_test.go\n---\n# TEST/a\n")
	writeExternal(t, "fixtures", 1)

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (parent leaf is in ancestors)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("expected first ancestor ROOT, got %s", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("expected second ancestor ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}

	// Target: TEST/a
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("expected target TEST/a, got %s", chain.Target.LogicalName)
	}

	// Dependencies: EXTERNAL/db and EXTERNAL/fixtures (sorted alphabetically)
	if len(chain.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("expected first dependency EXTERNAL/db, got %s", chain.Dependencies[0].LogicalName)
	}
	if chain.Dependencies[1].LogicalName != "EXTERNAL/fixtures" {
		t.Errorf("expected second dependency EXTERNAL/fixtures, got %s", chain.Dependencies[1].LogicalName)
	}
}

// TestTestNodeNoOwnDependencies verifies that a TEST/ node with no depends_on
// still inherits the parent leaf's dependencies.
func TestTestNodeNoOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1",
		"# ROOT/a\n")
	writeNode(t, "ROOT/b", "version: 1\nparent_version: 1", "# ROOT/b\n")

	// Test node with no depends_on
	writeFile(t, logicalNameToPath("TEST/a"),
		"---\nversion: 1\nparent_version: 1\nimplements:\n  - internal/a_test.go\n---\n# TEST/a\n")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("expected first ancestor ROOT, got %s", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("expected second ancestor ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}

	// Target: TEST/a
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("expected target TEST/a, got %s", chain.Target.LogicalName)
	}

	// Dependencies: ROOT/b (from parent leaf)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("expected dependency ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
}

// TestDependenciesSortedAlphabetically verifies alphabetical sorting of dependencies.
func TestDependenciesSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/z\n    version: 1\n  - path: ROOT/m\n    version: 1\n  - path: ROOT/b\n    version: 1",
		"# ROOT/a\n")
	writeNode(t, "ROOT/z", "version: 1\nparent_version: 1", "# ROOT/z\n")
	writeNode(t, "ROOT/m", "version: 1\nparent_version: 1", "# ROOT/m\n")
	writeNode(t, "ROOT/b", "version: 1\nparent_version: 1", "# ROOT/b\n")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(chain.Dependencies))
	}
	expectedOrder := []string{"ROOT/b", "ROOT/m", "ROOT/z"}
	for i, name := range expectedOrder {
		if chain.Dependencies[i].LogicalName != name {
			t.Errorf("expected Dependencies[%d]=%s, got %s", i, name, chain.Dependencies[i].LogicalName)
		}
	}
}

// TestLeafNodeImplementsFileExists verifies that existing implements files
// appear in Code.
func TestLeafNodeImplementsFileExists(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\nimplements:\n  - src/a.go",
		"# ROOT/a\n")

	// Create the implements file on disk
	writeFile(t, "src/a.go", "package a\n")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 1 {
		t.Fatalf("expected 1 code file, got %d", len(chain.Code))
	}
	if chain.Code[0] != "src/a.go" {
		t.Errorf("expected Code[0]=src/a.go, got %s", chain.Code[0])
	}
}

// TestLeafNodeImplementsFileNotExist verifies that non-existing implements
// files are not included in Code.
func TestLeafNodeImplementsFileNotExist(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\nimplements:\n  - src/a.go",
		"# ROOT/a\n")

	// Do NOT create src/a.go

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 0 {
		t.Errorf("expected 0 code files, got %d: %v", len(chain.Code), chain.Code)
	}
}

// TestLeafNodeSomeImplementsFilesExist verifies that only existing implements
// files are included in Code.
func TestLeafNodeSomeImplementsFilesExist(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\nimplements:\n  - src/a.go\n  - src/b.go",
		"# ROOT/a\n")

	// Create only src/a.go
	writeFile(t, "src/a.go", "package a\n")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Code) != 1 {
		t.Fatalf("expected 1 code file, got %d: %v", len(chain.Code), chain.Code)
	}
	if chain.Code[0] != "src/a.go" {
		t.Errorf("expected Code[0]=src/a.go, got %s", chain.Code[0])
	}
}

// --- Edge Case Tests ---

// TestTestNodeSharedExternalDependencyDeduplicated verifies that when both a
// leaf and its test node depend on the same EXTERNAL/ dependency, it appears
// only once in Dependencies.
func TestTestNodeSharedExternalDependencyDeduplicated(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1",
		"# ROOT/a\n")
	writeExternal(t, "db", 1)
	writeExternalFile(t, "db", "schema.sql", "CREATE TABLE test;")

	// Test node also depends on EXTERNAL/db
	writeFile(t, logicalNameToPath("TEST/a"),
		"---\nversion: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1\nimplements:\n  - internal/a_test.go\n---\n# TEST/a\n")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly one EXTERNAL/db dependency, not two
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("expected EXTERNAL/db, got %s", chain.Dependencies[0].LogicalName)
	}

	// FilePaths should contain _external.md and schema.sql
	if len(chain.Dependencies[0].FilePaths) != 2 {
		t.Fatalf("expected 2 file paths, got %d: %v", len(chain.Dependencies[0].FilePaths), chain.Dependencies[0].FilePaths)
	}
}

// TestDuplicateROOTDependencyDeduplicated verifies that duplicate ROOT/
// dependencies in depends_on are deduplicated.
func TestDuplicateROOTDependencyDeduplicated(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	// depends_on lists ROOT/b twice
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1\n  - path: ROOT/b\n    version: 1",
		"# ROOT/a\n")
	writeNode(t, "ROOT/b", "version: 1\nparent_version: 1", "# ROOT/b\n")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
}

// TestExternalWithOverlappingFiltersDedup verifies that when a leaf has an
// unfiltered EXTERNAL/ dep and the test node has a filtered reference to the
// same dep, file paths are merged and deduplicated.
func TestExternalWithOverlappingFiltersDedup(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	// Leaf depends on EXTERNAL/api with no filter (all files)
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1",
		"# ROOT/a\n")
	writeExternal(t, "api", 1)
	writeExternalFile(t, "api", "docs/ref.md", "reference")
	writeExternalFile(t, "api", "types.md", "types")

	// Test node depends on EXTERNAL/api with filter docs/*
	writeFile(t, logicalNameToPath("TEST/a"),
		"---\nversion: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"docs/*\"\nimplements:\n  - internal/a_test.go\n---\n# TEST/a\n")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have one EXTERNAL/api dependency
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/api" {
		t.Errorf("expected EXTERNAL/api, got %s", dep.LogicalName)
	}

	// The leaf's unfiltered reference imports all files; the test's filtered
	// reference merges but doesn't add duplicates. Result: _external.md,
	// docs/ref.md, types.md — each appearing once.
	if len(dep.FilePaths) != 3 {
		t.Fatalf("expected 3 file paths, got %d: %v", len(dep.FilePaths), dep.FilePaths)
	}
	expected := []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/docs/ref.md",
		"code-from-spec/external/api/types.md",
	}
	for i, e := range expected {
		if dep.FilePaths[i] != e {
			t.Errorf("expected FilePaths[%d]=%s, got %s", i, e, dep.FilePaths[i])
		}
	}
}

// --- Failure Case Tests ---

// TestInvalidLogicalName verifies that an unrecognized logical name prefix
// returns an error.
func TestInvalidLogicalName(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := ResolveChain("INVALID/something")
	if err == nil {
		t.Fatal("expected error for invalid logical name, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("expected error containing 'cannot resolve logical name', got: %v", err)
	}
}

// TestUnreadableFrontmatter verifies that invalid YAML in frontmatter
// produces an error from ParseFrontmatter.
func TestUnreadableFrontmatter(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	// Write invalid YAML frontmatter
	writeFile(t, logicalNameToPath("ROOT/a"), "---\n: invalid: yaml: [unclosed\n---\n# ROOT/a\n")

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for unreadable frontmatter, got nil")
	}
}

// TestUnresolvableDependency verifies that a dependency pointing to a
// non-existent node returns an error.
func TestUnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/nonexistent\n    version: 1",
		"# ROOT/a\n")
	// Do NOT create ROOT/nonexistent

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for unresolvable dependency, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("expected error containing 'cannot resolve logical name', got: %v", err)
	}
}

// TestInvalidGlobPatternInFilter verifies that an invalid glob pattern
// in a filter produces an appropriate error.
func TestInvalidGlobPatternInFilter(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	writeNode(t, "ROOT", "version: 1", "# ROOT\n")
	writeNode(t, "ROOT/a",
		"version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"[invalid\"",
		"# ROOT/a\n")
	writeExternal(t, "api", 1)
	writeExternalFile(t, "api", "data.txt", "data")

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for invalid glob pattern, got nil")
	}
	if !strings.Contains(err.Error(), "error evaluating filter") {
		t.Errorf("expected error containing 'error evaluating filter', got: %v", err)
	}
}
