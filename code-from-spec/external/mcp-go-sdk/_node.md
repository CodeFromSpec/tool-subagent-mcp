---
version: 4
parent_version: 2
---

# ROOT/external/mcp-go-sdk

Official Go SDK for the Model Context Protocol:
`github.com/modelcontextprotocol/go-sdk`.

# Public

## Import

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"
```

## Creating a server

```go
server := mcp.NewServer(&mcp.Implementation{
    Name: "server-name",
}, nil)
```

## Registering a tool

```go
func AddTool[In, Out any](s *Server, t *Tool, h ToolHandlerFor[In, Out])
```

Input schema is inferred from `In` struct fields. Use `json`
tags for field names and `jsonschema` tags for descriptions.

Tool with no parameters:

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "my_tool",
    Description: "...",
}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: "result"}},
    }, nil, nil
})
```

Tool with parameters:

```go
type Args struct {
    Path    string `json:"path" jsonschema:"file path"`
    Content string `json:"content" jsonschema:"file content"`
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "write_file",
    Description: "...",
}, func(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error) {
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: "wrote " + args.Path}},
    }, nil, nil
})
```

## Tool metadata

`mcp.Tool` embeds `mcp.Meta` (`map[string]any`), serialized as
`_meta` in JSON. Set entries directly on the `Meta` field:

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "my_tool",
    Description: "...",
    Meta:        mcp.Meta{"anthropic/maxResultSizeChars": 500000},
}, handler)
```

## Returning a success result

```go
&mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: "message"}},
}
```

## Returning a tool error

```go
&mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: "error message"}},
    IsError: true,
}
```

## Running the server over stdio

```go
if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
    // handle error
}
```

Blocks until the client disconnects.
