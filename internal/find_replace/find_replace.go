// code-from-spec: ROOT/tech_design/internal/tools/find_replace@v2

// Package find_replace implements the MCP tool handler that finds a unique
// string in an existing source file and replaces it with a new string, after
// validating the target path against the node's implements list and the
// project root.
//
// Security boundary: writes are restricted to the paths listed in the
// frontmatter `implements` field of the resolved spec node. This check must
// not be bypassable.
package find_replace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/pathvalidation"
)

// FindReplaceArgs holds the input parameters for the find_replace tool.
// The JSON tags map to the MCP parameter names; the jsonschema tags provide
// human-readable descriptions surfaced in the tool schema.
type FindReplaceArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	OldString   string `json:"old_string" jsonschema:"Exact string to find in the file. Must match exactly once."`
	NewString   string `json:"new_string" jsonschema:"Replacement string."`
}

// toolError constructs an MCP tool error result with IsError set to true.
// All expected error conditions use this instead of returning a Go error so
// that the MCP server keeps running after the failure.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

// toolSuccess constructs a successful MCP tool result containing text.
func toolSuccess(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// HandleFindReplace is the MCP tool handler for the "find_replace" tool.
//
// It:
//  1. Validates the logical name prefix (must start with ROOT/ or TEST/, or
//     equal ROOT or TEST).
//  2. Resolves the logical name to a spec file path via
//     logicalnames.PathFromLogicalName.
//  3. Parses the frontmatter of the resolved spec file to obtain the
//     implements list.
//  4. Validates and authorises the target path against the implements list
//     and the project working directory.
//  5. Reads the file, counts occurrences of old_string (must be exactly 1),
//     replaces it, and writes the file back.
//
// The returned Go error is reserved for catastrophic server failures; all
// expected error conditions are returned as tool errors (IsError: true).
func HandleFindReplace(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args FindReplaceArgs,
) (*mcp.CallToolResult, any, error) {
	// -----------------------------------------------------------------------
	// Step 1: Validate logical name prefix.
	// -----------------------------------------------------------------------
	// The logical name must identify a spec node in the ROOT or TEST
	// namespace. Any other value is rejected immediately.
	name := args.LogicalName
	validPrefix := name == "ROOT" || name == "TEST" ||
		strings.HasPrefix(name, "ROOT/") ||
		strings.HasPrefix(name, "TEST/")
	if !validPrefix {
		return toolError(fmt.Sprintf("invalid logical name: %s", name)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 2: Resolve logical name to a spec file path.
	// -----------------------------------------------------------------------
	specPath, ok := logicalnames.PathFromLogicalName(name)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", name)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 3: Parse the frontmatter of the resolved spec file.
	// -----------------------------------------------------------------------
	fm, err := frontmatter.ParseFrontmatter(specPath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse frontmatter for %s: %v", name, err)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 4a: Validate that the implements list is non-empty.
	// -----------------------------------------------------------------------
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", name)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 5: Normalise the target path to forward slashes.
	// Paths from agents may arrive with OS-native separators; we normalise
	// early so that all subsequent comparisons are consistent.
	// -----------------------------------------------------------------------
	normalizedPath := filepath.ToSlash(args.Path)

	// -----------------------------------------------------------------------
	// Step 6: Validate the path against the project root (traversal checks).
	// -----------------------------------------------------------------------
	// Obtain the working directory, which the spec guarantees is the project
	// root. Each handler resolves this independently (stateless requirement).
	workingDir, err := os.Getwd()
	if err != nil {
		// This is an OS-level failure; treat it as a tool error so the server
		// stays alive but there is nothing the agent can do to recover.
		return toolError(fmt.Sprintf("failed to determine working directory: %v", err)), nil, nil
	}

	if err := pathvalidation.ValidatePath(normalizedPath, workingDir); err != nil {
		// Include the list of allowed paths so the agent can correct itself.
		allowed := strings.Join(fm.Implements, ", ")
		return toolError(fmt.Sprintf("path validation failed: %v. allowed paths: %s", err, allowed)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 7: Check that the normalised path appears in the implements list.
	// This is the security boundary — only spec-declared paths may be written.
	// -----------------------------------------------------------------------
	inImplements := false
	for _, imp := range fm.Implements {
		if imp == normalizedPath {
			inImplements = true
			break
		}
	}
	if !inImplements {
		allowed := strings.Join(fm.Implements, ", ")
		return toolError(fmt.Sprintf("path not allowed: %s. allowed paths: %s", normalizedPath, allowed)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 8: Read the existing file.
	// find_replace does not create new files; the file must already exist.
	// -----------------------------------------------------------------------
	fileBytes, err := os.ReadFile(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return toolError(fmt.Sprintf("file does not exist: %s", normalizedPath)), nil, nil
		}
		// Other read errors (permissions, etc.) are also surfaced as tool errors.
		return toolError(fmt.Sprintf("failed to read %s: %v", normalizedPath, err)), nil, nil
	}
	content := string(fileBytes)

	// -----------------------------------------------------------------------
	// Step 9: Count occurrences of old_string.
	// Exactly one occurrence is required — zero or multiple are rejected.
	// -----------------------------------------------------------------------
	count := strings.Count(content, args.OldString)
	if count == 0 {
		return toolError(fmt.Sprintf("old_string not found in %s", normalizedPath)), nil, nil
	}
	if count > 1 {
		return toolError(fmt.Sprintf("old_string matches multiple locations in %s", normalizedPath)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 10: Replace the single occurrence.
	// strings.Replace with n=1 replaces only the first (and only) match.
	// -----------------------------------------------------------------------
	newContent := strings.Replace(content, args.OldString, args.NewString, 1)

	// -----------------------------------------------------------------------
	// Step 11: Write the modified content back to the file.
	// Use the same permission bits (0644) that write_file uses for consistency.
	// -----------------------------------------------------------------------
	if err := os.WriteFile(normalizedPath, []byte(newContent), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write %s: %v", normalizedPath, err)), nil, nil
	}

	// -----------------------------------------------------------------------
	// Step 12: Return a success result.
	// -----------------------------------------------------------------------
	return toolSuccess(fmt.Sprintf("edited %s", normalizedPath)), nil, nil
}
