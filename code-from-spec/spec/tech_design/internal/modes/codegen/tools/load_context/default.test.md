---
version: 8
parent_version: 29
implements:
  - internal/modes/codegen/load_context_test.go
---

# TEST/tech_design/internal/modes/codegen/tools/load_context

## Context

Each test uses `t.TempDir()` to create an isolated project
structure with the necessary spec files. The working directory
is changed to the temp dir for the duration of the test so that
path validation resolves correctly.

## Happy Path

### Valid ROOT/ leaf node

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements` and no dependencies). Call `handleLoadContext`
with `LogicalName: "ROOT/a"`.

Expect: success result. Text contains the chain content
with files from `ROOT` and `ROOT/a`.

### Valid TEST/ node

Create a spec tree: `ROOT`, `ROOT/a` (leaf), and `TEST/a`.
Call `handleLoadContext` with `LogicalName: "TEST/a"`.

Expect: success result.

### Node with dependencies

Create a spec tree: `ROOT`, `ROOT/a` (leaf with `depends_on`
referencing `EXTERNAL/db`). Create the external dependency
with `_external.md` and a data file. Call `handleLoadContext`
with `LogicalName: "ROOT/a"`.

Expect: success result. Chain content contains all files
from the chain including the external dependency files.

### Chain content uses heredoc format

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements`). Call `handleLoadContext` with
`LogicalName: "ROOT/a"`.

Expect: success result. Text contains `<<<FILE_` and
`<<<END_FILE_` delimiters with `node:` and `path:` headers.

### Repeated calls succeed

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements`). Call `handleLoadContext` twice with the
same `LogicalName`.

Expect: both calls return success with non-empty chain
content. Content may differ between calls because a new
UUID is generated each time.

## Failure Cases

### Invalid prefix

Call `handleLoadContext` with
`LogicalName: "EXTERNAL/something"`.

Expect: tool error containing `"codegen target must be a
ROOT/ or TEST/"`.

### Nonexistent spec file

Call `handleLoadContext` with
`LogicalName: "ROOT/nonexistent"`. Do not create the
corresponding spec file.

Expect: tool error (from `ParseFrontmatter` — file not found).

### No implements

Create a spec tree: `ROOT` and `ROOT/a` (leaf without
`implements`). Call `handleLoadContext` with
`LogicalName: "ROOT/a"`.

Expect: tool error containing `"has no implements"`.

### Invalid implements path — traversal

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`implements: ["../../etc/passwd"]`). Call `handleLoadContext`
with `LogicalName: "ROOT/a"`.

Expect: tool error from path validation.

### Unresolvable dependency

Create a spec tree: `ROOT` and `ROOT/a` (leaf with
`depends_on` referencing `ROOT/b`). Do not create `ROOT/b`'s
file. Call `handleLoadContext` with `LogicalName: "ROOT/a"`.

Expect: tool error from chain resolution.
