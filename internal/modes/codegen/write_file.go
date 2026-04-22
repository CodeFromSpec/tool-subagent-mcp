// spec: ROOT/tech_design/internal/modes/codegen/tools/write_file@v27

package codegen

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

// handleWriteFile is the write_file tool handler.
func handleWriteFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1 — validate logical name prefix.
	if !isValidCodegenTarget(args.LogicalName) {
		return toolError(fmt.Sprintf("codegen target must be a ROOT/ or TEST/ logical name: %s", args.LogicalName)), nil, nil
	}

	// Step 2 — resolve logical name to file path.
	filePath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName)), nil, nil
	}

	// Step 3 — parse frontmatter.
	fm, err := frontmatter.ParseFrontmatter(filePath)
	if err != nil {
		return toolError(fmt.Sprintf("error loading frontmatter: %v", err)), nil, nil
	}

	// Step 4 — validate implements is not empty.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", args.LogicalName)), nil, nil
	}

	allowedPaths := strings.Join(fm.Implements, ", ")

	// Step 5 — path traversal / containment validation.
	cwd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("failed to determine working directory: %v", err)), nil, nil
	}

	if err := pathvalidation.ValidatePath(args.Path, cwd); err != nil {
		return toolError(fmt.Sprintf("path validation failed: %v. allowed paths: %s", err, allowedPaths)), nil, nil
	}

	// Step 6 — implements allow-list check.
	if !containsPath(fm.Implements, args.Path) {
		return toolError(fmt.Sprintf("path not allowed: %s. allowed paths: %s", args.Path, allowedPaths)), nil, nil
	}

	// Step 7 — create any missing intermediate directories.
	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return toolError(fmt.Sprintf("failed to create directories for %s: %v", args.Path, err)), nil, nil
	}

	// Step 8 — write the content, overwriting any existing file.
	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %v", args.Path, err)), nil, nil
	}

	// Step 9 — success.
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
