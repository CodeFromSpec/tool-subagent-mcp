---
version: 16
parent_version: 34
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
---

# ROOT/tech_design/internal/modes/codegen/tools

## Intent

Technical context shared by both codegen tool handlers: how to
return results and errors.

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

## Constraints

- Tool handlers receive `context.Context`, `*mcp.CallToolRequest`,
  and a typed args struct, and return
  `(*mcp.CallToolResult, any, error)`. The returned Go error
  is reserved for catastrophic server failures; all expected error
  conditions must use `IsError: true` on the result instead.
