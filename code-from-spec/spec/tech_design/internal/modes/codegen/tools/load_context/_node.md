---
version: 27
parent_version: 17
depends_on:
  - path: EXTERNAL/mcp-go-sdk
    version: 1
  - path: ROOT/domain/modes/codegen
    version: 21
implements:
  - internal/modes/codegen/load_context.go
---

# ROOT/tech_design/internal/modes/codegen/tools/load_context

## Intent

Implements the `load_context` tool handler: returns the
pre-loaded chain content as a single MCP text response.

## Contracts

### Handler

Follows the `handleLoadContext` signature defined in the
parent codegen node. Returns `target.ChainContent` as a
success result. No I/O — the content was loaded during setup.
