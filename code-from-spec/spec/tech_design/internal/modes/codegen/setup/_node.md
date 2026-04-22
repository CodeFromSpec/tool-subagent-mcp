---
version: 13
parent_version: 36
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes/codegen
    version: 21
implements:
  - internal/modes/codegen/setup.go
---

# ROOT/tech_design/internal/modes/codegen/setup

## Intent

Implements the `Setup` function for the codegen mode: declares
the `Instructions` constant and registers the `load_context`
and `write_file` tools on the MCP server.

## Contracts

### Instructions constant

```go
const Instructions = "..."
```

Package-level constant holding the server instructions string defined in the
parent node. Declared in this file so `main` can reference it via
`codegen.Instructions` when creating the MCP server.

### Setup function

```go
func Setup(s *mcp.Server, args []string) error
```

If `len(args) > 0`, returns an error:
`"codegen mode does not accept arguments"`.

Registers `load_context` and `write_file` tools on `s` using
`mcp.AddTool`. Returns `nil` on success.
