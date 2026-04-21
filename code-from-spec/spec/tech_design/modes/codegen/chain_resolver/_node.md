---
version: 10
parent_version: 8
depends_on:
  - path: EXTERNAL/codefromspec
    version: 1
  - path: ROOT/tech_design/logical_names
    version: 12
  - path: ROOT/tech_design/frontmatter
    version: 9
implements:
  - cmd/subagent-mcp/modes/codegen/chainresolver.go
---

# ROOT/tech_design/modes/codegen/chain_resolver

## Intent

Resolves the ordered list of file paths that form the chain for a
given leaf node logical name.

## Contracts

### Types

```go
type ChainEntry struct {
    LogicalName string
    FilePath    string
}

func ResolveChain(leafLogicalName string) ([]ChainEntry, error)
```

`ResolveChain` returns the chain entries in the order defined by
`ROOT/domain/modes/codegen/chain`. Returns an error if the
chain cannot be built (unresolvable path, unreadable frontmatter).

### Algorithm

Execute in order. Maintain a `seen` set of file paths to enforce
uniqueness — silently skip any entry whose path is already in `seen`.

**Step 1 — Ancestor path**

Collect the ancestor path from `ROOT` down to `leafLogicalName`
by repeatedly calling `ParentLogicalName` to walk upward, then
reversing the collected names to get root-first order.

For each logical name in the ancestor path (including the leaf):
1. Call `PathFromLogicalName` to get the file path.
2. If path is already in `seen`, skip.
3. Add to chain and mark in `seen`.

**Step 2 — Internal references (ROOT/ depends_on)**

Read the leaf node's frontmatter using `ParseFrontmatter`.

For each entry in `DependsOn` whose `Path` starts with `ROOT/`:
1. Call `PathFromLogicalName` to get the file path.
2. If path is already in `seen`, skip.
3. Add to chain and mark in `seen`.

**Step 3 — External dependencies (EXTERNAL/ depends_on)**

For each entry in `DependsOn` whose `Path` starts with `EXTERNAL/`:
1. Call `PathFromLogicalName` to get the `_external.md` path.
2. If not already in `seen`, add to chain and mark in `seen`.
3. If the entry has a non-empty `Filter`, glob-match each pattern
   against files in the external dependency folder (the directory
   containing `_external.md`). For each matching file path (sorted,
   relative to project root): if not in `seen`, add to chain and
   mark in `seen`.

### Error handling

- If `PathFromLogicalName` returns false for any logical name →
  return error: `cannot resolve logical name: <name>`.
- If `ParseFrontmatter` fails for the leaf node → return error
  wrapping the underlying error.
- If a glob pattern fails to evaluate → return error:
  `error evaluating filter <pattern> for <EXTERNAL/name>: <err>`.
- `depends_on` entries with a `Path` that is neither `ROOT/` nor
  `EXTERNAL/` are silently ignored.
