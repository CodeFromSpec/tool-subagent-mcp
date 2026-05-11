// code-from-spec: ROOT/tech_design/internal/normalizename@v5

// Package normalizename provides heading and logical name text normalization
// for comparison. It applies the framework normalization rules defined in the
// Code from Spec methodology: trim, collapse whitespace, and Unicode case fold.
//
// Only space (U+0020) and horizontal tab (U+0009) are treated as whitespace.
// Other Unicode whitespace characters (e.g. U+00A0 non-breaking space) are
// not recognized — they are treated as ordinary text.
package normalizename

import (
	"strings"

	"golang.org/x/text/cases"
)

// NormalizeName applies the framework normalization rules to a raw heading
// or logical name qualifier text.
//
// The rules are applied in order:
//
//  1. Trim leading and trailing whitespace (space U+0020 and tab U+0009 only).
//  2. Collapse each interior sequence of one or more whitespace characters
//     to a single space (U+0020).
//  3. Apply Unicode simple case folding via cases.Fold().
//
// Only space (U+0020) and horizontal tab (U+0009) are recognized as
// whitespace. Any other Unicode whitespace (e.g. U+00A0) is left as-is.
func NormalizeName(raw string) string {
	// Step 1: Trim leading and trailing whitespace (space and tab only).
	// strings.Trim with the explicit cutset avoids treating other Unicode
	// whitespace as trimmable.
	trimmed := strings.Trim(raw, " \t")

	// Step 2: Collapse interior sequences of space and/or tab characters
	// to a single space. We iterate rune-by-rune to ensure that only
	// U+0020 and U+0009 are treated as whitespace — not other Unicode
	// whitespace characters such as U+00A0 (non-breaking space).
	var b strings.Builder
	b.Grow(len(trimmed)) // pre-allocate a reasonable capacity
	inWhitespace := false
	for _, r := range trimmed {
		if r == ' ' || r == '\t' {
			// We are inside a whitespace run; defer emitting a single space
			// until we encounter the next non-whitespace rune.
			inWhitespace = true
		} else {
			if inWhitespace {
				// Emit the single collapsed space before this non-whitespace rune.
				b.WriteByte(' ')
				inWhitespace = false
			}
			b.WriteRune(r)
		}
	}
	// Note: inWhitespace cannot be true here because Step 1 already trimmed
	// trailing whitespace, so there is no trailing whitespace run to flush.
	collapsed := b.String()

	// Step 3: Apply Unicode simple case folding.
	// cases.Fold() returns a Caser configured for Unicode simple case folding.
	// Caser.String returns the case-folded result.
	caser := cases.Fold()
	return caser.String(collapsed)
}
