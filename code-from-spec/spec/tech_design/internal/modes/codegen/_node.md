---
version: 37
parent_version: 22
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
  - path: ROOT/tech_design/internal/pathvalidation
    version: 8
---

# ROOT/tech_design/internal/modes/codegen

## Intent

Technical design for the codegen mode: tool registration
and handler contracts.

## Context

### Package

`package codegen`

Directory: `internal/modes/codegen/`

All leaf nodes under this subtree generate files in this
package and directory.

## Contracts

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
Usage: subagent-mcp codegen

Starts an MCP server over stdin/stdout that provides tools
for code generation.

The server exposes two tools:
  load_context   Loads the context for a spec node and returns it.
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
        "args": ["codegen"]
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

1. Call load_context with the logical name of the node
   to generate code for.
2. Generate the code from the returned context.
3. Call write_file once per file to write the result,
   passing the same logical_name used in load_context.
```

### Tool definitions

#### load_context

Name: `load_context`
Description: `"Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the node to generate code for. |

#### write_file

Name: `write_file`
Description: `"Write a generated source file to disk. The path must be one of the files declared in the node's implements list. Overwrites existing content."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the node whose implements list authorizes the write. |
| `path` | string | yes | Relative file path from project root. |
| `content` | string | yes | Complete file content to write. |

### WriteFileArgs type

```go
type WriteFileArgs struct {
    LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
    Path        string `json:"path" jsonschema:"Relative file path from project root."`
    Content     string `json:"content" jsonschema:"Complete file content to write."`
}
```

### LoadContextArgs type

```go
type LoadContextArgs struct {
    LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}
```

### Tool handler signatures

Tool handlers are defined as package-level functions in
sibling files and registered directly by `Setup`.

```go
func handleLoadContext(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args LoadContextArgs,
) (*mcp.CallToolResult, any, error)

func handleWriteFile(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args WriteFileArgs,
) (*mcp.CallToolResult, any, error)
```

### Path validation

File paths from `implements` are validated using `ValidatePath`
in both `load_context` and `write_file` before any write.
Each handler resolves the frontmatter independently and
validates the paths against the working directory.
