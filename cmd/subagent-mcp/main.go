// code-from-spec: ROOT/tech_design/server@v60
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/find_replace"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/load_chain"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/write_file"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// usageMessage is the help text printed when the binary is invoked
// with --help/-h/help (stdout, exit 0) or any other argument
// (stderr, exit 1). Defined as a constant so the same text is
// shared between both paths.
const usageMessage = `Usage: subagent-mcp

Starts an MCP server over stdin/stdout for Code from Spec
subagents.

Tools:
  load_chain     Load the spec chain for a node.
  write_file     Write a generated file to disk.
  find_replace   Replace a specific string in an existing file.

The subagent should have no other tools available — no file
read, write, or search capabilities beyond what this server
provides. This confinement ensures the subagent works only
from the provided context and writes only to declared outputs.

MCP configuration example:
  {
    "mcpServers": {
      "subagent-mcp": {
        "type": "stdio",
        "command": "<path-to-binary>"
      }
    }
  }
`

func main() {
	// Step 1: Handle help flags — print usage to stdout and exit 0.
	// Step 2: Reject any other argument — print usage to stderr and exit 1.
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" || arg == "help" {
			fmt.Print(usageMessage)
			os.Exit(0)
		}
		fmt.Fprint(os.Stderr, usageMessage)
		os.Exit(1)
	}

	// Step 3: Create the MCP server.
	// Implementation.Name is "subagent-mcp" as required by the spec.
	server := mcp.NewServer(&mcp.Implementation{
		Name: "subagent-mcp",
	}, nil)

	// Step 4: Register tools using mcp.AddTool.
	// Tool names and descriptions are taken verbatim from the
	// corresponding tool definition specs.

	// load_chain — loads the spec chain for a given logical name and
	// returns all relevant spec files concatenated in a single response.
	// Meta advertises the maximum result size (500000 chars) so that
	// tools/list exposes this constraint to the client (e.g. Claude).
	mcp.AddTool(server, &mcp.Tool{
		Name:        "load_chain",
		Description: "Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response.",
		Meta:        mcp.Meta{"anthropic/maxResultSizeChars": 500000},
	}, load_chain.HandleLoadChain)

	// write_file — writes a generated source file to disk after
	// validating the path against the node's implements list.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the node's implements list. Overwrites existing content.",
	}, write_file.HandleWriteFile)

	// find_replace — replaces a specific string in an existing source
	// file after validating the path against the node's implements list.
	// The old_string must appear exactly once in the file.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_replace",
		Description: "Replace a specific string in an existing source file. The old_string must appear exactly once. The path must be one of the files declared in the node's implements list. The file must already exist.",
	}, find_replace.HandleFindReplace)

	// Steps 5–7: Run the server over stdio. Blocks until the client
	// disconnects. On error, print to stderr and exit 1.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
