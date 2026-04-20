---
version: 2
parent_version: 2
depends_on:
  - path: ROOT/domain/operations/codegen/chain
    version: 2
implements:
  - cmd/subagent-mcp/frontmatter.go
---

# ROOT/tech_design/frontmatter

## Intent

Reads and parses the YAML frontmatter from spec node files and
external dependency files.

## Contracts

### Parsing

The frontmatter is the YAML block between the first `---` and the
second `---` at the top of the file. Everything after the second
`---` is ignored.

Fields extracted:

- `version` (pointer to integer, nil if absent)
- `depends_on` (list of objects; see below)
- `implements` (list of strings)

`parent_version` is present in spec files but is not needed by this
server — it is ignored.

Unknown fields are ignored.

### DependsOn structure

Each `depends_on` entry has:

| Field | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | Logical name of the dependency. |
| `version` | int | yes | Expected version. |
| `filter` | []string | no | Glob patterns for file selection within an external dep folder. |

Example frontmatter with an external dep with filter:

```yaml
---
version: 3
depends_on:
  - path: ROOT/architecture/config
    version: 2
  - path: EXTERNAL/celcoin-api
    version: 5
    filter:
      - "endpoints/*.md"
      - "types.md"
implements:
  - internal/payments/processor.go
  - internal/payments/processor_test.go
---
```

### Interface

```go
type DependsOn struct {
    Path    string
    Version int
    Filter  []string
}

type Frontmatter struct {
    Version    *int
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
