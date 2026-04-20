---
version: 1
parent_version: 1
depends_on:
  - path: ROOT/domain/operations/codegen/tools/load_context
    version: 1
  - path: ROOT/tech_design/operations/codegen/chain_resolver
    version: 1
implements:
  - cmd/subagent-mcp/operations/codegen/load_context.go
---

# ROOT/tech_design/operations/codegen/tools/load_context

## Intent

Implements the `load_context` tool handler: resolves the chain,
reads every file, and returns all content concatenated in a single
MCP text response.

## Contracts

### Tool registration

Name: `load_context`
Description: `"Load the full specification context for the current code generation task. Returns all relevant spec files concatenated in a single response. Call this once at the start of your task."`
No input parameters.

### Handler algorithm

1. Call `ResolveChain(session.LeafLogicalName)` to get the ordered
   `[]ChainEntry`.
2. For each entry, read the file at `entry.FilePath`. If any read
   fails, return a tool error identifying the file.
3. Concatenate the file contents using the separator format defined
   below.
4. Return the concatenated string as `mcp.NewToolResultText(result)`.

### Separator format

Each file in the chain is preceded by a boundary block. The boundary
string is `<<<CFS-BOUNDARY>>>` (fixed, not randomized).

The format for each file section:

```
<<<CFS-BOUNDARY>>>
node: <logical-name>
file: <file-path>

<file content>
```

The final file section is followed by a closing boundary line:

```
<<<CFS-BOUNDARY>>>
```

Full example with two files:

```
<<<CFS-BOUNDARY>>>
node: ROOT
file: code-from-spec/spec/_node.md

<content of ROOT/_node.md>
<<<CFS-BOUNDARY>>>
node: ROOT/payments/fees/calculation
file: code-from-spec/spec/payments/fees/calculation/_node.md

<content of leaf _node.md>
<<<CFS-BOUNDARY>>>
```

### Error handling

- `ResolveChain` error → tool error: `"failed to resolve chain: <err>"`.
- File read error → tool error: `"failed to read <file-path>: <err>"`.
