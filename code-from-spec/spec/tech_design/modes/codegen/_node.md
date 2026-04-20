---
version: 7
parent_version: 5
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 17
---

# ROOT/tech_design/modes/codegen

## Intent

Technical design for the codegen mode: argument parsing,
session initialization, tool registration, and MCP server startup.

## Contracts

### File organization

```
cmd/subagent-mcp/modes/codegen/
  codegen.go        ← Run(), Session type, argument parsing
  chainresolver.go  ← chain resolution algorithm
  load_context.go   ← load_context tool handler
  write_file.go     ← write_file tool handler
```

All files under `package codegen`.

### Run function

```go
func Run(args []string) error
```

1. Validate `args[0]` is present and non-empty (the leaf logical
   name). If not, return a usage error.
2. Validate the logical name starts with `ROOT`. If not, return
   an error.
3. Call `ParseFrontmatter` on the resolved file path.
4. Build a `Session` and register `load_context` and `write_file`
   with the MCP server.
5. Start the stdio server and block until the client disconnects.

### Session type

```go
type Session struct {
    LeafLogicalName string
    LeafFilePath    string
    LeafFrontmatter *Frontmatter
}
```

### Error handling

Errors returned from `Run` are printed to stderr by `main` and
cause an exit 1.
