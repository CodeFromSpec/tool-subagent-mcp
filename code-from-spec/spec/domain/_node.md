---
version: 1
parent_version: 1
---

# ROOT/domain

## Intent

Defines the domain concepts this server depends on: the Code from
Spec project structure it reads, and the runtime concepts (session,
operation) that govern its behavior.

## Context

### Code from Spec structure

A Code from Spec project organizes specifications as a tree under
`code-from-spec/spec/`. Each node is a directory with a `_node.md`
file. External dependencies live under `code-from-spec/external/`,
each with an `_external.md` entry point. Only leaf nodes generate
code (they carry an `implements` field in their frontmatter).

### Role of this server

The server mediates between the orchestrator and a subagent. The
orchestrator decides which operation to run and with which
parameters; it launches the server accordingly. The subagent knows
only what the server exposes — it has no direct filesystem access.

## Constraints

- Domain rules are independent of the MCP protocol mechanics.
- Domain rules apply regardless of implementation language or
  MCP library choice.
