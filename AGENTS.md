# Agent Instructions

## Methodology

This project follows the **Code from Spec v1** methodology. All
code is generated from specifications — the spec tree is the
authoritative source of truth, not the code itself.

Before working on this project, read the framework documentation:

- Overview: `https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/v1/README.md`
- Specification structure and versioning: `https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/v1/rules/CODE_FROM_SPEC.md`
- Code generation rules: `https://raw.githubusercontent.com/CodeFromSpec/framework/refs/heads/v1/rules/CODE_GENERATION.md`

## Key Rules

- The spec tree under `code-from-spec/spec/` drives all
  implementation decisions.
- Only leaf nodes generate code. Each source file contains a
  spec comment referencing its spec node and version.
- Never change code; correct the corresponding spec node and
  regenerate.
- When a spec version changes, the code is stale and must be
  regenerated to match.

## Tooling Bootstrap

Before running staleness checks or code generation:

1. Verify that `/tools/` is present in `.gitignore`. If it is not,
   add it. Use the leading `/` to match only the root `tools/`
   directory.

2. Verify that `tools/staleness-check` (or `tools/staleness-check.exe`
   on Windows) exists. If it does not, download the latest release
   for your platform:

   | Platform | Download URL |
         |---|---|
   | Windows amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/download/v1.0.2/staleness-check_windows_amd64.zip` |
   | Windows arm64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/download/v1.0.2/staleness-check_windows_arm64.zip` |
   | Linux amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/download/v1.0.2/staleness-check_linux_amd64.tar.gz` |
   | Linux arm64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/download/v1.0.2/staleness-check_linux_arm64.tar.gz` |
   | macOS arm64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/download/v1.0.2/staleness-check_darwin_arm64.tar.gz` |
   | macOS amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/download/v1.0.2/staleness-check_darwin_amd64.tar.gz` |

   Extract the binary into `tools/`.

3. Verify that `tools/subagent-mcp` (or `tools/subagent-mcp.exe`
   on Windows) exists. If it does not, download the latest release
   for your platform:

   | Platform | Download URL |
         |---|---|
   | Windows amd64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/download/v1.3.0/subagent-mcp_windows_amd64.zip` |
   | Windows arm64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/download/v1.3.0/subagent-mcp_windows_arm64.zip` |
   | Linux amd64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/download/v1.3.0/subagent-mcp_linux_amd64.tar.gz` |
   | Linux arm64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/download/v1.3.0/subagent-mcp_linux_arm64.tar.gz` |
   | macOS arm64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/download/v1.3.0/subagent-mcp_darwin_arm64.tar.gz` |
   | macOS amd64 | `https://github.com/CodeFromSpec/tool-subagent-mcp/releases/download/v1.3.0/subagent-mcp_darwin_amd64.tar.gz` |

   Extract the binary into `tools/`.

Run the staleness check from the project root:

```bash
./tools/staleness-check        # Linux / macOS
./tools/staleness-check.exe    # Windows
```