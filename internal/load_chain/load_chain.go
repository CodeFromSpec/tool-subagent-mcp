// spec: ROOT/tech_design/internal/tools/load_chain@v46

package load_chain

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

// LoadChainArgs defines the input parameters for the load_chain tool.
type LoadChainArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

// toolError returns an MCP tool error result with the given message.
func toolError(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}, nil, nil
}

// toolSuccess returns an MCP tool success result with the given content.
func toolSuccess(content string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}, nil, nil
}

// HandleLoadChain implements the load_chain tool handler.
// It validates the logical name, loads the spec chain, and returns
// the chain content as a single MCP text response.
func HandleLoadChain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args LoadChainArgs,
) (*mcp.CallToolResult, any, error) {
	name := args.LogicalName

	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST exactly).
	if !isValidTargetPrefix(name) {
		return toolError(fmt.Sprintf("target must be a ROOT/ or TEST/ logical name: %s", name))
	}

	// Step 2: Resolve the logical name to a file path.
	targetPath, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 3: Parse frontmatter from the target node.
	fm, err := frontmatter.ParseFrontmatter(targetPath)
	if err != nil {
		return toolError(fmt.Sprintf("error parsing frontmatter for %s: %v", name, err))
	}

	// Step 4a: Implements must not be empty.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name))
	}

	// Step 4b: Validate each implements path against the working directory.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("cannot determine working directory: %v", err))
	}
	for _, p := range fm.Implements {
		if err := pathvalidation.ValidatePath(p, wd); err != nil {
			return toolError(fmt.Sprintf("invalid implements path %q: %v", p, err))
		}
	}

	// Step 5: Generate a UUID for heredoc delimiters.
	delimID := uuid.New().String()

	// Step 6: Resolve the chain and build the output.
	chain, err := chainresolver.ResolveChain(name)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain for %s: %v", name, err))
	}

	var sb strings.Builder

	// Write ancestors — strip frontmatter from content.
	for _, item := range chain.Ancestors {
		for _, fp := range item.FilePaths {
			content, err := os.ReadFile(fp)
			if err != nil {
				return toolError(fmt.Sprintf("error reading file %s: %v", fp, err))
			}
			stripped := stripFrontmatter(string(content))
			writeSection(&sb, delimID, item.LogicalName, fp, stripped)
		}
	}

	// Write the target node — include frontmatter (no stripping).
	for _, fp := range chain.Target.FilePaths {
		content, err := os.ReadFile(fp)
		if err != nil {
			return toolError(fmt.Sprintf("error reading file %s: %v", fp, err))
		}
		writeSection(&sb, delimID, chain.Target.LogicalName, fp, string(content))
	}

	// Write dependencies — strip frontmatter from content.
	for _, item := range chain.Dependencies {
		for _, fp := range item.FilePaths {
			content, err := os.ReadFile(fp)
			if err != nil {
				return toolError(fmt.Sprintf("error reading file %s: %v", fp, err))
			}
			stripped := stripFrontmatter(string(content))
			writeSection(&sb, delimID, item.LogicalName, fp, stripped)
		}
	}

	// Write code files — no node header, no frontmatter stripping.
	for _, fp := range chain.Code {
		content, err := os.ReadFile(fp)
		if err != nil {
			return toolError(fmt.Sprintf("error reading file %s: %v", fp, err))
		}
		writeCodeSection(&sb, delimID, fp, string(content))
	}

	// Step 7: Return the chain content as a success result.
	return toolSuccess(sb.String())
}

// isValidTargetPrefix checks whether the logical name is a valid
// ROOT or TEST target (must be "ROOT", "TEST", or start with
// "ROOT/" or "TEST/").
func isValidTargetPrefix(name string) bool {
	if name == "ROOT" || name == "TEST" {
		return true
	}
	if strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/") {
		return true
	}
	return false
}

// stripFrontmatter removes the YAML frontmatter block (the content
// between the first and second "---" lines) from the text. If no
// frontmatter is found, the original text is returned unchanged.
func stripFrontmatter(text string) string {
	lines := strings.SplitAfter(text, "\n")
	state := 0 // 0 = before first ---, 1 = inside frontmatter
	startBody := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if state == 0 {
				// Found the opening delimiter.
				state = 1
			} else {
				// Found the closing delimiter — body starts after this line.
				startBody = i + 1
				break
			}
		}
	}

	if startBody < 0 || startBody >= len(lines) {
		// No complete frontmatter found; return original text.
		return text
	}

	// Rejoin remaining lines, skipping any leading blank line
	// right after the closing delimiter.
	body := strings.Join(lines[startBody:], "")

	// Trim a single leading newline if present (the blank line
	// separating frontmatter from content).
	body = strings.TrimLeft(body, "\n")

	return body
}

// writeSection writes a spec or dependency file section with both
// node: and path: headers.
func writeSection(sb *strings.Builder, delimID, logicalName, filePath, content string) {
	fmt.Fprintf(sb, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(sb, "node: %s\n", logicalName)
	fmt.Fprintf(sb, "path: %s\n", filePath)
	sb.WriteString("\n")
	sb.WriteString(content)
	sb.WriteString("\n")
	fmt.Fprintf(sb, "<<<END_FILE_%s>>>\n", delimID)
}

// writeCodeSection writes a code file section with only a path: header
// (no node: header).
func writeCodeSection(sb *strings.Builder, delimID, filePath, content string) {
	fmt.Fprintf(sb, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(sb, "path: %s\n", filePath)
	sb.WriteString("\n")
	sb.WriteString(content)
	sb.WriteString("\n")
	fmt.Fprintf(sb, "<<<END_FILE_%s>>>\n", delimID)
}
