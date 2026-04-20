---
version: 2
---

# ROOT

## Intent

Local MCP server for Code from Spec subagents. Runs as a stdio
process inside any Code from Spec project, exposes a set of tools
determined by the requested operation, and restricts the subagent
to exactly those tools.

## Context

Claude Code offers no native mechanism to restrict a subagent's
filesystem access or scope its actions to a specific task. This
server is the enforcement layer: the orchestrator launches it with
a specific operation and parameters, and the subagent can only do
what the server's tools allow.

Each operation defines its own tool set and its own argument schema.
The server selects the operation at startup and delegates everything
else to it.

## Contracts

### Invocation

```
subagent-mcp <operation> [args...]
subagent-mcp --help
```

If the first argument is `--help` or `-h`, the server prints a
usage message to stdout and exits 0. This takes precedence over
operation dispatch.

The server is launched by the orchestrator via MCP stdio config:

```json
{
  "mcpServers": {
    "subagent-mcp": {
      "type": "stdio",
      "command": "<path-to-subagent-mcp>",
      "args": ["<operation>", "<arg1>", "..."]
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

### Distribution

The binary may be placed inside the host project repository at a
path chosen by that project. No installation on the machine is
required.

### Concurrency

Multiple instances may run in parallel without conflict. Each is an
independent OS process with its own operation and session state.

## Constraints

- If the operation argument is absent or unrecognized, the server
  prints a usage message and exits 1.
- Each operation is responsible for its own argument validation and
  tool registration.

## Preconditions

This tool does not verify spec correctness or staleness. It assumes
the orchestrator has already run `staleness-check` and confirmed
that the target node and its dependencies are up to date before
invoking a codegen session. Generating code from a stale spec may
produce incorrect results — enforcing this precondition is the
orchestrator's responsibility.

## Decisions

### stdio transport

Each server instance is a child process of the orchestrator, tied
to a single subagent invocation. stdio requires zero port
configuration, has zero conflicts between parallel instances, and
couples the process lifetime automatically to the subagent's. The
isolation-per-process is a feature — server instances are not meant
to be shared.

### Positional CLI arguments, not environment variables

The operation and its parameters are positional CLI arguments. This
is idiomatic for CLI tools, makes the process visible in `ps` output
when debugging parallel runs, and makes local testing trivial
(`subagent-mcp codegen ROOT/x/y`).

### Operation as first argument

The first argument selects the operation. Each operation owns its
own argument schema and tool set. This allows the server to grow to
cover new subagent workflows without changing existing operations or
the dispatch mechanism.
