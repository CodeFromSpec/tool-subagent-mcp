---
version: 4
parent_version: 9
---

# ROOT/tech_design/internal

## Intent

Internal packages for the tool. Contains shared utilities and
mode implementations.

## Constraints

All leaf nodes under this subtree generate code inside the
`internal/` directory. The Go package path for each leaf mirrors
its position in the spec tree relative to this node.
