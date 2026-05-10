---
version: 30
parent_version: 12
depends_on:
  - path: ROOT/external/codefromspec
    version: 4
implements:
  - internal/logicalnames/logicalnames.go
---

# ROOT/tech_design/internal/logical_names

Centralizes conversion between logical names and file paths.

# Public

## Context

### Package

`package logicalnames`

## Contracts

### Interface

```go
func PathFromLogicalName(logicalName string) (string, bool)
func HasParent(logicalName string) (hasParent, ok bool)
func ParentLogicalName(logicalName string) (string, bool)
func HasQualifier(logicalName string) (hasQualifier, ok bool)
func QualifierName(logicalName string) (string, bool)
```

### PathFromLogicalName

Resolves a logical name to a file path relative to the
project root. Returned paths always use forward slashes as
separators, regardless of the operating system. Use
`filepath.ToSlash` on the result before returning.

If the logical name has a parenthetical qualifier, it is
stripped before resolving the path. `ROOT/x(y)` resolves
to the same path as `ROOT/x`.

| Logical name | File path |
|---|---|
| `ROOT` | `code-from-spec/_node.md` |
| `ROOT/x/y` | `code-from-spec/x/y/_node.md` |
| `ROOT/x/y(z)` | `code-from-spec/x/y/_node.md` |
| `TEST` | `code-from-spec/default.test.md` |
| `TEST/x` | `code-from-spec/x/default.test.md` |
| `TEST/x(name)` | `code-from-spec/x/name.test.md` |

Rules:
- `ROOT` → `code-from-spec/_node.md`
- `ROOT/<path>` → `code-from-spec/<path>/_node.md`
- `ROOT/<path>(<qualifier>)` → `code-from-spec/<path>/_node.md`
- `TEST` → `code-from-spec/default.test.md`
- `TEST/<path>` → `code-from-spec/<path>/default.test.md`
- `TEST/<path>(<name>)` → `code-from-spec/<path>/<name>.test.md`

### HasParent

Determines whether a logical name has a parent node.
Returns `(hasParent, ok)` where `ok` indicates whether
the input is a valid logical name.

| Logical name | hasParent | ok |
|---|---|---|
| `ROOT` | `false` | `true` |
| `ROOT/x` | `true` | `true` |
| `ROOT/x(y)` | `true` | `true` |
| `TEST` | `true` | `true` |
| `TEST/x` | `true` | `true` |
| `TEST/x(name)` | `true` | `true` |
| `""` | `false` | `false` |

Rules:
- `ROOT` → no parent
- `ROOT/<path>` and `ROOT/<path>(<qualifier>)` → has parent
- `TEST` and `TEST/<path>` and `TEST/<path>(<name>)` →
  has parent (parent is always in the ROOT namespace)
- Anything else → not a valid logical name

### ParentLogicalName

Derives the parent's logical name from a node's logical
name. Returns `(parent, true)` on success, `("", false)`
if the node has no parent. Only call after confirming
`HasParent` returns `true`.

The qualifier is stripped before deriving the parent.

| Logical name | Parent |
|---|---|
| `ROOT/x` | `ROOT` |
| `ROOT/x/y` | `ROOT/x` |
| `ROOT/x/y(z)` | `ROOT/x` |
| `TEST` | `ROOT` |
| `TEST/x` | `ROOT/x` |
| `TEST/x(name)` | `ROOT/x` |

Rules:
- `ROOT/<path>` → strip last segment. If only one
  segment remains, parent is `ROOT`.
- `ROOT/<path>(<qualifier>)` → strip qualifier, then
  strip last segment.
- `TEST` → `ROOT`
- `TEST/<path>` → `ROOT/<path>`
- `TEST/<path>(<name>)` → `ROOT/<path>`

### HasQualifier

Determines whether a logical name has a parenthetical
qualifier. Returns `(hasQualifier, ok)` where `ok`
indicates whether the input is a valid logical name.

| Logical name | hasQualifier | ok |
|---|---|---|
| `ROOT` | `false` | `true` |
| `ROOT/x` | `false` | `true` |
| `ROOT/x(y)` | `true` | `true` |
| `ROOT/x/y(z)` | `true` | `true` |
| `TEST` | `false` | `true` |
| `TEST/x` | `false` | `true` |
| `TEST/x(name)` | `true` | `true` |
| `""` | `false` | `false` |

### QualifierName

Extracts the qualifier from a logical name. Returns
`(qualifier, true)` on success, `("", false)` if there
is no qualifier. Only call after confirming `HasQualifier`
returns `true`.

| Logical name | Qualifier |
|---|---|
| `ROOT/x(y)` | `"y"` |
| `ROOT/x/y(z)` | `"z"` |
| `TEST/x(name)` | `"name"` |
| `ROOT/x` | `""`, `false` |
| `ROOT` | `""`, `false` |

### Error handling

These are pure functions operating on strings. They do
not perform I/O or return errors.
`PathFromLogicalName` returns `(result, true)` on success
and `("", false)` if the input does not match any known
pattern.
