---
name: staleness-resolution
description: Resolves spec and test staleness in a Code from Spec project. Use when the staleness-check tool reports spec_staleness or test_staleness items, or when the user asks to resolve staleness.
---

# Staleness Resolution

Resolve all `spec_staleness` and `test_staleness` items reported by the
staleness-check tool, one at a time, until both sections are empty.

## When invoked

Run this skill when the user asks to resolve spec/test staleness, or
invokes `/staleness-resolution`.

## Bootstrap check

Before starting, verify the staleness-check binary exists:
- Windows: `tools/staleness-check.exe`
- Linux/macOS: `tools/staleness-check`

If it does not exist, follow the bootstrap instructions in `AGENTS.md`.

## Algorithm

Repeat the following loop until `spec_staleness` is empty, then repeat
until `test_staleness` is empty:

1. Run the staleness-check tool and read its output.
2. If the target section (`spec_staleness` first, then `test_staleness`)
   is empty, stop.
3. Take the **first** item in the list.
4. Read the node file for that item.
5. For each reported status, resolve it:

   **`parent_changed`**
   - Determine the parent's file path from the node's logical name.
     - Spec nodes: strip the last segment — `ROOT/x/y` → `ROOT/x` →
       `code-from-spec/x/_node.md`.
     - Test nodes: the subject is the `_node.md` in the same directory.
   - Read the parent/subject file and get its current `version`.
   - Update the node's `parent_version` to that value.

   **`dependency_changed`**
   - For each `depends_on` entry whose version does not match the
     dependency's current `version`:
     - Resolve the dependency's file path from its logical name
       (strip any subsection qualifier in parentheses before resolving).
     - Read the dependency file and get its current `version`.
     - Update the `depends_on` entry's `version` to that value.

6. Review the node's content against updated parents and dependencies.
   If any contract described in the node is now inconsistent with the
   current specs it depends on, update the content accordingly. If
   everything is still consistent, no content changes are needed.

7. Increment the node's own `version` by 1.

8. Save the file.

9. Run the staleness-check tool again and go to step 2.

## Rules

- Never batch multiple nodes in one pass — fix one, verify, repeat.
- Do not modify `code_staleness` items — those are regeneration tasks,
  not staleness resolution.
- Never edit generated source files (those listed under `implements`).
- Path resolution for logical names:
  - `ROOT/x/y` → `code-from-spec/x/y/_node.md`
  - `ROOT/x/y(qualifier)` → same as `ROOT/x/y` (strip qualifier)
  - `TEST/x/y` → `code-from-spec/x/y/default.test.md`
  - `TEST/x/y(name)` → `code-from-spec/x/y/name.test.md`
