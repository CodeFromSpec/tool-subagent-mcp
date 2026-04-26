---
version: 71
parent_version: 12
depends_on:
  - path: EXTERNAL/codefromspec
    version: 3
  - path: ROOT/tech_design/internal/frontmatter
    version: 32
  - path: ROOT/tech_design/internal/logical_names
    version: 28
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
    FilePath    string
    Qualifier   *string
}

type Chain struct {
    Ancestors    []ChainItem
    Target       ChainItem
    Dependencies []ChainItem
    Code         []string
}

func ResolveChain(targetLogicalName string) (*Chain, error)
```

`ResolveChain` returns the chain separated into ancestors, target,
and dependencies. Returns an error if the chain cannot be built.

Each `ChainItem` has a single `FilePath` and an optional
`Qualifier`. When `Qualifier` is nil, the caller should use
the `# Public` section of the file. When `Qualifier` is
non-nil, the caller should use only the `## <qualifier>`
subsection within `# Public`.

### Algorithm

**Step 1 — Ancestors and Target**

Starting from the target logical name, repeatedly call
`logicalnames.ParentLogicalName` to walk upward, collecting each
logical name. Sort the list by logical name alphabetically.

For each logical name, call `logicalnames.PathFromLogicalName` to
resolve the file path and create a `ChainItem` with
`Qualifier` = nil.

The last item in the sorted list is the `Target`; the
remaining items form `Ancestors`.

**Step 2 — Dependencies**

Read the target node's frontmatter using `ParseFrontmatter`.
If the target is a `TEST/` node, also read the subject
node's frontmatter. Collect all `DependsOn` entries from both
and process them together.

For each entry in `DependsOn`:
1. Call `logicalnames.PathFromLogicalName` to get the file path.
2. Determine the qualifier: call `logicalnames.HasQualifier` and
   `logicalnames.QualifierName` on the logical name. If the logical name
   has a qualifier, set `Qualifier` to that value. Otherwise,
   set `Qualifier` to nil.
3. Verify the file exists on disk (using `os.Stat`). If it
   does not exist, return error:
   `"cannot resolve logical name: <name>"`.
4. Add a `ChainItem` with the file path and qualifier to
   `Dependencies`.

Sort `Dependencies` alphabetically by `FilePath`, then by
`Qualifier` (nil sorts before non-nil).

**Step 3 — Code**

Read the target node's frontmatter using `ParseFrontmatter`
and extract the `Implements` list. For each path in
`Implements`, check if the file exists on disk (using
`os.Stat`). If it exists, add the path to `Code`. If it does
not exist, skip it. `Code` contains only files that already
exist.

**Step 4 — Normalize file paths**

Convert all file paths in `Ancestors`, `Target`,
`Dependencies`, and `Code` to use forward slashes as
separators, regardless of the operating system. Use
`filepath.ToSlash`.

**Step 5 — Deduplicate**

Review `Ancestors` and `Dependencies` and remove duplicate
entries. Two entries are considered duplicates when they have
the same `FilePath` and the same `Qualifier`.

Additionally, when an entry exists with a given `FilePath`
and `Qualifier` = nil (meaning the entire `# Public` section),
any other entry with the same `FilePath` and a non-nil
`Qualifier` is redundant and must be removed — the full
`# Public` already includes every subsection.

When removing duplicates, keep the first occurrence.

### Error handling

- If `logicalnames.PathFromLogicalName` returns false for any logical name →
  return error: `"cannot resolve logical name: <name>"`.
- If `ParseFrontmatter` fails → return error wrapping the
  underlying error.
