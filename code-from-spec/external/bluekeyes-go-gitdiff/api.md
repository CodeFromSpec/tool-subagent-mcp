# bluekeyes/go-gitdiff — API used

## Import

```go
import "github.com/bluekeyes/go-gitdiff/gitdiff"
```

## Parsing a unified diff

```go
files, preamble, err := gitdiff.Parse(reader)
```

`Parse` reads a unified diff from an `io.Reader` and returns
a slice of `*gitdiff.File` values, one per file in the diff.
`preamble` is any text before the first file header.

To parse from a string:

```go
files, _, err := gitdiff.Parse(strings.NewReader(diffText))
```

## Applying a patch

```go
var output bytes.Buffer
err := gitdiff.Apply(&output, source, file)
```

`Apply` writes the result of applying the patch in `file`
(a `*gitdiff.File`) to the content read from `source`
(an `io.ReaderAt`) into `output` (an `io.Writer`).

To apply to a byte slice:

```go
source := bytes.NewReader(originalContent)
var output bytes.Buffer
err := gitdiff.Apply(&output, source, file)
```

## Error handling

`Parse` returns an error if the diff is malformed.
`Apply` returns an error if the patch cannot be applied
(e.g. context lines do not match the source).
