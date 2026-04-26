---
version: 14
parent_version: 54
implements:
  - internal/load_chain/load_chain_test.go
---

# TEST/tech_design/internal/tools/load_chain

## Context

Each test uses `t.TempDir()` to create an isolated project
structure with the necessary spec files. The working directory
is changed to the temp dir for the duration of the test so that
path validation resolves correctly.

Spec files in tests must have valid CommonMark body structure:
frontmatter followed by `# <logical name>` heading, optionally
`# Public` with `##` subsections, and private sections.

## Happy Path

### Valid ROOT/ leaf node

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements` and no dependencies). Both have `# Public`
sections. Call `handleLoadChain` with
`LogicalName: "ROOT/a"`.

Expect: success result. Chain content contains:
- `ROOT` with only its `# Public` section content
- `ROOT/a` with reduced frontmatter and full body

### Valid TEST/ node

Create a spec tree: `ROOT`, `ROOT/a` (leaf with `# Public`),
and `TEST/a`. Call `handleLoadChain` with
`LogicalName: "TEST/a"`.

Expect: success result. `ROOT` and `ROOT/a` contain only
their `# Public` sections. `TEST/a` contains reduced
frontmatter and full body.

### Node with ROOT/ dependency, no qualifier

Create a spec tree: `ROOT`, `ROOT/a` (leaf with `depends_on`
referencing `ROOT/b`), `ROOT/b` (with `# Public` containing
`## Interface` and `## Constraints`). Call `handleLoadChain`
with `LogicalName: "ROOT/a"`.

Expect: success result. The dependency `ROOT/b` section
contains only its `# Public` content (both subsections).

### Node with ROOT/ dependency, with qualifier

Create a spec tree: `ROOT`, `ROOT/a` (leaf with `depends_on`
referencing `ROOT/b(interface)`), `ROOT/b` (with `# Public`
containing `## Interface` and `## Constraints`). Call
`handleLoadChain` with `LogicalName: "ROOT/a"`.

Expect: success result. The dependency `ROOT/b` section
contains only the `## Interface` subsection content, not
`## Constraints`.

### Chain content uses heredoc format

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements`). Call `handleLoadChain` with
`LogicalName: "ROOT/a"`.

Expect: success result. Text contains `<<<FILE_` and
`<<<END_FILE_` delimiters with `node:` and `path:` headers.

### Ancestors contain only # Public

Create a spec tree: `ROOT` (with `# Public` and private
sections), `ROOT/a`, `ROOT/a/b` (leaf with `implements`).
Call `handleLoadChain` with `LogicalName: "ROOT/a/b"`.

Expect: the sections for `ROOT` and `ROOT/a` contain only
their `# Public` content. Private sections and node name
sections are not present.

### Target has reduced frontmatter

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`version: 5`, `parent_version: 1`,
`depends_on: [{path: ROOT/b, version: 2}]`,
`implements: ["src/a.go"]`). Call `handleLoadChain` with
`LogicalName: "ROOT/a"`.

Expect: the target section contains frontmatter with only
`version: 5` and `implements: ["src/a.go"]`. The fields
`parent_version` and `depends_on` are not present.

### Existing code files included in output

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements: ["src/a.go"]`). Create `src/a.go` with known
content. Call `handleLoadChain` with `LogicalName: "ROOT/a"`.

Expect: success result. Chain content contains a file section
for `src/a.go` with `path:` header and no `node:` header.
The file content matches what was written to disk.

### Non-existing code files omitted from output

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements: ["src/a.go"]`). Do not create `src/a.go`.
Call `handleLoadChain` with `LogicalName: "ROOT/a"`.

Expect: success result. Chain content does not contain a
file section for `src/a.go`.

### Ancestor with no # Public section

Create a spec tree: `ROOT` (with no `# Public` — only node
name and private sections) and `ROOT/a` (leaf with
`implements`). Call `handleLoadChain` with
`LogicalName: "ROOT/a"`.

Expect: success result. The section for `ROOT` is included
but with empty content.

## Failure Cases

### Invalid prefix

Call `handleLoadChain` with
`LogicalName: "INVALID/something"`.

Expect: tool error containing `"target must be a
ROOT/ or TEST/"`.

### Nonexistent spec file

Call `handleLoadChain` with
`LogicalName: "ROOT/nonexistent"`. Do not create the
corresponding spec file.

Expect: tool error (from `ParseFrontmatter` — file not found).

### No implements

Create a spec tree: `ROOT` and `ROOT/a` (leaf without
`implements`). Call `handleLoadChain` with
`LogicalName: "ROOT/a"`.

Expect: tool error containing `"has no implements"`.

### Invalid implements path — traversal

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements: ["../../etc/passwd"]`). Call `handleLoadChain`
with `LogicalName: "ROOT/a"`.

Expect: tool error from path validation.

### Unresolvable dependency

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`depends_on` referencing `ROOT/b`). Do not create `ROOT/b`'s
file. Call `handleLoadChain` with `LogicalName: "ROOT/a"`.

Expect: tool error from chain resolution.
