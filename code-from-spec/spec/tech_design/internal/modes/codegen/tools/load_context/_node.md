---
version: 28
parent_version: 18
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes/codegen
    version: 21
  - path: ROOT/tech_design/internal/chain_resolver
    version: 48
  - path: ROOT/tech_design/internal/frontmatter
    version: 25
  - path: ROOT/tech_design/internal/logical_names
    version: 22
  - path: ROOT/tech_design/internal/pathvalidation
    version: 8
implements:
  - internal/modes/codegen/load_context.go
---

# ROOT/tech_design/internal/modes/codegen/tools/load_context

## Intent

Implements the `load_context` tool handler: validates the
logical name, loads the spec chain, populates `currentTarget`,
and returns the chain content as a single MCP text response.

## Contracts

### Handler

Follows the `handleLoadContext` signature defined in the
parent codegen node. Accepts `LoadContextArgs` with a
`logical_name` string field.

### Algorithm

1. If `currentTarget != nil`, return a tool error:
   `"load_context already called for this session"`.
2. Validate that `args.LogicalName` starts with `ROOT/` or
   `TEST/` (or equals `ROOT` or `TEST`). If not, return a
   tool error: `"codegen target must be a ROOT/ or TEST/
   logical name: <name>"`.
3. Call `PathFromLogicalName`. If it returns false, return a
   tool error: `"invalid logical name: <name>"`.
4. Call `ParseFrontmatter` on the resolved path. If it fails,
   return a tool error wrapping the underlying error.
5. Validate `Implements`:
   a. Must not be empty → tool error: `"node <name> has no
      implements"`.
   b. Call `ValidatePath` for each path against the working
      directory. If any fails, return a tool error.
6. Generate a UUID. Call `ResolveChain` to resolve the full
   chain and read every file in the chain into memory. Build
   the concatenated chain content using the UUID and the chain
   output format defined in the parent codegen node. If any
   step fails, return a tool error.
7. Build a `Target` struct with `LogicalName`, `FilePath`,
   `Frontmatter`, and `ChainContent` populated. Set
   `currentTarget` to this struct.
8. Return the chain content as a success result.
