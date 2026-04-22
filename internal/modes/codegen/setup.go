// spec: ROOT/tech_design/internal/modes/codegen/setup@v13

package codegen

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
)

const Instructions = `How to use this MCP server:

1. Call load_context with the logical name of the node
   to generate code for. This loads the context and must be
   called exactly once per session.
2. Generate the code from the returned context.
3. Call write_file once per file to write the result.`

type Target struct {
	LogicalName  string
	FilePath     string
	Frontmatter  *frontmatter.Frontmatter
	ChainContent string
}

var currentTarget *Target

type LoadContextArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

type WriteFileArgs struct {
	Path    string `json:"path" jsonschema:"Relative file path from project root."`
	Content string `json:"content" jsonschema:"Complete file content to write."`
}

func Setup(s *mcp.Server, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("codegen mode does not accept arguments")
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "load_context",
		Description: "Load the spec chain context for a given logical name. Must be called exactly once before write_file. Returns all relevant spec files concatenated in a single response.",
	}, handleLoadContext)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the current node's implements list. Overwrites existing content.",
	}, handleWriteFile)
	return nil
}
