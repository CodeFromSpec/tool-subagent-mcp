---
version: 15
parent_version: 9
---

# ROOT/tech_design

Technical design decisions for implementing the MCP server in Go.

# Public

## Context

### Go module

The module path is `github.com/CodeFromSpec/tool-subagent-mcp/v2`.
All internal package imports must use this prefix.

## Contracts

### Language

Go (minimum 1.24).

### Dependencies

- Standard library unless explicitly stated otherwise.
- `github.com/modelcontextprotocol/go-sdk` — Official MCP SDK
  (stdio transport, tool registration with generics, request
  handling).

### Error handling

- **Startup errors** (unexpected arguments) — print to stderr and
  exit 1. The tool does not start if it cannot be configured.
- **Tool errors** — returned as MCP tool error responses. The tool
  continues running after a tool error.

### Project root

The tool is always executed from the project root directory.
The working directory of the process is the project root.
All relative paths — spec files, generated source files — are
resolved against it.

# Decisions

### Subagent guidance via MCP protocol

Use the MCP protocol's built-in tool descriptions to give the
subagent relevant context about how to use the server. This
complements the prompt the subagent receives from the orchestrator
and reduces the amount of guidance the orchestrator's prompt needs
to carry.

## Constraints

- Every error return value must be checked.
- No test framework beyond the standard `testing` package.
- No configuration files.
- All test helper functions and types must be prefixed with `test`
  (e.g., `testMakeFM`, `testIntPtr`, `testCase`). This prevents
  name collisions with unexported functions and types in the
  package under test when using internal test files (same package
  as the implementation).
