---
version: 1
parent_version: 1
---

# ROOT/domain/operations/codegen/chain

## Intent

Defines what the chain is: the ordered set of files that together
form the full context for generating the session's leaf node.

## Context

The Code from Spec methodology requires that a code generation agent
receive the complete specification chain — ancestor nodes that
establish constraints and conventions, plus any cross-tree references
the leaf node declares. The chain is deterministic given a leaf node;
it is not a user decision.

## Contracts

### Composition

The chain for a leaf node `L` consists of three ordered parts:

1. **Ancestor path** — the `_node.md` files on the path from `ROOT`
   down to `L`, inclusive. If `L` is `ROOT/a/b/c`, the ancestor
   path is: `ROOT`, `ROOT/a`, `ROOT/a/b`, `ROOT/a/b/c` (four files,
   in this order).

2. **Internal references** — `_node.md` files referenced in `L`'s
   `depends_on` whose path begins with `ROOT/`. Included in
   `depends_on` declaration order. These are cross-tree specs that
   `L` explicitly depends on but that are not ancestors of `L`.

3. **External dependencies** — entries in `L`'s `depends_on` whose
   path begins with `EXTERNAL/`. For each:
   - Always include the `_external.md` entry point.
   - If the entry has a `filter` field, also include all files
     within the external dependency folder that match any of the
     glob patterns, in filesystem order.
   Included in `depends_on` declaration order.

### Source of depends_on

Only the leaf node's own `depends_on` is used to build parts 2 and
3. Ancestor nodes' `depends_on` entries are not included.

### Uniqueness

Each file path appears at most once in the chain. If a file would
appear in multiple parts (e.g., an ancestor is also listed in
`depends_on`), the first occurrence wins and the duplicate is
silently skipped.

## Constraints

- The chain is read-only. No chain file may be modified.
- If the leaf node has no `depends_on`, parts 2 and 3 are empty.
- If the leaf node's frontmatter cannot be read, the chain cannot
  be built; this is an operational error.
