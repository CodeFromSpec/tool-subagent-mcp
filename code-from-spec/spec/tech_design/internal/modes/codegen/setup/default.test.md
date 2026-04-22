---
version: 7
parent_version: 13
implements:
  - internal/modes/codegen/setup_test.go
---

# TEST/tech_design/internal/modes/codegen/setup

## Context

Each test creates a `mcp.NewServer` instance and calls `Setup`.
For tests that check tool registration, use the MCP SDK's
in-memory transport to verify tools were registered correctly.

## Happy Path

### Registers load_context and write_file tools

Call `Setup` with `args = []string{}`.

Expect: no error. Server has `load_context` and `write_file`
tools registered with the correct names and descriptions as
defined in the parent codegen node.

### Nil args treated as empty

Call `Setup` with `args = nil`.

Expect: no error.

## Failure Cases

### Rejects unexpected arguments

Call `Setup` with `args = []string{"unexpected"}`.

Expect: error containing `"codegen mode does not accept arguments"`.
