---
name: code-from-spec-staleness-resolution
description: Use this agent to resolve spec and test staleness in a Code from Spec project.
tools: "Read, Edit, Bash"
model: "claude-haiku-4-5-20251001"
effort: medium
---

## Your task

You are a staleness resolution agent for a Code from Spec project.
Your task is to resolve spec and test staleness.
Run the staleness-check tool via Bash: `./tools/staleness-check.exe` (Windows) or `./tools/staleness-check` (Linux/macOS).


# Code From Spec

**Code From Spec** is a methodology where code is a generated
artifact, not the source of truth. The source of truth is a hierarchy
of specification files. To change behavior, you change the spec and
regenerate. You never edit generated code directly.

This methodology is designed for AI agent participation at every
stage — writing specs, managing versions, detecting and resolving
staleness, generating code, and assisting non-technical contributors
with spec authoring.

## The Model

Specifications are organized as a tree. Each node adds precision
to its parent — high-level intent at the root, implementation
detail at the leaves. Only leaf nodes generate code.

```
root/
└── payments/
    └── fees/
        ├── calculation/   ← leaf, implemented
        └── rounding/      ← leaf, implemented
```

## File Format

Specification files use CommonMark for Markdown formatting and are
UTF-8 encoded, without BOM.

### YAML frontmatter

Each file begins with a YAML frontmatter block. Frontmatter is not
part of CommonMark — it is an extension adopted by this framework.

The frontmatter block starts with a line containing exactly `---`
(three hyphens, nothing else) as the first line of the file, and
ends with the next line containing exactly `---`. The content
between the two delimiters is parsed as YAML.

## Specifications

Specifications are the source of truth from which code is generated.

### Location

Specifications live under `<project root>/code-from-spec/`.

### Structure

Specifications are organized as a hierarchical tree of nodes. Child
nodes inherit the public content of all their ancestors — only what
is explicitly marked public propagates down the tree (see Body).
This inheritance is automatic and mandatory.

### Nodes

Every spec node is a directory containing a `_node.md` file. The
directory structure is the spec tree — a node's position in the
filesystem is its position in the hierarchy.

Each `_node.md` describes one aspect of the system at a specific
level of abstraction.

A node with child directories is an **intermediate node**. A node
without children is a **leaf node**. Intermediate nodes provide
context and constraints to their descendants. Only leaf nodes may
generate code. Not all leaf nodes do; some serve as documentation
only.

A **test node** is a file ending in `.test.md` placed inside the
directory of the node it tests (its **subject**). The canonical test
node is named `default.test.md`. Additional test nodes use
`<name>.test.md`. Any node may have test nodes.

Test nodes are not children of their subject — they have no parent
in the tree. However, they receive the same inherited context as
their subject: the public content of all ancestors of the subject
node. Since test nodes are not part of the tree hierarchy, they may
declare `depends_on` to children of their subject without creating
circular dependencies.

```
config/
  _node.md             ← spec node (leaf)
  default.test.md      ← canonical test node
  edge_cases.test.md   ← additional test node
```

### Logical names

Every node has a logical name derived from its position in the tree.
Spec nodes use the `ROOT/` prefix; test nodes use the `TEST/`
prefix. A `ROOT/` reference may include a parenthetical qualifier
to target a specific public subsection of the node (see Body).

| Logical name | Resolves to |
|---|---|
| `ROOT` | `code-from-spec/_node.md` |
| `ROOT/architecture/backend` | `code-from-spec/architecture/backend/_node.md` |
| `ROOT/architecture/backend/config` | `code-from-spec/architecture/backend/config/_node.md` |
| `ROOT/architecture/backend/config(interface)` | `## Interface` subsection of `# Public` in `code-from-spec/architecture/backend/config/_node.md` |
| `TEST/architecture/backend/config` | `code-from-spec/architecture/backend/config/default.test.md` |
| `TEST/architecture/backend/config(edge_cases)` | `code-from-spec/architecture/backend/config/edge_cases.test.md` |

Resolution rules:
- `ROOT/x` → `code-from-spec/x/_node.md` (`# Public`)
- `ROOT/x(y)` → `## y` subsection of `# Public` in `code-from-spec/x/_node.md`
- `TEST/x(y)` → `code-from-spec/x/y.test.md`
- `TEST/x` is an alias for `TEST/x(default)`

### Frontmatter

Every node begins with a YAML frontmatter block.

| Field | Description | Notes |
|---|---|---|
| `version` | See Versioning and Staleness. | All nodes |
| `parent_version` | The version of the parent node this node was written against. | Root node and test nodes have no parent |
| `subject_version` | The version of the node this test was written against — the `_node.md` in the same directory. | Test nodes only |
| `depends_on` | Cross-tree dependencies with their known versions. Uses logical names. | Optional |
| `implements` | Source files generated by this node. Filesystem paths relative to the project root. | Leaf and test nodes |

Frontmatter is metadata for the framework — it is not part of the
node's content and does not participate in inheritance or
`depends_on`.

Content imported via `depends_on` does not propagate to descendant
nodes. Each node must declare its own `depends_on` for the content
it needs.

A `depends_on` entry using `ROOT/x/y` imports the `# Public` section
of the referenced node. An entry using `ROOT/x/y(z)` imports only the `## z` subsection
of `# Public` of the referenced node — useful when a node needs
a specific part of the public context rather than all of it.

`depends_on` may only reference nodes in other branches of the tree.
Pointing to an ancestor would be redundant — its content is already
available via inheritance. Pointing to a descendant would create a
circular dependency.

Example — root node:

```yaml
---
version: 3
---
```

Example — intermediate node without dependencies:

```yaml
---
version: 2
parent_version: 3
---
```

Example — leaf node with dependencies:

```yaml
---
version: 1
parent_version: 1
depends_on:
  - path: ROOT/external/payments-api/create-transfer
    version: 5
  - path: ROOT/architecture/backend/api-gateway
    version: 6
implements:
  - internal/transfers/transfers.go
---
```

Example — test node:

```yaml
---
version: 1
subject_version: 2
implements:
  - internal/configuration/config_test.go
---
```

### Body

The body of a node is divided into top-level sections, each starting
with a `#` heading. A section ends when the next `#` heading begins
or the file ends. Two sections have special meaning: the **node name section** and
the **public section** (`# Public`). All other sections are treated
as private — not available via inheritance or `depends_on`.

#### Node name section

Must be the first section in the file, immediately after the
frontmatter — nothing may appear between the frontmatter and this
heading. The heading is the node's logical name (e.g.
`# ROOT/architecture/backend/config`). Its content serves as
intent — what this node does and why it exists. This section is
not available to other nodes.


---

## Versioning and Staleness

Every versioned file has a `version` field in its YAML frontmatter.
Version numbers are integers.

### Which files are versioned

| File | Location |
|---|---|
| Spec node | `code-from-spec/**/_node.md` |
| Test node | `code-from-spec/**/*.test.md` |

### When to increment the version field

The `version` field must be incremented on every change to the
file — no exceptions. A single added space, a corrected typo, a
reformatted line, a bumped dependency version in the frontmatter —
all require a version increment. The rule is mechanical: if
computing a hash of the file before and after the change would
produce different results, the version must change. Semantic
significance is irrelevant. Never decide that a change is "too
small" to warrant a version increment.

### How to increment

Add 1 to the current value. Version 3 becomes 4, not 5 or 10.

### What is staleness

A file is stale when it references a version that is no longer
current — meaning something it depends on has changed since it was
last updated. Staleness is never declared — it is always
calculated by comparing declared versions against current versions.

### Which files can become stale

| File | Stale when |
|---|---|
| Spec node (`_node.md`) | Parent or dependency version changed |
| Test node (`*.test.md`) | Subject or dependency version changed |
| Generated source file | The node that implements it has changed version since last generation |

### How to determine if a file is stale

A node is stale when:

```
parent.version != node.parent_version
depends_on[x].current_version != node.depends_on[x].version
```

For test nodes, replace `parent_version` with `subject_version`.

A generated source file is stale when:

```
node.version != version in the file's spec comment
```

Staleness verification is automated by the `staleness-check` tool.
The tool reports stale items in a fixed order: spec nodes first
(top-down), then test nodes, then generated source files.

### Staleness Resolution

Resolving staleness means reviewing each stale node in light of
how the parent or dependency that triggered the staleness changed,
and determining whether the node's own content needs to be updated.
The version bump is the consequence of that review, not the act
itself. Skipping the content review defeats the purpose of versioning.

The resolution process is iterative: call `staleness-check`, address
the first item it reports, call the tool again, repeat. Because the
tool reports top-down, resolving a parent before its children avoids
cascading rework. If a resolution introduces ambiguity or requires
human judgment, stop and consult the user.

Spec node staleness must be resolved before test node staleness.
Both must be clean before generating code (see Code Generation) —
generating from stale specs is wasteful, as the output will be
stale before it is written.

## Path Separator

All paths in the framework use forward slash (`/`) as the
separator, regardless of the operating system. This applies to
logical names, `implements` entries, and file paths in the chain.
Backslash (`\`) is never used as a separator. Tools that interact
with the OS filesystem must normalize paths to forward slashes
before returning or comparing them.

