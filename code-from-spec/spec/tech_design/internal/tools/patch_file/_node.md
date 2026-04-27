---
version: 5
parent_version: 5
depends_on:
  - path: EXTERNAL/bluekeyes-go-gitdiff
    version: 1
  - path: EXTERNAL/mcp-go-sdk
    version: 2
  - path: ROOT/tech_design/internal/frontmatter
    version: 32
  - path: ROOT/tech_design/internal/logical_names
    version: 29
  - path: ROOT/tech_design/internal/pathvalidation
    version: 11
implements:
  - internal/patch_file/patch_file.go
---

# ROOT/tech_design/internal/tools/patch_file

## Intent

Implements the `patch_file` tool handler: applies a unified diff
to an existing file, after validating the target path against the
node's `implements` list and the project root.

## Context

### Package

`package patch_file`

### Dependencies

- `github.com/bluekeyes/go-gitdiff/gitdiff` — parsing and
  applying unified diffs.

## Contracts

### Tool definition

Name: `patch_file`
Description: `"Apply a unified diff to an existing source file. The path must be one of the files declared in the node's implements list. The file must already exist."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the node whose implements list authorizes the write. |
| `path` | string | yes | Relative file path from project root. |
| `diff` | string | yes | Unified diff to apply to the file. |

### PatchFileArgs type

```go
type PatchFileArgs struct {
    LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
    Path        string `json:"path" jsonschema:"Relative file path from project root."`
    Diff        string `json:"diff" jsonschema:"Unified diff to apply to the file."`
}
```

### Handler

```go
func HandlePatchFile(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args PatchFileArgs,
) (*mcp.CallToolResult, any, error)
```

### Algorithm

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
9. Parse the unified diff from `args.Diff` using
   `gitdiff.Parse`. If parsing fails (e.g. the diff is
   semantically invalid), return a tool error:
   `"failed to parse diff: <err>"`. Note: completely
   malformed input that `gitdiff.Parse` cannot interpret
   as a diff will result in zero file entries (no error),
   which is caught by step 10.
10. The parse result must contain exactly one file entry.
    If it contains zero or more than one, return a tool error:
    `"diff must contain exactly one file"`. This also covers
    diffs that are so malformed that `gitdiff.Parse` produces
    no file entries.
11. Apply the patch to the file content using `gitdiff.Apply`.
    If application fails, return a tool error:
    `"failed to apply diff to <path>: <err>"`. Note: context
    mismatches between the diff and the file may be caught
    by `gitdiff.Parse` (step 9) rather than `gitdiff.Apply`.
12. Write the patched content back to the file, overwriting
    the original.
13. Return a success result with text `"patched <path>"`.

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
- Diff parse failure (semantic error from `gitdiff.Parse`) →
  tool error: `"failed to parse diff: <err>"`.
- Diff has wrong number of files (including malformed input
  that produces zero entries) → tool error:
  `"diff must contain exactly one file"`.
- Apply failure → tool error:
  `"failed to apply diff to <path>: <err>"`. Context
  mismatches may surface as parse errors (step 9) instead.
- Write failure → tool error:
  `"failed to write <path>: <err>"`.

## Constraints

- The target argument must be a logical name that resolves to a
  node with `implements`. Absent, empty, or invalid values cause
  the tool to report an error.
- Writes are limited to `implements`.
- The validation against `implements` is the security boundary.
  It must not be bypassable.
- The file must already exist. `patch_file` does not create new
  files — use `write_file` for that.
- Exactly one file is patched per `patch_file` call.
- The diff must target exactly one file.
