---
name: staleness-resolution
description: Resolves spec and test staleness in a Code from Spec project. Use when the staleness-check tool reports spec_staleness or test_staleness items, or when the user asks to resolve staleness.
---

# Staleness Resolution

Dispatch a subagent to resolve all `spec_staleness` and
`test_staleness` items reported by the staleness-check tool.

## When invoked

Run this skill when the user asks to resolve spec/test staleness, or
invokes `/staleness-resolution`.

## Prerequisites

Verify the staleness-check binary exists (`tools/staleness-check.exe`
on Windows, `tools/staleness-check` elsewhere). If not found, tell
the user it is missing and stop.

## Algorithm

1. Run the staleness-check tool to confirm there are items to resolve.
2. If both `spec_staleness` and `test_staleness` are empty, report
   that everything is up to date and stop.
3. Dispatch a `code-from-spec-staleness-resolution` subagent with
   the following prompt:

   > Resolve all spec and test staleness in this project.
   >
   > Resolve `spec_staleness` first (all items), then
   > `test_staleness` (all items).
   >
   > When done, run the staleness-check tool one final time and
   > report the result.

4. After the subagent completes, run the staleness-check tool to
   verify. Report the remaining `spec_staleness` and `test_staleness`
   items (if any) to the user.

## Rules

- Dispatch exactly one subagent — it handles the full loop internally.
- Do not modify `code_staleness` items — those are regeneration tasks.
- Never edit generated source files.
- If the subagent reports ambiguity or issues requiring human judgment,
  surface them to the user.
