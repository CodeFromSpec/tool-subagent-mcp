---
name: code-generation
description: Generates or regenerates source files from the Code from Spec tree. Use when code_staleness items exist, or when the user asks to generate code, regenerate files, or run code generation.
---

# Code Generation

Generate source files for all stale nodes reported in `code_staleness`
by the staleness-check tool.

## When invoked

Run this skill when the user asks to generate code, regenerate files,
or when `code_staleness` has items.

## Prerequisites

1. Verify the staleness-check binary exists (`tools/staleness-check.exe`
   on Windows, `tools/staleness-check` elsewhere). If not, follow the
   bootstrap instructions in `AGENTS.md`.

2. Verify the subagent-mcp binary exists (`tools/subagent-mcp.exe` on
   Windows, `tools/subagent-mcp` elsewhere). If not, follow the
   bootstrap instructions in `AGENTS.md`.

3. Run the staleness-check tool. If `spec_staleness` or `test_staleness`
   are not empty, stop and tell the user to run `/staleness-resolution`
   first — code generation requires a clean spec tree.

## Algorithm

1. Run the staleness-check tool and collect all `code_staleness` items.
2. If `code_staleness` is empty, report that everything is up to date
   and stop.
3. Group items by `node` — each unique logical name is one generation
   task.
4. For each stale node (in the order reported), dispatch a `codegen`
   subagent with the following prompt:

   > You are a confined code generation subagent.
   > Your only task is to generate the source file(s) for the node
   > `<logical-name>`.
   >
   > Steps:
   > 1. Call `load_chain` with logical_name `<logical-name>` to receive
   >    the complete spec chain.
   > 2. Read the chain carefully. Identify the target node's spec
   >    (its intent, contracts, and interface), the constraints from
   >    ancestor nodes, and any dependency specs.
   > 3. For each file declared in the node's `implements` list, generate
   >    the complete file content. The first line where a comment is
   >    allowed must be the spec comment:
   >    `// code-from-spec: <logical-name>@v<version>`
   >    where `<version>` is the node's current `version` field.
   > 4. Call `write_file` once per file, passing the logical name, the
   >    relative file path, and the complete content.
   > 5. If the spec has gaps or contradictions that prevent generation,
   >    do not guess — report the problem clearly instead of writing a
   >    file.
   >
   > Do not read any file not provided by `load_chain`. Do not call any
   > tool other than `load_chain` and `write_file`.

5. After all subagents complete, run the staleness-check tool again.
   Report the remaining `code_staleness` items (if any) to the user.

## Rules

- Dispatch one subagent per node logical name, not per file.
- Independent nodes may be dispatched in parallel (single message with
  multiple Agent tool calls).
- Never edit generated files manually — always regenerate via a
  subagent.
- If a subagent reports a spec gap, surface it to the user. Do not
  attempt to fill the gap by reading the codebase yourself.
- After generation, do not automatically run build or tests unless the
  user asks — report what was generated and let the user decide.
