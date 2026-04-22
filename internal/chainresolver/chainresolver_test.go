// spec: TEST/tech_design/internal/chain_resolver@v12
package chainresolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: createFile creates a file at the given path (relative to dir)
// with the given content. It creates all necessary parent directories.
func createFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	// Always use OS-native path for filesystem operations.
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// helper: chdirTemp changes the working directory to dir for the
// duration of the test. Restores the original directory on cleanup.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// frontmatter generates a minimal frontmatter block with the given
// version and optional depends_on entries.
func nodeFrontmatter(version int, parentVersion int, dependsOn string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("version: ")
	b.WriteString(itoa(version))
	b.WriteString("\n")
	if parentVersion > 0 {
		b.WriteString("parent_version: ")
		b.WriteString(itoa(parentVersion))
		b.WriteString("\n")
	}
	if dependsOn != "" {
		b.WriteString(dependsOn)
	}
	b.WriteString("---\n")
	return b.String()
}

// itoa converts an int to its string representation (avoids importing strconv).
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

// --- Happy Path ---

// TestLeafNodeAncestorsOnly verifies that a leaf node with no
// dependencies returns only ancestors and the target.
func TestLeafNodeAncestorsOnly(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a, ROOT/a/b (leaf)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, ""))
	createFile(t, dir, "code-from-spec/spec/a/b/_node.md",
		nodeFrontmatter(1, 1, ""))

	chain, err := ResolveChain("ROOT/a/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (sorted alphabetically)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestors[0] = %q, want ROOT", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestors[1] = %q, want ROOT/a", chain.Ancestors[1].LogicalName)
	}

	// Target: ROOT/a/b
	if chain.Target.LogicalName != "ROOT/a/b" {
		t.Errorf("target = %q, want ROOT/a/b", chain.Target.LogicalName)
	}
	if len(chain.Target.FilePaths) != 1 || chain.Target.FilePaths[0] != "code-from-spec/spec/a/b/_node.md" {
		t.Errorf("target file paths = %v, want [code-from-spec/spec/a/b/_node.md]", chain.Target.FilePaths)
	}

	// Dependencies: empty
	if len(chain.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(chain.Dependencies))
	}
}

// TestLeafNodeWithRootDependency verifies that a ROOT/ dependency
// is resolved correctly.
func TestLeafNodeWithRootDependency(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on ROOT/b), ROOT/b
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: ROOT/b\n    version: 1\n"))
	createFile(t, dir, "code-from-spec/spec/b/_node.md",
		nodeFrontmatter(1, 1, ""))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT
	if len(chain.Ancestors) != 1 {
		t.Fatalf("expected 1 ancestor, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestors[0] = %q, want ROOT", chain.Ancestors[0].LogicalName)
	}

	// Target: ROOT/a
	if chain.Target.LogicalName != "ROOT/a" {
		t.Errorf("target = %q, want ROOT/a", chain.Target.LogicalName)
	}

	// Dependencies: ROOT/b with single file path
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep[0] = %q, want ROOT/b", chain.Dependencies[0].LogicalName)
	}
	if len(chain.Dependencies[0].FilePaths) != 1 {
		t.Fatalf("expected 1 file path for dep ROOT/b, got %d", len(chain.Dependencies[0].FilePaths))
	}
	if chain.Dependencies[0].FilePaths[0] != "code-from-spec/spec/b/_node.md" {
		t.Errorf("dep file path = %q, want code-from-spec/spec/b/_node.md", chain.Dependencies[0].FilePaths[0])
	}
}

// TestLeafNodeWithExternalDependencyNoFilter verifies that an
// EXTERNAL/ dependency without a filter includes all files.
func TestLeafNodeWithExternalDependencyNoFilter(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on EXTERNAL/db)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: EXTERNAL/db\n    version: 1\n"))

	// Create external dependency: db with _external.md and schema.sql
	createFile(t, dir, "code-from-spec/external/db/_external.md",
		"---\nversion: 1\n---\n# EXTERNAL/db\n")
	createFile(t, dir, "code-from-spec/external/db/schema.sql",
		"CREATE TABLE t (id INT);")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dependencies: EXTERNAL/db with _external.md and schema.sql (sorted)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/db" {
		t.Errorf("dep = %q, want EXTERNAL/db", dep.LogicalName)
	}
	// File paths should be sorted and use forward slashes.
	expectedPaths := []string{
		"code-from-spec/external/db/_external.md",
		"code-from-spec/external/db/schema.sql",
	}
	if len(dep.FilePaths) != len(expectedPaths) {
		t.Fatalf("expected %d file paths, got %d: %v", len(expectedPaths), len(dep.FilePaths), dep.FilePaths)
	}
	for i, want := range expectedPaths {
		if dep.FilePaths[i] != want {
			t.Errorf("file path[%d] = %q, want %q", i, dep.FilePaths[i], want)
		}
	}
}

// TestLeafNodeWithExternalDependencyWithFilter verifies that an
// EXTERNAL/ dependency with a filter includes only matching files
// plus _external.md.
func TestLeafNodeWithExternalDependencyWithFilter(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on EXTERNAL/api with filter)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"endpoints/*.md\"\n"))

	// Create external dependency: api with _external.md, endpoints/create.md,
	// endpoints/delete.md, types.md
	createFile(t, dir, "code-from-spec/external/api/_external.md",
		"---\nversion: 1\n---\n# EXTERNAL/api\n")
	createFile(t, dir, "code-from-spec/external/api/endpoints/create.md", "create")
	createFile(t, dir, "code-from-spec/external/api/endpoints/delete.md", "delete")
	createFile(t, dir, "code-from-spec/external/api/types.md", "types")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	dep := chain.Dependencies[0]
	if dep.LogicalName != "EXTERNAL/api" {
		t.Errorf("dep = %q, want EXTERNAL/api", dep.LogicalName)
	}
	// _external.md always included; endpoints/create.md and endpoints/delete.md
	// match the filter; types.md does not match.
	expectedPaths := []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/endpoints/create.md",
		"code-from-spec/external/api/endpoints/delete.md",
	}
	if len(dep.FilePaths) != len(expectedPaths) {
		t.Fatalf("expected %d file paths, got %d: %v", len(expectedPaths), len(dep.FilePaths), dep.FilePaths)
	}
	for i, want := range expectedPaths {
		if dep.FilePaths[i] != want {
			t.Errorf("file path[%d] = %q, want %q", i, dep.FilePaths[i], want)
		}
	}
}

// TestTestNodeIncludesParentLeafDependencies verifies that a TEST/
// node includes its parent leaf's dependencies plus its own.
func TestTestNodeIncludesParentLeafDependencies(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on EXTERNAL/db)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: EXTERNAL/db\n    version: 1\n"))

	// Create test node TEST/a with depends_on EXTERNAL/fixtures
	createFile(t, dir, "code-from-spec/spec/a/default.test.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: EXTERNAL/fixtures\n    version: 1\n"))

	// Create external dependencies
	createFile(t, dir, "code-from-spec/external/db/_external.md",
		"---\nversion: 1\n---\n# EXTERNAL/db\n")
	createFile(t, dir, "code-from-spec/external/fixtures/_external.md",
		"---\nversion: 1\n---\n# EXTERNAL/fixtures\n")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (parent leaf in ancestors)
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestors[0] = %q, want ROOT", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestors[1] = %q, want ROOT/a", chain.Ancestors[1].LogicalName)
	}

	// Target: TEST/a
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("target = %q, want TEST/a", chain.Target.LogicalName)
	}

	// Dependencies: EXTERNAL/db and EXTERNAL/fixtures (sorted alphabetically)
	if len(chain.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("dep[0] = %q, want EXTERNAL/db", chain.Dependencies[0].LogicalName)
	}
	if chain.Dependencies[1].LogicalName != "EXTERNAL/fixtures" {
		t.Errorf("dep[1] = %q, want EXTERNAL/fixtures", chain.Dependencies[1].LogicalName)
	}
}

// TestTestNodeNoOwnDependencies verifies that a TEST/ node with no
// own depends_on still inherits the parent leaf's dependencies.
func TestTestNodeNoOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on ROOT/b), ROOT/b
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: ROOT/b\n    version: 1\n"))
	createFile(t, dir, "code-from-spec/spec/b/_node.md",
		nodeFrontmatter(1, 1, ""))

	// Create test node TEST/a with no depends_on
	createFile(t, dir, "code-from-spec/spec/a/default.test.md",
		nodeFrontmatter(1, 1, ""))

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a
	if len(chain.Ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(chain.Ancestors))
	}
	if chain.Ancestors[0].LogicalName != "ROOT" {
		t.Errorf("ancestors[0] = %q, want ROOT", chain.Ancestors[0].LogicalName)
	}
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestors[1] = %q, want ROOT/a", chain.Ancestors[1].LogicalName)
	}

	// Target: TEST/a
	if chain.Target.LogicalName != "TEST/a" {
		t.Errorf("target = %q, want TEST/a", chain.Target.LogicalName)
	}

	// Dependencies: ROOT/b (from parent leaf)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep[0] = %q, want ROOT/b", chain.Dependencies[0].LogicalName)
	}
}

// TestDependenciesSortedAlphabetically verifies that dependencies
// are returned in alphabetical order by logical name.
func TestDependenciesSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on ROOT/z, ROOT/m, ROOT/b)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: ROOT/z\n    version: 1\n  - path: ROOT/m\n    version: 1\n  - path: ROOT/b\n    version: 1\n"))
	createFile(t, dir, "code-from-spec/spec/z/_node.md",
		nodeFrontmatter(1, 1, ""))
	createFile(t, dir, "code-from-spec/spec/m/_node.md",
		nodeFrontmatter(1, 1, ""))
	createFile(t, dir, "code-from-spec/spec/b/_node.md",
		nodeFrontmatter(1, 1, ""))

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dependencies sorted: ROOT/b, ROOT/m, ROOT/z
	if len(chain.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(chain.Dependencies))
	}
	expected := []string{"ROOT/b", "ROOT/m", "ROOT/z"}
	for i, want := range expected {
		if chain.Dependencies[i].LogicalName != want {
			t.Errorf("dep[%d] = %q, want %q", i, chain.Dependencies[i].LogicalName, want)
		}
	}
}

// --- Failure Cases ---

// TestInvalidLogicalName verifies that an unrecognized logical name
// prefix produces an error.
func TestInvalidLogicalName(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	_, err := ResolveChain("INVALID/something")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error = %q, want it to contain 'cannot resolve logical name'", err.Error())
	}
}

// TestUnreadableFrontmatter verifies that invalid YAML in a node's
// frontmatter produces an error.
func TestUnreadableFrontmatter(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (leaf with invalid frontmatter)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	// Write invalid YAML frontmatter for ROOT/a
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		"---\n: invalid: yaml: [broken\n---\n")

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error from ParseFrontmatter, got nil")
	}
}

// TestUnresolvableDependency verifies that a dependency pointing to
// a nonexistent node produces an error.
func TestUnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on ROOT/nonexistent)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: ROOT/nonexistent\n    version: 1\n"))

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve logical name") {
		t.Errorf("error = %q, want it to contain 'cannot resolve logical name'", err.Error())
	}
}

// TestInvalidGlobPatternInFilter verifies that a malformed glob
// pattern in a filter produces an error.
func TestInvalidGlobPatternInFilter(t *testing.T) {
	dir := t.TempDir()
	chdirTemp(t, dir)

	// Create spec tree: ROOT, ROOT/a (depends on EXTERNAL/api with bad filter)
	createFile(t, dir, "code-from-spec/spec/_node.md",
		nodeFrontmatter(1, 0, ""))
	createFile(t, dir, "code-from-spec/spec/a/_node.md",
		nodeFrontmatter(1, 1, "depends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"[invalid\"\n"))

	// Create external dependency
	createFile(t, dir, "code-from-spec/external/api/_external.md",
		"---\nversion: 1\n---\n# EXTERNAL/api\n")
	createFile(t, dir, "code-from-spec/external/api/data.txt", "data")

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error evaluating filter") {
		t.Errorf("error = %q, want it to contain 'error evaluating filter'", err.Error())
	}
}
