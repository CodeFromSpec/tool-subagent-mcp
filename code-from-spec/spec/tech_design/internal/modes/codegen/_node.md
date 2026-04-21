---
version: 20
parent_version: 11
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
---

# ROOT/tech_design/internal/modes/codegen

## Intent

Technical design for the codegen mode: argument parsing,
tool registration, and MCP server startup.

## Context

### Package

`package codegen`

Directory: `internal/modes/codegen/`

All leaf nodes under this subtree generate files in this
package and directory.

## Contracts

### Target type

```go
type Target struct {
    LogicalName  string
    FilePath     string
    Frontmatter  *Frontmatter
    ChainContent string
}
```

`ChainContent` holds the fully concatenated chain loaded during
setup, ready to be returned by `load_context`.
