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
