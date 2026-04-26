# golang.org/x/text — API used

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
