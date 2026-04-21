---
version: 2
parent_version: 15
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes/codegen
    version: 21
  - path: ROOT/tech_design/internal/chain_resolver
    version: 36
  - path: ROOT/tech_design/internal/frontmatter
    version: 18
  - path: ROOT/tech_design/internal/logical_names
    version: 16
implements:
  - internal/modes/codegen/setup.go
---

# ROOT/tech_design/internal/modes/codegen/setup

## Intent

Implements the `Setup` function for the codegen mode: validates
arguments, loads and pre-caches the full spec chain, and registers
the `load_context` and `write_file` tools on the MCP server.

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
   `ResolveChain` and reading every file in the chain. Build
   the concatenated chain content using the separator format
   defined by `load_context`. If any step fails, return an error.
6. Build a `Target` struct and register `load_context` and
   `write_file` on the provided server. `load_context` serves
   the pre-loaded content without further I/O.

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
`Setup`, ready to be returned by `load_context`.

### Error handling

All errors returned from `Setup` are initialization failures —
printed to stderr by `main` and cause an exit 1.

- Missing or extra args → `"usage: subagent-mcp codegen <logical-name>"`.
- Invalid logical name → `"invalid logical name: <name>"`.
- Frontmatter parse error → wrapped error from `ParseFrontmatter`.
- Empty implements → `"node <name> has no implements"`.
- Invalid implements path → `"invalid implements path: <path>: <reason>"`.
- Chain resolution error → wrapped error from `ResolveChain`.
- Chain file read error → `"failed to read <file-path>: <err>"`.
