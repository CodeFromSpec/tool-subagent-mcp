// Package write_file implements the write_file MCP tool handler.
// spec: ROOT/tech_design/internal/tools/write_file@v30
package write_file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WriteFileArgs defines the input parameters for the write_file tool.
type WriteFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	Content     string `json:"content" jsonschema:"Complete file content to write."`
}

// toolError returns an MCP tool error result with the given message.
func toolError(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}, nil, nil
}

// HandleWriteFile is the MCP tool handler for write_file.
//
// It resolves the node's frontmatter from the provided logical name,
// validates the target path against the implements list and the project
// root, then writes the file to disk.
func HandleWriteFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST exactly).
	if !isValidPrefix(args.LogicalName) {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName))
	}

	// Step 2: Resolve the logical name to a file path.
	specPath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName))
	}

	// Step 3: Parse the frontmatter from the resolved spec file.
	fm, err := frontmatter.ParseFrontmatter(specPath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse frontmatter for %s: %s", args.LogicalName, err.Error()))
	}

	// Step 4: Validate that the node has an implements list.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", args.LogicalName))
	}

	// Step 5: Validate the path against the working directory using ValidatePath.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("failed to determine working directory: %s", err.Error()))
	}
	if err := pathvalidation.ValidatePath(args.Path, wd); err != nil {
		return toolError(fmt.Sprintf(
			"path validation failed: %s. allowed paths: %s",
			err.Error(),
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 6: Check that the path appears in the implements list (exact match).
	if !containsPath(fm.Implements, args.Path) {
		return toolError(fmt.Sprintf(
			"path not allowed: %s. allowed paths: %s",
			args.Path,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 7: Create any missing intermediate directories.
	dir := filepath.Dir(args.Path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return toolError(fmt.Sprintf("failed to create directories for %s: %s", args.Path, err.Error()))
		}
	}

	// Step 8: Write the content to the file, overwriting if it exists.
	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %s", args.Path, err.Error()))
	}

	// Step 9: Return success.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("wrote %s", args.Path)}},
	}, nil, nil
}

// isValidPrefix checks that a logical name starts with ROOT/ or TEST/,
// or equals ROOT or TEST exactly.
func isValidPrefix(name string) bool {
	if name == "ROOT" || name == "TEST" {
		return true
	}
	if strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/") {
		return true
	}
	return false
}

// containsPath checks whether the given path appears in the list (exact match).
func containsPath(list []string, path string) bool {
	for _, p := range list {
		if p == path {
			return true
		}
	}
	return false
}
