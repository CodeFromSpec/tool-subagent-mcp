---
version: 29
parent_version: 13
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/tech_design/internal/tools
    version: 2
implements:
  - cmd/subagent-mcp/main.go
---

# ROOT/tech_design/server

## Intent

Entry point: handles argument validation, creates and configures
the MCP server, registers tools, and runs the server.

## Context

### Package

`package main`

## Contracts

### Startup sequence

1. If `len(os.Args) > 1` and `os.Args[1]` is `--help`, `-h`, or
   `help`, print the usage message to stdout and exit 0.
2. If `len(os.Args) > 1` (any other argument), print the usage
   message to stderr and exit 1.
3. Create the MCP server via `mcp.NewServer` with
   `Implementation.Name` = `"subagent-mcp"`.
4. Register `load_chain` and `write_file` tools on the server
   using `mcp.AddTool`.
5. Call `s.Run(context.Background(), &mcp.StdioTransport{})`.
6. If `Run` returns an error, print it to stderr and exit 1.
7. Otherwise exit 0.

### Usage message

```
Usage: subagent-mcp

Starts an MCP server over stdin/stdout for Code from Spec
subagents.

Tools:
  load_chain     Load the spec chain for a node.
  write_file     Write a generated file to disk.

The subagent should have no other tools available — no file
read, write, or search capabilities beyond what this server
provides. This confinement ensures the subagent works only
from the provided context and writes only to declared outputs.

MCP configuration example:
  {
    "mcpServers": {
      "subagent-mcp": {
        "type": "stdio",
        "command": "<path-to-binary>"
      }
    }
  }
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Clean shutdown. |
| 1 | Startup error or server error. |
