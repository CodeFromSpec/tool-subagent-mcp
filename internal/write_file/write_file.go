// code-from-spec: ROOT/tech_design/internal/tools/write_file@v34

package write_file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/pathvalidation"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WriteFileArgs defines the input parameters for the write_file tool.
// Each field maps directly to a required input parameter declared in the
// tool definition.
type WriteFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	Content     string `json:"content" jsonschema:"Complete file content to write."`
}

// toolError returns an MCP tool error result with the given message.
// The returned Go error is nil because all expected error conditions are
// communicated via IsError on the result, keeping the server running.
func toolError(message string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}, nil, nil
}

// HandleWriteFile implements the write_file tool handler.
//
// It resolves the node's frontmatter from the provided logical name,
// validates the target path against the implements list and the project
// root, then writes the file to disk.
//
// Algorithm:
//  1. Validate namespace (ROOT or TEST).
//  2. Resolve logical name to a spec file path.
//  3. Parse frontmatter from the resolved spec file.
//  4. Confirm implements is non-empty.
//  5. Normalize the supplied path to forward slashes.
//  6. Validate the normalized path against the working directory.
//  7. Confirm the normalized path appears in implements (exact match).
//  8. Create missing intermediate directories.
//  9. Write the content to disk.
//  10. Return success.
func HandleWriteFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name belongs to the ROOT or TEST
	// namespace. EXTERNAL nodes do not have implements lists.
	if !isValidNamespace(args.LogicalName) {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName))
	}

	// Step 2: Resolve the logical name to a file path on disk.
	specPath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName))
	}

	// Step 3: Parse the frontmatter from the resolved spec file.
	fm, err := frontmatter.ParseFrontmatter(specPath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse frontmatter for %s: %s", args.LogicalName, err.Error()))
	}

	// Step 4: Validate that the node declares at least one output file.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", args.LogicalName))
	}

	// Step 5: Normalize the caller-supplied path to forward slashes.
	// This ensures consistent comparison regardless of OS conventions.
	normalizedPath := filepath.ToSlash(args.Path)

	// Step 6: Validate the path against the project root (working directory).
	// The tool is always executed from the project root per ROOT/tech_design.
	// ValidatePath rejects traversal attempts, absolute paths, and symlink
	// escapes — this is the security boundary for file writes.
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

	// Step 7: Confirm the normalized path appears in the node's implements
	// list using an exact string match. This is the authoritative check —
	// only declared output files may be written.
	if !containsPath(fm.Implements, normalizedPath) {
		return toolError(fmt.Sprintf(
			"path not allowed: %s. allowed paths: %s",
			normalizedPath,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 8: Create any missing intermediate directories for the target path.
	// filepath.FromSlash converts back to OS-native separators for disk ops.
	targetPath := filepath.FromSlash(normalizedPath)
	dir := filepath.Dir(targetPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return toolError(fmt.Sprintf("failed to create directories for %s: %s", normalizedPath, err.Error()))
		}
	}

	// Step 9: Write the content to the file, overwriting any existing content.
	if err := os.WriteFile(targetPath, []byte(args.Content), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %s", normalizedPath, err.Error()))
	}

	// Step 10: Return a success result identifying the written path.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "wrote " + normalizedPath}},
	}, nil, nil
}

// isValidNamespace reports whether name belongs to the ROOT or TEST
// namespace. EXTERNAL nodes are excluded because they do not have
// implements lists.
func isValidNamespace(name string) bool {
	if name == "ROOT" || name == "TEST" {
		return true
	}
	return strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/")
}

// containsPath reports whether path appears in the implements slice
// using an exact string match.
func containsPath(implements []string, path string) bool {
	for _, p := range implements {
		if p == path {
			return true
		}
	}
	return false
}
