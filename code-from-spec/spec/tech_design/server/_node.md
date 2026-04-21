---
version: 9
parent_version: 6
depends_on:
  - path: ROOT/domain/modes
    version: 12
  - path: ROOT/tech_design/modes
    version: 8
implements:
  - cmd/subagent-mcp/main.go
---

# ROOT/tech_design/server

## Intent

Entry point: reads the mode argument, dispatches to the
corresponding mode handler, and exits with the appropriate
code.

## Contracts

### Startup sequence

1. Read `os.Args[1]` as the mode name. If absent or empty,
   print a usage message to stderr and exit 1.
2. If `os.Args[1]` is `--help`, `-h`, or `help`, print the usage
   message to stdout and exit 0.
3. Look up the mode by name. If unrecognized, print a usage
   message listing valid modes and exit 1.
4. Call `mode.Run(os.Args[2:])`.
5. If `Run` returns an error, print it to stderr and exit 1.
6. Otherwise exit 0.

### Usage message

```
Usage: subagent-mcp <mode> [args...]

Modes:
  codegen <leaf-logical-name>   Generate code for a spec leaf node.
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Clean shutdown. |
| 1 | Startup error or mode error. |
