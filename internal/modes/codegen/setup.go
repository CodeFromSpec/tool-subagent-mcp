// spec: ROOT/tech_design/internal/modes/codegen/setup@v13

package codegen

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Instructions = `How to use this MCP server:

1. Call load_context with the logical name of the node
   to generate code for.
2. Generate the code from the returned context.
3. Call write_file once per file to write the result,
   passing the same logical_name used in load_context.`

type LoadContextArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

type WriteFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	Content     string `json:"content" jsonschema:"Complete file content to write."`
}

func Setup(s *mcp.Server, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("codegen mode does not accept arguments")
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "load_context",
		Description: "Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response.",
	}, handleLoadContext)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the node's implements list. Overwrites existing content.",
	}, handleWriteFile)
	return nil
}
