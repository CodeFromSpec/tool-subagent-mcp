// spec: ROOT/tech_design/internal/modes/codegen/tools/write_file@v26

package codegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

// handleWriteFile is the write_file tool handler.
func handleWriteFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1 — ensure load_context has been called.
	if currentTarget == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "load_context must be called before write_file",
			}},
			IsError: true,
		}, nil, nil
	}

	allowedPaths := strings.Join(currentTarget.Frontmatter.Implements, ", ")

	// Step 2 — path traversal / containment validation.
	cwd, err := os.Getwd()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("failed to determine working directory: %v", err),
			}},
			IsError: true,
		}, nil, nil
	}

	if err := pathvalidation.ValidatePath(args.Path, cwd); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("path validation failed: %v. allowed paths: %s", err, allowedPaths),
			}},
			IsError: true,
		}, nil, nil
	}

	// Step 3 — implements allow-list check.
	if !containsPath(currentTarget.Frontmatter.Implements, args.Path) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("path not allowed: %s. allowed paths: %s", args.Path, allowedPaths),
			}},
			IsError: true,
		}, nil, nil
	}

	// Step 4 — create any missing intermediate directories.
	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("failed to create directories for %s: %v", args.Path, err),
			}},
			IsError: true,
		}, nil, nil
	}

	// Step 5 — write the content, overwriting any existing file.
	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("failed to write %s: %v", args.Path, err),
			}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "wrote " + args.Path}},
	}, nil, nil
}

// containsPath reports whether s contains an exact match for target.
func containsPath(s []string, target string) bool {
	for _, p := range s {
		if p == target {
			return true
		}
	}
	return false
}
