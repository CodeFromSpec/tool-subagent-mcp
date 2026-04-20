---
version: 4
parent_version: 8
---

# ROOT/domain/operations/codegen

## Intent

Defines the codegen operation: provides the subagent with the
context it needs to generate code for a spec leaf node, and
restricts its writes to the node's declared outputs.

## Context

### The problem this operation solves

In Code from Spec, a subagent given unrestricted file access will
compensate for perceived gaps in its context by exploring the
repository — reading generated source files, unrelated specs, or
framework documentation — rather than stopping to report ambiguity.
This produces hallucinated or inconsistent output and makes the
generation process unpredictable.

The codegen operation enforces confinement by design: the subagent
receives exactly the spec chain for its leaf node, nothing more. If
that context is insufficient to generate correct code, the only
available action is to stop and report the problem. Exploration is
not possible because the tools to explore do not exist in this
session.

On the write side, the subagent is restricted to the files declared
in the node's `implements` list. This prevents hallucinated writes —
extra files, wrong paths, or unrelated modifications — from
corrupting the project.

### Intended workflow

1. The subagent calls `load_context` once to receive the full chain.
2. The subagent generates the files declared in `implements`.
3. The subagent calls `write_file` once per file.

## Contracts

### CLI signature

```
subagent-mcp codegen <target-logical-name>
```

`os.Args[2]` is the logical name of the target node — either a leaf
spec node (`ROOT/...`) or a test node (`TEST/...`).
Examples: `ROOT/payments/fees/calculation`,
`TEST/payments/fees/calculation`.

### MCP config

```json
{
  "mcpServers": {
    "subagent-mcp": {
      "type": "stdio",
      "command": "<path-to-subagent-mcp>",
      "args": ["codegen", "ROOT/payments/fees/calculation"]
    }
  }
}
```

### Tools exposed

| Tool | Purpose |
|---|---|
| `load_context` | Returns the full spec chain as a single response. |
| `write_file` | Writes a generated file, validated against `implements`. |

## Constraints

- `os.Args[2]` must be a valid `ROOT/` or `TEST/` logical name.
  Absent, empty, or invalid values cause the tool to exit 1 with
  a clear error.
- Native Claude Code file tools (Read, Write, Glob) must be withheld
  from the subagent. The MCP config must not grant them.

## Decisions

### Two tools, minimal surface

`load_context` and `write_file` are the only tools the subagent can
call. This is intentional: the surface area of available actions
defines the agent's behavior space. With only these two tools, the
correct workflow — load context, generate, write — is also the only
possible workflow.

### load_context returns everything in one call

Loading the chain file-by-file via separate tool calls would
accumulate context in the conversation history, increasing token
cost with each roundtrip. A single call returns the entire chain,
keeping the cost flat regardless of chain size.

### write_file validates against implements

The leaf node's `implements` field is the authoritative list of
files this operation may produce. Validating every write against it
prevents the subagent from writing to paths outside the declared
scope, whether by mistake or hallucination.
