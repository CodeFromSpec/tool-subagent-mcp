---
version: 1
parent_version: 1
---

# ROOT/domain/operations/codegen/tools

## Intent

Defines the two tools exposed by the codegen operation and their
shared behavioral constraints.

## Context

The subagent operates with only these two tools available — no
native file access from Claude Code. This is intentional: the
tool's surface area defines the agent's action space, and the
minimal surface ensures correct behavior by construction.

## Constraints

- Both tools operate within the session's scope. They may not
  access files outside the chain (for reads) or outside `implements`
  (for writes).
- Tool errors must be reported as MCP tool errors, not as process
  crashes. The tool continues serving after a tool error.
- Tool responses must be deterministic given the same inputs and
  filesystem state.
