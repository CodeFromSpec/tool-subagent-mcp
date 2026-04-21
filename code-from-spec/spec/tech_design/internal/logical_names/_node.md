---
version: 19
parent_version: 5
depends_on:
  - path: EXTERNAL/codefromspec
    version: 1
implements:
  - internal/logicalnames/logicalnames.go
---

# ROOT/tech_design/internal/logical_names

## Intent

Centralizes conversion between logical names and file paths.

## Context

### Package

`package logicalnames`

## Contracts

### Interface

```go
func PathFromLogicalName(logicalName string) (string, bool)
func HasParent(logicalName string) (hasParent, ok bool)
func ParentLogicalName(logicalName string) (string, bool)
```

### PathFromLogicalName

Resolves a logical name to a file path relative to the
project root.

| Logical name | File path |
|---|---|
| `ROOT` | `code-from-spec/spec/_node.md` |
| `ROOT/x/y` | `code-from-spec/spec/x/y/_node.md` |
| `TEST` | `code-from-spec/spec/default.test.md` |
| `TEST/x` | `code-from-spec/spec/x/default.test.md` |
| `TEST/x(name)` | `code-from-spec/spec/x/name.test.md` |
| `EXTERNAL/x` | `code-from-spec/external/x/_external.md` |

Rules:
- `ROOT` → `code-from-spec/spec/_node.md`
- `ROOT/<path>` → `code-from-spec/spec/<path>/_node.md`
- `TEST` → `code-from-spec/spec/default.test.md`
- `TEST/<path>` → `code-from-spec/spec/<path>/default.test.md`
- `TEST/<path>(<name>)` → `code-from-spec/spec/<path>/<name>.test.md`
- `EXTERNAL/<name>` → `code-from-spec/external/<name>/_external.md`

### HasParent

Determines whether a logical name has a parent node.
Returns `(hasParent, ok)` where `ok` indicates whether
the input is a valid logical name.

| Logical name | hasParent | ok |
|---|---|---|
| `ROOT` | `false` | `true` |
| `ROOT/x` | `true` | `true` |
| `TEST` | `true` | `true` |
| `TEST/x` | `true` | `true` |
| `TEST/x(name)` | `true` | `true` |
| `EXTERNAL/x` | `false` | `true` |
| `EXTERNAL` | `false` | `false` |
| `""` | `false` | `false` |

Rules:
- `ROOT` → no parent
- `ROOT/<path>` → has parent
- `TEST` and `TEST/<path>` and `TEST/<path>(<name>)` →
  has parent (parent is always in the ROOT namespace)
- `EXTERNAL/<name>` → no parent
- Anything else → not a valid logical name

### ParentLogicalName

Derives the parent's logical name from a node's logical
name. Returns `(parent, true)` on success, `("", false)`
if the node has no parent. Only call after confirming
`HasParent` returns `true`.

| Logical name | Parent |
|---|---|
| `ROOT/x` | `ROOT` |
| `ROOT/x/y` | `ROOT/x` |
| `TEST` | `ROOT` |
| `TEST/x` | `ROOT/x` |
| `TEST/x(name)` | `ROOT/x` |

Rules:
- `ROOT/<path>` → strip last segment. If only one
  segment remains, parent is `ROOT`.
- `TEST` → `ROOT`
- `TEST/<path>` → `ROOT/<path>`
- `TEST/<path>(<name>)` → `ROOT/<path>`

### Error handling

These are pure functions operating on strings. They do
not perform I/O or return errors.
`PathFromLogicalName` returns `(result, true)` on success
and `("", false)` if the input does not match any known
pattern.
