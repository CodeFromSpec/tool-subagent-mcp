---
version: 20
parent_version: 8
---

# ROOT/tech_design/internal/modes

## Intent

Technical context for implementing the mode dispatch layer
and individual modes.

## Contracts

### Setup function convention

Each mode exposes a `Setup` function:

```go
func Setup(s *mcp.Server, args []string) error
```

`s` is the MCP server instance created by the server entry point.
`args` contains `os.Args[2:]` — the arguments after the mode
name. Each mode parses and validates its own args.

The server calls `Setup` after creating the MCP server and
selecting the mode. `Setup` is responsible for registering
tools on the server. It does not start or run the server —
that is the entry point's responsibility.

### Help message convention

Each mode also exposes a `HelpMessage` function:

```go
func HelpMessage() string
```

The server calls it and prints the result when the user runs
`subagent-mcp <mode> --help` (or `-h` or `help`).
The server handles help detection before calling `Setup`.

### State sharing

Tool handlers share state through package-level variables.
The variable is initialized to its zero value and populated
at runtime by the first tool call that establishes the state.

#### Decision: package-level variables

`StdioTransport` means a single client per process with no
concurrent access. Package-level variables are sufficient
and simple under these constraints.

