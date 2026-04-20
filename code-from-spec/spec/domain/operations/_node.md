---
version: 8
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

The tool prints a usage message listing all available operations in
two cases:

- **Help requested** (`--help`, `-h`, or `help` as the first
  argument): prints the usage message and exits 0.
- **Operation absent, empty, or unrecognized**: prints the usage
  message and exits 1.

### Extensibility

New operations are added by defining a new argument value and
implementing the corresponding tool set. Existing operations
are unaffected.

## Constraints

- Operation names are lowercase, single words.
