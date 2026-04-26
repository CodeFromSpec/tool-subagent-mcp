// code-from-spec: ROOT/tech_design/internal/tools/patch_file@v2

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
	logicalnames "github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logical_names"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/pathvalidation"
)

// PatchFileArgs holds the input parameters for the patch_file tool.
type PatchFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	Diff        string `json:"diff" jsonschema:"Unified diff to apply to the file."`
}

// toolError is a convenience helper that returns a tool error result.
// Tool errors are returned to the agent with IsError: true so the
// server continues running after an expected failure.
func toolError(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}, nil, nil
}

// HandlePatchFile is the handler for the patch_file MCP tool.
// It validates the logical name, resolves the spec file, checks the
// implements list, then applies the provided unified diff to the file.
func HandlePatchFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args PatchFileArgs,
) (*mcp.CallToolResult, any, error) {

	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST). These are the only valid namespaces.
	name := args.LogicalName
	validPrefix := strings.HasPrefix(name, "ROOT/") ||
		strings.HasPrefix(name, "TEST/") ||
		name == "ROOT" ||
		name == "TEST"
	if !validPrefix {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 2: Resolve the logical name to a spec file path.
	specPath, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 3: Parse the frontmatter of the resolved spec file.
	fm, err := frontmatter.ParseFrontmatter(specPath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse frontmatter for %s: %v", name, err))
	}

	// Step 4: The implements list must not be empty.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name))
	}

	// Step 5: Normalize the provided path to forward slashes.
	// This ensures consistent comparison regardless of OS separator.
	normalizedPath := filepath.ToSlash(args.Path)

	// Step 6: Validate the path against the project root (working directory).
	// This is the security boundary — it prevents writes outside the project.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("failed to determine working directory: %v", err))
	}
	if err := pathvalidation.ValidatePath(normalizedPath, wd); err != nil {
		return toolError(fmt.Sprintf(
			"path validation failed: %v. allowed paths: %s",
			err,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 7: Check that the normalized path is present in the frontmatter's
	// implements list (exact string match). This enforces the write boundary.
	allowed := false
	for _, imp := range fm.Implements {
		if imp == normalizedPath {
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

	// Step 8: Read the existing file. patch_file requires the file to exist;
	// use write_file to create new files.
	existingContent, err := os.ReadFile(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return toolError(fmt.Sprintf("file does not exist: %s", normalizedPath))
		}
		return toolError(fmt.Sprintf("failed to read %s: %v", normalizedPath, err))
	}

	// Step 9: Parse the unified diff. The diff must be valid and parseable.
	files, _, err := gitdiff.Parse(strings.NewReader(args.Diff))
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse diff: %v", err))
	}

	// Step 10: Exactly one file must be present in the diff.
	// Applying a multi-file diff to a single path would be ambiguous.
	if len(files) != 1 {
		return toolError("diff must contain exactly one file")
	}

	// Step 11: Apply the patch to the existing file content.
	source := bytes.NewReader(existingContent)
	var output bytes.Buffer
	if err := gitdiff.Apply(&output, source, files[0]); err != nil {
		return toolError(fmt.Sprintf("failed to apply diff to %s: %v", normalizedPath, err))
	}

	// Step 12: Write the patched content back to the file, overwriting the original.
	// os.WriteFile preserves the path; the file must already exist (checked above).
	if err := os.WriteFile(normalizedPath, output.Bytes(), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %v", normalizedPath, err))
	}

	// Step 13: Return success.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("patched %s", normalizedPath)}},
	}, nil, nil
}
