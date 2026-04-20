---
version: 1
parent_version: 1
---

# ROOT/domain/operations

## Intent

Defines what an operation is and how the server selects which
operation to run based on the CLI argument.

## Contracts

### Operation selection

The first CLI argument (`os.Args[1]`) identifies the operation.
Each operation defines its own set of MCP tools and its own
additional arguments.

Currently defined operations:

| Argument | Operation |
|---|---|
| `codegen` | Code generation from a spec leaf node. |

### Extensibility

New operations are added by defining a new argument value and
implementing the corresponding tool set. Existing operations
are unaffected.

## Constraints

- If the operation argument is absent, empty, or unrecognized,
  the server prints a usage message to stderr and exits 1.
- Operation names are lowercase, single words.
