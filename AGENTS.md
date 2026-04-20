# Agent Instructions

## Methodology

This project follows the **Code from Spec** methodology. All code is generated from specifications — the spec tree is the authoritative source of truth, not the code itself.

Before working on this project, read the framework documentation:

- Overview: `https://raw.githubusercontent.com/CodeFromSpec/.github/refs/heads/main/README.md`
- Specification structure and versioning: `https://raw.githubusercontent.com/CodeFromSpec/.github/refs/heads/main/framework/CODE_FROM_SPEC.md`
- Code generation rules: `https://raw.githubusercontent.com/CodeFromSpec/.github/refs/heads/main/framework/AGENT_CODE_GENERATION.md`

## Key Rules

- The spec tree under `code-from-spec/spec/` drives all implementation decisions.
- Only leaf nodes generate code. Each Go file begins with a `// spec:` comment referencing its spec node and version.
- Never change code without understanding the corresponding spec node first.
- When a spec version changes, the code is stale and must be regenerated to match.

## Tooling Bootstrap

Before running staleness checks:

1. Verify that `tools/` is present in `.gitignore`. If it is not, add it.
2. Verify that `tools/staleness-check` (or `tools/staleness-check.exe` on
   Windows) exists. If it does not, download the latest release for your
   platform:

| Platform | Download URL |
|---|---|
| Windows amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_windows_amd64.zip` |
| Linux amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_linux_amd64.tar.gz` |
| macOS arm64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_darwin_arm64.tar.gz` |
| macOS amd64 | `https://github.com/CodeFromSpec/tool-staleness-check/releases/latest/download/staleness-check_darwin_amd64.tar.gz` |

Extract the binary into `tools/`. The `tools/` directory is gitignored — do
not commit binaries.

Run the staleness check from the project root:

```bash
./tools/staleness-check        # Linux / macOS
./tools/staleness-check.exe    # Windows
```
