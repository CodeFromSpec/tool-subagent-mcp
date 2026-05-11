---
name: check-meta-language
description: Read spec nodes and identify meta-language — content that references the spec tree structure itself rather than describing the system. These expressions confuse code generation subagents which receive a flat chain and have no concept of the tree.
---

# Check Meta Language

Read all spec nodes and identify content that references the Code
From Spec tree structure rather than describing the system being
built. A code generation subagent receives a linear chain — it
has no concept of nodes, parents, children, or tree position.
Meta-language that leaks tree structure into spec content produces
confusing or incorrect generated code.

## When invoked

Run this skill when the user asks to check for meta-language, or
invokes `/check-meta-language`.

## The core question

For every sentence in a spec node, ask:

> If a developer read this sentence with no knowledge that spec
> files exist or that they are organized in a tree, would the
> sentence still make sense?

If the answer is no, the sentence contains meta-language.

This applies regardless of language (English, Portuguese, or any
other).

## Categories of meta-language

These are conceptual categories, not grep patterns. The spec may
express these ideas in any phrasing or language.

### Self-reference to the spec artifact

The content refers to itself as a "node", "spec", or
"specification" — concepts that belong to the framework, not to
the system being described.

### References to tree structure

The content references hierarchical relationships between specs:
parents, children, siblings, ancestors, descendants, leaves,
intermediate nodes, subtrees, branches (of the spec tree).

### Redundant positional scoping

The content describes scope relative to the current position in
the tree ("below this node", "under this subtree", "from this
level down"). The public content of a node is only visible to
its descendants, so positional scoping is always redundant — the
reader is already "below". The content should state the rule
directly without qualifying where it applies in the tree.

### Spatial deixis

The content describes itself as a "place" where things "live",
"go", or "are defined" — as if the spec node were a physical
location. Public content is inherited as context, not as a
description of a location. Phrases like "lives here", "defined
here", "in this section" treat the spec as a container rather
than stating rules or facts directly.

The fix is to remove the spatial reference entirely and state
the attribute or rule as a direct declaration. If descendants
inherit "Non-exported service code lives here", they receive a
sentence about a place they cannot see. If they inherit
"Service code is non-exported", they receive a rule.

### Framework mechanics

The content references how the Code From Spec framework
propagates content: inheritance between nodes, public/private
sections (in the spec sense, not code visibility), version
tracking mechanics.

## What does NOT count

Content that uses similar words but describes the system, not the
spec tree:

- Database parent/child relationships
- OOP inheritance
- Git branches
- Domain uses of "node" (DOM nodes, tree data structures)
- Code visibility (public/private functions or fields)

Use judgment. The question is always whether the phrase describes
the system or the spec framework.

## Algorithm

1. Collect all `_node.md` files under `code-from-spec/`.

2. Read each file. For every sentence, apply the core question.
   Focus on the `# Public` section — that is what descendants
   and dependents receive, so meta-language there is most
   harmful. But check private sections too — they reach the
   code generation subagent for leaf nodes.

3. For each finding, determine whether it is true meta-language
   or a false positive (see "What does NOT count").

4. Report findings grouped by file. For each finding, show:
   - File path and line number
   - The problematic text
   - Why it is meta-language
   - A suggested rewrite that removes the meta-language while
     preserving the intended meaning

   If no findings, report that all specs are clean.

5. Do NOT edit any files automatically. Present findings and
   suggested rewrites for the user to review. Only apply changes
   if the user confirms.

## Examples

**Before:**
> Each child node documents one third-party or internal-platform
> dependency.

**After:**
> Each dependency documented here is a third-party or
> internal-platform service.

**Why:** "child node" is a tree concept. The reader does not know
what a child node is.

---

**Before:**
> When the schema changes in the migration repository, update this
> node and increment `version`.

**After:**
> When the schema changes in the migration repository, update this
> document and increment `version`.

**Why:** "this node" is a framework concept. "This document" says
the same thing without leaking the abstraction.

---

**Before:**
> Leaf nodes under this subtree generate files into
> `internal/<package>/`.

**After:**
> Generated files go into `internal/<package>/`.

**Why:** "Leaf nodes under this subtree" is redundant — the
content is only visible to descendants, so "under this subtree"
adds nothing, and "leaf nodes" is a framework concept. The rule
is simply where files go.

---

**Before:**
> All non-exported service code lives here.

**After:**
> All non-exported service code.

**Why:** "lives here" is a spatial reference to the spec node.
Remove it — what remains is the actual content. Descendants
inherit the attribute directly; they do not need to know where
it was declared.

---

**Before:**
> O mapeamento de status é definido no nó pai.

**After:**
> O mapeamento de status é definido em
> ROOT/architecture/backend/internal/api.

**Why:** "nó pai" is a tree concept. If a specific node is meant,
use its logical name so the chain resolver includes it. If the
content is already inherited, just state the rule directly.
