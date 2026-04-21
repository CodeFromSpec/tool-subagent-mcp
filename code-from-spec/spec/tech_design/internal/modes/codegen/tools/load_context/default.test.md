---
version: 3
parent_version: 26
implements:
  - internal/modes/codegen/load_context_test.go
---

# TEST/tech_design/internal/modes/codegen/tools/load_context

## Context

Tests call `handleLoadContext` with a `Target` containing
a known `ChainContent` string and verify the MCP response.

## Happy Path

### Returns pre-loaded chain content

Create a `Target` with `ChainContent` set to a known string.
Call the handler.

Expect: success result with text equal to the `ChainContent`.

### Empty chain content

Create a `Target` with `ChainContent` set to `""`.
Call the handler.

Expect: success result with empty text.
