---
version: 1
parent_version: 34
depends_on:
  - path: ROOT/domain/modes/codegen
    version: 21
implements:
  - internal/modes/codegen/help_message.go
---

# ROOT/tech_design/internal/modes/codegen/help_message

## Intent

Implements the `HelpMessage` function for the codegen mode.
Returns usage and configuration instructions aimed at the
orchestrator agent.

## Contracts

### Function

```go
func HelpMessage() string
```

Returns the help text defined in the parent codegen node.
