---
version: 60
parent_version: 5
depends_on:
  - path: ROOT/external/google-uuid
    version: 3
  - path: ROOT/external/mcp-go-sdk
    version: 3
  - path: ROOT/tech_design/internal/chain_resolver
    version: 73
  - path: ROOT/tech_design/internal/frontmatter
    version: 32
  - path: ROOT/tech_design/internal/logical_names
    version: 29
  - path: ROOT/tech_design/internal/normalizename
    version: 2
  - path: ROOT/tech_design/internal/parsenode
    version: 32
  - path: ROOT/tech_design/internal/pathvalidation
    version: 11
implements:
  - internal/load_chain/load_chain.go
---

# ROOT/tech_design/internal/tools/load_chain

Implements the `load_chain` tool handler: validates the
logical name, loads the spec chain, and returns the chain
content as a single MCP text response.

# Public

## Context

### Package

`package load_chain`

### Target node

The target node is identified by its logical name — either a leaf
spec node (`ROOT/...`) or a test node (`TEST/...`). Examples:
`ROOT/payments/fees/calculation`,
`TEST/payments/fees/calculation`.

## Contracts

### Tool definition

Name: `load_chain`
Description: `"Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response."`

Input parameters:

| Name | Type | Required | Description |
|---|---|---|---|
| `logical_name` | string | yes | Logical name of the node to generate code for. |

### LoadChainArgs type

```go
type LoadChainArgs struct {
    LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}
```

### Handler

```go
func HandleLoadChain(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args LoadChainArgs,
) (*mcp.CallToolResult, any, error)
```

### Chain output format

The chain is serialized as a sequence of file sections using
heredoc-style delimiters with a UUID generated once per call
to avoid collisions with file content.

Opening delimiter: `<<<FILE_<uuid>>>`
Closing delimiter: `<<<END_FILE_<uuid>>>`

The same UUID is used for all files in the chain. Each section
includes `node:` and `path:` headers between the opening
delimiter and the file content, separated by a blank line.
Code files include only `path:`.

```
<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
node: ROOT
path: code-from-spec/spec/_node.md

<Public section body — no # Public heading>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>

<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
node: ROOT/payments/fees/calculation
path: code-from-spec/spec/payments/fees/calculation/_node.md

<target content with reduced frontmatter>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>

<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
node: ROOT/architecture/backend
path: code-from-spec/spec/architecture/backend/_node.md

<Public section body — no # Public heading>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>

<<<FILE_550e8400-e29b-41d4-a716-446655440000>>>
path: internal/payments/fees/calculation.go

<existing source file content>
<<<END_FILE_550e8400-e29b-41d4-a716-446655440000>>>
```

### Algorithm

1. Validate that `args.LogicalName` starts with `ROOT/` or
   `TEST/` (or equals `ROOT` or `TEST`). If not, return a
   tool error: `"target must be a ROOT/ or TEST/
   logical name: <name>"`.
2. Call `logicalnames.PathFromLogicalName`. If it returns false, return a
   tool error: `"invalid logical name: <name>"`.
3. Call `ParseFrontmatter` on the resolved path. If it fails,
   return a tool error wrapping the underlying error.
4. Validate `Implements`:
   a. Must not be empty → tool error: `"node <name> has no
      implements"`.
   b. Call `ValidatePath` for each path against the working
      directory. If any fails, return a tool error.
5. Generate a UUID using `github.com/google/uuid`.
6. Call `ResolveChain` to resolve the full chain. If it fails,
   return a tool error.
7. Build the output by processing each part of the chain:

   **Ancestors** — for each ancestor, call `ParseNode` with
   the ancestor's logical name. Extract the `# Public`
   section. If `Public` is nil, or if `Public` has no
   content and no subsections, skip this ancestor entirely —
   do not emit a file section for it. Otherwise, emit a
   file section whose content is the `# Public` section's
   own body content followed by each subsection reconstructed
   as markdown — **without** the `# Public` heading itself.

   **Target** — read the file and include it with a reduced
   frontmatter containing only `version` and `implements`.
   All other frontmatter fields are stripped.

   **Dependencies** — group the dependency items by
   `FilePath`, preserving first-occurrence order. For each
   group, call `ParseNode` once using the base logical name
   of any item in the group (without qualifier). Use the base
   logical name (qualifier stripped) as the `node:` header
   for the emitted file section.

   Build the group's content as follows:
   - If any item in the group has `Qualifier` = nil, include
     the `# Public` section's body content and subsections —
     **without** the `# Public` heading itself.
   - Otherwise, for each item in the group (in order), find
     the `## <qualifier>` subsection within `# Public` using
     `normalizename.NormalizeName` for comparison. If the
     subsection has no body content (blank after trimming),
     treat it as absent and contribute nothing. Otherwise,
     append the `##` heading followed by the body content.
     Each subsection is appended in sequence.

   If the consolidated content is empty (blank after
   trimming), skip this group entirely — do not emit a file
   section for it. Otherwise, emit a single file section for
   the group containing the consolidated content.

   **Code** — for each code file, read the file and include
   it as-is.

   If any file cannot be read or parsed, return a tool error.

8. Return the chain content as a success result.

### Reduced frontmatter

The target file's frontmatter is reduced to only the fields
needed for code generation:

```yaml
---
version: <version>
implements:
  - <path>
  - <path>
---
```

All other fields (`parent_version`, `subject_version`,
`depends_on`) are stripped to save tokens and avoid
confusing the subagent.

## Constraints

- The target argument must be a logical name that resolves to a
  node with `implements`. Absent, empty, or invalid values cause
  the tool to report an error.
- Reads are limited to the chain.
- If any chain file cannot be read, `load_chain` returns an error
  identifying the missing file; it does not return partial results.

# Decisions

### load_chain returns everything in one call

Loading the chain file-by-file via separate tool calls would
accumulate context in the conversation history, increasing token
cost with each roundtrip. A single call returns the entire chain,
minimizing roundtrip overhead.
