---
version: 9
parent_version: 7
---

# ROOT/tech_design

## Intent

Technical design decisions for implementing the MCP server in Go.

## Contracts

### Language

Go (minimum 1.24).

### Dependencies

- `github.com/modelcontextprotocol/go-sdk` — Official MCP SDK
  (stdio transport, tool registration with generics, request
  handling).
- Standard library for everything else.

### Error handling

- **Startup errors** (missing or invalid mode argument, unresolvable
  target node, unreadable frontmatter) — print to stderr and
  exit 1. The tool does not start if it cannot be configured.
- **Tool errors** — returned as MCP tool error responses. The tool
  continues running after a tool error.

## Constraints

- Every error return value must be checked.
- No test framework beyond the standard `testing` package.
- No configuration files. All behavior is determined by CLI
  arguments.
