---
version: 3
parent_version: 2
---

# ROOT/external/golang-x-text

Unicode text processing packages for Go:
`golang.org/x/text`.

Part of the official Go project extended libraries.
MIT licensed.

# Public

## Import

```go
import "golang.org/x/text/cases"
```

## Case folding

```go
caser := cases.Fold()
folded := caser.String(input)
```

`cases.Fold()` returns a `Caser` that applies Unicode simple
case folding. `Caser.String` returns the folded string.
