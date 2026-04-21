// spec: ROOT/tech_design/internal/modes/codegen/help_message@v2

// Package codegen implements the codegen mode for the subagent-mcp tool.
// This file exposes the HelpMessage function, which prints usage and MCP
// configuration instructions for the orchestrator agent when the user runs:
//
//	subagent-mcp codegen --help  (or -h / help)
//
// The server handles help-flag detection before calling Setup, so this
// function only needs to return the text — it has no side effects.
package codegen

// HelpMessage returns the usage and configuration help text for the codegen
// mode, as defined in ROOT/tech_design/internal/modes/codegen (the parent
// spec node). The server prints this when the user passes --help, -h, or
// the literal word "help" as the first argument after the mode name.
func HelpMessage() string {
	// The help text below is specified verbatim in the parent codegen node
	// (ROOT/tech_design/internal/modes/codegen § Help message). It is
	// intentionally plain text — no ANSI escapes — so it renders correctly
	// regardless of how the binary is invoked (terminal, CI log, etc.).
	return `Usage: subagent-mcp codegen <logical-name>

Starts an MCP server over stdin/stdout that provides tools
for code generation. The logical name identifies a spec or
test node that implements source code files.

The server exposes two tools:
  load_context   Returns the context for code generation.
  write_file     Writes a generated file to disk.

The subagent should have no other tools available — no file
read, write, or search capabilities beyond what this server
provides. This confinement ensures the subagent works only
from the provided context and writes only to declared outputs.

MCP configuration example:
  {
    "mcpServers": {
      "subagent-mcp": {
        "type": "stdio",
        "command": "<path-to-binary>",
        "args": ["codegen", "<logical-name>"]
      }
    }
  }`
}
