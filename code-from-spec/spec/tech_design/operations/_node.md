---
version: 2
parent_version: 2
---

# ROOT/tech_design/operations

## Intent

Technical context for implementing the operation dispatch layer
and individual operations.

## Contracts

### Operation interface

Each operation implements the following interface:

```go
type Operation interface {
    Run(args []string) error
}
```

`args` contains `os.Args[2:]` — the arguments after the operation
name. Each operation parses and validates its own args.

The server calls `Run` after selecting the operation. `Run` is
responsible for registering tools, starting the MCP server, and
blocking until the client disconnects.

## Constraints

- Each operation is implemented in its own subdirectory under
  `cmd/subagent-mcp/operations/`.
- Operations do not share state. All state is initialized within
  `Run` and passed explicitly to tool handlers.
