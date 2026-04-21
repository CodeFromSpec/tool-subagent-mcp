---
version: 2
parent_version: 22
implements:
  - cmd/subagent-mcp/main_test.go
---

# TEST/tech_design/server

## Context

Tests invoke the compiled binary as a subprocess using
`os/exec` and verify its behavior: exit codes, stderr
output, and stdout output. For tests that verify MCP
server behavior, connect via the MCP SDK's in-memory
or stdio transport.

## Happy Path

### Help flag prints usage to stdout

Run the binary with `--help`.

Expect: exit 0, stdout contains the usage message.

### Help word prints usage to stdout

Run the binary with `help`.

Expect: exit 0, stdout contains the usage message.

### Short help flag prints usage to stdout

Run the binary with `-h`.

Expect: exit 0, stdout contains the usage message.

### Codegen mode sets correct server instructions

Run the binary with `codegen` and a valid logical name.
Connect as an MCP client.

Expect: the server's initialize response contains
`instructions` matching `codegen.Instructions`.

## Failure Cases

### No arguments prints usage to stderr

Run the binary with no arguments.

Expect: exit 1, stderr contains the usage message.

### Unrecognized mode prints usage to stderr

Run the binary with `unknownmode`.

Expect: exit 1, stderr contains the usage message
and lists valid modes.

### Codegen setup error prints to stderr

Run the binary with `codegen` and no logical name.

Expect: exit 1, stderr contains the setup error.
