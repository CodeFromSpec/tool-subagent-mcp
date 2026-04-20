---
version: 2
parent_version: 2
depends_on:
  - path: ROOT/domain/operations/codegen/tools
    version: 2
---

# ROOT/tech_design/operations/codegen/tools

## Intent

Technical context shared by both codegen tool handlers: how to
return results and errors via mcp-go.

## Contracts

### Tool result — success

Return a `mcp.NewToolResultText(content)` response.

### Tool result — error

Return a `mcp.NewToolResultError(message)` response. Do not panic
or return a Go error from a tool handler — always return an MCP
tool error so the server remains running.

### Error messages

Error messages returned to the agent must be actionable: they must
identify what went wrong and what the agent can do about it (or
indicate that the error is unrecoverable for this session).

## Constraints

- Tool handlers receive `context.Context` and `mcp.CallToolRequest`
  and return `(*mcp.CallToolResult, error)`. The returned Go error
  is reserved for catastrophic server failures; all expected error
  conditions must use `mcp.NewToolResultError` instead.
