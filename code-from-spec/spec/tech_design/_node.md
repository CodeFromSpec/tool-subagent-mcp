---
version: 3
parent_version: 3
---

# ROOT/tech_design

## Intent

Technical design decisions for implementing the MCP server in Go.

## Context

This is a single-purpose, long-running stdio process. It serves MCP
tool calls from one subagent until that subagent exits. The design
prioritizes simplicity and correctness over extensibility.

## Contracts

### Language

Go (minimum 1.24).

### Dependencies

- `github.com/mark3labs/mcp-go` — MCP server implementation
  (stdio transport, tool registration, request handling).
- `gopkg.in/yaml.v3` — YAML frontmatter parsing.
- Standard library for everything else.

### File organization

```
cmd/subagent-mcp/
  main.go             ← startup and mode dispatch
  logicalnames.go     ← logical name ↔ file path conversions
  frontmatter.go      ← YAML frontmatter parsing
  modes/
    codegen/
      codegen.go      ← Run(), Session type, argument parsing
      chainresolver.go
      load_context.go
      write_file.go
```

### Error handling

- **Startup errors** (missing or invalid mode argument, unresolvable
  target node, unreadable leaf frontmatter) — print to stderr and
  exit 1. The tool does not start if the session cannot be
  established.
- **Tool errors** — returned as MCP tool error responses. The tool
  continues running after a tool error.
- The tool never panics. All errors are handled explicitly.

## Constraints

- No global mutable state. Session configuration is initialized once
  at startup and passed explicitly to tool handlers.
- No concurrency beyond what the MCP library manages internally.
  Tool handlers execute serially.
- No configuration files. All behavior is determined by CLI arguments
  and the filesystem.
