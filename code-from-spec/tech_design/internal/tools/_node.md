---
version: 8
parent_version: 14
---

# ROOT/tech_design/internal/tools

Technical context shared by all tool handlers: how to return
results and errors.

# Public

## Contracts

### Tool result — success

Return a `*mcp.CallToolResult` with a `mcp.TextContent` entry:

```go
&mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: content}},
}
```

### Tool result — error

Return a `*mcp.CallToolResult` with `IsError: true`:

```go
&mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: message}},
    IsError: true,
}
```

Do not use panic for error handling — always return an MCP tool
error so the server remains running.

### Error messages

Error messages returned to the agent must be actionable: they must
identify what went wrong and what the agent can do about it (or
indicate that the error is unrecoverable).

### Path validation

File paths from `implements` are validated using `ValidatePath`
in `load_chain`, `write_file`, and `find_replace` before any
write. Each handler resolves the frontmatter independently and
validates the paths against the working directory.

## Constraints

- Tool handlers must be stateless — each call resolves its
  own inputs independently. The MCP host (e.g. Claude Code)
  keeps a single server process for the entire session, and
  multiple subagents may call tools on it concurrently.
- Tool handlers receive `context.Context`, `*mcp.CallToolRequest`,
  and a typed args struct, and return
  `(*mcp.CallToolResult, any, error)`. The returned Go error
  is reserved for catastrophic server failures; all expected error
  conditions must use `IsError: true` on the result instead.
