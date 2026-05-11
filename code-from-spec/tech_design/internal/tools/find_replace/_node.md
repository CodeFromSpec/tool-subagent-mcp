---
version: 2
parent_version: 8
depends_on:
  - path: ROOT/external/mcp-go-sdk
    version: 4
  - path: ROOT/tech_design/internal/frontmatter
    version: 36
  - path: ROOT/tech_design/internal/logical_names
    version: 33
  - path: ROOT/tech_design/internal/pathvalidation
    version: 14
implements:
  - internal/find_replace/find_replace.go
---

# ROOT/tech_design/internal/tools/find_replace

Implements the `find_replace` tool handler: finds a unique
string in an existing file and replaces it, after validating
the target path against the node's `implements` list and the
project root.

# Public

## Package

`package find_replace`

## Target node

The target node is identified by its logical name — either a leaf
spec node (`ROOT/...`) or a test node (`TEST/...`). Examples:
`ROOT/payments/fees/calculation`,
`TEST/payments/fees/calculation`.

## Interface

### Tool definition

Name: `find_replace`
Description: `"Replace a specific string in an existing source file. The old_string must appear exactly once. The path must be one of the files declared in the node's implements list. The file must already exist."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the node whose implements list authorizes the write. |
| `path` | string | yes | Relative file path from project root. |
| `old_string` | string | yes | Exact string to find in the file. Must match exactly once. |
| `new_string` | string | yes | Replacement string. |

### FindReplaceArgs type

```go
type FindReplaceArgs struct {
    LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
    Path        string `json:"path" jsonschema:"Relative file path from project root."`
    OldString   string `json:"old_string" jsonschema:"Exact string to find in the file. Must match exactly once."`
    NewString   string `json:"new_string" jsonschema:"Replacement string."`
}
```

### Handler

```go
func HandleFindReplace(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args FindReplaceArgs,
) (*mcp.CallToolResult, any, error)
```

# Private

## Implementation

1. Validate that `args.LogicalName` starts with `ROOT/` or
   `TEST/` (or equals `ROOT` or `TEST`). If not, return a
   tool error.
2. Call `logicalnames.PathFromLogicalName`. If it returns false, return a
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
8. Read the existing file at the target path. If the file does
   not exist, return a tool error:
   `"file does not exist: <path>"`.
9. Count occurrences of `args.OldString` in the file content
   using `strings.Count`. If count is 0, return a tool error:
   `"old_string not found in <path>"`. If count is greater
   than 1, return a tool error:
   `"old_string matches multiple locations in <path>"`.
10. Replace the single occurrence using
    `strings.Replace(content, args.OldString, args.NewString, 1)`.
11. Write the result back to the file, overwriting the original.
12. Return a success result with text `"edited <path>"`.

### Error handling

- Invalid logical name → tool error with the name.
- Frontmatter parse failure → tool error wrapping the error.
- No implements → tool error: `"node <name> has no implements"`.
- Path validation failure → tool error with the violation and
  the list of allowed paths.
- Path not in implements → tool error: `"path not allowed:
  <path>. allowed paths: <list>"`.
- File does not exist → tool error:
  `"file does not exist: <path>"`.
- old_string not found → tool error:
  `"old_string not found in <path>"`.
- old_string matches multiple locations → tool error:
  `"old_string matches multiple locations in <path>"`.
- Write failure → tool error:
  `"failed to write <path>: <err>"`.

## Constraints

- The target argument must be a logical name that resolves to a
  node with `implements`. Absent, empty, or invalid values cause
  the tool to report an error.
- Writes are limited to `implements`.
- The validation against `implements` is the security boundary.
  It must not be bypassable.
- The file must already exist. `find_replace` does not create new
  files — use `write_file` for that.
- `old_string` must match exactly once. Zero or multiple matches
  are rejected.
- Exactly one file is edited per `find_replace` call.
