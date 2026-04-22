---
version: 27
parent_version: 18
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes/codegen
    version: 21
  - path: ROOT/tech_design/internal/frontmatter
    version: 25
  - path: ROOT/tech_design/internal/logical_names
    version: 22
  - path: ROOT/tech_design/internal/pathvalidation
    version: 8
implements:
  - internal/modes/codegen/write_file.go
---

# ROOT/tech_design/internal/modes/codegen/tools/write_file

## Intent

Implements the `write_file` tool handler: resolves the node's
frontmatter from the provided logical name, validates the
target path against the `implements` list and the project
root, then writes the file to disk.

## Contracts

### Handler

Follows the `handleWriteFile` signature defined in the parent
codegen node. Receives `WriteFileArgs` with `LogicalName`,
`Path`, and `Content` already deserialized by the MCP SDK.

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
5. Call `ValidatePath` on `args.Path` against the working
   directory. If it fails, return a tool error with the
   validation error and the list of valid `implements` paths.
6. Check that `args.Path` appears in the frontmatter's
   `Implements` (exact string match). If not, return a tool
   error listing the valid paths.
7. Create any missing intermediate directories for the target
   path.
8. Write `args.Content` to the file, overwriting if it exists.
9. Return a success result with text `"wrote <path>"`.

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
