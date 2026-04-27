---
version: 23
parent_version: 72
implements:
  - internal/chainresolver/chainresolver_test.go
---

# TEST/tech_design/internal/chain_resolver

## Context

Tests use `t.TempDir()` to create an isolated project structure
with spec nodes and test nodes. Each test builds the minimal
filesystem needed and calls `ResolveChain`.

Spec files are created at paths matching `logicalnames.PathFromLogicalName`:
- `ROOT` → `<tmpdir>/code-from-spec/_node.md`
- `ROOT/a` → `<tmpdir>/code-from-spec/a/_node.md`
- `TEST/a` → `<tmpdir>/code-from-spec/a/default.test.md`

The working directory must be changed to `<tmpdir>` before calling
`ResolveChain` and restored after.

File paths in `ChainItem.FilePath` use forward slashes regardless
of the OS. Test assertions must use forward slashes.

## Happy Path

### Leaf node — ancestors only, no dependencies

Create a spec tree: `ROOT`, `ROOT/a`, `ROOT/a/b` (leaf).

Input: `"ROOT/a/b"`

Expect:
- `Ancestors`: `ROOT`, `ROOT/a` (sorted alphabetically),
  each with `Qualifier` = nil
- `Target`: `ROOT/a/b` with `Qualifier` = nil
- `Dependencies`: empty
- `Code`: empty

### Leaf node — with ROOT/ dependency, no qualifier

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b`), `ROOT/b`.

Input: `"ROOT/a"`

Expect:
- `Ancestors`: `ROOT`
- `Target`: `ROOT/a`
- `Dependencies`: one item `ROOT/b` with `Qualifier` = nil

### Leaf node — with ROOT/ dependency, with qualifier

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b(interface)`), `ROOT/b`.

Input: `"ROOT/a"`

Expect:
- `Dependencies`: one item with `LogicalName` =
  `"ROOT/b(interface)"`, `FilePath` pointing to `ROOT/b`'s
  `_node.md`, `Qualifier` = pointer to `"interface"`

### Test node — includes subject's dependencies

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/c`), `ROOT/c`. Create test node `TEST/a` with its own
depends_on `ROOT/d`, `ROOT/d`.

Input: `"TEST/a"`

Expect:
- `Ancestors`: `ROOT`, `ROOT/a` (subject in ancestors)
- `Target`: `TEST/a`
- `Dependencies`: items from both `ROOT/c` and `ROOT/d`

### Test node — no own dependencies

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b`), `ROOT/b`. Create test node `TEST/a` with no
depends_on.

Input: `"TEST/a"`

Expect:
- `Ancestors`: `ROOT`, `ROOT/a`
- `Target`: `TEST/a`
- `Dependencies`: one item `ROOT/b` (from subject)

### Dependencies sorted

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/z`, `ROOT/m`, `ROOT/b`).

Input: `"ROOT/a"`

Expect:
- `Dependencies` sorted by `FilePath`

### Leaf node — implements file exists on disk

Create a spec tree: `ROOT`, `ROOT/a` (leaf with
`implements: ["src/a.go"]`). Create the file `src/a.go` on
disk.

Input: `"ROOT/a"`

Expect:
- `Code`: `["src/a.go"]`

### Leaf node — implements file does not exist

Create a spec tree: `ROOT`, `ROOT/a` (leaf with
`implements: ["src/a.go"]`). Do not create `src/a.go`.

Input: `"ROOT/a"`

Expect:
- `Code`: empty

### Multiple qualifiers for same file

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b(interface)` and `ROOT/b(constraints)`), `ROOT/b`.

Input: `"ROOT/a"`

Expect:
- `Dependencies`: two items, both pointing to `ROOT/b`'s
  file, one with `Qualifier` = `"interface"`, the other
  with `Qualifier` = `"constraints"`

## Edge Cases

### Dedup: same file, same qualifier

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b`). Create test node `TEST/a` with depends_on
`ROOT/b`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: one item `ROOT/b` with `Qualifier` = nil
  (not two)

### Dedup: same file, different qualifiers — both kept

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b(interface)`). Create test node `TEST/a` with
depends_on `ROOT/b(constraints)`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: two items for `ROOT/b`, one with
  `Qualifier` = `"interface"`, one with `"constraints"`

### Dedup: nil qualifier subsumes specific qualifiers

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b`). Create test node `TEST/a` with depends_on
`ROOT/b(interface)`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: one item `ROOT/b` with `Qualifier` = nil.
  The `ROOT/b(interface)` entry is removed because nil
  (whole `# Public`) already includes `## interface`.

### Dedup: specific qualifier appears before nil — nil wins

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b(interface)`). Create test node `TEST/a` with
depends_on `ROOT/b`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: one item `ROOT/b` with `Qualifier` = nil.
  Even though the specific qualifier appeared first, the nil
  entry subsumes it.

### Dedup: repeated qualifier for same file

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b(interface)`). Create test node `TEST/a` with
depends_on `ROOT/b(interface)`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: one item with `Qualifier` = `"interface"`
  (not two)

## Failure Cases

### Invalid logical name

Input: `"INVALID/something"`

Expect error containing `"cannot resolve logical name"`.

### Unreadable frontmatter

Create a spec tree: `ROOT`, `ROOT/a` (leaf). Write invalid
YAML in `ROOT/a`'s frontmatter.

Input: `"ROOT/a"`

Expect error from `ParseFrontmatter`.

### Unresolvable dependency

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/nonexistent`).

Input: `"ROOT/a"`

Expect error containing `"cannot resolve logical name"`.
