// spec: ROOT/tech_design/internal/modes/codegen/help_message@v3

package codegen

func HelpMessage() string {
	return `Usage: subagent-mcp codegen

Starts an MCP server over stdin/stdout that provides tools
for code generation.

The server exposes two tools:
  load_context   Loads the context for a spec node and returns it.
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
        "args": ["codegen"]
      }
    }
  }`
}
