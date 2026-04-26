// code-from-spec: TEST/tech_design/internal/normalizename@v2

// Package normalizename tests the NormalizeName function.
// These are pure function tests — no filesystem or temp directories needed.
package normalizename

import "testing"

// testCase holds a single input/expected pair for table-driven tests.
type testCase struct {
	name   string
	input  string
	expect string
}

func TestNormalizeName(t *testing.T) {
	cases := []testCase{
		// Identity
		{
			name:   "already normalized",
			input:  "public",
			expect: "public",
		},
		{
			name:   "single word",
			input:  "Interface",
			expect: "interface",
		},

		// Trim
		{
			name:   "leading and trailing spaces",
			input:  "  Interface  ",
			expect: "interface",
		},
		{
			name:   "leading and trailing tabs",
			input:  "\tInterface\t",
			expect: "interface",
		},
		{
			name:   "mixed leading whitespace",
			input:  " \t Interface \t ",
			expect: "interface",
		},

		// Collapse
		{
			name:   "multiple spaces between words",
			input:  "Testes   de   aceitação",
			expect: "testes de aceitação",
		},
		{
			name:   "tabs between words",
			input:  "Testes\tde\taceitação",
			expect: "testes de aceitação",
		},
		{
			name:   "mixed whitespace between words",
			input:  "Testes \t de \t aceitação",
			expect: "testes de aceitação",
		},

		// Case folding
		{
			name:   "all uppercase",
			input:  "PUBLIC",
			expect: "public",
		},
		{
			name:   "mixed case",
			input:  "PuBLiC",
			expect: "public",
		},
		{
			name:   "unicode case folding",
			input:  "TESTES DE ACEITAÇÃO",
			expect: "testes de aceitação",
		},
		{
			// Unicode simple case folding maps ß (U+00DF) to ss.
			name:   "german sharp s",
			input:  "Straße",
			expect: "strasse",
		},

		// Combined
		{
			name:   "trim, collapse, and case fold together",
			input:  "  TESTES   DE   ACEITAÇÃO  ",
			expect: "testes de aceitação",
		},
		{
			name:   "logical name qualifier style",
			input:  "testes de ACEITAÇÃO",
			expect: "testes de aceitação",
		},
		{
			name:   "tabs and mixed case",
			input:  "\tROOT/payments/fees\t",
			expect: "root/payments/fees",
		},

		// Edge cases
		{
			name:   "empty string",
			input:  "",
			expect: "",
		},
		{
			name:   "only whitespace",
			input:  "   \t  ",
			expect: "",
		},
		{
			// U+00A0 (non-breaking space) is not treated as whitespace
			// by the framework — it is part of the text, not collapsed.
			name:   "non-breaking space is not whitespace",
			input:  "hello world",
			expect: "hello world",
		},
		{
			name:   "single character",
			input:  "X",
			expect: "x",
		},
	}

	for _, tc := range cases {
		tc := tc // capture loop variable for parallel subtests
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeName(tc.input)
			if got != tc.expect {
				t.Errorf("NormalizeName(%q) = %q; want %q", tc.input, got, tc.expect)
			}
		})
	}
}
