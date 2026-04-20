---
version: 9
parent_version: 4
depends_on:
  - path: EXTERNAL/codefromspec
    version: 1
implements:
  - cmd/subagent-mcp/logicalnames.go
---

# ROOT/tech_design/logical_names

## Intent

Centralizes conversion between logical names and file paths. Used
by chain resolution and write_file validation.

## Context

This tool deals with two namespaces: `ROOT/` for spec nodes and
`EXTERNAL/` for external dependencies. Test nodes (`TEST/`) are not
part of the chain and are not handled here.

## Contracts

### Interface

```go
func PathFromLogicalName(logicalName string) (string, bool)
func HasParent(logicalName string) bool
func ParentLogicalName(logicalName string) (string, bool)
```

### PathFromLogicalName

Resolves a logical name to a file path relative to the project root.

| Logical name | File path |
|---|---|
| `ROOT` | `code-from-spec/spec/_node.md` |
| `ROOT/x` | `code-from-spec/spec/x/_node.md` |
| `ROOT/x/y` | `code-from-spec/spec/x/y/_node.md` |
| `EXTERNAL/x` | `code-from-spec/external/x/_external.md` |

Returns `(path, true)` on success, `("", false)` if the input does
not match any known pattern.

### HasParent

Determines whether a logical name has a parent in the ancestor chain.

| Logical name | Result |
|---|---|
| `ROOT` | `false` |
| `ROOT/x` | `true` |
| `ROOT/x/y` | `true` |
| `EXTERNAL/x` | `false` |

Returns `false` for any input that is not a valid logical name.

### ParentLogicalName

Derives the parent's logical name. Only call after confirming
`HasParent` returns `true`.

| Logical name | Parent |
|---|---|
| `ROOT/x` | `ROOT` |
| `ROOT/x/y` | `ROOT/x` |
| `ROOT/x/y/z` | `ROOT/x/y` |

Rules:
- `ROOT/<path>` → strip last `/`-separated segment. If no segment
  remains, parent is `ROOT`.

Returns `(parent, true)` on success, `("", false)` if the node has
no parent or the input is invalid.

### Error handling

Pure functions operating on strings. No I/O, no errors.
