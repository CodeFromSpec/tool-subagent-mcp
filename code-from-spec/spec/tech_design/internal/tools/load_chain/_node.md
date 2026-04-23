---
version: 44
parent_version: 3
depends_on:
  - path: EXTERNAL/google-uuid
    version: 2
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/tech_design/internal/chain_resolver
    version: 56
  - path: ROOT/tech_design/internal/frontmatter
    version: 27
  - path: ROOT/tech_design/internal/logical_names
    version: 24
  - path: ROOT/tech_design/internal/pathvalidation
    version: 10
implements:
  - internal/load_chain/load_chain.go
---

# ROOT/tech_design/internal/tools/load_chain

## Intent

Implements the `load_chain` tool handler: validates the
logical name, loads the spec chain, and returns the chain
content as a single MCP text response.

## Context

### Package

`package load_chain`

### Target node

The target node is identified by its logical name — either a leaf
spec node (`ROOT/...`) or a test node (`TEST/...`). Examples:
`ROOT/payments/fees/calculation`,
`TEST/payments/fees/calculation`.

## Contracts

### Tool definition

Name: `load_chain`
Description: `"Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the node to generate code for. |

### LoadChainArgs type

```go
type LoadChainArgs struct {
    LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}
```

### Handler

```go
func HandleLoadChain(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args LoadChainArgs,
) (*mcp.CallToolResult, any, error)
```

### Chain output format

The chain is serialized as a sequence of file sections using
heredoc-style delimiters with a UUID generated once per call
to avoid collisions with file content.

Opening delimiter: `<<<FILE_<uuid>>>`
Closing delimiter: `<<<END_FILE_<uuid>>>`

The same UUID is used for all files in the chain. Each section
includes `path:` and optionally `node:` headers between the
opening delimiter and the file content, separated by a blank
line. Spec and dependency files include both `node:` and
`path:`; code files include only `path:`.

```
<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
node: ROOT
path: code-from-spec/spec/_node.md

<file content>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>

<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
node: EXTERNAL/database
path: code-from-spec/external/database/_external.md

<content of _external.md>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>

<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
node: EXTERNAL/database
path: code-from-spec/external/database/schema.sql

<content of schema.sql>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>

<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
path: internal/payments/fees/calculation.go

<existing source file content>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>
```

### Algorithm

1. Validate that `args.LogicalName` starts with `ROOT/` or
   `TEST/` (or equals `ROOT` or `TEST`). If not, return a
   tool error: `"target must be a ROOT/ or TEST/
   logical name: <name>"`.
2. Call `PathFromLogicalName`. If it returns false, return a
   tool error: `"invalid logical name: <name>"`.
3. Call `ParseFrontmatter` on the resolved path. If it fails,
   return a tool error wrapping the underlying error.
4. Validate `Implements`:
   a. Must not be empty → tool error: `"node <name> has no
      implements"`.
   b. Call `ValidatePath` for each path against the working
      directory. If any fails, return a tool error.
5. Generate a UUID using `github.com/google/uuid`.
6. Call `ResolveChain` to resolve the full chain and read every
   file in the chain into memory. For ancestors and dependencies, strip the YAML frontmatter
   before including the content. Build
   the concatenated chain content using the UUID and the chain
   output format, appending the code files after the
   dependencies. If any step fails, return a tool error.
7. Return the chain content as a success result.

## Constraints

- The target argument must be a logical name that resolves to a
  node with `implements`. Absent, empty, or invalid values cause
  the tool to report an error.
- Reads are limited to the chain.
- If any chain file cannot be read, `load_chain` returns an error
  identifying the missing file; it does not return partial results.

## Decisions

### load_chain returns everything in one call

Loading the chain file-by-file via separate tool calls would
accumulate context in the conversation history, increasing token
cost with each roundtrip. A single call returns the entire chain,
minimizing roundtrip overhead.
