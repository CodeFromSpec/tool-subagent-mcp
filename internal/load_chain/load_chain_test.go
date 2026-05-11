// code-from-spec: TEST/tech_design/internal/tools/load_chain@v24
package load_chain

// This test file verifies the HandleLoadChain tool handler.
// Each test creates an isolated project structure using t.TempDir(),
// changes the working directory to the temp dir for the duration of
// the test, and calls HandleLoadChain (the exported handler) to exercise
// the full code path.
//
// Spec files are placed at the paths that logicalnames.PathFromLogicalName
// produces:
//   ROOT          → code-from-spec/_node.md
//   ROOT/a        → code-from-spec/a/_node.md
//   TEST/a        → code-from-spec/a/default.test.md

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --------------------------------------------------------------------------
// Test helpers
// --------------------------------------------------------------------------

// testChdir changes the current working directory to dir and registers a
// cleanup function that restores the original directory at the end of the
// test. This is necessary because path validation uses os.Getwd().
func testChdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("testChdir: os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("testChdir: os.Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			// Best-effort restore; flag as fatal so future tests don't run
			// in the wrong directory.
			t.Fatalf("testChdir cleanup: os.Chdir(%q): %v", orig, err)
		}
	})
}

// testWriteFile writes content to a file, creating parent directories as
// needed.
func testWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("testWriteFile: MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("testWriteFile: WriteFile(%q): %v", path, err)
	}
}

// testCallHandler invokes HandleLoadChain with the given logical name and
// returns the result.
func testCallHandler(t *testing.T, logicalName string) *mcp.CallToolResult {
	t.Helper()
	args := LoadChainArgs{LogicalName: logicalName}
	result, _, err := HandleLoadChain(context.Background(), &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("HandleLoadChain returned unexpected Go error: %v", err)
	}
	return result
}

// testResultText extracts the text from the first content entry of the result.
// Fails the test if no text content is available.
func testResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("testResultText: result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("testResultText: content[0] is %T, want *mcp.TextContent", result.Content[0])
	}
	return tc.Text
}

// testAssertSuccess fails the test if the result is an error result.
func testAssertSuccess(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success result, got tool error: %s", testResultText(t, result))
	}
}

// testAssertToolError fails the test if the result is not an error result, or
// if the error message does not contain the expected substring.
func testAssertToolError(t *testing.T, result *mcp.CallToolResult, wantSubstr string) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("expected tool error, got success: %s", testResultText(t, result))
	}
	text := testResultText(t, result)
	if !strings.Contains(text, wantSubstr) {
		t.Fatalf("tool error message %q does not contain %q", text, wantSubstr)
	}
}

// --------------------------------------------------------------------------
// Spec file content builders
// --------------------------------------------------------------------------

// testRootSpec returns a minimal ROOT spec with a # Public section.
func testRootSpec() string {
	return `---
version: 1
---

# ROOT

Some private content.

# Public

Public content for ROOT.
`
}

// testRootSpecNoPublic returns a ROOT spec with no # Public section.
func testRootSpecNoPublic() string {
	return `---
version: 1
---

# ROOT

Some private content only.
`
}

// testRootSpecEmptyPublic returns a ROOT spec with a # Public section that
// has no content and no subsections (blank after trimming).
func testRootSpecEmptyPublic() string {
	return `---
version: 1
---

# ROOT

Some private content.

# Public
`
}

// testRootSpecWithPrivate returns a ROOT spec that has both # Public and
// additional private sections, used to verify that only # Public is exposed.
func testRootSpecWithPrivate() string {
	return `---
version: 1
---

# ROOT

Node name section content.

# Public

This is public.

# Private Details

This should not appear in ancestor output.
`
}

// testLeafSpec returns a simple leaf spec under ROOT/a with implements.
func testLeafSpec(implements ...string) string {
	implLines := ""
	for _, p := range implements {
		implLines += fmt.Sprintf("  - %s\n", p)
	}
	return fmt.Sprintf(`---
version: 3
parent_version: 1
implements:
%s---

# ROOT/a

Leaf node content.

# Public

Public leaf content.
`, implLines)
}

// testLeafSpecWithDepends returns a leaf spec that depends on another node.
func testLeafSpecWithDepends(dependsOn string, implements ...string) string {
	implLines := ""
	for _, p := range implements {
		implLines += fmt.Sprintf("  - %s\n", p)
	}
	return fmt.Sprintf(`---
version: 3
parent_version: 1
depends_on:
  - path: %s
    version: 1
implements:
%s---

# ROOT/a

Leaf node content.

# Public

Public leaf content.
`, dependsOn, implLines)
}

// testLeafSpecWithMultipleDepends returns a leaf spec that depends on two
// entries for the same node with different qualifiers.
func testLeafSpecWithMultipleDepends(dep1, dep2 string, implements ...string) string {
	implLines := ""
	for _, p := range implements {
		implLines += fmt.Sprintf("  - %s\n", p)
	}
	return fmt.Sprintf(`---
version: 3
parent_version: 1
depends_on:
  - path: %s
    version: 1
  - path: %s
    version: 1
implements:
%s---

# ROOT/a

Leaf node content.

# Public

Public leaf content.
`, dep1, dep2, implLines)
}

// testDepSpec returns a spec for ROOT/b with # Public containing two subsections.
func testDepSpec() string {
	return `---
version: 1
---

# ROOT/b

Dependency node.

# Public

Public intro.

## Interface

Interface subsection content.

## Constraints

Constraints subsection content.
`
}

// testDepSpecEmptyInterfaceSubsection returns a spec for ROOT/b with # Public
// containing an ## Interface subsection that has no body content.
func testDepSpecEmptyInterfaceSubsection() string {
	return `---
version: 1
---

# ROOT/b

Dependency node.

# Public

Public intro.

## Interface
`
}

// testIntermediateSpec returns a spec for an intermediate node (ROOT/a)
// with a # Public section but no implements.
func testIntermediateSpec() string {
	return `---
version: 2
---

# ROOT/a

Intermediate node.

# Public

Public intermediate content.
`
}

// testLeafSpecLevel2 returns a leaf spec for ROOT/a/b.
func testLeafSpecLevel2(implements ...string) string {
	implLines := ""
	for _, p := range implements {
		implLines += fmt.Sprintf("  - %s\n", p)
	}
	return fmt.Sprintf(`---
version: 4
parent_version: 2
implements:
%s---

# ROOT/a/b

Deep leaf content.

# Public

Deep public content.
`, implLines)
}

// testTestSpec returns a TEST/a spec.
func testTestSpec() string {
	return `---
version: 2
parent_version: 3
implements:
  - internal/a/a_test.go
---

# TEST/a

Test node content.

# Public

Test public content.
`
}

// --------------------------------------------------------------------------
// Happy path tests
// --------------------------------------------------------------------------

// TestHandleLoadChain_ValidRootLeaf verifies that a valid ROOT/ leaf node
// returns a success result whose chain content contains the ancestor ROOT
// with only its # Public section and the target ROOT/a with reduced
// frontmatter and full body.
func TestHandleLoadChain_ValidRootLeaf(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	// Create spec tree.
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// ROOT section should be present and contain only # Public content.
	if !strings.Contains(text, "node: ROOT") {
		t.Error("expected ROOT section in chain")
	}
	if !strings.Contains(text, "Public content for ROOT") {
		t.Error("expected ROOT # Public content in chain")
	}

	// TARGET section for ROOT/a should be present.
	if !strings.Contains(text, "node: ROOT/a") {
		t.Error("expected ROOT/a section in chain")
	}

	// Target content should include leaf body.
	if !strings.Contains(text, "Leaf node content") {
		t.Error("expected leaf body in chain")
	}
}

// TestHandleLoadChain_ValidTestNode verifies that a TEST/ node returns a
// success result, with ROOT and ROOT/a providing only their # Public
// sections and TEST/a providing reduced frontmatter and full body.
func TestHandleLoadChain_ValidTestNode(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "default.test.md"), testTestSpec())

	result := testCallHandler(t, "TEST/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// ROOT and ROOT/a should contain only their # Public content.
	if !strings.Contains(text, "Public content for ROOT") {
		t.Error("expected ROOT # Public content")
	}
	if !strings.Contains(text, "Public leaf content") {
		t.Error("expected ROOT/a # Public content")
	}

	// TEST/a target section should be present.
	if !strings.Contains(text, "node: TEST/a") {
		t.Error("expected TEST/a section in chain")
	}
	if !strings.Contains(text, "Test node content") {
		t.Error("expected TEST/a body in chain")
	}
}

// TestHandleLoadChain_DependencyNoQualifier verifies that a dependency
// without a qualifier includes the full # Public section.
func TestHandleLoadChain_DependencyNoQualifier(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"),
		testLeafSpecWithDepends("ROOT/b", "src/a.go"))
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "b", "_node.md"), testDepSpec())

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// The ROOT/b dependency section should contain both subsections.
	if !strings.Contains(text, "Interface subsection content") {
		t.Error("expected ## Interface content from ROOT/b dependency")
	}
	if !strings.Contains(text, "Constraints subsection content") {
		t.Error("expected ## Constraints content from ROOT/b dependency")
	}
}

// TestHandleLoadChain_DependencyWithQualifier verifies that a dependency
// with a qualifier includes only the matching ## subsection, not others.
func TestHandleLoadChain_DependencyWithQualifier(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// ROOT/a depends on ROOT/b(interface) — only the ## Interface subsection.
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"),
		testLeafSpecWithDepends("ROOT/b(interface)", "src/a.go"))
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "b", "_node.md"), testDepSpec())

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// Only the Interface subsection content should be present.
	if !strings.Contains(text, "Interface subsection content") {
		t.Error("expected ## Interface content in chain")
	}
	// Constraints must NOT be present when qualifier is "interface".
	if strings.Contains(text, "Constraints subsection content") {
		t.Error("unexpected ## Constraints content in chain (should be excluded by qualifier)")
	}
}

// TestHandleLoadChain_HeredocFormat verifies that the chain output uses
// heredoc-style delimiters with node: and path: headers.
func TestHandleLoadChain_HeredocFormat(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// Must contain heredoc delimiters.
	if !strings.Contains(text, "<<<FILE_") {
		t.Error("expected <<<FILE_ delimiter in chain output")
	}
	if !strings.Contains(text, "<<<END_FILE_") {
		t.Error("expected <<<END_FILE_ delimiter in chain output")
	}

	// Sections must have node: and path: headers.
	if !strings.Contains(text, "node:") {
		t.Error("expected node: header in chain output")
	}
	if !strings.Contains(text, "path:") {
		t.Error("expected path: header in chain output")
	}
}

// TestHandleLoadChain_AncestorsContainOnlyPublic verifies that ancestor
// sections in the chain contain only # Public content. Private sections and
// node name sections must not appear.
func TestHandleLoadChain_AncestorsContainOnlyPublic(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpecWithPrivate())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testIntermediateSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "b", "_node.md"), testLeafSpecLevel2("src/b.go"))

	result := testCallHandler(t, "ROOT/a/b")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// Public content should be present.
	if !strings.Contains(text, "This is public") {
		t.Error("expected # Public content from ROOT in chain")
	}
	if !strings.Contains(text, "Public intermediate content") {
		t.Error("expected # Public content from ROOT/a in chain")
	}

	// Private section content must NOT appear in ancestor sections.
	// Note: "Private Details" heading or its content must not appear.
	if strings.Contains(text, "This should not appear") {
		t.Error("private section content must not appear in ancestor output")
	}
}

// TestHandleLoadChain_TargetHasReducedFrontmatter verifies that the target
// node's frontmatter in the chain is reduced to only version and implements.
func TestHandleLoadChain_TargetHasReducedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// Build a leaf spec with extra frontmatter fields that should be stripped.
	leafContent := `---
version: 5
parent_version: 1
depends_on:
  - path: ROOT/b
    version: 2
implements:
  - src/a.go
---

# ROOT/a

Leaf body.
`
	// Also create ROOT/b so chain resolution does not fail.
	depContent := `---
version: 2
---

# ROOT/b

Dep body.

# Public

Dep public.
`
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), leafContent)
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "b", "_node.md"), depContent)

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// The target section must contain version and implements.
	if !strings.Contains(text, "version: 5") {
		t.Error("expected version: 5 in reduced frontmatter")
	}
	if !strings.Contains(text, "src/a.go") {
		t.Error("expected implements entry in reduced frontmatter")
	}

	// Fields parent_version and depends_on must NOT appear in target frontmatter.
	// They may appear in other parts of the chain text (e.g., comments), but the
	// key signal is that the stripped fields are absent from the target section.
	//
	// We locate the target file section and inspect it specifically.
	targetIdx := strings.Index(text, "node: ROOT/a\n")
	if targetIdx < 0 {
		t.Fatal("could not locate target section in chain output")
	}
	// Find the closing delimiter after the target section start.
	endDelimIdx := strings.Index(text[targetIdx:], "<<<END_FILE_")
	if endDelimIdx < 0 {
		t.Fatal("could not locate end delimiter for target section")
	}
	targetSection := text[targetIdx : targetIdx+endDelimIdx]

	if strings.Contains(targetSection, "parent_version") {
		t.Error("parent_version must not appear in reduced frontmatter of target section")
	}
	if strings.Contains(targetSection, "depends_on") {
		t.Error("depends_on must not appear in reduced frontmatter of target section")
	}
}

// TestHandleLoadChain_ExistingCodeFilesIncluded verifies that when an
// implements file exists on disk, it is included as a code section in the
// chain output with a path: header but no node: header.
func TestHandleLoadChain_ExistingCodeFilesIncluded(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))

	// Create the implements file with known content.
	const knownContent = "package a\n\n// existing source\n"
	testWriteFile(t, filepath.Join(dir, "src", "a.go"), knownContent)

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// Must have a section with path: src/a.go but no node: header.
	if !strings.Contains(text, "path: src/a.go") {
		t.Error("expected path: src/a.go in chain output")
	}
	if !strings.Contains(text, "existing source") {
		t.Error("expected existing source file content in chain output")
	}

	// Verify the code section does NOT have a node: header directly before the path: header.
	// Find the section for src/a.go and check it has no node: line.
	pathIdx := strings.Index(text, "path: src/a.go")
	if pathIdx < 0 {
		t.Fatal("path: src/a.go not found in chain")
	}
	// Look back from the path: line to find the opening delimiter for this section.
	textBefore := text[:pathIdx]
	lastOpenIdx := strings.LastIndex(textBefore, "<<<FILE_")
	if lastOpenIdx < 0 {
		t.Fatal("could not find opening delimiter for src/a.go section")
	}
	sectionHeader := textBefore[lastOpenIdx:]
	if strings.Contains(sectionHeader, "node:") {
		t.Error("code section for src/a.go must not have a node: header")
	}
}

// TestHandleLoadChain_NonExistingCodeFilesOmitted verifies that when an
// implements file does NOT exist on disk, it is not included in the chain.
func TestHandleLoadChain_NonExistingCodeFilesOmitted(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))

	// Do NOT create src/a.go.

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// The chain must not contain a section for src/a.go.
	if strings.Contains(text, "path: src/a.go") {
		t.Error("chain must not include section for non-existing code file src/a.go")
	}
}

// TestHandleLoadChain_AncestorWithNoPublicSectionOmitted verifies that an
// ancestor with no # Public section is omitted from the chain entirely.
// Per spec: "The chain content does not contain a file section for ROOT."
func TestHandleLoadChain_AncestorWithNoPublicSectionOmitted(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	// ROOT has no # Public section.
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpecNoPublic())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// ROOT section must NOT be present — ancestors with no # Public are omitted entirely.
	if strings.Contains(text, "node: ROOT\n") {
		t.Error("chain must not contain a file section for ROOT when it has no # Public section")
	}

	// The private content of ROOT must not appear at all.
	if strings.Contains(text, "Some private content only") {
		t.Error("private content must not appear in chain when ROOT has no # Public")
	}
}

// TestHandleLoadChain_AncestorWithEmptyPublicSectionOmitted verifies that an
// ancestor with a # Public section that has no content and no subsections
// (blank after trimming) is omitted from the chain entirely.
func TestHandleLoadChain_AncestorWithEmptyPublicSectionOmitted(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	// ROOT has an empty # Public section (no content, no subsections).
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpecEmptyPublic())
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), testLeafSpec("src/a.go"))

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// ROOT section must NOT be present — ancestors with empty # Public are omitted.
	if strings.Contains(text, "node: ROOT\n") {
		t.Error("chain must not contain a file section for ROOT when its # Public section is empty")
	}
}

// TestHandleLoadChain_DependencyWithEmptyExtractedContentOmitted verifies that
// when a dependency's extracted content (after applying the qualifier filter)
// is empty (blank after trimming), the dependency is omitted from the chain.
func TestHandleLoadChain_DependencyWithEmptyExtractedContentOmitted(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// ROOT/a depends on ROOT/b(interface), but ROOT/b's ## Interface has no body.
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"),
		testLeafSpecWithDepends("ROOT/b(interface)", "src/a.go"))
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "b", "_node.md"),
		testDepSpecEmptyInterfaceSubsection())

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// ROOT/b section must NOT be present — the extracted ## Interface content is empty.
	if strings.Contains(text, "node: ROOT/b\n") {
		t.Error("chain must not contain a file section for ROOT/b when its extracted content is empty")
	}
}

// TestHandleLoadChain_MultipleQualifiersConsolidated verifies that when a
// node depends on the same dependency with multiple qualifiers, the chain
// contains exactly one file section for that dependency, and that section
// includes the content of all matched subsections in order.
func TestHandleLoadChain_MultipleQualifiersConsolidated(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// ROOT/a depends on ROOT/b(interface) and ROOT/b(constraints).
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"),
		testLeafSpecWithMultipleDepends("ROOT/b(interface)", "ROOT/b(constraints)", "src/a.go"))
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "b", "_node.md"), testDepSpec())

	result := testCallHandler(t, "ROOT/a")
	testAssertSuccess(t, result)

	text := testResultText(t, result)

	// Both subsections must appear in the chain.
	if !strings.Contains(text, "Interface subsection content") {
		t.Error("expected ## Interface content in consolidated ROOT/b section")
	}
	if !strings.Contains(text, "Constraints subsection content") {
		t.Error("expected ## Constraints content in consolidated ROOT/b section")
	}

	// There must be exactly one file section for ROOT/b (not two).
	firstOccurrence := strings.Index(text, "node: ROOT/b\n")
	if firstOccurrence < 0 {
		t.Fatal("expected at least one ROOT/b section in chain")
	}
	// Look for a second occurrence of "node: ROOT/b" after the first.
	secondOccurrence := strings.Index(text[firstOccurrence+1:], "node: ROOT/b\n")
	if secondOccurrence >= 0 {
		t.Error("ROOT/b must appear in exactly one file section, but found duplicate sections")
	}
}

// --------------------------------------------------------------------------
// Failure case tests
// --------------------------------------------------------------------------

// TestHandleLoadChain_InvalidPrefix verifies that a logical name that does
// not start with ROOT/ or TEST/ returns a tool error.
func TestHandleLoadChain_InvalidPrefix(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	result := testCallHandler(t, "INVALID/something")
	testAssertToolError(t, result, "target must be a ROOT/ or TEST/")
}

// TestHandleLoadChain_NonexistentSpecFile verifies that referencing a logical
// name whose spec file does not exist on disk returns a tool error.
func TestHandleLoadChain_NonexistentSpecFile(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	// Do not create the spec file for ROOT/nonexistent.
	result := testCallHandler(t, "ROOT/nonexistent")
	// Must be a tool error (from ParseFrontmatter — file not found).
	if !result.IsError {
		t.Fatal("expected tool error for nonexistent spec file, got success")
	}
}

// TestHandleLoadChain_NoImplements verifies that a node without an implements
// field returns a tool error containing "has no implements".
func TestHandleLoadChain_NoImplements(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// ROOT/a has no implements.
	noImplSpec := `---
version: 1
parent_version: 1
---

# ROOT/a

No implements field.
`
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), noImplSpec)

	result := testCallHandler(t, "ROOT/a")
	testAssertToolError(t, result, "has no implements")
}

// TestHandleLoadChain_InvalidImplementsPathTraversal verifies that a path
// traversal attack in implements is rejected as a tool error.
func TestHandleLoadChain_InvalidImplementsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// ROOT/a has a traversal path in implements.
	traversalSpec := `---
version: 1
parent_version: 1
implements:
  - ../../etc/passwd
---

# ROOT/a

Traversal attempt.
`
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"), traversalSpec)

	result := testCallHandler(t, "ROOT/a")
	// Must be a tool error from path validation.
	if !result.IsError {
		t.Fatal("expected tool error for traversal path in implements, got success")
	}
}

// TestHandleLoadChain_UnresolvableDependency verifies that if a depends_on
// entry references a node whose file does not exist, a tool error is returned.
func TestHandleLoadChain_UnresolvableDependency(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	testWriteFile(t, filepath.Join(dir, "code-from-spec", "_node.md"), testRootSpec())
	// ROOT/a depends on ROOT/b but ROOT/b's spec file does not exist.
	testWriteFile(t, filepath.Join(dir, "code-from-spec", "a", "_node.md"),
		testLeafSpecWithDepends("ROOT/b", "src/a.go"))

	result := testCallHandler(t, "ROOT/a")
	// Must be a tool error from chain resolution.
	if !result.IsError {
		t.Fatal("expected tool error for unresolvable dependency, got success")
	}
}
