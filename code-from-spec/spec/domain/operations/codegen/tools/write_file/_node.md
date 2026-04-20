---
version: 2
parent_version: 2
---

# ROOT/domain/operations/codegen/tools/write_file

## Intent

Writes a generated file to disk, enforcing that the target path
is declared in the session node's `implements` field.

## Contracts

### Inputs

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `path`    | string | Relative path to write (from project root). |
| `content` | string | Full file content to write. |

### Validation

Before writing, the tool checks that `path` appears in the
`implements` list of the session leaf node's frontmatter. If it
does not, the tool returns an error and does not write anything.

### Output

On success: a confirmation message indicating the file was written.

On validation failure: an error identifying the rejected path and
the valid `implements` paths for this session.

### File creation

If the target file does not exist, it is created (including any
intermediate directories). If it does exist, it is overwritten.
The write is unconditional once validation passes.

## Constraints

- Exactly one file is written per call. The tool does not support
  writing multiple files in a single call.
- `path` is always relative to the project root. Absolute paths
  and paths with `..` components are rejected.
- The validation against `implements` is the security boundary of
  this tool. It must not be bypassable.
