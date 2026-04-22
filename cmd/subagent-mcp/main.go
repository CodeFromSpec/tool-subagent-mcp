// spec: ROOT/tech_design/server@v30
package main

// Entry point for the subagent-mcp MCP server.
//
// Spec ref: ROOT/tech_design/server § "Startup sequence"
//   1. Handle --help / -h / help  → print usage to stdout, exit 0.
//   2. Any other argument         → print usage to stderr, exit 1.
//   3. Create MCP server with Implementation.Name = "subagent-mcp".
//   4. Register load_chain and write_file tools via mcp.AddTool.
//   5. Run the server over stdio.
//   6. On Run error → print to stderr, exit 1.

import (
	"context"
	"fmt"
	"os"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/load_chain"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/write_file"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// usageMessage is printed on --help/-h/help (stdout, exit 0) and on an
// unrecognised argument (stderr, exit 1).
//
// Spec ref: ROOT/tech_design/server § "Usage message"
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
	// Spec ref: ROOT/tech_design/server § "Startup sequence" steps 1–2.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			// Help flags: print usage to stdout and exit cleanly.
			fmt.Print(usageMessage)
			os.Exit(0)
		default:
			// Any other argument is an error: print usage to stderr and exit 1.
			fmt.Fprint(os.Stderr, usageMessage)
			os.Exit(1)
		}
	}

	// Spec ref: ROOT/tech_design/server § "Startup sequence" step 3.
	// Create the MCP server with the required implementation name.
	s := mcp.NewServer(&mcp.Implementation{
		Name: "subagent-mcp",
	}, nil)

	// Spec ref: ROOT/tech_design/server § "Startup sequence" step 4.
	// Register tools. Each package exposes its own registration function.
	load_chain.RegisterTool(s)
	write_file.Register(s)

	// Spec ref: ROOT/tech_design/server § "Startup sequence" steps 5–7.
	// Run blocks until the client disconnects. Any error is fatal.
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "subagent-mcp: %v\n", err)
		os.Exit(1)
	}
}
