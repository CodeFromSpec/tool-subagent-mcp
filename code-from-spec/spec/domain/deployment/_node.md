---
version: 3
parent_version: 5
---

# ROOT/domain/deployment

## Intent

Describes how the orchestrator is expected to launch and manage
server instances. This is informational context — it records how
the tool is intended to be used but does not define contracts that
drive the implementation.

## Context

The server is launched by the orchestrator via MCP config:

```json
{
  "mcpServers": {
    "subagent-mcp": {
      "type": "stdio",
      "command": "<path-to-subagent-mcp>",
      "args": ["<mode>", "<arg1>", "..."]
    }
  }
}
```

The orchestrator writes that JSON to a temporary file and invokes
the subagent pointing to it. Managing this file — creation, path
choice, and cleanup — is the orchestrator's responsibility, not
the server's. A practical pattern:

```bash
# orchestrator generates a UUID to avoid collisions between
# parallel subagent invocations
CONFIG=/tmp/subagent-mcp-<uuid>.json
echo '{ ... }' > "$CONFIG"
claude --mcp-config "$CONFIG" "generate the code for this node"
rm "$CONFIG"
```

Using a per-invocation UUID in the filename ensures multiple
subagents running in parallel each get their own config file with
no risk of collision. A dedicated scratch directory (e.g. `.tmp/`
at the project root, gitignored) is a clean alternative to `/tmp`
for projects that prefer to keep temporary files local.
