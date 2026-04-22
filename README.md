# tool-subagent-mcp

MCP server for [Code from Spec](https://github.com/CodeFromSpec/framework)
subagents. Provides confined tools for code generation — the
subagent can only read the spec chain and write declared output
files.

## Tools

- **load_chain** — returns the complete spec chain for a given
  logical name
- **write_file** — writes a generated file to disk, validated
  against the node's `implements` list

## Install

Download the latest release for your platform from
[Releases](https://github.com/CodeFromSpec/tool-subagent-mcp/releases)
and extract the binary into your project's `tools/` directory.

Or build from source:

```bash
go build -o tools/subagent-mcp ./cmd/subagent-mcp
```

## Configure

Register the server in `.claude/settings.json`:

```json
{
  "mcpServers": {
    "subagent-mcp": {
      "type": "stdio",
      "command": "tools/subagent-mcp"
    }
  }
}
```

On Windows, use `tools/subagent-mcp.exe` as the command.

## Usage

The server takes no arguments. Run `subagent-mcp --help` for
usage information.

```
Usage: subagent-mcp

Starts an MCP server over stdin/stdout for Code from Spec
subagents.
```

## Documentation

- [Code from Spec framework](https://github.com/CodeFromSpec/framework)
- [Getting Started](https://github.com/CodeFromSpec/framework/blob/main/docs/GETTING_STARTED.md)
- [Code Generation with Subagents](https://github.com/CodeFromSpec/framework/blob/main/rules/CODE_GENERATION.md)
