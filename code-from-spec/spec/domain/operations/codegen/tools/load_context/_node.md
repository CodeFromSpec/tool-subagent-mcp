---
version: 1
parent_version: 1
---

# ROOT/domain/operations/codegen/tools/load_context

## Intent

Returns the complete chain for the session's leaf node as a single
response, so the subagent receives all context in one tool call.

## Contracts

### Inputs

None. The tool takes no arguments. The chain is fully determined by
the session's leaf node argument.

### Output

A single text response containing every file in the chain,
concatenated in chain order (as defined by
`ROOT/domain/operations/codegen/chain`).

Files are separated by a structured boundary so the agent can
distinguish where one file ends and another begins. The exact
format of the boundary is defined in
`ROOT/tech_design/operations/codegen/tools/load_context`.

### Idempotency

Calling `load_context` multiple times within a session returns the
same response (assuming the filesystem has not changed). The tool
does not modify any state.

## Constraints

- The tool must read all chain files at call time, not at server
  startup. Startup failure is not acceptable for a file-not-found
  condition in the chain.
- If any chain file cannot be read, the tool returns an error
  identifying the missing file; it does not return partial results.
