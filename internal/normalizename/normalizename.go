// code-from-spec: ROOT/tech_design/internal/normalizename@v2

// Package normalizename provides heading and logical name text normalization
// for comparison. It applies the framework normalization rules defined in the
// Code from Spec methodology: trim, collapse whitespace, and Unicode case fold.
package normalizename

import (
	"strings"

	"golang.org/x/text/cases"
)

// NormalizeName applies the framework normalization rules to a raw heading
// or logical name qualifier text:
//
//  1. Trim leading and trailing whitespace (space U+0020 and tab U+0009).
//  2. Collapse each sequence of one or more whitespace characters to a
//     single space (U+0020).
//  3. Apply Unicode simple case folding via cases.Fold().
//
// Only space (U+0020) and horizontal tab (U+0009) are treated as whitespace —
// other Unicode whitespace characters are not recognized and are left as-is.
func NormalizeName(raw string) string {
	// Step 1: Trim leading and trailing whitespace (space and tab only).
	trimmed := strings.Trim(raw, " \t")

	// Step 2: Collapse interior sequences of spaces and/or tabs to a single space.
	// We build the result manually to avoid treating other Unicode whitespace
	// (e.g. U+00A0 non-breaking space) as collapsible.
	var b strings.Builder
	b.Grow(len(trimmed))
	inWhitespace := false
	for _, r := range trimmed {
		if r == ' ' || r == '\t' {
			// Mark that we are inside a whitespace run; emit one space later.
			inWhitespace = true
		} else {
			if inWhitespace {
				// Emit the single collapsed space before the non-whitespace rune.
				b.WriteByte(' ')
				inWhitespace = false
			}
			b.WriteRune(r)
		}
	}
	collapsed := b.String()

	// Step 3: Apply Unicode simple case folding.
	caser := cases.Fold()
	return caser.String(collapsed)
}
