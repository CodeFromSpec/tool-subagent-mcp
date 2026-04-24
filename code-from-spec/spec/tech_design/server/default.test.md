---
version: 19
parent_version: 41
implements:
  - cmd/subagent-mcp/main_test.go
---

# TEST/tech_design/server

## Context

Tests invoke the compiled binary as a subprocess using
`os/exec` and verify its behavior: exit codes, stderr
output, and stdout output.

The binary is built once in `TestMain` into a temp directory.
On Windows, the binary name must include the `.exe` extension:
use `runtime.GOOS == "windows"` to detect the platform and
append `.exe` to the output path when building.

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

## Failure Cases

### Unrecognized argument prints usage to stderr

Run the binary with `something`.

Expect: exit 1, stderr contains the usage message.

### Multiple arguments prints usage to stderr

Run the binary with `foo bar`.

Expect: exit 1, stderr contains the usage message.
