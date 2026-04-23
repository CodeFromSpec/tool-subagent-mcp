---
version: 33
parent_version: 3
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/tech_design/internal/frontmatter
    version: 27
  - path: ROOT/tech_design/internal/logical_names
    version: 26
  - path: ROOT/tech_design/internal/pathvalidation
    version: 10
implements:
  - internal/write_file/write_file.go
---

# ROOT/tech_design/internal/tools/write_file

## Intent

Implements the `write_file` tool handler: resolves the node's
frontmatter from the provided logical name, validates the
target path against the `implements` list and the project
root, then writes the file to disk.

## Context

### Package

`package write_file`

### Target node

The target node is identified by its logical name — either a leaf
spec node (`ROOT/...`) or a test node (`TEST/...`). Examples:
`ROOT/payments/fees/calculation`,
`TEST/payments/fees/calculation`.

## Contracts

### Tool definition

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

### Handler

```go
func HandleWriteFile(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args WriteFileArgs,
) (*mcp.CallToolResult, any, error)
```

### Algorithm

1. Validate that `args.LogicalName` starts with `ROOT/` or
   `TEST/` (or equals `ROOT` or `TEST`). If not, return a
   tool error.
2. Call `PathFromLogicalName`. If it returns false, return a
   tool error: `"invalid logical name: <name>"`.
3. Call `ParseFrontmatter` on the resolved path. If it fails,
   return a tool error wrapping the underlying error.
4. Validate `Implements` is not empty → tool error:
   `"node <name> has no implements"`.
5. Normalize `args.Path` to forward slashes using
   `filepath.ToSlash`.
6. Call `ValidatePath` on the normalized path against the
   working directory. If it fails, return a tool error with
   the validation error and the list of valid `implements`
   paths.
7. Check that the normalized path appears in the frontmatter's
   `Implements` (exact string match). If not, return a tool
   error listing the valid paths.
8. Create any missing intermediate directories for the target
   path.
9. Write `args.Content` to the file, overwriting if it exists.
10. Return a success result with text `"wrote <path>"`.

### Error handling

- Invalid logical name → tool error with the name.
- Frontmatter parse failure → tool error wrapping the error.
- No implements → tool error: `"node <name> has no implements"`.
- Path validation failure → tool error with the violation and
  the list of allowed paths.
- Path not in implements → tool error: `"path not allowed:
  <path>. allowed paths: <list>"`.
- Directory creation failure → tool error: `"failed to create
  directories for <path>: <err>"`.
- Write failure → tool error: `"failed to write <path>:
  <err>"`.

## Constraints

- The target argument must be a logical name that resolves to a
  node with `implements`. Absent, empty, or invalid values cause
  the tool to report an error.
- Writes are limited to `implements`.
- The validation against `implements` is the security boundary of
  `write_file`. It must not be bypassable.
- Exactly one file is written per `write_file` call.

## Decisions

### write_file validates against implements

The target node's `implements` field is the authoritative list of
files this tool may produce. Validating every write against it
prevents the subagent from writing to paths outside the declared
scope, whether by mistake or hallucination.
