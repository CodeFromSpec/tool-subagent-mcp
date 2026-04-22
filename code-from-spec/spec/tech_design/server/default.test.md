---
version: 9
parent_version: 26
implements:
  - cmd/subagent-mcp/main_test.go
---

# TEST/tech_design/server

## Context

Tests invoke the compiled binary as a subprocess using
`os/exec` and verify its behavior: exit codes, stderr
output, and stdout output. For tests that verify MCP
server behavior, use `mcp.Client` with `mcp.CommandTransport`
to connect to the server binary over stdio.

The binary is built once in `TestMain` into a temp directory.
On Windows, the binary name must include the `.exe` extension:
use `runtime.GOOS == "windows"` to detect the platform and
append `.exe` to the output path when building.

`go test` sets the working directory to the package directory
(`cmd/subagent-mcp/`). Any subprocess that reads spec files
via relative paths must have its working directory set to the
project root. Derive the project root from `os.Getwd()` and
walking up two directory levels (`filepath.Join(wd, "..", "..")`).
Set `cmd.Dir` to this project root for any subprocess that
needs access to spec files (i.e. the MCP initialize test).

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

### Codegen help flag prints mode help to stdout

Run the binary with `codegen --help`.

Expect: exit 0, stdout contains the codegen help message.

### Codegen mode sets correct server instructions

Run the binary with `codegen` using `mcp.CommandTransport`.
Build the command with `cmd.Dir` set to the project root
(derived by walking up two levels from `os.Getwd()`).
Connect an `mcp.Client` to the server using `client.Connect`.

After the connection is established, call
`cs.InitializeResult()` and assert that `.Instructions`
matches `codegen.Instructions`.

Close the session and wait for the subprocess to exit.

Import `"github.com/modelcontextprotocol/go-sdk/mcp"` for the
client transport types.

## Failure Cases

### No arguments prints usage to stderr

Run the binary with no arguments.

Expect: exit 1, stderr contains the usage message.

### Unrecognized mode prints usage to stderr

Run the binary with `unknownmode`.

Expect: exit 1, stderr contains the usage message
and lists valid modes.

### Codegen with extra arguments prints error to stderr

Run the binary with `codegen extraarg`.

Expect: exit 1, stderr contains
`"codegen mode does not accept arguments"`.
