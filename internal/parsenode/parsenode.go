// code-from-spec: ROOT/tech_design/internal/parsenode@v30

// Package parsenode parses the body of a spec node file and returns
// a structured representation of all sections.
//
// The file format is CommonMark with YAML frontmatter. Only level-1
// and level-2 headings are structural delimiters. Level-3+ headings
// are treated as content.
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

	logicalnames "github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logical_names"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/normalizename"
)

// Sentinel errors. All errors returned by ParseNode wrap one of these
// so callers can use errors.Is().
var (
	ErrRead                = errors.New("error reading file")
	ErrFrontmatterMissing  = errors.New("frontmatter not found")
	ErrUnexpectedContent   = errors.New("unexpected content before first heading")
	ErrInvalidNodeName     = errors.New("node name section does not match logical name")
	ErrDuplicatePublic     = errors.New("duplicate public section")
	ErrDuplicateSubsection = errors.New("duplicate subsection in public")
)

// Subsection represents a level-2 heading and its content within a Section.
type Subsection struct {
	Heading string // normalized heading text
	Content string // raw markdown content, leading/trailing blank lines trimmed
}

// Section represents a level-1 heading and its content, including any
// level-2 subsections.
type Section struct {
	Heading     string       // normalized heading text
	Content     string       // raw markdown between the level-1 heading and the first level-2 heading (or end), trimmed
	Subsections []Subsection // level-2 subsections within this section
}

// NodeBody is the structured result of parsing a spec node file.
// Public is nil when no "# Public" section exists.
type NodeBody struct {
	NameSection Section  // the first section (node name section)
	Public      *Section // the "# Public" section, or nil
	Private     []Section // all other sections (not name, not public)
}

// ParseNode resolves logicalName to a file path, reads and parses the
// spec node file, validates the structure, and returns a NodeBody.
//
// All returned errors wrap one of the sentinel variables so callers can
// use errors.Is().
func ParseNode(logicalName string) (*NodeBody, error) {
	// Step 1 — Resolve logical name to a file path.
	filePath, ok := logicalnames.PathFromLogicalName(logicalName)
	if !ok {
		return nil, fmt.Errorf("%w: cannot resolve logical name %q", ErrRead, logicalName)
	}

	// Step 2 — Read the file.
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRead, filePath, err)
	}

	// Step 3 — Skip frontmatter.
	// Frontmatter is delimited by the first "---" line and the second "---" line.
	// We discard everything up to and including the closing "---".
	body, err := skipFrontmatter(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFrontmatterMissing, filePath)
	}

	// Step 4 — Parse body as CommonMark using goldmark.
	md := goldmark.New()
	source := body
	doc := md.Parser().Parse(text.NewReader(source))

	// Step 5 — Validate: the first direct child of the document root must
	// be a level-1 heading.
	firstChild := doc.FirstChild()
	if firstChild == nil {
		return nil, fmt.Errorf("%w: %s: document is empty", ErrUnexpectedContent, filePath)
	}
	firstHeading, ok := firstChild.(*ast.Heading)
	if !ok || firstHeading.Level != 1 {
		return nil, fmt.Errorf("%w: %s: first element is not a level-1 heading", ErrUnexpectedContent, filePath)
	}

	// Step 6 — Validate: node name section heading matches logical name.
	// Apply NormalizeName to both the heading text and the logical name.
	firstHeadingText := headingText(firstHeading, source)
	normalizedHeading := normalizename.NormalizeName(firstHeadingText)
	normalizedLogicalName := normalizename.NormalizeName(logicalName)
	if normalizedHeading != normalizedLogicalName {
		return nil, fmt.Errorf(
			"%w: %s: heading %q does not match logical name %q",
			ErrInvalidNodeName, filePath, firstHeadingText, logicalName,
		)
	}

	// Step 7 — Validate: no duplicate "# Public" sections.
	// Walk all level-1 headings and count how many normalize to "public".
	publicCount := 0
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if h, ok := child.(*ast.Heading); ok && h.Level == 1 {
			if normalizename.NormalizeName(headingText(h, source)) == "public" {
				publicCount++
			}
		}
	}
	if publicCount > 1 {
		return nil, fmt.Errorf("%w: %s", ErrDuplicatePublic, filePath)
	}

	// Step 8 — Validate: no duplicate level-2 subsections within "# Public".
	// Collect level-2 headings within the public section and check for duplicates.
	if err := validatePublicSubsections(doc, source, filePath); err != nil {
		return nil, err
	}

	// Step 9 — Extract sections from the document.
	sections, err := extractSections(doc, source, filePath)
	if err != nil {
		return nil, err
	}

	// Step 10 — Classify sections.
	// The first section is always the node name section.
	// A section normalizing to "public" is the public section.
	// All others are private.
	result := &NodeBody{}
	if len(sections) == 0 {
		// This should not happen because we validated a first heading exists.
		return nil, fmt.Errorf("%w: %s: no sections found", ErrUnexpectedContent, filePath)
	}

	result.NameSection = sections[0]

	for i := 1; i < len(sections); i++ {
		s := sections[i]
		if s.Heading == "public" {
			// Already validated there is at most one; assign directly.
			sCopy := s
			result.Public = &sCopy
		} else {
			result.Private = append(result.Private, s)
		}
	}

	return result, nil
}

// skipFrontmatter finds the closing "---" delimiter and returns the bytes
// that follow it. Returns an error if the frontmatter is not found.
//
// The frontmatter format requires:
//   - The file begins with a line that is exactly "---".
//   - A second line that is exactly "---" closes the frontmatter.
func skipFrontmatter(raw []byte) ([]byte, error) {
	// Split into lines preserving endings so we can find exact positions.
	// We look for the first "---" line, then the second "---" line.

	const delimiter = "---"

	// Find the first "---" at the start.
	rest := raw
	firstEnd := findDelimiterLine(rest)
	if firstEnd < 0 {
		return nil, errors.New("opening delimiter not found")
	}
	rest = raw[firstEnd:]

	// Find the second "---" (the closing delimiter).
	secondEnd := findDelimiterLine(rest)
	if secondEnd < 0 {
		return nil, errors.New("closing delimiter not found")
	}

	// Return everything after the closing "---" line.
	_ = delimiter
	return rest[secondEnd:], nil
}

// findDelimiterLine finds the first line in data that is exactly "---"
// (possibly followed by a newline) and returns the byte offset immediately
// after that line (i.e., the start of the next line). Returns -1 if not found.
func findDelimiterLine(data []byte) int {
	offset := 0
	for offset < len(data) {
		// Find the end of the current line.
		lineEnd := bytes.IndexByte(data[offset:], '\n')
		var line []byte
		var nextOffset int
		if lineEnd < 0 {
			// Last line with no trailing newline.
			line = data[offset:]
			nextOffset = len(data)
		} else {
			line = data[offset : offset+lineEnd]
			nextOffset = offset + lineEnd + 1
		}

		// Trim carriage return for Windows-style line endings.
		line = bytes.TrimRight(line, "\r")

		if string(line) == "---" {
			return nextOffset
		}
		offset = nextOffset
	}
	return -1
}

// headingText extracts the plain text content of a heading node by walking
// its inline children and concatenating *ast.Text segments.
// This returns the text without the "#" prefix.
func headingText(h *ast.Heading, source []byte) string {
	var buf bytes.Buffer
	for c := h.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
	}
	return buf.String()
}

// validatePublicSubsections checks that no two level-2 headings within the
// "# Public" section have the same normalized text.
func validatePublicSubsections(doc ast.Node, source []byte, filePath string) error {
	// Walk direct children to find the public section, then check its level-2 headings.
	inPublic := false
	seen := make(map[string]bool)

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		h, ok := child.(*ast.Heading)
		if !ok {
			continue
		}

		if h.Level == 1 {
			// Check if we're entering the public section.
			normalized := normalizename.NormalizeName(headingText(h, source))
			inPublic = normalized == "public"
			continue
		}

		if h.Level == 2 && inPublic {
			normalized := normalizename.NormalizeName(headingText(h, source))
			if seen[normalized] {
				return fmt.Errorf("%w: %s: duplicate subsection %q in public section", ErrDuplicateSubsection, filePath, normalized)
			}
			seen[normalized] = true
		}
	}

	return nil
}

// extractSections iterates the direct children of the document root and
// builds a slice of Section values. Each level-1 heading starts a new
// section. Level-2 headings within a section become subsections.
func extractSections(doc ast.Node, source []byte, filePath string) ([]Section, error) {
	var sections []Section

	// currentSection accumulates the current level-1 section being built.
	// currentSubsection accumulates the current level-2 subsection being built.
	var (
		currentSection    *sectionBuilder
		currentSubsection *subsectionBuilder
	)

	// finishSubsection closes the current subsection and appends it to
	// the current section (if any).
	finishSubsection := func(endOffset int) {
		if currentSubsection == nil {
			return
		}
		content := trimBlankLines(source[currentSubsection.contentStart:endOffset])
		currentSection.subsections = append(currentSection.subsections, Subsection{
			Heading: currentSubsection.heading,
			Content: string(content),
		})
		currentSubsection = nil
	}

	// finishSection closes the current section and appends it to sections.
	finishSection := func(endOffset int) {
		if currentSection == nil {
			return
		}
		// Close any open subsection first.
		finishSubsection(endOffset)

		// Section content is from the end of the level-1 heading line to either
		// the first level-2 heading or endOffset if there are no subsections.
		content := trimBlankLines(source[currentSection.contentStart:currentSection.contentEnd])
		sections = append(sections, Section{
			Heading:     currentSection.heading,
			Content:     string(content),
			Subsections: currentSection.subsections,
		})
		currentSection = nil
	}

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		h, ok := child.(*ast.Heading)
		if !ok {
			// Non-heading node: not a structural delimiter.
			// If it appears before the first level-1 heading, it is an error.
			// (This is already checked before extractSections is called, but
			// guard here for safety against unexpected document structures.)
			continue
		}

		if h.Level == 1 {
			// The start offset for content is the byte after the heading line.
			headingLineStop := headingLineStop(h)

			// Close previous section at the start of this heading line.
			finishSection(headingLineStart(h))

			// Begin new section. Content starts after the heading line.
			// contentEnd will be updated when we encounter the first level-2
			// heading (or end of section/document).
			currentSection = &sectionBuilder{
				heading:      normalizename.NormalizeName(headingText(h, source)),
				contentStart: headingLineStop,
				contentEnd:   headingLineStop, // default: no content until updated
			}
			continue
		}

		if h.Level == 2 {
			if currentSection == nil {
				// Orphan level-2 heading before any level-1 heading — treat as content.
				// (The invariants say this should not happen in valid files.)
				continue
			}

			headingStart := headingLineStart(h)
			headingStop := headingLineStop(h)

			if currentSubsection == nil {
				// This is the first level-2 heading in the current section.
				// The section's content ends here.
				currentSection.contentEnd = headingStart
			} else {
				// Close the previous subsection.
				finishSubsection(headingStart)
			}

			currentSubsection = &subsectionBuilder{
				heading:      normalizename.NormalizeName(headingText(h, source)),
				contentStart: headingStop,
			}
			continue
		}

		// Level 3+ headings are not structural; they are content nodes but
		// goldmark represents them as siblings at the top level. They fall
		// through here and are handled by content extraction via byte offsets.
	}

	// Close the last open section/subsection at the end of the source.
	finishSection(len(source))

	return sections, nil
}

// headingLineStart returns the byte offset of the start of the heading line
// in the source. For ATX headings, Lines() contains one segment.
func headingLineStart(h *ast.Heading) int {
	lines := h.Lines()
	if lines.Len() == 0 {
		return 0
	}
	return lines.At(0).Start
}

// headingLineStop returns the byte offset immediately after the heading line
// in the source (i.e., the first byte of the next line).
func headingLineStop(h *ast.Heading) int {
	lines := h.Lines()
	if lines.Len() == 0 {
		return 0
	}
	return lines.At(0).Stop
}

// trimBlankLines removes leading and trailing blank lines (lines containing
// only whitespace) from content. Mirrors the spec requirement: "Leading and
// trailing blank lines in content are trimmed."
func trimBlankLines(content []byte) []byte {
	s := string(content)
	lines := strings.Split(s, "\n")

	// Trim leading blank lines.
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	// Trim trailing blank lines.
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	if start >= end {
		return nil
	}

	return []byte(strings.Join(lines[start:end], "\n"))
}

// sectionBuilder accumulates data for a Section while parsing.
type sectionBuilder struct {
	heading      string       // normalized heading
	contentStart int          // byte offset of content start (after heading line)
	contentEnd   int          // byte offset of content end (before first ## or end)
	subsections  []Subsection // accumulated subsections
}

// subsectionBuilder accumulates data for a Subsection while parsing.
type subsectionBuilder struct {
	heading      string // normalized heading
	contentStart int    // byte offset of content start (after ## heading line)
}
