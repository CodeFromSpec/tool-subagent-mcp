---
version: 6
---

# ROOT

## Intent

Local MCP server for Code from Spec subagents. Runs as a process
inside any Code from Spec project, exposes a set of tools determined
by the requested mode, and restricts the subagent to exactly those
tools.

## Context

AI agent frameworks typically offer no native mechanism to restrict
a subagent's filesystem access or scope its actions to a specific
task. This server is the enforcement layer: the orchestrator
launches it with a specific mode and parameters, and the subagent
can only do what the server's tools allow.

## Contracts

### Invocation

```
subagent-mcp <mode> [args...]
```

### Distribution

The binary may be placed inside the host project repository at a
path chosen by that project. No installation on the machine is
required.

### Concurrency

Multiple instances may run in parallel without conflict. Each is an
independent OS process with its own mode and state.

## Constraints

- If the mode argument is absent or unrecognized, the server
  prints a usage message and exits 1.
- Each mode is responsible for its own argument validation and
  tool registration.

## Preconditions

This tool does not verify spec correctness or staleness. It assumes
the orchestrator has already run `staleness-check` and confirmed
that the target node and its dependencies are up to date before
invoking the subagent. Generating code from a stale spec may
produce incorrect results — enforcing this precondition is the
orchestrator's responsibility.

## Decisions

### Extensible by design

Although the tool was created for code generation, it is natural
to extend it to support other subagent workflows in the future.
The mode-based architecture allows adding new capabilities without
changing existing ones.
