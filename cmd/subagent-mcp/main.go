// spec: ROOT/tech_design/server@v34
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	load_chain "github.com/CodeFromSpec/tool-subagent-mcp/internal/load_chain"
	write_file "github.com/CodeFromSpec/tool-subagent-mcp/internal/write_file"
)

// usageMessage is the exact text printed for help or invalid arguments.
const usageMessage = `Usage: subagent-mcp

Starts an MCP server over stdin/stdout for Code from Spec
subagents.

Tools:
  load_chain     Load the spec chain for a node.
  write_file     Write a generated file to disk.

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
	// Step 1-2: Argument validation.
	if len(os.Args) > 1 {
		arg := os.Args[1]
		// Help flags print to stdout and exit 0.
		if arg == "--help" || arg == "-h" || arg == "help" {
			fmt.Print(usageMessage)
			os.Exit(0)
		}
		// Any other argument prints to stderr and exits 1.
		fmt.Fprint(os.Stderr, usageMessage)
		os.Exit(1)
	}

	// Step 3: Create the MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name: "subagent-mcp",
	}, nil)

	// Step 4: Register tools with their handlers from the
	// corresponding internal packages.

	// Register load_chain tool.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "load_chain",
		Description: "Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response.",
	}, load_chain.HandleLoadChain)

	// Register write_file tool.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the node's implements list. Overwrites existing content.",
	}, write_file.HandleWriteFile)

	// Step 5-7: Run the server over stdio. Blocks until the
	// client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
