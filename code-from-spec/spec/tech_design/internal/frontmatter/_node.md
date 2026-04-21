---
version: 18
parent_version: 3
depends_on:
  - path: EXTERNAL/codefromspec
    version: 1
  - path: EXTERNAL/goccy-go-yaml
    version: 1
implements:
  - internal/frontmatter/frontmatter.go
---

# ROOT/tech_design/internal/frontmatter

## Intent

Reads and parses the YAML frontmatter from spec nodes, test nodes,
and external dependency files.

## Context

### YAML dependency

Uses `github.com/goccy/go-yaml` for YAML parsing.

## Contracts

### Parsing

The frontmatter is the YAML block between the first `---` and the
second `---` at the top of the file. Everything after the second
`---` is ignored.

Fields extracted:

- `depends_on` (list of objects; see below)
- `implements` (list of strings)

All fields are optional at the parsing level — validation
of required fields happens elsewhere. Unknown fields are
ignored.

### DependsOn structure

Each `depends_on` entry has:

| Field | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the dependency. |
| `filter` | []string | no | Glob patterns for file selection within an external dep folder. |

### Interface

```go
type DependsOn struct {
    LogicalName string
    Filter      []string
}

type Frontmatter struct {
    DependsOn  []DependsOn
    Implements []string
}

func ParseFrontmatter(filePath string) (*Frontmatter, error)
```

`ParseFrontmatter` reads the file, extracts the frontmatter block,
and returns the parsed result. It does not cache — caching is the
caller's responsibility.

### Efficiency

The parser reads line by line, extracts the frontmatter block, and
stops as soon as the closing `---` is found. The file body is never
read.

### Error handling

- `error reading <path>: <underlying error>`
- `error parsing frontmatter in <path>: <underlying error>`
- `frontmatter not found in <path>`
