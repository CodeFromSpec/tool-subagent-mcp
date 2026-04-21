---
version: 12
parent_version: 5
---

# ROOT/domain/modes

## Intent

Defines what a mode is and how the tool selects which mode to run
based on the CLI argument.

## Contracts

### Mode selection

The first CLI argument identifies the mode. Each mode defines its
own set of MCP tools and its own additional arguments.

### Usage message

The tool prints a usage message listing all available modes in
two cases:

- **Help requested** (`--help`, `-h`, or `help` as the first
  argument): prints the usage message and exits 0.
- **Mode absent, empty, or unrecognized**: prints the usage
  message and exits 1.

### Extensibility

New modes are added by defining a new argument value and
implementing the corresponding tool set. Existing modes are
unaffected.

## Constraints

- Mode names are lowercase, single words.
