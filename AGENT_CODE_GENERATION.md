# Code Generation Agent Rules

This document outlines procedures for an AI code generation agent that implements specifications using the `subagent-mcp` server.

## Core Workflow

The orchestrator provides the logical name of the target node in the agent's prompt.

1. Call `load_chain` with that logical name. This returns all relevant spec files concatenated in a single response. Must be called exactly once.
2. Generate the source files declared in the node's `implements` list from the returned context.
3. Write each file using the `write_file` tool.

## Key Principles

**Minimize changes.** Only modify code as needed to meet the spec — no unnecessary reformatting or restructuring.

**Prioritize readability.** Write straightforward, well-commented code that reviewers can easily verify against the specification.

**Abundant comments.** Explain intent and reference spec sections. Comments should help a reviewer trace each piece of code back to the spec that mandates it (e.g., `// Spec ref: ROOT/tech_design/server § "Startup sequence"`).

**Respect all constraints.** Every rule and convention provided is mandatory; nothing is optional.

**Use only provided input.** The agent may only use context returned by `load_chain` and the content of existing target files. No external searches or assumptions.

## Specification Comment Format

Every generated file must include a comment on its first line (where syntactically allowed):

```
// spec: <logical-name>@v<version>
```

The logical name identifies the spec node (e.g., `ROOT/architecture/backend/config` or `TEST/architecture/backend/config`). The version must match the node's current version.

## File Processing Logic

For each target file:

1. **Check existence** — Does the file already exist?
2. **New file** — Generate from scratch using the context returned by `load_chain`.
3. **Existing file** — Compare the `// spec:` version comment:
   - Versions match → No action needed.
   - Versions differ → Update comment only if code already complies; otherwise make minimal modifications to satisfy the current spec.

## Stop Condition

If instructions are ambiguous, information is missing, or constraints conflict, the agent must stop and report the issue rather than assume or invent solutions.
