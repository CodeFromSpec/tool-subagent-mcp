---
version: 13
parent_version: 9
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
---

# ROOT/tech_design/modes/codegen

## Intent

Technical design for the codegen mode: argument parsing,
tool registration, and MCP server startup.

## Contracts

### Setup function

```go
func Setup(s *mcp.Server, args []string) error
```

1. Validate `args` has exactly one element (the target logical
   name). If not, return a usage error.
2. Call `PathFromLogicalName` to resolve the file path. If it
   returns false, return `"invalid logical name: <name>"`.
3. Call `ParseFrontmatter` on the resolved file path.
4. Build a `Target` struct and register `load_context` and
   `write_file` on the provided server.

### Target type

```go
type Target struct {
    LogicalName string
    FilePath    string
    Frontmatter *Frontmatter
}
```

### Error handling

Errors returned from `Run` are printed to stderr by `main` and
cause an exit 1.
