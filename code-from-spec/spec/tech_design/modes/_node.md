---
version: 5
parent_version: 4
---

# ROOT/tech_design/modes

## Intent

Technical context for implementing the mode dispatch layer
and individual modes.

## Contracts

### Mode interface

Each mode implements the following interface:

```go
type Mode interface {
    Run(args []string) error
}
```

`args` contains `os.Args[2:]` — the arguments after the mode
name. Each mode parses and validates its own args.

The server calls `Run` after selecting the mode. `Run` is
responsible for registering tools, starting the MCP server, and
blocking until the client disconnects.

## Constraints

- Each mode is implemented in its own subdirectory under
  `cmd/subagent-mcp/modes/`.
- Modes do not share state. All state is initialized within
  `Run` and passed explicitly to tool handlers.
