// code-from-spec: ROOT/tech_design/server@v51
// spec: ROOT/tech_design/server@v51
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/load_chain"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/patch_file"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/write_file"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// usageMessage is the help text printed when the binary is invoked
// with arguments. Defined as a constant so it is shared between
// the help (stdout) and error (stderr) paths.
const usageMessage = `Usage: subagent-mcp

Starts an MCP server over stdin/stdout for Code from Spec
subagents.

Tools:
  load_chain     Load the spec chain for a node.
  write_file     Write a generated file to disk.
  patch_file     Apply a unified diff to an existing file.

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
	// Step 1–2: Argument validation.
	// Any argument triggers the usage message. Help flags go to
	// stdout with exit 0; anything else goes to stderr with exit 1.
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
	server := mcp.NewServer(&mcp.Implementation{
		Name: "subagent-mcp",
	}, nil)

	// Step 4: Register tools.
	// Tool names and descriptions come from the corresponding tool
	// definition specs (load_chain, write_file, patch_file).

	// Register load_chain — loads the spec chain for a given
	// logical name and returns all relevant spec files concatenated.
	// Meta advertises the maximum result size so clients (e.g.
	// Claude) know the tool may return large responses.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "load_chain",
		Description: "Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response.",
		Meta:        mcp.Meta{"anthropic/maxResultSizeChars": 500000},
	}, load_chain.HandleLoadChain)

	// Register write_file — writes a generated source file to disk,
	// validating the path against the node's implements list.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the node's implements list. Overwrites existing content.",
	}, write_file.HandleWriteFile)

	// Register patch_file — applies a unified diff to an existing
	// source file, validating the path against the node's implements
	// list. The file must already exist; use write_file to create new
	// files.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "patch_file",
		Description: "Apply a unified diff to an existing source file. The path must be one of the files declared in the node's implements list. The file must already exist.",
	}, patch_file.HandlePatchFile)

	// Step 5–7: Run the server over stdio. Blocks until the client
	// disconnects. If Run returns an error, print it and exit 1.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
