// code-from-spec: TEST/tech_design/internal/parsenode@v7

// Package parsenode tests cover the ParseNode function.
// Tests use t.TempDir() to create isolated spec file trees and change the
// working directory before calling ParseNode so that logicalnames.PathFromLogicalName
// resolves against the temp root.
package parsenode

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// testWriteNode creates the file expected by logicalnames.PathFromLogicalName for
// the given logicalName under dir, writing content as the file body.
// For example, logicalName "ROOT/x/y" → <dir>/code-from-spec/x/y/_node.md
// For "ROOT" → <dir>/code-from-spec/_node.md
// For "TEST/x" → <dir>/code-from-spec/x/default.test.md
// For "TEST/x(name)" → <dir>/code-from-spec/x/name.test.md
func testWriteNode(t *testing.T, dir, logicalName, content string) {
	t.Helper()

	// Derive the relative path the same way logicalnames.PathFromLogicalName does.
	relPath := testPathForLogicalName(t, logicalName)

	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("testWriteNode: MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("testWriteNode: WriteFile: %v", err)
	}
}

// testPathForLogicalName computes the relative file path for a logical name,
// mirroring the rules in logicalnames.PathFromLogicalName.
// This is used only by the test helper to create fixture files.
func testPathForLogicalName(t *testing.T, logicalName string) string {
	t.Helper()
	// Strip qualifier (parenthetical suffix) if present.
	base := logicalName
	qualifier := ""
	if idx := indexByte(base, '('); idx >= 0 {
		qualifier = base[idx+1 : len(base)-1] // inside the parens
		base = base[:idx]
	}

	switch {
	case base == "ROOT":
		return "code-from-spec/_node.md"
	case len(base) > 5 && base[:5] == "ROOT/":
		path := base[5:]
		return "code-from-spec/" + path + "/_node.md"
	case base == "TEST":
		return "code-from-spec/default.test.md"
	case len(base) > 5 && base[:5] == "TEST/":
		path := base[5:]
		if qualifier != "" {
			return "code-from-spec/" + path + "/" + qualifier + ".test.md"
		}
		return "code-from-spec/" + path + "/default.test.md"
	default:
		t.Fatalf("testPathForLogicalName: unrecognized logical name %q", logicalName)
		return ""
	}
}

// indexByte returns the index of the first occurrence of b in s, or -1.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// testChdir changes the working directory to dir for the duration of the test,
// restoring the original directory when the test ends.
func testChdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("testChdir: Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("testChdir: Chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("testChdir cleanup: Chdir: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Happy path tests
// ---------------------------------------------------------------------------

// TestMinimalNode verifies parsing of a node that only has a name section.
func TestMinimalNode(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/x"
	content := `---
version: 3
parent_version: 1
---
# ROOT/x

This node has only a name section.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	// NameSection checks.
	if got, want := body.NameSection.Heading, "root/x"; got != want {
		t.Errorf("NameSection.Heading = %q, want %q", got, want)
	}
	if got, want := body.NameSection.Content, "This node has only a name section."; got != want {
		t.Errorf("NameSection.Content = %q, want %q", got, want)
	}
	if body.NameSection.Subsections != nil {
		t.Errorf("NameSection.Subsections = %v, want nil", body.NameSection.Subsections)
	}

	// Public and Private must be absent.
	if body.Public != nil {
		t.Errorf("Public = %v, want nil", body.Public)
	}
	if body.Private != nil {
		t.Errorf("Private = %v, want nil", body.Private)
	}
}

// TestFullNode verifies parsing of a node with name, public, and private sections.
func TestFullNode(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/payments/fees"
	content := `---
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
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	// NameSection.
	if got, want := body.NameSection.Heading, "root/payments/fees"; got != want {
		t.Errorf("NameSection.Heading = %q, want %q", got, want)
	}
	if got, want := body.NameSection.Content, "Calculates transaction fees."; got != want {
		t.Errorf("NameSection.Content = %q, want %q", got, want)
	}

	// Public section.
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := body.Public.Heading, "public"; got != want {
		t.Errorf("Public.Heading = %q, want %q", got, want)
	}
	// Public.Content should be empty — no content before the first ##.
	if got, want := body.Public.Content, ""; got != want {
		t.Errorf("Public.Content = %q, want %q", got, want)
	}
	if got, want := len(body.Public.Subsections), 2; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d", got, want)
	}
	if got, want := body.Public.Subsections[0].Heading, "interface"; got != want {
		t.Errorf("Public.Subsections[0].Heading = %q, want %q", got, want)
	}
	if got, want := body.Public.Subsections[0].Content, "Fee calculation types and functions."; got != want {
		t.Errorf("Public.Subsections[0].Content = %q, want %q", got, want)
	}
	if got, want := body.Public.Subsections[1].Heading, "constraints"; got != want {
		t.Errorf("Public.Subsections[1].Heading = %q, want %q", got, want)
	}
	if got, want := body.Public.Subsections[1].Content, "Maximum fee is 5%."; got != want {
		t.Errorf("Public.Subsections[1].Content = %q, want %q", got, want)
	}

	// Private sections.
	if got, want := len(body.Private), 2; got != want {
		t.Fatalf("len(Private) = %d, want %d", got, want)
	}
	if got, want := body.Private[0].Heading, "implementation"; got != want {
		t.Errorf("Private[0].Heading = %q, want %q", got, want)
	}
	if got, want := body.Private[0].Content, "Step-by-step logic for fee calculation."; got != want {
		t.Errorf("Private[0].Content = %q, want %q", got, want)
	}
	if got, want := body.Private[1].Heading, "decisions"; got != want {
		t.Errorf("Private[1].Heading = %q, want %q", got, want)
	}
	if got, want := body.Private[1].Content, "Chose percentage-based over flat fees."; got != want {
		t.Errorf("Private[1].Content = %q, want %q", got, want)
	}
}

// TestTestNodeBody verifies parsing of a TEST/ node whose name section has subsections.
func TestTestNodeBody(t *testing.T) {
	dir := t.TempDir()
	logicalName := "TEST/x"
	// The heading must match the logical name. TEST/x → first heading "TEST/x".
	content := `---
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
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	if got, want := body.NameSection.Heading, "test/x"; got != want {
		t.Errorf("NameSection.Heading = %q, want %q", got, want)
	}
	if got, want := body.NameSection.Content, "Test cases for x."; got != want {
		t.Errorf("NameSection.Content = %q, want %q", got, want)
	}
	if got, want := len(body.NameSection.Subsections), 1; got != want {
		t.Fatalf("len(NameSection.Subsections) = %d, want %d", got, want)
	}
	if got, want := body.NameSection.Subsections[0].Heading, "happy path"; got != want {
		t.Errorf("NameSection.Subsections[0].Heading = %q, want %q", got, want)
	}
	// The subsection content must include the ### Case one heading and its text.
	subContent := body.NameSection.Subsections[0].Content
	if subContent == "" {
		t.Error("NameSection.Subsections[0].Content is empty, want raw markdown including ### Case one")
	}
	// Check that the level-3 heading is present as raw markdown in the content.
	if !containsString(subContent, "### Case one") {
		t.Errorf("subsection content %q does not contain '### Case one'", subContent)
	}
	if !containsString(subContent, "Check basic behavior.") {
		t.Errorf("subsection content %q does not contain 'Check basic behavior.'", subContent)
	}
}

// TestNoPublicSection verifies that Public is nil when no # Public heading exists.
func TestNoPublicSection(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/decisions"
	content := `---
version: 1
---
# ROOT/decisions

Architecture decisions.

# Rationale

Why we chose this approach.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	if body.Public != nil {
		t.Errorf("Public = %v, want nil", body.Public)
	}
	if got, want := body.NameSection.Heading, "root/decisions"; got != want {
		t.Errorf("NameSection.Heading = %q, want %q", got, want)
	}
	if got, want := len(body.Private), 1; got != want {
		t.Fatalf("len(Private) = %d, want %d", got, want)
	}
	if got, want := body.Private[0].Heading, "rationale"; got != want {
		t.Errorf("Private[0].Heading = %q, want %q", got, want)
	}
}

// TestPublicContentBeforeFirstSubsection verifies content directly under # Public
// (before any ## heading) is captured in Public.Content.
func TestPublicContentBeforeFirstSubsection(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/a"
	content := `---
version: 1
---
# ROOT/a

Intent.

# Public

This is direct content of the public section.

## Interface

Types and functions.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}

	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := body.Public.Content, "This is direct content of the public section."; got != want {
		t.Errorf("Public.Content = %q, want %q", got, want)
	}
	if got, want := len(body.Public.Subsections), 1; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d", got, want)
	}
	if got, want := body.Public.Subsections[0].Heading, "interface"; got != want {
		t.Errorf("Public.Subsections[0].Heading = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Heading normalization tests
// ---------------------------------------------------------------------------

// TestCaseInsensitivePublicDetection verifies that # PUBLIC is treated as public.
func TestCaseInsensitivePublicDetection(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/c"
	content := `---
version: 1
---
# ROOT/c

Intent.

# PUBLIC

## Interface

Content.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := body.Public.Heading, "public"; got != want {
		t.Errorf("Public.Heading = %q, want %q", got, want)
	}
}

// TestPublicMixedCaseAndWhitespace verifies normalization of leading/trailing
// whitespace and mixed case on the # Public heading.
func TestPublicMixedCaseAndWhitespace(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/d"
	content := `---
version: 1
---
# ROOT/d

Intent.

#   PuBLiC

## Interface

Content.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := body.Public.Heading, "public"; got != want {
		t.Errorf("Public.Heading = %q, want %q", got, want)
	}
}

// TestNodeNameVariedWhitespace verifies normalization of whitespace in the
// node name heading.
func TestNodeNameVariedWhitespace(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/e"
	content := `---
version: 1
---
#    ROOT/e

Intent.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if got, want := body.NameSection.Heading, "root/e"; got != want {
		t.Errorf("NameSection.Heading = %q, want %q", got, want)
	}
}

// TestSubsectionHeadingsNormalized verifies that ## headings in # Public are
// normalized (whitespace + case folding).
func TestSubsectionHeadingsNormalized(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/f"
	content := `---
version: 1
---
# ROOT/f

Intent.

# Public

##   Interface

Types.

## CONSTRAINTS

Rules.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := len(body.Public.Subsections), 2; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d", got, want)
	}
	if got, want := body.Public.Subsections[0].Heading, "interface"; got != want {
		t.Errorf("Public.Subsections[0].Heading = %q, want %q", got, want)
	}
	if got, want := body.Public.Subsections[1].Heading, "constraints"; got != want {
		t.Errorf("Public.Subsections[1].Heading = %q, want %q", got, want)
	}
}

// TestTabCharactersInHeadingWhitespace verifies that tab characters around the
// heading text are treated as whitespace during normalization.
func TestTabCharactersInHeadingWhitespace(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/g"
	// The ## line intentionally contains tab characters around "Interface".
	content := "---\nversion: 1\n---\n# ROOT/g\n\nIntent.\n\n# Public\n\n## \tInterface\t\n\nContent.\n"
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := len(body.Public.Subsections), 1; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d", got, want)
	}
	if got, want := body.Public.Subsections[0].Heading, "interface"; got != want {
		t.Errorf("Public.Subsections[0].Heading = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Content extraction tests
// ---------------------------------------------------------------------------

// TestLevel3AndDeeperAreContent verifies that ### and #### headings inside a
// subsection appear as raw markdown in the subsection content.
func TestLevel3AndDeeperAreContent(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/h"
	content := `---
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
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := len(body.Public.Subsections), 2; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d", got, want)
	}

	// Interface subsection must contain the ### and #### headings as raw markdown.
	iface := body.Public.Subsections[0].Content
	for _, want := range []string{"### Types", "Type definitions here.", "#### Nested detail", "Even deeper content."} {
		if !containsString(iface, want) {
			t.Errorf("interface subsection content %q does not contain %q", iface, want)
		}
	}

	// Constraints subsection must contain ### Rule one as raw markdown.
	constraints := body.Public.Subsections[1].Content
	for _, want := range []string{"### Rule one", "Details."} {
		if !containsString(constraints, want) {
			t.Errorf("constraints subsection content %q does not contain %q", constraints, want)
		}
	}
}

// TestFencedCodeBlockWithHeadingLikeContent verifies that # and ## inside a
// fenced code block are not treated as structural headings.
func TestFencedCodeBlockWithHeadingLikeContent(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/i"
	content := "---\nversion: 1\n---\n# ROOT/i\n\nIntent.\n\n# Public\n\n## Interface\n\n~~~go\n// # This is not a heading\n// ## Neither is this\nfunc Foo() {}\n~~~\n\nAfter the code block.\n"
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	// There must be exactly one subsection — the # and ## inside the code block
	// must NOT have been treated as headings.
	if got, want := len(body.Public.Subsections), 1; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d — code block headings may have been parsed as structural", got, want)
	}

	iface := body.Public.Subsections[0].Content
	// The fenced code block and the text after it must both appear in the content.
	for _, want := range []string{"# This is not a heading", "Neither is this", "func Foo()", "After the code block."} {
		if !containsString(iface, want) {
			t.Errorf("interface subsection content %q does not contain %q", iface, want)
		}
	}
}

// TestContentTrimmed verifies that leading and trailing blank lines in section
// and subsection content are trimmed.
func TestContentTrimmed(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/j"
	content := `---
version: 1
---
# ROOT/j

Intent.

# Public



Content with surrounding blank lines.



## Interface

Also surrounded.

`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	body, err := ParseNode(logicalName)
	if err != nil {
		t.Fatalf("ParseNode returned unexpected error: %v", err)
	}
	if body.Public == nil {
		t.Fatal("Public = nil, want non-nil")
	}
	if got, want := body.Public.Content, "Content with surrounding blank lines."; got != want {
		t.Errorf("Public.Content = %q, want %q", got, want)
	}
	if got, want := len(body.Public.Subsections), 1; got != want {
		t.Fatalf("len(Public.Subsections) = %d, want %d", got, want)
	}
	if got, want := body.Public.Subsections[0].Content, "Also surrounded."; got != want {
		t.Errorf("Public.Subsections[0].Content = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Validation error tests
// ---------------------------------------------------------------------------

// TestFileDoesNotExist verifies ErrRead is returned when the file is absent.
func TestFileDoesNotExist(t *testing.T) {
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

// TestNoFrontmatterDelimiters verifies ErrFrontmatterMissing when the file has
// no --- delimiters.
func TestNoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/m"
	// File deliberately omits frontmatter delimiters.
	content := `# ROOT/m

Just text.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrFrontmatterMissing")
	}
	if !errors.Is(err, ErrFrontmatterMissing) {
		t.Errorf("errors.Is(err, ErrFrontmatterMissing) = false; err = %v", err)
	}
}

// TestContentBeforeFirstHeading verifies ErrUnexpectedContent when non-heading
// text appears before the first # heading.
func TestContentBeforeFirstHeading(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/o"
	content := `---
version: 1
---
Some text before any heading.

# ROOT/o

Intent.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrUnexpectedContent")
	}
	if !errors.Is(err, ErrUnexpectedContent) {
		t.Errorf("errors.Is(err, ErrUnexpectedContent) = false; err = %v", err)
	}
}

// TestLevel2HeadingBeforeAnyLevel1 verifies ErrUnexpectedContent when a ##
// heading appears before any # heading.
func TestLevel2HeadingBeforeAnyLevel1(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/p"
	content := `---
version: 1
---
## Orphan subsection

# ROOT/p

Intent.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrUnexpectedContent")
	}
	if !errors.Is(err, ErrUnexpectedContent) {
		t.Errorf("errors.Is(err, ErrUnexpectedContent) = false; err = %v", err)
	}
}

// TestNodeNameDoesNotMatchLogicalName verifies ErrInvalidNodeName when the first
// level-1 heading text does not match the logical name.
func TestNodeNameDoesNotMatchLogicalName(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/q"
	content := `---
version: 1
---
# ROOT/wrong

Intent.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrInvalidNodeName")
	}
	if !errors.Is(err, ErrInvalidNodeName) {
		t.Errorf("errors.Is(err, ErrInvalidNodeName) = false; err = %v", err)
	}
}

// TestNodeNameCaseMismatchIsNotError verifies that case differences between the
// heading and the logical name are tolerated by normalization.
func TestNodeNameCaseMismatchIsNotError(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/q"
	// Heading "root/Q" normalizes to "root/q", same as logical name "ROOT/q".
	content := `---
version: 1
---
# root/Q

Intent.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err != nil {
		t.Errorf("ParseNode returned unexpected error: %v", err)
	}
}

// TestDuplicatePublicSameCaseerifies ErrDuplicatePublic when two # Public
// headings with the same case exist.
func TestDuplicatePublicSameCase(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/r"
	content := `---
version: 1
---
# ROOT/r

Intent.

# Public

First public.

# Public

Second public.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicatePublic")
	}
	if !errors.Is(err, ErrDuplicatePublic) {
		t.Errorf("errors.Is(err, ErrDuplicatePublic) = false; err = %v", err)
	}
}

// TestDuplicatePublicDifferentCase verifies ErrDuplicatePublic when two public
// headings differ only in case.
func TestDuplicatePublicDifferentCase(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/s"
	content := `---
version: 1
---
# ROOT/s

Intent.

# Public

First.

# PUBLIC

Second.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicatePublic")
	}
	if !errors.Is(err, ErrDuplicatePublic) {
		t.Errorf("errors.Is(err, ErrDuplicatePublic) = false; err = %v", err)
	}
}

// TestDuplicateSubsectionSameCase verifies ErrDuplicateSubsection when two ##
// headings inside # Public have identical text.
func TestDuplicateSubsectionSameCase(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/t"
	content := `---
version: 1
---
# ROOT/t

Intent.

# Public

## Interface

First interface.

## Interface

Second interface.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicateSubsection")
	}
	if !errors.Is(err, ErrDuplicateSubsection) {
		t.Errorf("errors.Is(err, ErrDuplicateSubsection) = false; err = %v", err)
	}
}

// TestDuplicateSubsectionDifferentCase verifies ErrDuplicateSubsection when two
// ## headings in # Public differ only by case.
func TestDuplicateSubsectionDifferentCase(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/u"
	content := `---
version: 1
---
# ROOT/u

Intent.

# Public

## Interface

First.

## INTERFACE

Second.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicateSubsection")
	}
	if !errors.Is(err, ErrDuplicateSubsection) {
		t.Errorf("errors.Is(err, ErrDuplicateSubsection) = false; err = %v", err)
	}
}

// TestDuplicateSubsectionWhitespaceVariation verifies ErrDuplicateSubsection when
// two ## headings in # Public differ only by whitespace (normalized to equal).
func TestDuplicateSubsectionWhitespaceVariation(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/v"
	content := `---
version: 1
---
# ROOT/v

Intent.

# Public

## Interface

First.

##   Interface

Second.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrDuplicateSubsection")
	}
	if !errors.Is(err, ErrDuplicateSubsection) {
		t.Errorf("errors.Is(err, ErrDuplicateSubsection) = false; err = %v", err)
	}
}

// TestFirstElementIsParagraph verifies ErrUnexpectedContent when the body starts
// with a paragraph rather than a level-1 heading.
func TestFirstElementIsParagraph(t *testing.T) {
	dir := t.TempDir()
	logicalName := "ROOT/w"
	content := `---
version: 1
---
This is a paragraph, not a heading.
`
	testWriteNode(t, dir, logicalName, content)
	testChdir(t, dir)

	_, err := ParseNode(logicalName)
	if err == nil {
		t.Fatal("ParseNode returned nil error, want ErrUnexpectedContent")
	}
	if !errors.Is(err, ErrUnexpectedContent) {
		t.Errorf("errors.Is(err, ErrUnexpectedContent) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test utility
// ---------------------------------------------------------------------------

// containsString reports whether substr appears anywhere in s.
func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
