---
version: 1
parent_version: 1
---

# ROOT/domain/operations/codegen/chain

## Intent

Defines what the chain is: the ordered set of files that together
form the full context for generating the session's target node
(a leaf spec node or a test node).

## Context

The Code from Spec methodology requires that a code generation agent
receive the complete specification chain — ancestor nodes that
establish constraints and conventions, plus any cross-tree references
the target node declares. The chain is deterministic given a target
node; it is not a user decision.

## Contracts

### Chain for a leaf spec node

The chain for a leaf node `L` (a `ROOT/` logical name) consists of
four ordered parts:

1. **Ancestor path** — the `_node.md` files on the path from `ROOT`
   down to `L`, inclusive. If `L` is `ROOT/a/b/c`, the ancestor
   path is: `ROOT`, `ROOT/a`, `ROOT/a/b`, `ROOT/a/b/c`.

2. **Internal references** — `_node.md` files in `L`'s `depends_on`
   whose path begins with `ROOT/`. Included in `depends_on`
   declaration order.

3. **External dependencies** — entries in `L`'s `depends_on` whose
   path begins with `EXTERNAL/`. For each: always include
   `_external.md`; if a `filter` is present, also include matching
   files within the external dependency folder, in filesystem order.
   Included in `depends_on` declaration order.

### Chain for a test node

The chain for a test node `T` (a `TEST/` logical name) extends the
chain of its parent leaf node:

1. **Ancestor path of the parent** — `_node.md` files from `ROOT`
   down to `T`'s parent leaf node, inclusive.

2. **Parent's internal references** — `ROOT/` entries in the parent
   leaf node's `depends_on`, in declaration order.

3. **Parent's external dependencies** — `EXTERNAL/` entries in the
   parent leaf node's `depends_on`, with filters applied, in
   declaration order.

4. **The test node itself** — the `.test.md` file for `T`.

5. **Test node's own dependencies** — `depends_on` entries from the
   test node (both `ROOT/` and `EXTERNAL/`), in declaration order,
   following the same rules as parts 2 and 3 above. These cover only
   what the test node needs beyond the parent's context.

The parent leaf node's chain is always included implicitly — the
test node depends fundamentally on its test subject.

### Uniqueness

Each file path appears at most once in the chain. If a file would
appear in multiple parts, the first occurrence wins and the duplicate
is silently skipped.

## Constraints

- The chain is read-only. No chain file may be modified.
- If the target node's frontmatter cannot be read, the chain cannot
  be built; this is an operational error.
- If the target node is a test node and its parent's frontmatter
  cannot be read, the chain cannot be built; this is an operational
  error.
