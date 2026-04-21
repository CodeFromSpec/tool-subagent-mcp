---
version: 2
parent_version: 8
implements:
  - internal/modes/codegen/setup_test.go
---

# TEST/tech_design/internal/modes/codegen/setup

## Context

Each test creates a minimal spec tree under `t.TempDir()` with
the necessary `_node.md`, `_external.md`, and `*.test.md` files.
`Setup` is called with a `mcp.NewServer` instance and the test
verifies the result.

For tests that check tool registration, use the MCP SDK's
in-memory transport to verify tools were registered correctly.

## Happy Path

### Valid ROOT/ leaf node

Create a spec tree with `ROOT` and `ROOT/a` (leaf with
`implements` and no dependencies). Call `Setup` with
`args = ["ROOT/a"]`.

Expect: no error. Server has `load_context` and `write_file`
tools registered.

### Valid TEST/ node

Create a spec tree with `ROOT`, `ROOT/a` (leaf), and `TEST/a`.
Call `Setup` with `args = ["TEST/a"]`.

Expect: no error.

### Node with dependencies

Create a spec tree with `ROOT`, `ROOT/a` (leaf with
`depends_on` referencing `EXTERNAL/db`). Create the external
dependency with `_external.md` and a data file. Call `Setup`
with `args = ["ROOT/a"]`.

Expect: no error. Chain content contains all files from the
chain including the external dependency files.

## Failure Cases

### No arguments

Call `Setup` with `args = []`.

Expect: error containing `"usage:"`.

### Too many arguments

Call `Setup` with `args = ["ROOT/a", "extra"]`.

Expect: error containing `"usage:"`.

### Invalid prefix

Call `Setup` with `args = ["EXTERNAL/something"]`.

Expect: error containing `"ROOT/ or TEST/"`.

### Nonexistent spec file

Call `Setup` with `args = ["ROOT/nonexistent"]`.
Do not create the corresponding spec file.

Expect: error from `ParseFrontmatter` (file not found).

### No implements

Create a spec tree with `ROOT` and `ROOT/a` (leaf without
`implements`). Call `Setup` with `args = ["ROOT/a"]`.

Expect: error containing `"has no implements"`.

### Invalid implements path — traversal

Create a spec tree with `ROOT` and `ROOT/a` (leaf with
`implements: ["../../etc/passwd"]`). Call `Setup` with
`args = ["ROOT/a"]`.

Expect: error from path validation.

### Unreadable chain file

Create a spec tree with `ROOT` and `ROOT/a` (leaf with
`depends_on` referencing `ROOT/b`). Do not create `ROOT/b`'s
file. Call `Setup` with `args = ["ROOT/a"]`.

Expect: error from chain resolution.
