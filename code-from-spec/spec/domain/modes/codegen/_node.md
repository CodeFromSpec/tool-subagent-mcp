---
version: 18
parent_version: 11
---

# ROOT/domain/modes/codegen

## Intent

Defines the codegen mode: provides the subagent with the
context it needs to generate code for a spec leaf node, and
restricts its writes to the node's declared outputs.

## Context

### The problem this mode solves

Given unrestricted file access, a subagent will compensate for
perceived gaps in its context by exploring the repository — reading
generated source files, unrelated specs, or framework documentation
— rather than stopping to report ambiguity. This can produce
hallucinated or inconsistent output and make the code generation
process unpredictable.

The codegen mode enforces confinement by design: the subagent
receives exactly the spec chain for its leaf node, nothing more. If
that context is insufficient to generate correct code, the only
available action is to stop and report the problem. Exploration is
not possible because the tools to explore do not exist.

On the write side, the subagent is restricted to the files declared
in the node's `implements` list. This prevents hallucinated writes —
extra files, wrong paths, or unrelated modifications — from
corrupting the project.

### Intended workflow

1. The subagent calls `load_context` once to receive the full chain.
2. The subagent generates the files declared in `implements`.
3. The subagent calls `write_file` once per file.

## Contracts

### Target node

The target node is identified by its logical name — either a leaf
spec node (`ROOT/...`) or a test node (`TEST/...`). Examples:
`ROOT/payments/fees/calculation`,
`TEST/payments/fees/calculation`.

### MCP tools exposed

The subagent has access to exactly two tools — no native file
access. The minimal surface ensures correct behavior by
construction: the surface area of available actions defines the
agent's action space.

#### load_context

Returns the complete chain as a single text response, so the
subagent receives all context in one tool call. Takes no arguments.
The chain is fully determined by the target node. Files in the chain
are returned with a structured separator between them so the
subagent can distinguish boundaries.

#### write_file

Writes a generated file to disk. Takes two arguments:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `path`    | string | Relative path to write (from project root). |
| `content` | string | Full file content to write. |

Before writing, validates that `path` appears in the `implements`
list of the target node's frontmatter. Once validated, writes the
file to disk, creating any intermediate directories if needed.
Overwrites if the file already exists.

## Constraints

- The target argument must be a logical name that resolves to a
  node with `implements`. Absent, empty, or invalid values cause
  the tool to exit 1 with a clear error.
- Reads are limited to the chain; writes are limited to `implements`.
- The validation against `implements` is the security boundary of
  `write_file`. It must not be bypassable.
- Exactly one file is written per `write_file` call.
- If any chain file cannot be read, `load_context` returns an error
  identifying the missing file; it does not return partial results.

## Decisions

### Two tools, minimal surface

`load_context` and `write_file` are the only tools the subagent can
call. With only these two tools, the correct workflow — load context,
generate, write — is also the only possible workflow.

### load_context returns everything in one call

Loading the chain file-by-file via separate tool calls would
accumulate context in the conversation history, increasing token
cost with each roundtrip. A single call returns the entire chain,
minimizing roundtrip overhead.

### write_file validates against implements

The leaf node's `implements` field is the authoritative list of
files this mode may produce. Validating every write against it
prevents the subagent from writing to paths outside the declared
scope, whether by mistake or hallucination.
