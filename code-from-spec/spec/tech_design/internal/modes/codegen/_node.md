---
version: 15
parent_version: 10
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
---

# ROOT/tech_design/internal/modes/codegen

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
4. Validate `Implements`:
   a. Must not be empty.
   b. Each path must be relative (no leading `/`, no `:`).
   c. Must not contain `..` as a path component.
   If any validation fails, return an error describing the
   violation.
5. Resolve and pre-load the full chain into memory by calling
   `ResolveChain` and reading every file in the chain. If any
   step fails, return an error.
6. Build a `Target` struct and register `load_context` and
   `write_file` on the provided server. `load_context` serves
   the pre-loaded content without further I/O.

### Target type

```go
type Target struct {
    LogicalName string
    FilePath    string
    Frontmatter *Frontmatter
    ChainContent string
}
```

`ChainContent` holds the fully concatenated chain loaded during
`Setup`, ready to be returned by `load_context`.

### Error handling

Errors returned from `Setup` are printed to stderr by `main` and
cause an exit 1.
