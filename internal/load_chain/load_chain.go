// spec: ROOT/tech_design/internal/tools/load_chain@v31

// Package load_chain implements the load_chain MCP tool handler.
//
// The tool validates a target logical name, loads the spec chain for that
// node, and returns the chain content as a single concatenated MCP text
// response. See spec § "Algorithm" and § "Chain output format".
package load_chain

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	chainresolver "github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	logicalnames "github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

// LoadChainArgs holds the input parameters for the load_chain tool.
// Spec ref: ROOT/tech_design/internal/tools/load_chain § "LoadChainArgs type"
type LoadChainArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

// toolError returns a CallToolResult representing a tool-level error.
// Spec ref: ROOT/tech_design/internal/tools § "Tool result — error"
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

// toolSuccess returns a CallToolResult representing a successful response.
// Spec ref: ROOT/tech_design/internal/tools § "Tool result — success"
func toolSuccess(content string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}
}

// handleLoadChain is the handler for the load_chain tool.
//
// It validates the logical name, resolves the chain, reads each file, and
// returns the full chain content in the heredoc-style format defined in the spec.
//
// Spec ref: ROOT/tech_design/internal/tools/load_chain § "Handler" and § "Algorithm"
func handleLoadChain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args LoadChainArgs,
) (*mcp.CallToolResult, any, error) {
	name := args.LogicalName

	// Step 1 — Validate that the logical name starts with ROOT/ or TEST/ (or equals ROOT or TEST).
	// Spec ref: § "Algorithm" step 1.
	if name != "ROOT" && name != "TEST" &&
		!strings.HasPrefix(name, "ROOT/") &&
		!strings.HasPrefix(name, "TEST/") {
		return toolError(fmt.Sprintf(
			"target must be a ROOT/ or TEST/ logical name: %s", name,
		)), nil, nil
	}

	// Step 2 — Resolve the logical name to a file path.
	// Spec ref: § "Algorithm" step 2.
	_, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name)), nil, nil
	}

	// Step 3 — Parse frontmatter from the resolved path.
	// Spec ref: § "Algorithm" step 3.
	resolvedPath, _ := logicalnames.PathFromLogicalName(name)
	fm, err := frontmatter.ParseFrontmatter(resolvedPath)
	if err != nil {
		return toolError(fmt.Sprintf("error reading frontmatter: %v", err)), nil, nil
	}

	// Step 4 — Validate Implements list.
	// Spec ref: § "Algorithm" step 4.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name)), nil, nil
	}

	// Determine the project root (working directory).
	// Spec ref: ROOT/tech_design § "Project root"
	projectRoot, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("error getting working directory: %v", err)), nil, nil
	}

	// Step 4b — Validate each path in Implements against the working directory.
	// Spec ref: § "Algorithm" step 4b; ROOT/tech_design/internal/tools § "Path validation"
	for _, implPath := range fm.Implements {
		if err := pathvalidation.ValidatePath(implPath, projectRoot); err != nil {
			return toolError(fmt.Sprintf("invalid implements path %q: %v", implPath, err)), nil, nil
		}
	}

	// Step 5 — Generate UUID, resolve chain, build concatenated output.
	// Spec ref: § "Algorithm" step 5; § "Chain output format"
	// Generate a UUID v4 using crypto/rand (stdlib only, no external UUID package).
	var uuidBytes [16]byte
	if _, err := rand.Read(uuidBytes[:]); err != nil {
		return toolError(fmt.Sprintf("error generating UUID: %v", err)), nil, nil
	}
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40 // version 4
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80 // variant 1
	chainUUID := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])

	chain, err := chainresolver.ResolveChain(name)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain: %v", err)), nil, nil
	}

	// Collect all chain items in order: ancestors, target, dependencies.
	// Spec ref: § "Chain output format"
	var allItems []chainresolver.ChainItem
	allItems = append(allItems, chain.Ancestors...)
	allItems = append(allItems, chain.Target)
	allItems = append(allItems, chain.Dependencies...)

	var sb strings.Builder

	for i, item := range allItems {
		for _, filePath := range item.FilePaths {
			// Read the chain file content.
			// Spec ref: § "Constraints" — partial results are not returned on read failure.
			data, err := os.ReadFile(filePath)
			if err != nil {
				return toolError(fmt.Sprintf(
					"error reading chain file %q: %v", filePath, err,
				)), nil, nil
			}

			// Write the heredoc-style section for this file.
			// Spec ref: § "Chain output format"
			if i > 0 || filePath != item.FilePaths[0] {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("<<<FILE_%s>>>\n", chainUUID))
			sb.WriteString(fmt.Sprintf("node: %s\n", item.LogicalName))
			sb.WriteString(fmt.Sprintf("path: %s\n", filePath))
			sb.WriteString("\n")
			sb.Write(data)
			sb.WriteString(fmt.Sprintf("<<<END_FILE_%s>>>", chainUUID))
		}
	}

	// Step 6 — Return the chain content as a success result.
	// Spec ref: § "Algorithm" step 6
	return toolSuccess(sb.String()), nil, nil
}

// RegisterTool registers the load_chain tool on the given MCP server.
//
// Tool name and description match the spec exactly.
// Spec ref: ROOT/tech_design/internal/tools/load_chain § "Tool definition"
func RegisterTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "load_chain",
		Description: "Load the spec chain context for a given logical name. Returns all relevant spec files concatenated in a single response.",
	}, handleLoadChain)
}
