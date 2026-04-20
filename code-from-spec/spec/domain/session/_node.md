---
version: 1
parent_version: 1
---

# ROOT/domain/session

## Intent

Defines what constitutes a session: the unit of isolation that ties
one server process to one target node (a leaf spec node or a test
node).

## Contracts

### Session configuration

A session is defined by two implicit inputs available at startup:

- **First CLI argument** (`os.Args[1]`) — the logical name of the
  target node to generate. May be a leaf spec node (`ROOT/...`) or
  a test node (`TEST/...`). Example: `ROOT/payments/fees/calculation`
  or `TEST/payments/fees/calculation`. This is set by the
  orchestrator; the subagent has no access to it.
- **Working directory** — the project root. The server is always
  launched from the project root, so all relative paths resolve
  correctly against the spec tree and the filesystem.

### Lifecycle

One server process = one session = one target node. The server
starts, serves tool calls for a single subagent invocation, and
exits when the subagent is done. There is no session state beyond
what is loaded at startup.

### Isolation

Multiple sessions may run concurrently. Each is an independent OS
process with its own target node. They share no in-process state.

## Constraints

- If the CLI argument is absent or empty at startup, the server must
  report a clear error and exit. A session without a node is invalid.
- The argument must be a valid `ROOT/` or `TEST/` logical name.
  `EXTERNAL/` logical names are not valid session targets.
