// spec: ROOT/tech_design/internal/modes/codegen/setup@v11

// Package codegen implements the codegen mode for the subagent-mcp server.
//
// The codegen mode:
//   - Accepts a single logical-name argument identifying the target spec node.
//   - Validates the logical name and resolves it to a file path.
//   - Parses the node's frontmatter to discover the implements list.
//   - Pre-loads the full spec chain into memory so load_context can return
//     everything in one call.
//   - Registers two MCP tools on the server: load_context and write_file.
package codegen

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

// Instructions is the server instructions string passed to mcp.ServerOptions.Instructions
// when creating the MCP server. It tells the subagent how to use the two exposed tools
// in the correct order. (ROOT/tech_design/internal/modes/codegen §Server instructions)
const Instructions = `How to use this MCP server:

1. Call load_context once to receive the context for code
   generation. Multiple calls are wasteful as it always
   returns the same content.
2. Generate the code.
3. Call write_file once per file to write the result.`

// Setup validates arguments, pre-loads the full spec chain, and registers
// the load_context and write_file tools on the provided MCP server.
//
// Steps (from spec ROOT/tech_design/internal/modes/codegen/setup, §Contracts):
//  1. Validate args length == 1.
//  2. Validate logical name prefix is ROOT/ or TEST/ (or ROOT / TEST exactly).
//  3. Resolve logical name to a file path via PathFromLogicalName.
//  4. Parse frontmatter from the resolved path.
//  5. Validate that Implements is non-empty and each path passes ValidatePath.
//  6. Generate a UUID, resolve and pre-load the full chain into ChainContent.
//  7. Build a Target and register the two tools.
func Setup(s *mcp.Server, args []string) error {
	// Step 1 — Validate argument count.
	if len(args) != 1 {
		return fmt.Errorf("usage: subagent-mcp codegen <logical-name>")
	}

	logicalName := args[0]

	// Step 2 — Validate that the logical name targets ROOT or TEST namespace.
	// Other prefixes (e.g., EXTERNAL) are not valid codegen targets.
	if !isValidCodegenTarget(logicalName) {
		return fmt.Errorf(
			"codegen target must be a ROOT/ or TEST/ logical name: %s",
			logicalName,
		)
	}

	// Step 3 — Resolve the logical name to a file path.
	filePath, ok := logicalnames.PathFromLogicalName(logicalName)
	if !ok {
		return fmt.Errorf("invalid logical name: %s", logicalName)
	}

	// Step 4 — Parse frontmatter from the resolved spec file.
	fm, err := frontmatter.ParseFrontmatter(filePath)
	if err != nil {
		return err
	}

	// Step 5a — Implements must not be empty; a codegen node must declare
	// at least one output file.
	if len(fm.Implements) == 0 {
		return fmt.Errorf("node %s has no implements", logicalName)
	}

	// Step 5b — Validate each implements path against the project root.
	// The working directory is the project root (see ROOT/tech_design).
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	for _, implPath := range fm.Implements {
		if err := pathvalidation.ValidatePath(implPath, projectRoot); err != nil {
			return err
		}
	}

	// Step 6 — Generate a UUID and build the chain content.
	// The UUID is used as a collision-resistant delimiter throughout the chain
	// so that file delimiters cannot be confused with actual file content.
	uuid, err := generateUUID()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Resolve the ordered list of chain items (ancestors + target + dependencies).
	chain, err := chainresolver.ResolveChain(logicalName)
	if err != nil {
		return fmt.Errorf("failed to resolve chain: %w", err)
	}

	// Read every file in the chain and build the concatenated content string.
	// Format is defined in ROOT/tech_design/internal/modes/codegen §Chain output format.
	chainContent, err := buildChainContent(chain, uuid)
	if err != nil {
		return err
	}

	// Step 7 — Build the Target struct and register the tools.
	target := Target{
		LogicalName:  logicalName,
		FilePath:     filePath,
		Frontmatter:  fm,
		ChainContent: chainContent,
	}

	// Register load_context — returns the pre-built chain in a single call.
	mcp.AddTool(s, &mcp.Tool{
		Name:        "load_context",
		Description: "Load the context for code generation. Returns all relevant spec files concatenated in a single response.",
	}, handleLoadContext(&target))

	// Register write_file — writes a generated file to an implements-declared path.
	mcp.AddTool(s, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the current node's implements list. Overwrites existing content.",
	}, handleWriteFile(&target))

	return nil
}

// isValidCodegenTarget returns true when the logical name belongs to the
// ROOT or TEST namespace (the only valid codegen targets per spec step 2).
func isValidCodegenTarget(logicalName string) bool {
	return logicalName == "ROOT" ||
		logicalName == "TEST" ||
		strings.HasPrefix(logicalName, "ROOT/") ||
		strings.HasPrefix(logicalName, "TEST/")
}

// buildChainContent iterates over all chain items and their file paths,
// reads each file, and assembles the heredoc-delimited output string.
//
// Chain output format (ROOT/tech_design/internal/modes/codegen §Chain output format):
//
//	<<<FILE_<uuid>>>>
//	node: <logical-name>
//	path: <file-path>
//
//	<file content>
//	<<<END_FILE_<uuid>>>>
func buildChainContent(chain *chainresolver.Chain, uuid string) (string, error) {
	var sb strings.Builder

	// Helper: append one file section to the builder.
	writeSection := func(logicalName, filePath string) error {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		// Opening delimiter with node and path headers.
		fmt.Fprintf(&sb, "<<<FILE_%s>>>\n", uuid)
		fmt.Fprintf(&sb, "node: %s\n", logicalName)
		fmt.Fprintf(&sb, "path: %s\n", filePath)
		sb.WriteString("\n")
		sb.Write(content)
		// Ensure the closing delimiter is on its own line.
		if len(content) > 0 && content[len(content)-1] != '\n' {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "<<<END_FILE_%s>>>\n", uuid)
		sb.WriteString("\n")
		return nil
	}

	// Write ancestors first (sorted alphabetically by ResolveChain).
	for _, item := range chain.Ancestors {
		for _, fp := range item.FilePaths {
			if err := writeSection(item.LogicalName, fp); err != nil {
				return "", err
			}
		}
	}

	// Write the target node.
	for _, fp := range chain.Target.FilePaths {
		if err := writeSection(chain.Target.LogicalName, fp); err != nil {
			return "", err
		}
	}

	// Write dependencies (sorted alphabetically by ResolveChain).
	for _, item := range chain.Dependencies {
		for _, fp := range item.FilePaths {
			if err := writeSection(item.LogicalName, fp); err != nil {
				return "", err
			}
		}
	}

	return sb.String(), nil
}

// generateUUID generates a random UUID v4 (RFC 4122) using the crypto/rand
// package. The UUID is used once per Setup call to produce unique delimiters
// that will not collide with actual file content.
func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// Set version 4 bits (0100xxxx in the 7th byte).
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits (10xxxxxx in the 9th byte).
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	), nil
}
