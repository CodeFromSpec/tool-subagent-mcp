// spec: TEST/tech_design/internal/chain_resolver@v17
package chainresolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	// ROOT -> code-from-spec/spec/_node.md
	// ROOT/a -> code-from-spec/spec/a/_node.md
	// ROOT/a/b -> code-from-spec/spec/a/b/_node.md
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

// externalDep creates an _external.md in the external dependency folder.
func externalDep(t *testing.T, dir, name string) {
	t.Helper()
	relPath := "code-from-spec/external/" + name + "/_external.md"
	content := "---\nversion: 1\n---\n\n# EXTERNAL/" + name + "\n"
	writeFile(t, dir, relPath, content)
}

// externalFile creates an arbitrary file inside an external dependency folder.
func externalFile(t *testing.T, dir, depName, fileRelPath, content string) {
	t.Helper()
	relPath := "code-from-spec/external/" + depName + "/" + fileRelPath
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

// assertFilePaths checks that a ChainItem's FilePaths match the expected list exactly (order matters).
func assertFilePaths(t *testing.T, label string, item ChainItem, expected []string) {
	t.Helper()
	if len(item.FilePaths) != len(expected) {
		t.Errorf("%s: expected %d file paths, got %d\n  expected: %v\n  got:      %v",
			label, len(expected), len(item.FilePaths), expected, item.FilePaths)
		return
	}
	for i, e := range expected {
		if item.FilePaths[i] != e {
			t.Errorf("%s: file path [%d] expected %q, got %q", label, i, e, item.FilePaths[i])
		}
	}
}

// --- Happy Path Tests ---

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
	if chain.Ancestors[1].LogicalName != "ROOT/a" {
		t.Errorf("ancestor[1] expected ROOT/a, got %s", chain.Ancestors[1].LogicalName)
	}

	// Target: ROOT/a/b
	if chain.Target.LogicalName != "ROOT/a/b" {
		t.Errorf("target expected ROOT/a/b, got %s", chain.Target.LogicalName)
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

func TestLeafNode_WithRootDependency(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on ROOT/b
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

	// Dependencies: ROOT/b
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dependency[0] expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
	assertFilePaths(t, "ROOT/b", chain.Dependencies[0], []string{
		"code-from-spec/spec/b/_node.md",
	})
}

func TestLeafNode_ExternalDependency_NoFilter(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1")
	externalDep(t, dir, "db")
	externalFile(t, dir, "db", "schema.sql", "CREATE TABLE t;")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dependencies: EXTERNAL/db with _external.md and schema.sql (sorted)
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("dependency expected EXTERNAL/db, got %s", chain.Dependencies[0].LogicalName)
	}
	assertFilePaths(t, "EXTERNAL/db", chain.Dependencies[0], []string{
		"code-from-spec/external/db/_external.md",
		"code-from-spec/external/db/schema.sql",
	})
}

func TestLeafNode_ExternalDependency_WithFilter(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"endpoints/*.md\"")
	externalDep(t, dir, "api")
	externalFile(t, dir, "api", "endpoints/create.md", "create endpoint")
	externalFile(t, dir, "api", "endpoints/delete.md", "delete endpoint")
	externalFile(t, dir, "api", "types.md", "type definitions")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	// _external.md is always included, plus files matching endpoints/*.md
	// types.md should be excluded by filter
	assertFilePaths(t, "EXTERNAL/api", chain.Dependencies[0], []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/endpoints/create.md",
		"code-from-spec/external/api/endpoints/delete.md",
	})
}

func TestTestNode_IncludesParentLeafDependencies(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1")
	testNode(t, dir, "TEST/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/fixtures\n    version: 1")
	externalDep(t, dir, "db")
	externalDep(t, dir, "fixtures")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ancestors: ROOT, ROOT/a (parent leaf in ancestors)
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

	// Dependencies: EXTERNAL/db and EXTERNAL/fixtures (sorted)
	if len(chain.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("dep[0] expected EXTERNAL/db, got %s", chain.Dependencies[0].LogicalName)
	}
	if chain.Dependencies[1].LogicalName != "EXTERNAL/fixtures" {
		t.Errorf("dep[1] expected EXTERNAL/fixtures, got %s", chain.Dependencies[1].LogicalName)
	}
}

func TestTestNode_NoOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")
	testNode(t, dir, "TEST/a", "version: 1\nparent_version: 1")

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

	// Dependencies: ROOT/b from parent leaf
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep[0] expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
}

func TestDependencies_SortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

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
	expected := []string{"ROOT/b", "ROOT/m", "ROOT/z"}
	for i, e := range expected {
		if chain.Dependencies[i].LogicalName != e {
			t.Errorf("dep[%d] expected %s, got %s", i, e, chain.Dependencies[i].LogicalName)
		}
	}
}

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

func TestLeafNode_SomeImplementsFilesExist(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\nimplements:\n  - src/a.go\n  - src/b.go")
	// Create only src/a.go
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

func TestExternal_WithFilter_SubdirectoriesExcluded(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"endpoints/*\"")
	externalDep(t, dir, "api")
	externalFile(t, dir, "api", "endpoints/create.md", "create")
	// Create a subdirectory with a file that should NOT match the filter
	externalFile(t, dir, "api", "endpoints/drafts/notes.md", "notes")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	// endpoints/drafts/notes.md should NOT match "endpoints/*" because path.Match
	// does not match path separators in *.
	assertFilePaths(t, "EXTERNAL/api", chain.Dependencies[0], []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/endpoints/create.md",
	})
}

// --- Edge Cases ---

func TestTestNode_SharedExternalDependency_Deduplicated(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1")
	testNode(t, dir, "TEST/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/db\n    version: 1")
	externalDep(t, dir, "db")
	externalFile(t, dir, "db", "schema.sql", "CREATE TABLE t;")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly one EXTERNAL/db, not two
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/db" {
		t.Errorf("dep expected EXTERNAL/db, got %s", chain.Dependencies[0].LogicalName)
	}
	assertFilePaths(t, "EXTERNAL/db", chain.Dependencies[0], []string{
		"code-from-spec/external/db/_external.md",
		"code-from-spec/external/db/schema.sql",
	})
}

func TestDuplicateRootDependency_Deduplicated(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: ROOT/b\n    version: 1\n  - path: ROOT/b\n    version: 1")
	specNode(t, dir, "ROOT/b", "version: 1\nparent_version: 1")

	chain, err := ResolveChain("ROOT/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be deduplicated to one item
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "ROOT/b" {
		t.Errorf("dep expected ROOT/b, got %s", chain.Dependencies[0].LogicalName)
	}
}

func TestExternal_OverlappingFilters_FilesDeduplicated(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	// ROOT/a depends on EXTERNAL/api (no filter = all files)
	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1")
	// TEST/a depends on EXTERNAL/api with filter ["docs/*"]
	testNode(t, dir, "TEST/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"docs/*\"")
	externalDep(t, dir, "api")
	externalFile(t, dir, "api", "docs/ref.md", "reference")
	externalFile(t, dir, "api", "types.md", "types")

	chain, err := ResolveChain("TEST/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be one EXTERNAL/api with all files, each appearing only once
	if len(chain.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(chain.Dependencies))
	}
	if chain.Dependencies[0].LogicalName != "EXTERNAL/api" {
		t.Errorf("dep expected EXTERNAL/api, got %s", chain.Dependencies[0].LogicalName)
	}
	assertFilePaths(t, "EXTERNAL/api", chain.Dependencies[0], []string{
		"code-from-spec/external/api/_external.md",
		"code-from-spec/external/api/docs/ref.md",
		"code-from-spec/external/api/types.md",
	})
}

// --- Failure Cases ---

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
}

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

func TestInvalidGlobPattern(t *testing.T) {
	dir := t.TempDir()
	withWorkDir(t, dir)

	specNode(t, dir, "ROOT", "version: 1")
	specNode(t, dir, "ROOT/a", "version: 1\nparent_version: 1\ndepends_on:\n  - path: EXTERNAL/api\n    version: 1\n    filter:\n      - \"[invalid\"")
	externalDep(t, dir, "api")
	externalFile(t, dir, "api", "data.txt", "data")

	_, err := ResolveChain("ROOT/a")
	if err == nil {
		t.Fatal("expected error for invalid glob pattern")
	}
	if !strings.Contains(err.Error(), "error evaluating filter") {
		t.Errorf("error should contain 'error evaluating filter', got: %s", err.Error())
	}
}
