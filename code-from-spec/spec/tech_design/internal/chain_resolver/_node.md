---
version: 55
parent_version: 11
depends_on:
  - path: EXTERNAL/codefromspec
    version: 1
  - path: ROOT/tech_design/internal/frontmatter
    version: 27
  - path: ROOT/tech_design/internal/logical_names
    version: 24
implements:
  - internal/chainresolver/chainresolver.go
---

# ROOT/tech_design/internal/chain_resolver

## Intent

Resolves the ordered list of files that form the chain for a
given target logical name.

## Context

### Package

`package chainresolver`

## Contracts

### Types

```go
type ChainItem struct {
    LogicalName string
    FilePaths   []string
}

type Chain struct {
    Ancestors    []ChainItem
    Target       ChainItem
    Dependencies []ChainItem
}

func ResolveChain(targetLogicalName string) (*Chain, error)
```

`ResolveChain` returns the chain separated into ancestors, target,
and dependencies. `Ancestors` and `Dependencies` are sorted by
logical name alphabetically. Returns an error if the chain cannot
be built.

### Algorithm

**Step 1 — Ancestors and Target**

Starting from the target logical name, repeatedly call
`ParentLogicalName` to walk upward, collecting each
logical name. Sort the list by logical name alphabetically.

For each logical name, call `PathFromLogicalName` to
resolve the file path and create a `ChainItem` with a
single-element `FilePaths` list.

The last item in the sorted list is the `Target`; the
remaining items form `Ancestors`.

**Step 2 — Dependencies**

Read the target node's frontmatter using `ParseFrontmatter`.
If the target is a `TEST/` node, also read the parent leaf
node's frontmatter. Collect all `DependsOn` entries from both
and process them together.

For each entry in `DependsOn` whose `LogicalName` starts with
`ROOT/`:
1. Call `PathFromLogicalName` to get the file path.
2. Verify the file exists on disk (using `os.Stat`). If it does
   not exist, return error: `"cannot resolve logical name: <name>"`.
3. Add a `ChainItem` with a single-element `FilePaths` list to
   `Dependencies`.

For each entry in `DependsOn` whose `LogicalName` starts with
`EXTERNAL/`:
1. Call `PathFromLogicalName` to get the `_external.md` path.
2. If the entry has a non-empty `Filter`, include `_external.md`
   plus files in the dependency folder and any subfolders matching
   any pattern. If no `Filter` is present, include all files in
   the dependency folder and any subfolders. File paths are sorted
   and relative to project root.
3. Add a `ChainItem` with the collected `FilePaths` list to
   `Dependencies`.

Sort `Dependencies` by logical name alphabetically.

**Step 3 — Deduplicate file paths**

Review the `Ancestors` and `Dependencies` lists and remove
duplicate file paths. Each file path must appear only once
across the chain. When a path appears more than once, keep the
first occurrence and discard subsequent ones.

### Error handling

- If `PathFromLogicalName` returns false for any logical name →
  return error: `"cannot resolve logical name: <name>"`.
- If `ParseFrontmatter` fails → return error wrapping the
  underlying error.
- If a glob pattern fails to evaluate → return error:
  `"error evaluating filter <pattern> for <EXTERNAL/name>: <err>"`.
