# Agent Instructions

## Methodology — Read First

This project follows the **Code from Spec** methodology. All code
is generated from specifications — the spec tree is the
authoritative source of truth, not the code itself.

**You MUST fetch and read the documents below if you have not already
done so.** Do not proceed until you have read
all three. This is a blocking prerequisite — no code generation,
staleness checks, or spec modifications until the methodology is
loaded.

- Overview: `https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/main/README.md`
- Specification structure and versioning: `https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/main/rules/CODE_FROM_SPEC.md`
- Code generation rules: `https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/main/rules/CODE_GENERATION.md`

## Tooling Bootstrap

**You MUST ensure tooling is properly installed if you have not already
done so:**

1. Verify that `/tools/` is present in `.gitignore`. If it is not,
   add it. Use the leading `/` to match only the root `tools/`
   directory.

2. Verify that `tools/staleness-check` (or `tools/staleness-check.exe`
   on Windows) exists. If it does not, download the latest release
   for your platform:

   | Platform | Download URL |
      |---|---|
   | Windows amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_windows_amd64.zip` |
   | Linux amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_linux_amd64.tar.gz` |
   | Linux arm64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_linux_arm64.tar.gz` |
   | macOS arm64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_darwin_arm64.tar.gz` |
   | macOS amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_darwin_amd64.tar.gz` |

   Extract the binary into `tools/`.

3. Verify that `tools/subagent-mcp` (or `tools/subagent-mcp.exe`
   on Windows) exists. If it does not, download the latest release
   for your platform:

   | Platform | Download URL |
      |---|---|
   | Windows amd64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/latest/download/subagent-mcp_windows_amd64.zip` |
   | Windows arm64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/latest/download/subagent-mcp_windows_arm64.zip` |
   | Linux amd64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/latest/download/subagent-mcp_linux_amd64.tar.gz` |
   | Linux arm64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/latest/download/subagent-mcp_linux_arm64.tar.gz` |
   | macOS arm64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/latest/download/subagent-mcp_darwin_arm64.tar.gz` |
   | macOS amd64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/latest/download/subagent-mcp_darwin_amd64.tar.gz` |

   Extract the binary into `tools/`.

4. Verify that `.claude/agents/codegen.md` exists. If it does
   not, download it from the framework repository:

   ```
   https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/main/subagents/code_generation_subagent.md
   ```

   Save the contents to `.claude/agents/codegen.md`.

5. Verify that the `subagent-mcp` MCP server is configured.
   Check both `.mcp.json` and `.claude/settings.json`. If
   neither contains the configuration, create or update
   `.mcp.json` in the project root with:

   ```json
   {
     "mcpServers": {
       "subagent-mcp": {
         "type": "stdio",
         "command": "tools/subagent-mcp.exe"
       }
     }
   }
   ```

   On Linux and macOS, use `tools/subagent-mcp` (without
   the `.exe` extension).

   If you created or modified the MCP configuration, inform
   the user that a restart of Claude Code is required for the
   new MCP server to become available.

## Workflow Rules

- **Do not** run staleness checks, resolve staleness, or
  generate code unless the user explicitly requests it.
- Reading the methodology documents (the three URLs above)
  remains a prerequisite before any of these actions.
