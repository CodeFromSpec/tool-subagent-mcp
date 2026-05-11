---
version: 4
parent_version: 12
depends_on:
  - path: ROOT/external/codefromspec
    version: 4
  - path: ROOT/external/golang-x-text
    version: 2
implements:
  - internal/normalizename/normalizename.go
---

# ROOT/tech_design/internal/normalizename

Normalizes heading and logical name text for comparison.

# Public

## Package

`package normalizename`

## Dependencies

- `golang.org/x/text/cases` — Unicode simple case folding.

## Interface

```go
func NormalizeName(raw string) string
```

Applies the framework normalization rules to a raw heading
or logical name qualifier text:

1. Trim leading and trailing whitespace.
2. Collapse each sequence of one or more whitespace characters
   to a single space (`U+0020`).
3. Apply Unicode simple case folding using `cases.Fold()` from
   `golang.org/x/text/cases`.

Whitespace characters are space (`U+0020`) and horizontal tab
(`U+0009`).
