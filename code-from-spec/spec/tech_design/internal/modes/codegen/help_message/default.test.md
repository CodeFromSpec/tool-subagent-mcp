---
version: 1
parent_version: 1
implements:
  - internal/modes/codegen/help_message_test.go
---

# TEST/tech_design/internal/modes/codegen/help_message

## Happy Path

### Contains usage line

Call `HelpMessage()`.

Expect: result contains `"Usage: subagent-mcp codegen"`.

### Contains tool descriptions

Call `HelpMessage()`.

Expect: result contains `"load_context"` and `"write_file"`.

### Contains MCP configuration example

Call `HelpMessage()`.

Expect: result contains `"mcpServers"` and `"stdio"`.

### Contains confinement guidance

Call `HelpMessage()`.

Expect: result contains `"no other tools available"`.
