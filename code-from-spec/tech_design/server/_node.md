---
version: 56
parent_version: 17
depends_on:
  - path: ROOT/external/mcp-go-sdk
    version: 3
  - path: ROOT/tech_design/internal/tools
    version: 5
  - path: ROOT/tech_design/internal/tools/load_chain
    version: 62
  - path: ROOT/tech_design/internal/tools/patch_file
    version: 7
  - path: ROOT/tech_design/internal/tools/write_file
    version: 38
implements:
  - cmd/subagent-mcp/main.go
---

# ROOT/tech_design/server

Entry point: handles argument validation, creates and configures
the MCP server, registers tools, and runs the server.

# Public

## Package

`package main`

## Startup sequence

1. If `len(os.Args) > 1` and `os.Args[1]` is `--help`, `-h`, or
   `help`, print the usage message to stdout and exit 0.
2. If `len(os.Args) > 1` (any other argument), print the usage
   message to stderr and exit 1.
3. Create the MCP server via `mcp.NewServer` with
   `Implementation.Name` = `"subagent-mcp"`.
4. Register tools using `mcp.AddTool`. For each tool, construct
   the `mcp.Tool` inline with the name and description from the
   corresponding tool definition spec, and pass the exported
   handler from the package:
   - `load_chain.HandleLoadChain` with `LoadChainArgs`.
     Set `Meta: mcp.Meta{"anthropic/maxResultSizeChars": 500000}`
     on the tool so that `tools/list` advertises the maximum
     result size to the client.
   - `write_file.HandleWriteFile` with `WriteFileArgs`
   - `patch_file.HandlePatchFile` with `PatchFileArgs`
5. Call `s.Run(context.Background(), &mcp.StdioTransport{})`.
6. If `Run` returns an error, print it to stderr and exit 1.
7. Otherwise exit 0.

## Usage message

```
Usage: subagent-mcp

Starts an MCP server over stdin/stdout for Code from Spec
subagents.

Tools:
  load_chain     Load the spec chain for a node.
  write_file     Write a generated file to disk.
  patch_file     Apply a unified diff to an existing file.

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

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Clean shutdown. |
| 1 | Startup error or server error. |
