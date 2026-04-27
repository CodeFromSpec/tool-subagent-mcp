// code-from-spec: ROOT/tech_design/internal/tools/patch_file@v5

// Package patch_file implements the patch_file MCP tool handler.
// It applies a unified diff to an existing file, after validating
// the target path against the node's implements list and the project root.
package patch_file

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/pathvalidation"
)

// PatchFileArgs holds the input parameters for the patch_file tool.
type PatchFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	Diff        string `json:"diff" jsonschema:"Unified diff to apply to the file."`
}

// toolError returns an MCP tool error result with the given message.
// Tool errors are returned as IsError: true so the server keeps running.
func toolError(message string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}, nil, nil
}

// HandlePatchFile is the handler for the patch_file MCP tool.
// It validates the path against the node's implements list and applies
// a unified diff to the target file.
func HandlePatchFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args PatchFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST). This is a basic sanity check before resolving.
	name := args.LogicalName
	if name != "ROOT" && name != "TEST" &&
		!strings.HasPrefix(name, "ROOT/") &&
		!strings.HasPrefix(name, "TEST/") {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 2: Resolve the logical name to a file path.
	nodePath, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 3: Parse the frontmatter from the resolved node file.
	fm, err := frontmatter.ParseFrontmatter(nodePath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to read node %s: %v", name, err))
	}

	// Step 4: Validate that the node has an implements list.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name))
	}

	// Step 5: Normalize the path to forward slashes for consistent comparison.
	// This handles Windows paths where backslashes may be used.
	normalizedPath := filepath.ToSlash(args.Path)

	// Step 6: Validate the path against the working directory to prevent
	// directory traversal attacks and writes outside the project root.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("failed to get working directory: %v", err))
	}

	if err := pathvalidation.ValidatePath(normalizedPath, wd); err != nil {
		return toolError(fmt.Sprintf(
			"path validation failed: %v. allowed paths: %s",
			err,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 7: Check that the normalized path appears in the node's implements list.
	// This is the security boundary — only files declared in implements may be written.
	allowed := false
	for _, impl := range fm.Implements {
		if impl == normalizedPath {
			allowed = true
			break
		}
	}
	if !allowed {
		return toolError(fmt.Sprintf(
			"path not allowed: %s. allowed paths: %s",
			normalizedPath,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 8: Read the existing file content.
	// patch_file does not create new files — the file must already exist.
	existingContent, err := os.ReadFile(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return toolError(fmt.Sprintf("file does not exist: %s", normalizedPath))
		}
		return toolError(fmt.Sprintf("failed to read %s: %v", normalizedPath, err))
	}

	// Step 9: Parse the unified diff provided by the caller.
	// Note: completely malformed input that gitdiff.Parse cannot interpret as a diff
	// will result in zero file entries (no error), which is caught by step 10.
	files, _, err := gitdiff.Parse(strings.NewReader(args.Diff))
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse diff: %v", err))
	}

	// Step 10: The diff must target exactly one file — no more, no less.
	// This also catches malformed diffs that produce zero entries.
	if len(files) != 1 {
		return toolError("diff must contain exactly one file")
	}

	// Step 11: Apply the patch to the existing file content.
	// Note: context mismatches may be caught by gitdiff.Parse (step 9) rather
	// than gitdiff.Apply.
	source := bytes.NewReader(existingContent)
	var output bytes.Buffer
	if err := gitdiff.Apply(&output, source, files[0]); err != nil {
		return toolError(fmt.Sprintf("failed to apply diff to %s: %v", normalizedPath, err))
	}

	// Step 12: Write the patched content back to the file, overwriting the original.
	if err := os.WriteFile(normalizedPath, output.Bytes(), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %v", normalizedPath, err))
	}

	// Step 13: Return success.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("patched %s", normalizedPath)}},
	}, nil, nil
}
