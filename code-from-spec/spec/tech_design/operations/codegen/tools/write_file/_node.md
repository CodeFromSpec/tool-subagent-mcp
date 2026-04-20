---
version: 2
parent_version: 2
depends_on:
  - path: ROOT/domain/operations/codegen/tools/write_file
    version: 2
  - path: ROOT/tech_design/frontmatter
    version: 2
implements:
  - cmd/subagent-mcp/operations/codegen/write_file.go
---

# ROOT/tech_design/operations/codegen/tools/write_file

## Intent

Implements the `write_file` tool handler: validates the target path
against the session's `implements` list, then writes the file.

## Contracts

### Tool registration

Name: `write_file`
Description: `"Write a generated source file to disk. The path must be one of the files declared in the current node's implements list. Overwrites existing content."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | Relative file path from project root. |
| `content` | string | yes | Complete file content to write. |

### Handler algorithm

1. Read `path` and `content` from the tool request arguments.
2. Validate `path`:
   a. Must not be empty.
   b. Must not be an absolute path (must not start with `/` or a
      drive letter like `C:`).
   c. Must not contain `..` as a path component.
   d. Must appear in `session.LeafFrontmatter.Implements`. Comparison
      is exact string match.
   If any validation fails, return a tool error describing the
   violation and listing the valid `implements` paths.
3. Create any missing intermediate directories for the target path.
4. Write `content` to the file, overwriting if it exists.
5. Return `mcp.NewToolResultText("wrote <path>")`.

### Error handling

- Validation errors → tool error with the specific violation and the
  list of allowed paths.
- Directory creation failure → tool error: `"failed to create
  directories for <path>: <err>"`.
- Write failure → tool error: `"failed to write <path>: <err>"`.
