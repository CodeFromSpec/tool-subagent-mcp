---
version: 21
parent_version: 10
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes
    version: 12
  - path: ROOT/tech_design/internal/modes
    version: 13
  - path: ROOT/tech_design/internal/modes/codegen
    version: 30
implements:
  - cmd/subagent-mcp/main.go
---

# ROOT/tech_design/server

## Intent

Entry point: reads the mode argument, dispatches to the
corresponding mode handler, and exits with the appropriate
code.

## Context

### Package

`package main`

## Contracts

### Startup sequence

1. If `len(os.Args) < 2` or `os.Args[1]` is empty, print a
   usage message to stderr and exit 1.
2. If `os.Args[1]` is `--help`, `-h`, or `help`, print the usage
   message to stdout and exit 0.
3. Match the mode name:
   - `"codegen"`:
     a. If `os.Args[2]` is `--help`, `-h`, or `help`, print
        `codegen.HelpMessage()` to stdout and exit 0.
     b. Create the MCP server via `mcp.NewServer` with
        `Implementation.Name` = `"subagent-mcp"` and
        `ServerOptions.Instructions` = `codegen.Instructions`.
     c. Call `codegen.Setup(s, os.Args[2:])`.
   - Unrecognized → print a usage message listing valid modes
     to stderr and exit 1.
5. If `Setup` returns an error, print it to stderr and exit 1.
6. Call `s.Run(context.Background(), &mcp.StdioTransport{})`.
7. If `Run` returns an error, print it to stderr and exit 1.
8. Otherwise exit 0.

### Usage message

```
Usage: subagent-mcp <mode> [args...]

Modes:
  codegen <logical-name>   Generate code for a spec or test node.

Run subagent-mcp <mode> --help for mode-specific help.
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Clean shutdown. |
| 1 | Startup error or mode error. |
