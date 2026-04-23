---
version: 14
parent_version: 56
implements:
  - internal/chainresolver/chainresolver_test.go
---

# TEST/tech_design/internal/chain_resolver

## Context

Tests use `t.TempDir()` to create an isolated project structure
with spec nodes, test nodes, and external dependencies. Each test
builds the minimal filesystem needed and calls `ResolveChain`.

File paths in `ChainItem.FilePaths` use forward slashes regardless
of the OS. Test assertions must use forward slashes.

## Happy Path

### Leaf node — ancestors only, no dependencies

Create a spec tree: `ROOT`, `ROOT/a`, `ROOT/a/b` (leaf).

Input: `"ROOT/a/b"`

Expect:
- `Ancestors`: `ROOT`, `ROOT/a` (sorted alphabetically)
- `Target`: `ROOT/a/b`
- `Dependencies`: empty
- `Code`: empty

### Leaf node — with ROOT/ dependency

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b`), `ROOT/b`.

Input: `"ROOT/a"`

Expect:
- `Ancestors`: `ROOT`
- `Target`: `ROOT/a`
- `Dependencies`: one item `ROOT/b` with single file path

### Leaf node — with EXTERNAL/ dependency, no filter

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`EXTERNAL/db`). Create external dependency `db` with
`_external.md` and `schema.sql`.

Input: `"ROOT/a"`

Expect:
- `Dependencies`: one item `EXTERNAL/db` with `FilePaths`
  containing `_external.md` and `schema.sql` (sorted)

### Leaf node — with EXTERNAL/ dependency, with filter

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`EXTERNAL/api` with filter `["endpoints/*.md"]`). Create
external dependency `api` with `_external.md`,
`endpoints/create.md`, `endpoints/delete.md`, `types.md`.

Input: `"ROOT/a"`

Expect:
- `Dependencies`: one item `EXTERNAL/api` with `FilePaths`
  containing `_external.md`, `endpoints/create.md`,
  `endpoints/delete.md` (sorted). `types.md` excluded by filter.

### Test node — includes parent leaf's dependencies

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`EXTERNAL/db`). Create test node `TEST/a` with its own
depends_on `EXTERNAL/fixtures`.

Input: `"TEST/a"`

Expect:
- `Ancestors`: `ROOT`, `ROOT/a` (parent leaf in ancestors)
- `Target`: `TEST/a`
- `Dependencies`: `EXTERNAL/db` and `EXTERNAL/fixtures`
  (sorted alphabetically)

### Test node — no own dependencies

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b`), `ROOT/b`. Create test node `TEST/a` with no
depends_on.

Input: `"TEST/a"`

Expect:
- `Ancestors`: `ROOT`, `ROOT/a`
- `Target`: `TEST/a`
- `Dependencies`: one item `ROOT/b` (from parent leaf)

### Dependencies sorted alphabetically

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/z`, `ROOT/m`, `ROOT/b`).

Input: `"ROOT/a"`

Expect:
- `Dependencies` sorted: `ROOT/b`, `ROOT/m`, `ROOT/z`

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

### Leaf node — some implements files exist, some do not

Create a spec tree: `ROOT`, `ROOT/a` (leaf with
`implements: ["src/a.go", "src/b.go"]`). Create only
`src/a.go` on disk.

Input: `"ROOT/a"`

Expect:
- `Code`: `["src/a.go"]`

## Edge Cases

### Test node — shared EXTERNAL/ dependency is deduplicated

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`EXTERNAL/db`). Create test node `TEST/a` with depends_on
`EXTERNAL/db`. Create external dependency `db` with
`_external.md` and `schema.sql`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: one item `EXTERNAL/db` (not two), with
  `FilePaths` containing `_external.md` and `schema.sql`

### Duplicate ROOT/ dependency is deduplicated

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`ROOT/b` listed twice), `ROOT/b`.

Input: `"ROOT/a"`

Expect:
- `Dependencies`: one item `ROOT/b` (not two)

### EXTERNAL/ with overlapping filters — files deduplicated

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`EXTERNAL/api`). Create test node `TEST/a` with depends_on
`EXTERNAL/api` with filter `["docs/*"]`. Create external
dependency `api` with `_external.md`, `docs/ref.md`,
`types.md`.

Input: `"TEST/a"`

Expect:
- `Dependencies`: one item `EXTERNAL/api`. `FilePaths`
  contains `_external.md`, `docs/ref.md`, `types.md` — each
  appearing only once. The leaf's unfiltered reference imports
  the full folder; the test node's filtered reference does not
  add duplicates.

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

### Invalid glob pattern in filter

Create a spec tree: `ROOT`, `ROOT/a` (leaf with depends_on
`EXTERNAL/api` with filter `["[invalid"]`). Create external
dependency `api` with `_external.md` and `data.txt`.

Input: `"ROOT/a"`

Expect error containing `"error evaluating filter"`.
