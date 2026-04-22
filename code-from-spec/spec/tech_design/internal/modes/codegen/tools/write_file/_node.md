---
version: 26
parent_version: 18
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes/codegen
    version: 21
  - path: ROOT/tech_design/internal/pathvalidation
    version: 8
implements:
  - internal/modes/codegen/write_file.go
---

# ROOT/tech_design/internal/modes/codegen/tools/write_file

## Intent

Implements the `write_file` tool handler: validates the target
path against the node's `implements` list and the project root,
then writes the file to disk.

## Contracts

### Handler

Follows the `handleWriteFile` signature defined in the parent
codegen node. Receives `WriteFileArgs` with `Path` and `Content`
already deserialized by the MCP SDK.

### Algorithm

1. If `currentTarget == nil`, return a tool error:
   `"load_context must be called before write_file"`.
2. Call `ValidatePath` on `args.Path` against the working
   directory. If it fails, return a tool error with the
   validation error and the list of valid `implements` paths.
3. Check that `args.Path` appears in
   `currentTarget.Frontmatter.Implements` (exact string match).
   If not, return a tool error listing the valid paths.
4. Create any missing intermediate directories for the target
   path.
5. Write `args.Content` to the file, overwriting if it exists.
6. Return a success result with text `"wrote <path>"`.

### Error handling

- Path validation failure → tool error with the violation and
  the list of allowed paths.
- Path not in implements → tool error: `"path not allowed:
  <path>. allowed paths: <list>"`.
- Directory creation failure → tool error: `"failed to create
  directories for <path>: <err>"`.
- Write failure → tool error: `"failed to write <path>:
  <err>"`.
