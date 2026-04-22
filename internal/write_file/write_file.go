// spec: ROOT/tech_design/internal/tools/write_file@v28

// Package write_file implements the write_file MCP tool handler.
//
// The tool accepts a logical name, a relative file path, and file content.
// It resolves the node's frontmatter from the logical name, validates the
// target path against the node's implements list and the project root, then
// writes the file to disk.
//
// Spec ref: ROOT/tech_design/internal/tools/write_file § "Intent"
package write_file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

// WriteFileArgs holds the input parameters for the write_file tool.
//
// Spec ref: ROOT/tech_design/internal/tools/write_file § "WriteFileArgs type"
type WriteFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path"         jsonschema:"Relative file path from project root."`
	Content     string `json:"content"      jsonschema:"Complete file content to write."`
}

// toolError returns a *mcp.CallToolResult with IsError: true and the given
// message as its text content.
//
// Spec ref: ROOT/tech_design/internal/tools § "Tool result — error"
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

// toolSuccess returns a *mcp.CallToolResult with the given message as its
// text content.
//
// Spec ref: ROOT/tech_design/internal/tools § "Tool result — success"
func toolSuccess(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}
}

// handleWriteFile is the MCP tool handler for write_file.
//
// Algorithm (spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm"):
//  1. Validate that args.LogicalName starts with ROOT/ or TEST/ (or equals ROOT or TEST).
//  2. Call PathFromLogicalName; if it returns false, return a tool error.
//  3. Call ParseFrontmatter on the resolved path; if it fails, return a tool error.
//  4. Validate Implements is not empty.
//  5. Call ValidatePath on args.Path against the working directory.
//  6. Check that args.Path appears in the frontmatter's Implements (exact string match).
//  7. Create any missing intermediate directories for the target path.
//  8. Write args.Content to the file, overwriting if it exists.
//  9. Return a success result with text "wrote <path>".
func handleWriteFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name has an accepted prefix or equals a
	// known root token.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 1
	name := args.LogicalName
	validPrefix := strings.HasPrefix(name, "ROOT/") ||
		strings.HasPrefix(name, "TEST/") ||
		name == "ROOT" ||
		name == "TEST"
	if !validPrefix {
		return toolError(fmt.Sprintf("invalid logical name: %s", name)), nil, nil
	}

	// Step 2: Resolve the logical name to a spec file path.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 2
	specPath, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name)), nil, nil
	}

	// Step 3: Parse frontmatter from the resolved spec file.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 3
	fm, err := frontmatter.ParseFrontmatter(specPath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse frontmatter for %s: %v", name, err)), nil, nil
	}

	// Step 4: Ensure the node declares at least one implements path.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 4
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name)), nil, nil
	}

	// Step 5: Validate the target path against the project root (security boundary).
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 5
	// and ROOT/tech_design/internal/tools § "Path validation"
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("failed to determine working directory: %v", err)), nil, nil
	}

	if err := pathvalidation.ValidatePath(args.Path, wd); err != nil {
		return toolError(fmt.Sprintf(
			"invalid path %q: %v. allowed paths: %s",
			args.Path, err, strings.Join(fm.Implements, ", "),
		)), nil, nil
	}

	// Step 6: Verify that args.Path is in the node's implements list (exact match).
	// This is the security boundary — only declared files may be written.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 6
	// and § "Decisions: write_file validates against implements"
	allowed := false
	for _, imp := range fm.Implements {
		if imp == args.Path {
			allowed = true
			break
		}
	}
	if !allowed {
		return toolError(fmt.Sprintf(
			"path not allowed: %s. allowed paths: %s",
			args.Path, strings.Join(fm.Implements, ", "),
		)), nil, nil
	}

	// Step 7: Create any missing intermediate directories.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 7
	absPath := filepath.Join(wd, args.Path)
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return toolError(fmt.Sprintf(
			"failed to create directories for %s: %v", args.Path, err,
		)), nil, nil
	}

	// Step 8: Write content to the file, overwriting if it already exists.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 8
	if err := os.WriteFile(absPath, []byte(args.Content), 0644); err != nil {
		return toolError(fmt.Sprintf(
			"failed to write %s: %v", args.Path, err,
		)), nil, nil
	}

	// Step 9: Return success.
	// Spec ref: ROOT/tech_design/internal/tools/write_file § "Algorithm" step 9
	return toolSuccess(fmt.Sprintf("wrote %s", args.Path)), nil, nil
}

// Register adds the write_file tool to the given MCP server.
//
// Tool name and description follow the spec exactly.
// Spec ref: ROOT/tech_design/internal/tools/write_file § "Tool definition"
func Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write a generated source file to disk. The path must be one of the files declared in the node's implements list. Overwrites existing content.",
	}, handleWriteFile)
}
