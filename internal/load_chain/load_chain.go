// spec: ROOT/tech_design/internal/tools/load_chain@v49
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

// HandleLoadChain is the MCP tool handler for load_chain. It validates the
// logical name, loads the spec chain, and returns the concatenated chain
// content as a single MCP text response.
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
	_, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError("invalid logical name: " + args.LogicalName), nil, nil
	}

	// Step 3: Parse frontmatter from the resolved path.
	nodePath, _ := logicalnames.PathFromLogicalName(args.LogicalName)
	fm, err := frontmatter.ParseFrontmatter(nodePath)
	if err != nil {
		return toolError(fmt.Sprintf("error parsing frontmatter for %s: %v", args.LogicalName, err)), nil, nil
	}

	// Step 4a: Implements must not be empty.
	if len(fm.Implements) == 0 {
		return toolError("node " + args.LogicalName + " has no implements"), nil, nil
	}

	// Step 4b: Validate each implements path against the working directory.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("error getting working directory: %v", err)), nil, nil
	}
	for _, implPath := range fm.Implements {
		if validateErr := pathvalidation.ValidatePath(implPath, wd); validateErr != nil {
			return toolError(fmt.Sprintf("invalid implements path %q: %v", implPath, validateErr)), nil, nil
		}
	}

	// Step 5: Generate a UUID for heredoc delimiters.
	delimID := uuid.New().String()

	// Step 6: Resolve the chain and build the output.
	chain, err := chainresolver.ResolveChain(args.LogicalName)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain for %s: %v", args.LogicalName, err)), nil, nil
	}

	var buf strings.Builder

	// Append ancestors — strip frontmatter from content.
	for _, item := range chain.Ancestors {
		for _, fp := range item.FilePaths {
			content, readErr := os.ReadFile(fp)
			if readErr != nil {
				return toolError(fmt.Sprintf("error reading file %s: %v", fp, readErr)), nil, nil
			}
			stripped := stripFrontmatter(string(content))
			writeSection(&buf, delimID, item.LogicalName, fp, stripped)
		}
	}

	// Append target — do NOT strip frontmatter (subagent needs it).
	for _, fp := range chain.Target.FilePaths {
		content, readErr := os.ReadFile(fp)
		if readErr != nil {
			return toolError(fmt.Sprintf("error reading file %s: %v", fp, readErr)), nil, nil
		}
		writeSection(&buf, delimID, chain.Target.LogicalName, fp, string(content))
	}

	// Append dependencies — strip frontmatter from content.
	for _, item := range chain.Dependencies {
		for _, fp := range item.FilePaths {
			content, readErr := os.ReadFile(fp)
			if readErr != nil {
				return toolError(fmt.Sprintf("error reading file %s: %v", fp, readErr)), nil, nil
			}
			stripped := stripFrontmatter(string(content))
			writeSection(&buf, delimID, item.LogicalName, fp, stripped)
		}
	}

	// Append code files — no node header, no frontmatter stripping.
	for _, fp := range chain.Code {
		content, readErr := os.ReadFile(fp)
		if readErr != nil {
			return toolError(fmt.Sprintf("error reading file %s: %v", fp, readErr)), nil, nil
		}
		writeCodeSection(&buf, delimID, fp, string(content))
	}

	// Step 7: Return the chain content as a success result.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: buf.String()}},
	}, nil, nil
}

// isValidTargetPrefix checks that the logical name starts with ROOT/ or TEST/,
// or equals ROOT or TEST exactly.
func isValidTargetPrefix(name string) bool {
	return name == "ROOT" || name == "TEST" ||
		strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/")
}

// stripFrontmatter removes the YAML frontmatter block (between the first and
// second "---" lines) from a file's content. If no frontmatter is found, the
// content is returned unchanged.
func stripFrontmatter(content string) string {
	lines := strings.SplitAfter(content, "\n")
	// Find first "---" line.
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		i++
	}
	if i >= len(lines) {
		// No opening delimiter found — return as-is.
		return content
	}
	// Find second "---" line.
	j := i + 1
	for j < len(lines) {
		if strings.TrimSpace(lines[j]) == "---" {
			break
		}
		j++
	}
	if j >= len(lines) {
		// No closing delimiter found — return as-is.
		return content
	}
	// Return everything after the closing "---" line.
	// Preserve any content before the opening "---" as well.
	before := strings.Join(lines[:i], "")
	after := strings.Join(lines[j+1:], "")
	return before + after
}

// writeSection writes a spec/dependency file section with both node: and path: headers.
func writeSection(buf *strings.Builder, delimID, logicalName, filePath, content string) {
	fmt.Fprintf(buf, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(buf, "node: %s\n", logicalName)
	fmt.Fprintf(buf, "path: %s\n", filePath)
	fmt.Fprintf(buf, "\n%s\n", content)
	fmt.Fprintf(buf, "<<<END_FILE_%s>>>\n", delimID)
}

// writeCodeSection writes a code file section with only a path: header (no node:).
func writeCodeSection(buf *strings.Builder, delimID, filePath, content string) {
	fmt.Fprintf(buf, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(buf, "path: %s\n", filePath)
	fmt.Fprintf(buf, "\n%s\n", content)
	fmt.Fprintf(buf, "<<<END_FILE_%s>>>\n", delimID)
}

// toolError creates an MCP tool error result with the given message.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
