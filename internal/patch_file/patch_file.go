// code-from-spec: ROOT/tech_design/internal/tools/patch_file@v9

// Package patch_file implements the patch_file MCP tool handler.
// It applies a unified diff to an existing file, after validating
// the target path against the node's implements list and the project root.
//
// Security boundary: only files declared in the node's `implements`
// frontmatter field may be patched. The path is also validated against
// the working directory to prevent directory traversal attacks.
//
// This tool does not create new files — use write_file for that.
// Exactly one file per call is patched.
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
// All three fields are required — the tool returns a tool error if any are
// missing or invalid.
type PatchFileArgs struct {
	// LogicalName is the logical name of the node whose implements list
	// authorizes the write. Must start with ROOT/ or TEST/ (or equal ROOT/TEST).
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`

	// Path is the relative file path from the project root to the file to patch.
	// Must be one of the paths declared in the node's implements list.
	Path string `json:"path" jsonschema:"Relative file path from project root."`

	// Diff is the unified diff to apply to the file. Must contain exactly one
	// file entry.
	Diff string `json:"diff" jsonschema:"Unified diff to apply to the file."`
}

// toolError constructs an MCP tool error result with the given message.
// Tool errors use IsError: true so the MCP server keeps running — they are
// expected error conditions, not catastrophic failures. The returned Go error
// is always nil to signal that the server is healthy.
func toolError(message string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}, nil, nil
}

// HandlePatchFile is the MCP tool handler for the patch_file tool.
//
// It validates the logical name and path, then applies a unified diff to an
// existing file. The validation against the node's implements list is the
// security boundary: only files declared there may be written.
//
// The returned Go error is reserved for catastrophic server failures. All
// expected error conditions are returned as IsError: true tool results so the
// server remains running and can serve subsequent requests.
func HandlePatchFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args PatchFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST exactly). This is an early sanity check before
	// attempting to resolve the path, providing a clearer error message.
	name := args.LogicalName
	if name != "ROOT" && name != "TEST" &&
		!strings.HasPrefix(name, "ROOT/") &&
		!strings.HasPrefix(name, "TEST/") {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 2: Resolve the logical name to its corresponding _node.md file path
	// (or .test.md for TEST nodes). Returns false if the name does not match
	// any known pattern.
	nodePath, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name))
	}

	// Step 3: Parse the frontmatter from the resolved node file to get the
	// implements list. Errors are wrapped with context so the agent knows
	// which node caused the failure.
	fm, err := frontmatter.ParseFrontmatter(nodePath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to read node %s: %v", name, err))
	}

	// Step 4: The node must declare at least one implements entry. Without an
	// implements list there is no authorization to write any file.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name))
	}

	// Step 5: Normalize the caller-supplied path to forward slashes. This
	// ensures consistent comparison on all platforms (e.g. Windows may supply
	// backslash-separated paths).
	normalizedPath := filepath.ToSlash(args.Path)

	// Step 6: Validate the normalized path against the working directory to
	// prevent directory traversal (../../etc/passwd), absolute paths, and
	// other escape attempts. This check is independent of the implements list.
	wd, err := os.Getwd()
	if err != nil {
		// Failing to obtain the working directory is unexpected but must be
		// handled — do not proceed if we cannot establish the project root.
		return toolError(fmt.Sprintf("failed to get working directory: %v", err))
	}

	if err := pathvalidation.ValidatePath(normalizedPath, wd); err != nil {
		// Report the specific validation error and the set of allowed paths so
		// the agent knows exactly what went wrong and what values are acceptable.
		return toolError(fmt.Sprintf(
			"path validation failed: %v. allowed paths: %s",
			err,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 7: Check that the normalized path is explicitly listed in the node's
	// implements field. This is the primary security boundary — a path that
	// passes ValidatePath but is not in implements is still rejected.
	// Comparison is exact string match (both sides are forward-slash-normalized).
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
	// patch_file must not create new files — the caller should use write_file
	// for that. os.IsNotExist distinguishes "file not found" from other I/O
	// errors so the agent receives a precise message.
	existingContent, err := os.ReadFile(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return toolError(fmt.Sprintf("file does not exist: %s", normalizedPath))
		}
		return toolError(fmt.Sprintf("failed to read %s: %v", normalizedPath, err))
	}

	// Step 9: Parse the unified diff.
	// gitdiff.Parse returns an error for semantically invalid diffs. Completely
	// malformed input that cannot be interpreted as a diff at all results in
	// zero file entries with no error — that case is caught by step 10.
	files, _, err := gitdiff.Parse(strings.NewReader(args.Diff))
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse diff: %v", err))
	}

	// Step 10: Require exactly one file entry in the diff.
	// Zero entries means the input was not a recognizable diff.
	// More than one entry would mean the caller supplied a multi-file patch,
	// which is not supported — one call patches one file.
	if len(files) != 1 {
		return toolError("diff must contain exactly one file")
	}

	// Step 11: Apply the parsed patch to the existing file content.
	// bytes.NewReader implements io.ReaderAt as required by gitdiff.Apply.
	// Note: context-line mismatches may have already been caught by
	// gitdiff.Parse in step 9 rather than appearing here.
	source := bytes.NewReader(existingContent)
	var output bytes.Buffer
	if err := gitdiff.Apply(&output, source, files[0]); err != nil {
		return toolError(fmt.Sprintf("failed to apply diff to %s: %v", normalizedPath, err))
	}

	// Step 12: Overwrite the original file with the patched content.
	// Permissions 0644 match typical source-file permissions.
	if err := os.WriteFile(normalizedPath, output.Bytes(), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %v", normalizedPath, err))
	}

	// Step 13: Return a success message confirming which file was patched.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("patched %s", normalizedPath)}},
	}, nil, nil
}
