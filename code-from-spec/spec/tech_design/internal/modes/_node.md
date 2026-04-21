---
version: 12
parent_version: 6
---

# ROOT/tech_design/internal/modes

## Intent

Technical context for implementing the mode dispatch layer
and individual modes.

## Contracts

### Mode interface

Each mode implements the following interface:

```go
type Mode interface {
    Setup(s *mcp.Server, args []string) error
}
```

`s` is the MCP server instance created by the server entry point.
`args` contains `os.Args[2:]` — the arguments after the mode
name. Each mode parses and validates its own args.

The server calls `Setup` after creating the MCP server and
selecting the mode. `Setup` is responsible for registering
tools on the server. It does not start or run the server —
that is the entry point's responsibility.

### State sharing via closures

`Setup` builds any state that tool handlers need, then
registers the handlers as closures that capture that state.
This avoids global variables and explicit struct methods —
the handler function closes over the variables it needs.

