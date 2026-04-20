---
version: 4
parent_version: 2
depends_on:
  - path: ROOT/domain/operations
    version: 8
  - path: ROOT/tech_design/operations
    version: 2
implements:
  - cmd/subagent-mcp/main.go
---

# ROOT/tech_design/server

## Intent

Entry point: reads the operation argument, dispatches to the
corresponding operation handler, and exits with the appropriate
code.

## Contracts

### Startup sequence

1. Read `os.Args[1]` as the operation name. If absent or empty,
   print a usage message to stderr and exit 1.
2. Look up the operation by name. If unrecognized, print a usage
   message listing valid operations and exit 1.
3. Call `operation.Run(os.Args[2:])`.
4. If `Run` returns an error, print it to stderr and exit 1.
5. Otherwise exit 0.

### Usage message

```
Usage: subagent-mcp <operation> [args...]

Operations:
  codegen <leaf-logical-name>   Generate code for a spec leaf node.
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Clean shutdown. |
| 1 | Startup error or operation error. |
