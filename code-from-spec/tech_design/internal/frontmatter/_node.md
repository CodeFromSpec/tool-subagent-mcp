---
version: 32
parent_version: 12
depends_on:
  - path: ROOT/external/codefromspec
    version: 3
  - path: ROOT/external/goccy-go-yaml
    version: 1
implements:
  - internal/frontmatter/frontmatter.go
---

# ROOT/tech_design/internal/frontmatter

## Intent

Reads and parses the YAML frontmatter from spec nodes, test nodes,
and external dependency files.

## Context

### Package

`package frontmatter`

### YAML dependency

Uses `github.com/goccy/go-yaml` for YAML parsing.

## Contracts

### Interface

```go
type DependsOn struct {
    LogicalName string
    Version     int
}

type Frontmatter struct {
    Version        int
    ParentVersion  *int
    SubjectVersion *int
    DependsOn      []DependsOn
    Implements     []string
}

var (
    ErrRead               = errors.New("error reading file")
    ErrFrontmatterParse   = errors.New("error parsing frontmatter")
    ErrFrontmatterMissing = errors.New("frontmatter not found")
    ErrMissingVersion     = errors.New("version field is required")
)

func ParseFrontmatter(filePath string) (*Frontmatter, error)
```

`ParseFrontmatter` reads the file, extracts the frontmatter block,
and returns the parsed result.

Errors returned by `ParseFrontmatter` wrap the sentinel with
context (file path, underlying error) using `fmt.Errorf`, so
callers can match with `errors.Is()`.

### Parsing

The frontmatter is the YAML block between the first `---` and the
second `---` at the top of the file. Everything after the second
`---` is ignored.

Fields extracted:

| Field | Type | Description |
|---|---|---|
| `version` | int | Node version. Required. |
| `parent_version` | *int | Parent version. Nil if absent. |
| `subject_version` | *int | Subject version (test nodes). Nil if absent. |
| `depends_on` | []DependsOn | Cross-tree dependencies. |
| `implements` | []string | Output files. |

Unknown fields are ignored.

### DependsOn structure

Each `depends_on` entry has:

| YAML key | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | Logical name of the dependency. |
| `version` | int | yes | Known version of the dependency. |

### Efficiency

The parser reads line by line, extracts the frontmatter block, and
stops as soon as the closing `---` is found. The file body is never
read.

### Error handling

All errors wrap a sentinel so callers can use `errors.Is()`:

| Sentinel | Returned when |
|---|---|
| `ErrRead` | The file cannot be read. |
| `ErrFrontmatterParse` | The YAML frontmatter is malformed. |
| `ErrFrontmatterMissing` | No `---` delimiters found at the top of the file. |
| `ErrMissingVersion` | The `version` field is absent or zero. |
