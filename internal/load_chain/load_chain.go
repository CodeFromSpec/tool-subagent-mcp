// Package load_chain implements the load_chain MCP tool handler.
// It validates a logical name, resolves its spec chain, and returns
// the concatenated chain content as a single MCP text response.
//
// spec: ROOT/tech_design/internal/tools/load_chain@v34
package load_chain

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoadChainArgs defines the input parameters for the load_chain tool.
type LoadChainArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

// HandleLoadChain is the MCP tool handler for load_chain. It validates
// the logical name, resolves the spec chain, reads all files, and
// returns the concatenated content using heredoc-style delimiters.
func HandleLoadChain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args LoadChainArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST exactly).
	if !isValidTargetPrefix(args.LogicalName) {
		return toolError("target must be a ROOT/ or TEST/ logical name: " + args.LogicalName), nil, nil
	}

	// Step 2: Resolve the logical name to a file path.
	targetPath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError("invalid logical name: " + args.LogicalName), nil, nil
	}

	// Step 3: Parse frontmatter from the target node.
	fm, err := frontmatter.ParseFrontmatter(targetPath)
	if err != nil {
		return toolError(fmt.Sprintf("error parsing frontmatter for %s: %v", args.LogicalName, err)), nil, nil
	}

	// Step 4a: Validate that implements is not empty.
	if len(fm.Implements) == 0 {
		return toolError("node " + args.LogicalName + " has no implements"), nil, nil
	}

	// Step 4b: Validate each implements path against the working directory.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("cannot determine working directory: %v", err)), nil, nil
	}
	for _, implPath := range fm.Implements {
		if validErr := pathvalidation.ValidatePath(implPath, wd); validErr != nil {
			return toolError(fmt.Sprintf("invalid implements path %q: %v", implPath, validErr)), nil, nil
		}
	}

	// Step 5: Generate a UUID and resolve the full chain.
	id := uuid.New()
	delimiter := id.String()

	chain, err := chainresolver.ResolveChain(args.LogicalName)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain for %s: %v", args.LogicalName, err)), nil, nil
	}

	// Build the concatenated chain content by reading all files.
	var builder strings.Builder

	// Helper to append a single chain item's files to the output.
	appendItem := func(item chainresolver.ChainItem) error {
		for _, filePath := range item.FilePaths {
			data, readErr := os.ReadFile(filePath)
			if readErr != nil {
				return fmt.Errorf("cannot read chain file %s: %v", filePath, readErr)
			}

			// Write opening delimiter with node and path headers.
			fmt.Fprintf(&builder, "<<<FILE_%s>>>\n", delimiter)
			fmt.Fprintf(&builder, "node: %s\n", item.LogicalName)
			fmt.Fprintf(&builder, "path: %s\n", filePath)
			builder.WriteString("\n")
			builder.Write(data)
			// Ensure content ends with a newline before the closing delimiter.
			if len(data) > 0 && data[len(data)-1] != '\n' {
				builder.WriteString("\n")
			}
			fmt.Fprintf(&builder, "<<<END_FILE_%s>>>", delimiter)
			builder.WriteString("\n")
		}
		return nil
	}

	// Append ancestors first.
	for _, ancestor := range chain.Ancestors {
		if err := appendItem(ancestor); err != nil {
			return toolError(err.Error()), nil, nil
		}
	}

	// Append target.
	if err := appendItem(chain.Target); err != nil {
		return toolError(err.Error()), nil, nil
	}

	// Append dependencies.
	for _, dep := range chain.Dependencies {
		if err := appendItem(dep); err != nil {
			return toolError(err.Error()), nil, nil
		}
	}

	// Step 6: Return the chain content as a success result.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: builder.String()}},
	}, nil, nil
}

// isValidTargetPrefix checks that the logical name is a ROOT or TEST
// node (either exactly "ROOT"/"TEST" or prefixed with "ROOT/"/"TEST/").
func isValidTargetPrefix(name string) bool {
	if name == "ROOT" || name == "TEST" {
		return true
	}
	if strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/") {
		return true
	}
	return false
}

// toolError creates an MCP tool error result with the given message.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
