// code-from-spec: TEST/tech_design/internal/parsenode@v4

// Package parsenode contains tests for the ParseNode function.
// These tests are internal (same package) so they can access unexported helpers.
//
// Each test creates a temporary directory, writes spec files at the expected
// path (code-from-spec/spec/<path>/_node.md), changes the working directory
// to the temp dir, calls ParseNode, and restores the working directory.
package parsenode

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testWriteNode creates the spec file for logicalName inside dir, using the
// file path that logicalnames.PathFromLogicalName would resolve.
//
// For logical name ROOT/x/y the file is:
//   <dir>/code-from-spec/spec/x/y/_node.md
//
// For logical name TEST/x the file is:
//   <dir>/code-from-spec/spec/x/default.test.md
//
// The function creates all parent directories as needed and writes content.
func testWriteNode(t *testing.T, dir, logicalName, content string) {
	t.Helper()

	// Derive the relative path from the logical name using the same rules as
	// logicalnames.PathFromLogicalName, replicated here so tests are
	// self-contained and do not depend on the live working directory.
	relPath := testPathFromLogicalName(t, logicalName)

	fullPath := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("testWriteNode: MkdirAll: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("testWriteNode: WriteFile: %v", err)
	}
}

// testPathFromLogicalName mirrors logicalnames.PathFromLogicalName for the
// purposes of setting up test fixtures. It only handles the ROOT and TEST
// prefixes used in these tests.
func testPathFromLogicalName(t *testing.T, logicalName string) string {
	t.Helper()

	// Strip parenthetical qualifier if present.
	base := logicalName
	if idx := strings.IndexByte(base, '('); idx != -1 {
		base = base[:idx]
	}

	if base == "ROOT" {
		return "code-from-spec/spec/_node.md"
	}
	if strings.HasPrefix(base, "ROOT/") {
		rest := strings.TrimPrefix(base, "ROOT/")
		return "code-from-spec/spec/" + rest + "/_node.md"
	}
	if base == "TEST" {
		return "code-from-spec/spec/default.test.md"
	}
	if strings.HasPrefix(base, "TEST/") {
		rest := strings.TrimPrefix(base, "TEST/")
		// TEST/x → code-from-spec/spec/x/default.test.md
		return "code-from-spec/spec/" + rest + "/default.test.md"
	}

	t.Fatalf("testPathFromLogicalName: unrecognised logical name %q", logicalName)
	return ""
}

// testChdir changes the working directory to dir for the duration of the test
// and restores the original directory via t.Cleanup.
func testChdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("testChdir: Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("testChdir: Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			// Log only — cleanup must not call t.Fatal after the test ends.
			t.Logf("testChdir cleanup: Chdir(%q): %v", orig, err)
		}
	})
}

// ---------------------------------------------------------------------------
// Happy path
// ---------------------------------------------------------------------------

// TestParseNode_MinimalNode verifies that a node with only a name section is
// parsed correctly: Public is nil, Private is nil, and NameSection reflects
// the single paragraph of content.
func TestParseNode_MinimalNode(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/x", `---
version: 3
parent_version: 1
---
# ROOT/x

This node has only a name section.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/x")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	// NameSection
	if got.NameSection.Heading != "root/x" {
		t.Errorf("NameSection.Heading = %q, want %q", got.NameSection.Heading, "root/x")
	}
	if got.NameSection.Content != "This node has only a name section." {
		t.Errorf("NameSection.Content = %q, want %q",
			got.NameSection.Content, "This node has only a name section.")
	}
	if got.NameSection.Subsections != nil {
		t.Errorf("NameSection.Subsections = %v, want nil", got.NameSection.Subsections)
	}

	// Public / Private
	if got.Public != nil {
		t.Errorf("Public = %v, want nil", got.Public)
	}
	if got.Private != nil {
		t.Errorf("Private = %v, want nil", got.Private)
	}
}

// TestParseNode_FullNode verifies parsing of a node with name, public, and
// multiple private sections, including public subsections.
func TestParseNode_FullNode(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/payments/fees", `---
version: 5
parent_version: 2
depends_on:
  - path: ROOT/architecture/backend
    version: 3
implements:
  - internal/fees/fees.go
---
# ROOT/payments/fees

Calculates transaction fees.

# Public

## Interface

Fee calculation types and functions.

## Constraints

Maximum fee is 5%.

# Implementation

Step-by-step logic for fee calculation.

# Decisions

Chose percentage-based over flat fees.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/payments/fees")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	// NameSection
	if got.NameSection.Heading != "root/payments/fees" {
		t.Errorf("NameSection.Heading = %q, want %q", got.NameSection.Heading, "root/payments/fees")
	}
	if got.NameSection.Content != "Calculates transaction fees." {
		t.Errorf("NameSection.Content = %q, want %q",
			got.NameSection.Content, "Calculates transaction fees.")
	}

	// Public section
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got.Public.Heading != "public" {
		t.Errorf("Public.Heading = %q, want %q", got.Public.Heading, "public")
	}
	// No content before first ## inside # Public
	if got.Public.Content != "" {
		t.Errorf("Public.Content = %q, want %q", got.Public.Content, "")
	}
	if len(got.Public.Subsections) != 2 {
		t.Fatalf("Public.Subsections length = %d, want 2", len(got.Public.Subsections))
	}
	if got.Public.Subsections[0].Heading != "interface" {
		t.Errorf("Public.Subsections[0].Heading = %q, want %q",
			got.Public.Subsections[0].Heading, "interface")
	}
	if got.Public.Subsections[0].Content != "Fee calculation types and functions." {
		t.Errorf("Public.Subsections[0].Content = %q, want %q",
			got.Public.Subsections[0].Content, "Fee calculation types and functions.")
	}
	if got.Public.Subsections[1].Heading != "constraints" {
		t.Errorf("Public.Subsections[1].Heading = %q, want %q",
			got.Public.Subsections[1].Heading, "constraints")
	}
	if got.Public.Subsections[1].Content != "Maximum fee is 5%." {
		t.Errorf("Public.Subsections[1].Content = %q, want %q",
			got.Public.Subsections[1].Content, "Maximum fee is 5%.")
	}

	// Private sections
	if len(got.Private) != 2 {
		t.Fatalf("Private length = %d, want 2", len(got.Private))
	}
	if got.Private[0].Heading != "implementation" {
		t.Errorf("Private[0].Heading = %q, want %q", got.Private[0].Heading, "implementation")
	}
	if got.Private[0].Content != "Step-by-step logic for fee calculation." {
		t.Errorf("Private[0].Content = %q, want %q",
			got.Private[0].Content, "Step-by-step logic for fee calculation.")
	}
	if got.Private[1].Heading != "decisions" {
		t.Errorf("Private[1].Heading = %q, want %q", got.Private[1].Heading, "decisions")
	}
	if got.Private[1].Content != "Chose percentage-based over flat fees." {
		t.Errorf("Private[1].Content = %q, want %q",
			got.Private[1].Content, "Chose percentage-based over flat fees.")
	}
}

// TestParseNode_TestNodeBody verifies that a test node (TEST/ prefix) is
// parsed correctly and that level-3 headings inside a subsection appear as
// raw markdown content.
func TestParseNode_TestNodeBody(t *testing.T) {
	dir := t.TempDir()
	// TEST/x resolves to code-from-spec/spec/x/default.test.md
	testWriteNode(t, dir, "TEST/x", `---
version: 2
subject_version: 5
implements:
  - internal/x/x_test.go
---
# TEST/x

Test cases for x.

## Happy path

### Case one

Check basic behavior.
`)
	testChdir(t, dir)

	got, err := ParseNode("TEST/x")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	if got.NameSection.Heading != "test/x" {
		t.Errorf("NameSection.Heading = %q, want %q", got.NameSection.Heading, "test/x")
	}
	if got.NameSection.Content != "Test cases for x." {
		t.Errorf("NameSection.Content = %q, want %q",
			got.NameSection.Content, "Test cases for x.")
	}
	if len(got.NameSection.Subsections) != 1 {
		t.Fatalf("NameSection.Subsections length = %d, want 1", len(got.NameSection.Subsections))
	}
	if got.NameSection.Subsections[0].Heading != "happy path" {
		t.Errorf("NameSection.Subsections[0].Heading = %q, want %q",
			got.NameSection.Subsections[0].Heading, "happy path")
	}
	// The subsection content must contain the ### heading and its text.
	sub0Content := got.NameSection.Subsections[0].Content
	if !strings.Contains(sub0Content, "### Case one") {
		t.Errorf("Subsections[0].Content does not contain '### Case one'; got: %q", sub0Content)
	}
	if !strings.Contains(sub0Content, "Check basic behavior.") {
		t.Errorf("Subsections[0].Content does not contain 'Check basic behavior.'; got: %q", sub0Content)
	}
}

// TestParseNode_NoPublicSection verifies that a node without a # Public
// section has Public = nil.
func TestParseNode_NoPublicSection(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/decisions", `---
version: 1
---
# ROOT/decisions

Architecture decisions.

# Rationale

Why we chose this approach.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/decisions")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	if got.Public != nil {
		t.Errorf("Public = %v, want nil", got.Public)
	}
	if got.NameSection.Heading != "root/decisions" {
		t.Errorf("NameSection.Heading = %q, want %q", got.NameSection.Heading, "root/decisions")
	}
	if len(got.Private) != 1 {
		t.Fatalf("Private length = %d, want 1", len(got.Private))
	}
	if got.Private[0].Heading != "rationale" {
		t.Errorf("Private[0].Heading = %q, want %q", got.Private[0].Heading, "rationale")
	}
}

// TestParseNode_PublicContentBeforeFirstSubsection verifies that content
// appearing before the first ## within # Public is captured in Public.Content.
func TestParseNode_PublicContentBeforeFirstSubsection(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/a", `---
version: 1
---
# ROOT/a

Intent.

# Public

This is direct content of the public section.

## Interface

Types and functions.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/a")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got.Public.Content != "This is direct content of the public section." {
		t.Errorf("Public.Content = %q, want %q",
			got.Public.Content, "This is direct content of the public section.")
	}
	if len(got.Public.Subsections) != 1 {
		t.Fatalf("Public.Subsections length = %d, want 1", len(got.Public.Subsections))
	}
	if got.Public.Subsections[0].Heading != "interface" {
		t.Errorf("Public.Subsections[0].Heading = %q, want %q",
			got.Public.Subsections[0].Heading, "interface")
	}
}

// ---------------------------------------------------------------------------
// Heading normalization
// ---------------------------------------------------------------------------

// TestParseNode_CaseInsensitivePublicDetection verifies that "# PUBLIC" is
// recognized as the public section via case-insensitive normalization.
func TestParseNode_CaseInsensitivePublicDetection(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/c", `---
version: 1
---
# ROOT/c

Intent.

# PUBLIC

## Interface

Content.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/c")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got.Public.Heading != "public" {
		t.Errorf("Public.Heading = %q, want %q", got.Public.Heading, "public")
	}
}

// TestParseNode_PublicMixedCaseAndWhitespace verifies that heading text with
// mixed case and extra whitespace still resolves to "public".
func TestParseNode_PublicMixedCaseAndWhitespace(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/d", `---
version: 1
---
# ROOT/d

Intent.

#   PuBLiC

## Interface

Content.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/d")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got.Public.Heading != "public" {
		t.Errorf("Public.Heading = %q, want %q", got.Public.Heading, "public")
	}
}

// TestParseNode_NodeNameWithVariedWhitespace verifies that the node name
// heading is normalized even when extra leading whitespace is present.
func TestParseNode_NodeNameWithVariedWhitespace(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/e", `---
version: 1
---
#    ROOT/e

Intent.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/e")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.NameSection.Heading != "root/e" {
		t.Errorf("NameSection.Heading = %q, want %q", got.NameSection.Heading, "root/e")
	}
}

// TestParseNode_SubsectionHeadingsNormalized verifies that subsection headings
// within # Public are normalized (lowercased and trimmed).
func TestParseNode_SubsectionHeadingsNormalized(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/f", `---
version: 1
---
# ROOT/f

Intent.

# Public

##   Interface

Types.

## CONSTRAINTS

Rules.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/f")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if len(got.Public.Subsections) != 2 {
		t.Fatalf("Public.Subsections length = %d, want 2", len(got.Public.Subsections))
	}
	if got.Public.Subsections[0].Heading != "interface" {
		t.Errorf("Subsections[0].Heading = %q, want %q", got.Public.Subsections[0].Heading, "interface")
	}
	if got.Public.Subsections[1].Heading != "constraints" {
		t.Errorf("Subsections[1].Heading = %q, want %q", got.Public.Subsections[1].Heading, "constraints")
	}
}

// TestParseNode_TabCharactersInHeadingWhitespace verifies that tab characters
// surrounding a subsection heading are treated as whitespace and stripped.
func TestParseNode_TabCharactersInHeadingWhitespace(t *testing.T) {
	dir := t.TempDir()
	// The ## line contains tab characters around "Interface".
	testWriteNode(t, dir, "ROOT/g", "---\nversion: 1\n---\n# ROOT/g\n\nIntent.\n\n# Public\n\n## \tInterface\t\n\nContent.\n")
	testChdir(t, dir)

	got, err := ParseNode("ROOT/g")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if len(got.Public.Subsections) != 1 {
		t.Fatalf("Public.Subsections length = %d, want 1", len(got.Public.Subsections))
	}
	if got.Public.Subsections[0].Heading != "interface" {
		t.Errorf("Subsections[0].Heading = %q, want %q", got.Public.Subsections[0].Heading, "interface")
	}
}

// ---------------------------------------------------------------------------
// Content extraction
// ---------------------------------------------------------------------------

// TestParseNode_Level3AndDeeperAreContent verifies that level-3 (###) and
// level-4 (####) headings appear as raw markdown inside subsection content
// and are not treated as structural delimiters.
func TestParseNode_Level3AndDeeperAreContent(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/h", `---
version: 1
---
# ROOT/h

Intent.

# Public

## Interface

### Types

Type definitions here.

#### Nested detail

Even deeper content.

## Constraints

### Rule one

Details.
`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/h")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if len(got.Public.Subsections) != 2 {
		t.Fatalf("Public.Subsections length = %d, want 2", len(got.Public.Subsections))
	}

	interfaceContent := got.Public.Subsections[0].Content
	for _, want := range []string{"### Types", "Type definitions here.", "#### Nested detail", "Even deeper content."} {
		if !strings.Contains(interfaceContent, want) {
			t.Errorf("interface subsection content missing %q; got: %q", want, interfaceContent)
		}
	}

	constraintsContent := got.Public.Subsections[1].Content
	for _, want := range []string{"### Rule one", "Details."} {
		if !strings.Contains(constraintsContent, want) {
			t.Errorf("constraints subsection content missing %q; got: %q", want, constraintsContent)
		}
	}
}

// TestParseNode_FencedCodeBlockWithHeadingLikeContent verifies that `#` and
// `##` inside a fenced code block are not treated as structural headings.
func TestParseNode_FencedCodeBlockWithHeadingLikeContent(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/i", "---\nversion: 1\n---\n# ROOT/i\n\nIntent.\n\n# Public\n\n## Interface\n\n~~~go\n// # This is not a heading\n// ## Neither is this\nfunc Foo() {}\n~~~\n\nAfter the code block.\n")
	testChdir(t, dir)

	got, err := ParseNode("ROOT/i")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	// There should be exactly 1 subsection (## Interface). The # and ## inside
	// the code block must NOT produce additional sections or subsections.
	if len(got.Public.Subsections) != 1 {
		t.Fatalf("Public.Subsections length = %d, want 1 (code block headings must not be structural)",
			len(got.Public.Subsections))
	}
	subContent := got.Public.Subsections[0].Content
	if !strings.Contains(subContent, "# This is not a heading") {
		t.Errorf("interface content missing code block line; got: %q", subContent)
	}
	if !strings.Contains(subContent, "After the code block.") {
		t.Errorf("interface content missing trailing text; got: %q", subContent)
	}
}

// TestParseNode_ContentBetweenSectionsIsTrimmed verifies that leading and
// trailing blank lines are stripped from Section.Content and Subsection.Content.
func TestParseNode_ContentBetweenSectionsIsTrimmed(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/j", `---
version: 1
---
# ROOT/j

Intent.

# Public



Content with surrounding blank lines.



## Interface

Also surrounded.

`)
	testChdir(t, dir)

	got, err := ParseNode("ROOT/j")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got.Public.Content != "Content with surrounding blank lines." {
		t.Errorf("Public.Content = %q, want %q",
			got.Public.Content, "Content with surrounding blank lines.")
	}
	if len(got.Public.Subsections) != 1 {
		t.Fatalf("Public.Subsections length = %d, want 1", len(got.Public.Subsections))
	}
	if got.Public.Subsections[0].Content != "Also surrounded." {
		t.Errorf("Subsections[0].Content = %q, want %q",
			got.Public.Subsections[0].Content, "Also surrounded.")
	}
}

// ---------------------------------------------------------------------------
// Validation errors
// ---------------------------------------------------------------------------

// TestParseNode_FileDoesNotExist verifies that ErrRead is returned when the
// logical name resolves to a file that does not exist.
func TestParseNode_FileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	_, err := ParseNode("ROOT/nonexistent")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrRead")
	}
	if !errors.Is(err, ErrRead) {
		t.Errorf("errors.Is(err, ErrRead) = false; err = %v", err)
	}
}

// TestParseNode_NoFrontmatterDelimiters verifies that ErrFrontmatterMissing is
// returned when the file contains no --- delimiters.
func TestParseNode_NoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/m", `# ROOT/m

Just text.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/m")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrFrontmatterMissing")
	}
	if !errors.Is(err, ErrFrontmatterMissing) {
		t.Errorf("errors.Is(err, ErrFrontmatterMissing) = false; err = %v", err)
	}
}

// TestParseNode_ContentBeforeFirstHeading verifies that ErrUnexpectedContent
// is returned when text appears before the first level-1 heading.
func TestParseNode_ContentBeforeFirstHeading(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/o", `---
version: 1
---
Some text before any heading.

# ROOT/o

Intent.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/o")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrUnexpectedContent")
	}
	if !errors.Is(err, ErrUnexpectedContent) {
		t.Errorf("errors.Is(err, ErrUnexpectedContent) = false; err = %v", err)
	}
}

// TestParseNode_Level2HeadingBeforeLevel1 verifies that ErrUnexpectedContent
// is returned when a ## heading appears before any # heading.
func TestParseNode_Level2HeadingBeforeLevel1(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/p", `---
version: 1
---
## Orphan subsection

# ROOT/p

Intent.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/p")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrUnexpectedContent")
	}
	if !errors.Is(err, ErrUnexpectedContent) {
		t.Errorf("errors.Is(err, ErrUnexpectedContent) = false; err = %v", err)
	}
}

// TestParseNode_NodeNameDoesNotMatchLogicalName verifies that ErrInvalidNodeName
// is returned when the first # heading does not normalize to the logical name.
func TestParseNode_NodeNameDoesNotMatchLogicalName(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/q", `---
version: 1
---
# ROOT/wrong

Intent.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/q")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrInvalidNodeName")
	}
	if !errors.Is(err, ErrInvalidNodeName) {
		t.Errorf("errors.Is(err, ErrInvalidNodeName) = false; err = %v", err)
	}
}

// TestParseNode_NodeNameCaseMismatchIsNotAnError verifies that normalization
// makes "root/Q" and "ROOT/q" equal, so no error is returned.
func TestParseNode_NodeNameCaseMismatchIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/q", `---
version: 1
---
# root/Q

Intent.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/q")
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v (case mismatch should be allowed via normalization)", err)
	}
}

// TestParseNode_DuplicatePublicSectionSameCase verifies that ErrDuplicatePublic
// is returned when two # Public headings with the same casing exist.
func TestParseNode_DuplicatePublicSectionSameCase(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/r", `---
version: 1
---
# ROOT/r

Intent.

# Public

First public.

# Public

Second public.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/r")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicatePublic")
	}
	if !errors.Is(err, ErrDuplicatePublic) {
		t.Errorf("errors.Is(err, ErrDuplicatePublic) = false; err = %v", err)
	}
}

// TestParseNode_DuplicatePublicSectionDifferentCase verifies that ErrDuplicatePublic
// is returned even when the two # Public headings differ in case.
func TestParseNode_DuplicatePublicSectionDifferentCase(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/s", `---
version: 1
---
# ROOT/s

Intent.

# Public

First.

# PUBLIC

Second.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/s")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicatePublic")
	}
	if !errors.Is(err, ErrDuplicatePublic) {
		t.Errorf("errors.Is(err, ErrDuplicatePublic) = false; err = %v", err)
	}
}

// TestParseNode_DuplicateSubsectionInPublicSameCase verifies that
// ErrDuplicateSubsection is returned when two ## headings in # Public share
// the same normalized name.
func TestParseNode_DuplicateSubsectionInPublicSameCase(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/t", `---
version: 1
---
# ROOT/t

Intent.

# Public

## Interface

First interface.

## Interface

Second interface.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/t")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicateSubsection")
	}
	if !errors.Is(err, ErrDuplicateSubsection) {
		t.Errorf("errors.Is(err, ErrDuplicateSubsection) = false; err = %v", err)
	}
}

// TestParseNode_DuplicateSubsectionInPublicDifferentCase verifies that
// ErrDuplicateSubsection is returned even when the duplicate headings differ
// in case (normalization makes them equal).
func TestParseNode_DuplicateSubsectionInPublicDifferentCase(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/u", `---
version: 1
---
# ROOT/u

Intent.

# Public

## Interface

First.

## INTERFACE

Second.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/u")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicateSubsection")
	}
	if !errors.Is(err, ErrDuplicateSubsection) {
		t.Errorf("errors.Is(err, ErrDuplicateSubsection) = false; err = %v", err)
	}
}

// TestParseNode_DuplicateSubsectionInPublicWhitespaceVariation verifies that
// ErrDuplicateSubsection is returned when two headings differ only in
// whitespace (normalization collapses and trims whitespace).
func TestParseNode_DuplicateSubsectionInPublicWhitespaceVariation(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/v", `---
version: 1
---
# ROOT/v

Intent.

# Public

## Interface

First.

##   Interface

Second.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/v")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicateSubsection")
	}
	if !errors.Is(err, ErrDuplicateSubsection) {
		t.Errorf("errors.Is(err, ErrDuplicateSubsection) = false; err = %v", err)
	}
}

// TestParseNode_FirstElementIsParagraph verifies that ErrUnexpectedContent is
// returned when the body starts with a paragraph rather than a # heading.
func TestParseNode_FirstElementIsParagraph(t *testing.T) {
	dir := t.TempDir()
	testWriteNode(t, dir, "ROOT/w", `---
version: 1
---
This is a paragraph, not a heading.
`)
	testChdir(t, dir)

	_, err := ParseNode("ROOT/w")
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrUnexpectedContent")
	}
	if !errors.Is(err, ErrUnexpectedContent) {
		t.Errorf("errors.Is(err, ErrUnexpectedContent) = false; err = %v", err)
	}
}
