---
version: 7
parent_version: 2
---

# ROOT/domain/operations

## Intent

Defines what an operation is and how the tool selects which
operation to run based on the CLI argument.

## Contracts

### Operation selection

The first CLI argument identifies the operation. Each operation
defines its own set of MCP tools and its own additional arguments.

### Usage message

When the operation argument is absent, empty, or unrecognized,
the tool prints a usage message listing all available operations
and exits 1.

### Extensibility

New operations are added by defining a new argument value and
implementing the corresponding tool set. Existing operations
are unaffected.

## Constraints

- If the operation argument is absent, empty, or unrecognized,
  the tool prints a usage message and exits 1.
- Operation names are lowercase, single words.
