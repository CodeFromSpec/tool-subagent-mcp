// spec: ROOT/tech_design/server@v23

// Package main is the entry point for the subagent-mcp binary.
//
// It reads the first CLI argument to determine the mode, sets up the
// corresponding MCP server via that mode's Setup function, and runs
// the server over stdin/stdout using the MCP stdio transport.
//
// Invocation:
//
//	subagent-mcp <mode> [args...]
//
// Currently supported modes:
//   - codegen: Generate code for a spec or test node.
//
// Error handling follows the spec contract:
//   - Startup errors (missing/invalid mode, failed Setup) → stderr + exit 1.
//   - Tool errors are returned as MCP tool error responses (server keeps running).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gustavo-neto/tool-subagent-mcp/internal/modes/codegen"
)

// usageMessage is the top-level help text printed when no mode is given,
// an unrecognized mode is given, or --help/-h/help is passed as the first
// argument. It lists every available mode.
//
// Spec ref: ROOT/tech_design/server § "Usage message"
const usageMessage = `Usage: subagent-mcp <mode> [args...]

Modes:
  codegen <logical-name>   Generate code for a spec or test node.

Run subagent-mcp <mode> --help for mode-specific help.
`

func main() {
	os.Exit(run())
}

// run contains the full startup sequence so main can delegate and still use
// os.Exit cleanly. Returns the exit code (0 = success, 1 = error).
//
// Startup sequence (spec ref: ROOT/tech_design/server § "Startup sequence"):
//  1. Validate that a mode argument was provided.
//  2. Handle top-level help flags.
//  3. Dispatch to the matching mode.
//  4. Call Setup; abort on error.
//  5. Run the MCP server over stdio.
func run() int {
	// Step 1: Require at least one argument (the mode name).
	if len(os.Args) < 2 || os.Args[1] == "" {
		fmt.Fprint(os.Stderr, usageMessage)
		return 1
	}

	mode := os.Args[1]

	// Step 2: Handle top-level help — print to stdout and exit 0.
	if mode == "--help" || mode == "-h" || mode == "help" {
		fmt.Print(usageMessage)
		return 0
	}

	// Step 3: Dispatch to the recognized mode.
	// Each branch may inspect os.Args[2:] for mode-specific help before
	// delegating further argument handling to Setup.
	var s *mcp.Server

	switch mode {
	case "codegen":
		// Step 3a: Check for mode-level help before doing any setup work.
		if len(os.Args) > 2 && isHelpFlag(os.Args[2]) {
			fmt.Print(codegen.HelpMessage())
			return 0
		}

		// Step 3b: Create the MCP server with the codegen instructions so the
		// subagent understands how to interact with the server from the start.
		// The Instructions constant is defined in the codegen package
		// (spec ref: ROOT/tech_design/internal/modes/codegen § "Server instructions").
		s = mcp.NewServer(&mcp.Implementation{
			Name: "subagent-mcp",
		}, &mcp.ServerOptions{
			Instructions: codegen.Instructions,
		})

		// Step 3c: Let the codegen mode register its tools and validate args.
		// os.Args[2:] contains everything after "codegen" (e.g. the logical name).
		if err := codegen.Setup(s, os.Args[2:]); err != nil {
			// Step 4: Setup failed → report to stderr and exit 1.
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}

	default:
		// Unrecognized mode — print usage listing valid modes and exit 1.
		fmt.Fprintf(os.Stderr, "error: unknown mode %q\n\n", mode)
		fmt.Fprint(os.Stderr, usageMessage)
		return 1
	}

	// Step 5: Run the MCP server over stdio. Blocks until the client disconnects.
	// spec ref: ROOT/tech_design/server § "Startup sequence" step 5
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		// Step 6: Server error → report to stderr and exit 1.
		fmt.Fprintf(os.Stderr, "error: server exited with error: %v\n", err)
		return 1
	}

	// Step 7: Clean shutdown.
	return 0
}

// isHelpFlag reports whether the given string is one of the recognized
// help request tokens: --help, -h, or help.
func isHelpFlag(s string) bool {
	return s == "--help" || s == "-h" || s == "help"
}
