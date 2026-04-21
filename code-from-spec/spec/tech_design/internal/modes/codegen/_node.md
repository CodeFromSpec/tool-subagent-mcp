---
version: 35
parent_version: 18
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
  - path: ROOT/tech_design/internal/pathvalidation
    version: 7
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

### Help message

Exposed as `HelpMessage()`. The server calls it and prints
the result when the user runs `subagent-mcp codegen --help`.

```
Usage: subagent-mcp codegen <logical-name>

Starts an MCP server over stdin/stdout that provides tools
for code generation. The logical name identifies a spec or
test node that implements source code files.

The server exposes two tools:
  load_context   Returns the context for code generation.
  write_file     Writes a generated file to disk.

The subagent should have no other tools available — no file
read, write, or search capabilities beyond what this server
provides. This confinement ensures the subagent works only
from the provided context and writes only to declared outputs.

MCP configuration example:
  {
    "mcpServers": {
      "subagent-mcp": {
        "type": "stdio",
        "command": "<path-to-binary>",
        "args": ["codegen", "<logical-name>"]
      }
    }
  }
```

### Server instructions

Exposed as a package-level constant `Instructions`. The server
passes it to `mcp.ServerOptions.Instructions` when creating the
MCP server.

```
How to use this MCP server:

1. Call load_context once to receive the context for code
   generation. Multiple calls are wasteful as it always
   returns the same content.
2. Generate the code.
3. Call write_file once per file to write the result.
```

### Tool definitions

#### load_context

Name: `load_context`
Description: `"Load the context for code generation. Returns all relevant spec files concatenated in a single response."`
No input parameters.

#### write_file

Name: `write_file`
Description: `"Write a generated source file to disk. The path must be one of the files declared in the current node's implements list. Overwrites existing content."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | Relative file path from project root. |
| `content` | string | yes | Complete file content to write. |

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

### Path validation — defense in depth

File paths from `implements` are validated using `ValidatePath`
at two points: once during setup (rejects the entire invocation
if any path is invalid) and again in the `write_file` handler
before each write. This ensures that even if the setup
validation is bypassed or the `Target` struct is constructed
incorrectly, the write is still blocked.
