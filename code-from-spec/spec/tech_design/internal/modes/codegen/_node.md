---
version: 27
parent_version: 11
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
---

# ROOT/tech_design/internal/modes/codegen

## Intent

Technical design for the codegen mode: argument validation,
chain pre-loading, and tool registration.

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
    Frontmatter  *frontmatter.Frontmatter
    ChainContent string
}
```

`ChainContent` holds the fully concatenated chain loaded during
setup, ready to be returned by `load_context`.

### Chain output format

The chain is serialized as a sequence of file sections using
heredoc-style delimiters with a UUID generated once per setup
to avoid collisions with file content.

Opening delimiter: `<<<FILE_<uuid>>>`
Closing delimiter: `<<<END_FILE_<uuid>>>`

The same UUID is used for all files in the chain. Each section
includes `node:` and `path:` headers between the opening
delimiter and the file content, separated by a blank line.

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
```

### WriteFileArgs type

```go
type WriteFileArgs struct {
    Path    string `json:"path" jsonschema:"Relative file path from project root."`
    Content string `json:"content" jsonschema:"Complete file content to write."`
}
```

### Tool handler signatures

Tool handlers are defined as package-level functions in
sibling files. `Setup` registers them as closures that
capture the `Target`.

```go
func handleLoadContext(target *Target) func(
    ctx context.Context,
    req *mcp.CallToolRequest,
    _ struct{},
) (*mcp.CallToolResult, any, error)

func handleWriteFile(target *Target) func(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args WriteFileArgs,
) (*mcp.CallToolResult, any, error)
```

Each function takes a `*Target` and returns the closure that
`Setup` passes to `mcp.AddTool`.
