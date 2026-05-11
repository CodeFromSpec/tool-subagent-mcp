// code-from-spec: ROOT/tech_design/internal/parsenode@v36

// Package parsenode parses the body of a spec node file and returns
// a structured representation of all sections.
//
// The file format is CommonMark with YAML frontmatter. Only level-1
// and level-2 headings are structural delimiters. Level-3+ headings
// are treated as content within their enclosing section or subsection.
package parsenode

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/normalizename"
)

// Sentinel errors. All errors returned by ParseNode wrap one of these so
// callers can use errors.Is() to match error kinds.
var (
	// ErrRead is returned when the spec file cannot be read.
	ErrRead = errors.New("error reading file")

	// ErrFrontmatterMissing is returned when no "---" delimiters are found
	// at the top of the file.
	ErrFrontmatterMissing = errors.New("frontmatter not found")

	// ErrUnexpectedContent is returned when non-heading content appears
	// before the first level-1 heading (or the document is empty).
	ErrUnexpectedContent = errors.New("unexpected content before first heading")

	// ErrInvalidNodeName is returned when the first level-1 heading does not
	// match the logical name supplied to ParseNode.
	ErrInvalidNodeName = errors.New("node name section does not match logical name")

	// ErrDuplicatePublic is returned when more than one level-1 heading
	// normalizes to "public".
	ErrDuplicatePublic = errors.New("duplicate public section")

	// ErrDuplicateSubsection is returned when two or more level-2 headings
	// within the "# Public" section have the same normalized text.
	ErrDuplicateSubsection = errors.New("duplicate subsection in public")
)

// Subsection represents a level-2 heading and its associated content within
// a Section. It is a structural unit within the "# Public" section.
type Subsection struct {
	// Heading is the normalized text of the level-2 heading.
	Heading string

	// Content is the raw markdown source between the end of the level-2
	// heading and the start of the next level-2 heading, the next level-1
	// heading, or the end of the document — whichever comes first.
	// Leading and trailing blank lines are trimmed.
	Content string
}

// Section represents a level-1 heading and everything that follows it until
// the next level-1 heading or the end of the document.
type Section struct {
	// Heading is the normalized text of the level-1 heading.
	Heading string

	// Content is the raw markdown source between the end of the level-1
	// heading and the start of the first level-2 heading within the section
	// (or the start of the next level-1 heading / end of document if there
	// are no level-2 headings). Leading and trailing blank lines are trimmed.
	Content string

	// Subsections holds all level-2 subsections found within this section.
	Subsections []Subsection
}

// NodeBody is the parsed representation of a spec node file.
//
// Invariants after a successful parse:
//   - NameSection is always populated (the first "# ..." heading).
//   - Public is nil when no "# Public" section exists.
//   - Private contains all other top-level sections, in document order.
type NodeBody struct {
	NameSection Section  // the first section — the node name section
	Public      *Section // the "# Public" section, or nil
	Private     []Section // all other top-level sections, in document order
}

// ParseNode resolves logicalName to a file path using PathFromLogicalName,
// reads the file, validates its structure, and returns a NodeBody.
//
// Steps:
//  1. Resolve logicalName → file path.
//  2. Read the file.
//  3. Skip the YAML frontmatter (two "---" delimiters).
//  4. Parse the body with goldmark.
//  5. Validate the first element is a level-1 heading.
//  6. Validate the first heading matches logicalName (after normalization).
//  7. Validate there are no duplicate "# Public" sections.
//  8. Validate there are no duplicate level-2 subsections inside "# Public".
//  9. Extract sections and subsections using raw byte offsets.
// 10. Classify sections into NameSection, Public, and Private.
//
// All returned errors wrap one of the sentinel variables so callers can use
// errors.Is().
func ParseNode(logicalName string) (*NodeBody, error) {
	// Step 1 — Resolve logical name to a file path.
	//
	// logicalnames.PathFromLogicalName strips any parenthetical qualifier
	// before resolving, so "ROOT/x(y)" and "ROOT/x" both map to the same
	// file. The returned path always uses forward slashes.
	filePath, ok := logicalnames.PathFromLogicalName(logicalName)
	if !ok {
		// The logical name format is not recognized; treat as read error.
		return nil, fmt.Errorf("%w: cannot resolve logical name %q", ErrRead, logicalName)
	}

	// Step 2 — Read the file.
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRead, filePath, err)
	}

	// Step 3 — Skip frontmatter.
	//
	// The frontmatter is enclosed between two "---" delimiter lines. We
	// discard everything up to and including the closing delimiter. The
	// rest is the body that will be parsed as CommonMark.
	body, err := skipFrontmatter(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFrontmatterMissing, filePath)
	}

	// Step 4 — Parse body as CommonMark using goldmark.
	//
	// The source byte slice is retained alongside the AST because goldmark
	// AST nodes hold byte offsets into it (not copies of the text).
	source := body
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(source))

	// Step 5 — Validate: the first direct child must be a level-1 heading.
	//
	// The spec requires that nothing (paragraphs, code blocks, etc.) appears
	// before the first "# ..." heading. An empty document is also invalid.
	firstChild := doc.FirstChild()
	if firstChild == nil {
		return nil, fmt.Errorf("%w: %s: document body is empty", ErrUnexpectedContent, filePath)
	}
	firstHeading, ok := firstChild.(*ast.Heading)
	if !ok || firstHeading.Level != 1 {
		return nil, fmt.Errorf(
			"%w: %s: first element after frontmatter is not a level-1 heading",
			ErrUnexpectedContent, filePath,
		)
	}

	// Step 6 — Validate: the first level-1 heading matches the logical name.
	//
	// Both the heading text and the logical name are normalized with
	// NormalizeName before comparison so that case and whitespace differences
	// are ignored.
	firstHeadingText := headingText(firstHeading, source)
	normalizedHeading := normalizename.NormalizeName(firstHeadingText)
	normalizedLogicalName := normalizename.NormalizeName(logicalName)
	if normalizedHeading != normalizedLogicalName {
		return nil, fmt.Errorf(
			"%w: %s: first heading %q does not match logical name %q",
			ErrInvalidNodeName, filePath, firstHeadingText, logicalName,
		)
	}

	// Step 7 — Validate: no duplicate "# Public" sections.
	//
	// Count level-1 headings whose normalized text equals "public". More
	// than one is a structural error.
	publicCount := 0
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if h, isHeading := child.(*ast.Heading); isHeading && h.Level == 1 {
			if normalizename.NormalizeName(headingText(h, source)) == "public" {
				publicCount++
			}
		}
	}
	if publicCount > 1 {
		return nil, fmt.Errorf("%w: %s", ErrDuplicatePublic, filePath)
	}

	// Step 8 — Validate: no duplicate level-2 subsections within "# Public".
	//
	// This is separate from section extraction so validation errors are
	// returned before any partial data is built.
	if err := validatePublicSubsections(doc, source, filePath); err != nil {
		return nil, err
	}

	// Step 9 — Extract sections from the AST using raw byte offsets.
	//
	// extractSections iterates direct children of the document root. Each
	// level-1 heading starts a new section; level-2 headings become
	// subsections of the current section. Content boundaries are determined
	// by byte offsets derived from goldmark's Lines() on heading nodes.
	sections, err := extractSections(doc, source)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrUnexpectedContent, filePath, err)
	}

	// Should never happen because we already validated a first heading exists.
	if len(sections) == 0 {
		return nil, fmt.Errorf("%w: %s: no sections found after extraction", ErrUnexpectedContent, filePath)
	}

	// Step 10 — Classify sections.
	//
	// Rule: the first section is the node name section. A section whose
	// normalized heading equals "public" is the public section (guaranteed
	// unique by step 7). All remaining sections are private.
	result := &NodeBody{
		NameSection: sections[0],
	}

	for i := 1; i < len(sections); i++ {
		s := sections[i]
		if s.Heading == "public" {
			// The spec guarantees at most one public section (validated above).
			// Take the address of a local copy so each iteration's address is unique.
			sCopy := s
			result.Public = &sCopy
		} else {
			result.Private = append(result.Private, s)
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Frontmatter helpers
// ---------------------------------------------------------------------------

// skipFrontmatter finds the closing "---" delimiter in raw and returns
// everything after it as the body. Returns an error when the expected two
// "---" delimiter lines cannot be found.
//
// The opening delimiter must appear at the very beginning of the file; the
// closing delimiter ends the frontmatter block. Both lines must contain
// exactly "---" (possibly followed by a newline or CRLF).
func skipFrontmatter(raw []byte) ([]byte, error) {
	// Consume the opening "---" line.
	afterOpen := consumeDelimiterLine(raw)
	if afterOpen < 0 {
		return nil, errors.New("opening '---' delimiter not found")
	}

	// Consume the closing "---" line (within the remainder).
	rest := raw[afterOpen:]
	afterClose := consumeDelimiterLine(rest)
	if afterClose < 0 {
		return nil, errors.New("closing '---' delimiter not found")
	}

	// The body is everything after the closing delimiter line.
	return rest[afterClose:], nil
}

// consumeDelimiterLine scans data line by line and returns the byte offset
// immediately after the first line that is exactly "---". Returns -1 when
// no such line is found. CRLF line endings are handled transparently.
func consumeDelimiterLine(data []byte) int {
	offset := 0
	for offset < len(data) {
		// Locate the end of the current line.
		nlIdx := bytes.IndexByte(data[offset:], '\n')
		var line []byte
		var nextOffset int
		if nlIdx < 0 {
			// Last line, no trailing newline.
			line = data[offset:]
			nextOffset = len(data)
		} else {
			line = data[offset : offset+nlIdx]
			nextOffset = offset + nlIdx + 1
		}

		// Strip a trailing carriage return so "\r\n" endings work correctly.
		line = bytes.TrimRight(line, "\r")

		if string(line) == "---" {
			return nextOffset
		}
		offset = nextOffset
	}
	return -1
}

// ---------------------------------------------------------------------------
// AST helpers
// ---------------------------------------------------------------------------

// headingText extracts the plain-text content of an ATX heading by walking
// its inline children and concatenating *ast.Text segments. The returned
// string does not include the leading "#" prefix characters.
func headingText(h *ast.Heading, source []byte) string {
	var buf bytes.Buffer
	for c := h.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
	}
	return buf.String()
}

// headingLineStart returns the byte offset of the first byte of the heading
// line in source — i.e., the offset of the leading "#" character.
//
// goldmark's Lines().At(0).Start for an ATX heading points to the heading's
// *text content* (after "# "), not to the "#" itself. We scan backward from
// that position to find the preceding newline; the byte after the newline is
// the true start of the heading line. If there is no preceding newline the
// heading begins at offset 0.
func headingLineStart(h *ast.Heading, source []byte) int {
	lines := h.Lines()
	if lines.Len() == 0 {
		return 0
	}
	pos := lines.At(0).Start
	for pos > 0 && source[pos-1] != '\n' {
		pos--
	}
	return pos
}

// headingLineStop returns the byte offset immediately after the heading line
// — i.e., the first byte of the next line (the start of content that follows
// the heading). This is goldmark's Lines().At(0).Stop.
func headingLineStop(h *ast.Heading) int {
	lines := h.Lines()
	if lines.Len() == 0 {
		return 0
	}
	return lines.At(0).Stop
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

// validatePublicSubsections checks that all level-2 headings within the
// "# Public" section have unique normalized text. Returns ErrDuplicateSubsection
// (wrapped with context) on the first duplicate found.
//
// Note: In goldmark's flat document model, ALL headings — regardless of level
// — are direct children of the document root. A level-2 heading is a sibling
// of the level-1 heading it conceptually belongs to. We track whether we are
// "inside" the public section by watching level-1 headings.
func validatePublicSubsections(doc ast.Node, source []byte, filePath string) error {
	inPublic := false
	seen := make(map[string]bool)

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		h, ok := child.(*ast.Heading)
		if !ok {
			continue
		}

		switch h.Level {
		case 1:
			// A new level-1 heading may enter or exit the public section.
			inPublic = normalizename.NormalizeName(headingText(h, source)) == "public"

		case 2:
			if !inPublic {
				continue
			}
			normalized := normalizename.NormalizeName(headingText(h, source))
			if seen[normalized] {
				return fmt.Errorf(
					"%w: %s: duplicate subsection %q within '# Public'",
					ErrDuplicateSubsection, filePath, normalized,
				)
			}
			seen[normalized] = true
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Section extraction
// ---------------------------------------------------------------------------

// sectionBuilder accumulates the data needed to construct a Section while
// iterating through AST nodes.
type sectionBuilder struct {
	// heading is the normalized text of the level-1 heading.
	heading string

	// contentStart is the byte offset immediately after the heading line
	// (the first byte of content within this section).
	contentStart int

	// contentEnd is the byte offset where the section's own (non-subsection)
	// content ends. Populated when the first level-2 heading is encountered.
	// If no level-2 headings exist, the content extends to the section's end
	// offset, which is supplied at close time.
	contentEnd int

	// hadSubsection is true once the first level-2 heading within this
	// section has been encountered.
	hadSubsection bool

	// subsections holds the completed Subsection values for this section.
	subsections []Subsection
}

// subsectionBuilder accumulates the data needed to construct a Subsection
// while iterating through AST nodes.
type subsectionBuilder struct {
	// heading is the normalized text of the level-2 heading.
	heading string

	// contentStart is the byte offset immediately after the heading line.
	contentStart int
}

// extractSections iterates the direct children of the document root node and
// builds a slice of Section values in document order.
//
// Algorithm:
//   - A level-1 heading closes the previous section and starts a new one.
//   - A level-2 heading closes the current subsection (if any), records the
//     end of the section's own content, and starts a new subsection.
//   - Level-3+ headings and non-heading nodes are silently captured by the
//     byte-range approach — no explicit action needed.
//   - At the end of the document, close the last open section/subsection
//     using len(source) as the end offset.
func extractSections(doc ast.Node, source []byte) ([]Section, error) {
	var sections []Section

	var (
		curSection    *sectionBuilder
		curSubsection *subsectionBuilder
	)

	// finishSubsection closes the current level-2 subsection using endOffset
	// as the exclusive end of its content in source, and appends the result
	// to curSection. Does nothing when curSubsection is nil.
	finishSubsection := func(endOffset int) {
		if curSubsection == nil {
			return
		}
		rawContent := source[curSubsection.contentStart:endOffset]
		curSection.subsections = append(curSection.subsections, Subsection{
			Heading: curSubsection.heading,
			Content: string(trimBlankLines(rawContent)),
		})
		curSubsection = nil
	}

	// finishSection closes the current level-1 section using endOffset as the
	// exclusive end of the section in source (byte where next heading's line
	// starts, or len(source)). Appends the completed Section to sections.
	// Does nothing when curSection is nil.
	finishSection := func(endOffset int) {
		if curSection == nil {
			return
		}

		// First, close any open subsection.
		finishSubsection(endOffset)

		// Determine the end of the section's own (non-subsection) content.
		contentEnd := curSection.contentEnd
		if !curSection.hadSubsection {
			// No level-2 headings were found; own content extends to endOffset.
			contentEnd = endOffset
		}

		rawContent := source[curSection.contentStart:contentEnd]
		sections = append(sections, Section{
			Heading:     curSection.heading,
			Content:     string(trimBlankLines(rawContent)),
			Subsections: curSection.subsections,
		})
		curSection = nil
	}

	// Iterate over direct children of the document root.
	// In goldmark's flat AST, all headings (levels 1–6) appear here as siblings.
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		h, ok := child.(*ast.Heading)
		if !ok {
			// Non-heading block nodes (paragraphs, code blocks, lists, etc.)
			// are content. Their bytes are captured by the offset ranges above,
			// so no explicit action is required here.
			continue
		}

		switch h.Level {
		case 1:
			// A new level-1 heading signals the end of the previous section.
			// The previous section ends at the start of this heading's line.
			finishSection(headingLineStart(h, source))

			// Start a new section. Content begins immediately after the heading line.
			stop := headingLineStop(h)
			curSection = &sectionBuilder{
				heading:      normalizename.NormalizeName(headingText(h, source)),
				contentStart: stop,
				contentEnd:   stop, // updated when the first ## heading is found
			}

		case 2:
			// Level-2 headings only have structural meaning within sections.
			// If we haven't seen a level-1 heading yet, this is an orphan — skip it
			// (the invariants say it should not happen in valid files).
			if curSection == nil {
				continue
			}

			// The start of this heading's line is the boundary between the
			// section's own content and this subsection.
			h2LineStart := headingLineStart(h, source)
			h2LineStop := headingLineStop(h)

			if !curSection.hadSubsection {
				// This is the first level-2 heading in the current section.
				// Record the end of the section's own content.
				curSection.contentEnd = h2LineStart
				curSection.hadSubsection = true
			} else {
				// Close the previously open subsection before starting a new one.
				finishSubsection(h2LineStart)
			}

			// Start a new subsection.
			curSubsection = &subsectionBuilder{
				heading:      normalizename.NormalizeName(headingText(h, source)),
				contentStart: h2LineStop,
			}

		default:
			// Level 3+ headings are not structural delimiters. They appear inside
			// Section.Content or Subsection.Content as raw markdown, captured
			// automatically by the byte-offset approach. No action needed.
		}
	}

	// Close the last open section (and any open subsection within it) at the
	// end of the source document.
	finishSection(len(source))

	return sections, nil
}

// ---------------------------------------------------------------------------
// Content trimming
// ---------------------------------------------------------------------------

// trimBlankLines removes leading and trailing lines that contain only
// whitespace from content. The spec requires this for both Section.Content
// and Subsection.Content.
//
// A "blank line" is any line where strings.TrimSpace returns an empty string.
// Returns nil when content is empty or contains only blank lines.
func trimBlankLines(content []byte) []byte {
	if len(content) == 0 {
		return nil
	}

	s := string(content)
	lines := strings.Split(s, "\n")

	// Find the first non-blank line.
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	// Find the last non-blank line.
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	if start >= end {
		return nil
	}

	return []byte(strings.Join(lines[start:end], "\n"))
}
