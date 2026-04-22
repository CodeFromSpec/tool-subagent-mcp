---
version: 9
---

# ROOT

## Intent

Local MCP server for Code from Spec subagents. Exposes tools
that subagents use to interact with the spec tree and generate
code.

## Context

Given unrestricted file access, a subagent will compensate for
perceived gaps in its context by exploring the repository — reading
generated source files, unrelated specs, or framework documentation
— rather than stopping to report ambiguity. This produces
hallucinated or inconsistent output and makes the process
unpredictable. This server exists to mediate the subagent's
access: it controls what the subagent can read and where it can
write, so that the correct workflow is the only possible workflow.

## Contracts

### Invocation

```
subagent-mcp
```

Any argument causes the tool to print a usage message and exit.
`--help`, `-h`, and `help` exit 0; any other argument exits 1.

### Distribution

The binary may be placed inside the host project repository at a
path chosen by that project. No installation on the machine is
required.

### Deployment

The server is registered once in the project's Claude Code
configuration (`.claude/settings.json`):

```json
{
  "mcpServers": {
    "subagent-mcp": {
      "type": "stdio",
      "command": "<path-to-subagent-mcp>"
    }
  }
}
```

Once configured, the server is available to all sessions and
subagents in that project. No per-invocation setup or teardown
is needed.

### Concurrency

Multiple instances may run in parallel without conflict. Each is an
independent OS process with its own state.

### Preconditions

The orchestrator must run `staleness-check` and confirm that the
target node and its dependencies are up to date before invoking
a subagent. Operating on a stale spec may produce incorrect
results — enforcing this is the orchestrator's responsibility.

## Decisions

### Confinement is the caller's responsibility

The server exposes every tool it has to every connection. If a
subagent should only use a subset, the orchestrator must enforce
that by configuring the subagent itself — not by asking the server
to hide tools. This keeps the server simple and the tool surface
predictable.

### Minimal tool surface

Purpose-built tools combined with caller-side restriction of
which tools a subagent can call constrain the agent's action
space, making correct behavior more likely by construction.
