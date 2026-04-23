// spec: ROOT/tech_design/internal/tools/write_file@v33

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
func toolError(message string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}, nil, nil
}

// HandleWriteFile implements the write_file tool handler.
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
	if !isValidNamespace(args.LogicalName) {
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

	// Step 4: Validate that implements is not empty.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", args.LogicalName))
	}

	// Step 5: Normalize the path to forward slashes.
	normalizedPath := filepath.ToSlash(args.Path)

	// Step 6: Validate the path against the working directory.
	// The tool is always executed from the project root, so use the
	// current working directory.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("failed to determine working directory: %s", err.Error()))
	}
	if valErr := pathvalidation.ValidatePath(normalizedPath, wd); valErr != nil {
		return toolError(fmt.Sprintf(
			"%s. allowed paths: %s",
			valErr.Error(),
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 7: Check that the normalized path appears in the implements list
	// (exact string match).
	if !containsPath(fm.Implements, normalizedPath) {
		return toolError(fmt.Sprintf(
			"path not allowed: %s. allowed paths: %s",
			normalizedPath,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 8: Create any missing intermediate directories for the target path.
	targetPath := filepath.FromSlash(normalizedPath)
	dir := filepath.Dir(targetPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return toolError(fmt.Sprintf("failed to create directories for %s: %s", normalizedPath, err.Error()))
		}
	}

	// Step 9: Write the content to the file, overwriting if it exists.
	if err := os.WriteFile(targetPath, []byte(args.Content), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %s", normalizedPath, err.Error()))
	}

	// Step 10: Return success.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "wrote " + normalizedPath}},
	}, nil, nil
}

// isValidNamespace checks whether the logical name belongs to the ROOT
// or TEST namespace.
func isValidNamespace(name string) bool {
	if name == "ROOT" || name == "TEST" {
		return true
	}
	if strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/") {
		return true
	}
	return false
}

// containsPath checks whether the given path appears in the list of
// allowed paths (exact string match).
func containsPath(implements []string, path string) bool {
	for _, p := range implements {
		if p == path {
			return true
		}
	}
	return false
}
