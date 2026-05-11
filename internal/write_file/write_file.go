// code-from-spec: ROOT/tech_design/internal/tools/write_file@v40

// Package write_file implements the write_file MCP tool handler.
//
// The tool accepts a logical node name, a relative file path, and file
// content. It resolves the node's spec frontmatter, validates that the
// provided path is declared in the node's implements list, and writes the
// content to disk. The implements list is the authoritative security
// boundary: no file outside it may be written.
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
// tool definition. Tags drive both JSON serialisation and MCP schema
// generation.
type WriteFileArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node whose implements list authorizes the write."`
	Path        string `json:"path" jsonschema:"Relative file path from project root."`
	Content     string `json:"content" jsonschema:"Complete file content to write."`
}

// toolError constructs an MCP tool error result.
//
// The returned Go error is always nil: all expected error conditions are
// signalled via IsError on the result so the server continues running after
// a per-call failure.
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
// Algorithm (matches spec step numbering):
//  1. Validate namespace — args.LogicalName must start with ROOT or TEST.
//  2. Resolve logical name to a spec file path via logicalnames.PathFromLogicalName.
//  3. Parse frontmatter from the resolved spec file.
//  4. Confirm implements is non-empty.
//  5. Normalize the supplied path to forward slashes (filepath.ToSlash).
//  6. Validate the normalized path against the working directory (ValidatePath).
//  7. Confirm the normalized path appears in implements (exact string match).
//  8. Create any missing intermediate directories (os.MkdirAll).
//  9. Write args.Content to the file, overwriting if it exists (os.WriteFile).
//  10. Return a success result with text "wrote <path>".
func HandleWriteFile(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1 — Validate that the logical name belongs to the ROOT or TEST
	// namespace. EXTERNAL nodes are excluded because they have no implements
	// list.
	if !isValidNamespace(args.LogicalName) {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName))
	}

	// Step 2 — Resolve the logical name to a file-system path for the spec
	// node file. PathFromLogicalName returns false when the name does not
	// match any known pattern.
	specPath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName))
	}

	// Step 3 — Parse the YAML frontmatter from the resolved spec file.
	// Errors here wrap one of the frontmatter sentinel errors (ErrRead,
	// ErrFrontmatterParse, ErrFrontmatterMissing, ErrMissingVersion).
	fm, err := frontmatter.ParseFrontmatter(specPath)
	if err != nil {
		return toolError(fmt.Sprintf("failed to parse frontmatter for %s: %s", args.LogicalName, err.Error()))
	}

	// Step 4 — A node without an implements list cannot authorize any write.
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", args.LogicalName))
	}

	// Step 5 — Normalize the caller-supplied path to forward slashes so that
	// comparison against the frontmatter implements list (which always uses
	// forward slashes) is consistent on all operating systems.
	normalizedPath := filepath.ToSlash(args.Path)

	// Step 6 — Validate the normalized path against the project root.
	// The tool is always executed from the project root (see ROOT/tech_design),
	// so os.Getwd() is the project root.
	// ValidatePath rejects empty paths, absolute paths, directory traversal
	// sequences, and paths that resolve outside the project root via symlinks.
	// This is the first layer of the security boundary.
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

	// Step 7 — Confirm the normalized path appears in the node's implements
	// list via exact string match.
	// This is the second and authoritative layer of the security boundary.
	// A path that passes ValidatePath but is not declared in implements is
	// still rejected — the subagent may only write files it was specified to
	// produce.
	if !containsPath(fm.Implements, normalizedPath) {
		return toolError(fmt.Sprintf(
			"path not allowed: %s. allowed paths: %s",
			normalizedPath,
			strings.Join(fm.Implements, ", "),
		))
	}

	// Step 8 — Create any missing intermediate directories for the target
	// path. filepath.FromSlash converts back to OS-native separators so that
	// directory creation works on Windows as well as POSIX systems.
	targetPath := filepath.FromSlash(normalizedPath)
	dir := filepath.Dir(targetPath)
	if dir != "." {
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			return toolError(fmt.Sprintf("failed to create directories for %s: %s", normalizedPath, mkErr.Error()))
		}
	}

	// Step 9 — Write the content to the file. os.WriteFile creates the file
	// if it does not exist and truncates it if it does, which gives us the
	// "overwrite if exists" behaviour required by the spec.
	if writeErr := os.WriteFile(targetPath, []byte(args.Content), 0644); writeErr != nil {
		return toolError(fmt.Sprintf("failed to write %s: %s", normalizedPath, writeErr.Error()))
	}

	// Step 10 — Return a success result identifying the written path.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "wrote " + normalizedPath}},
	}, nil, nil
}

// isValidNamespace reports whether name belongs to the ROOT or TEST
// namespace. EXTERNAL nodes are excluded because they do not have an
// implements list.
//
// Valid forms: "ROOT", "ROOT/<path>", "TEST", "TEST/<path>".
func isValidNamespace(name string) bool {
	if name == "ROOT" || name == "TEST" {
		return true
	}
	return strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/")
}

// containsPath reports whether target appears in the implements slice
// using an exact string match. Both sides are expected to already be
// normalised to forward slashes before this call.
func containsPath(implements []string, target string) bool {
	for _, p := range implements {
		if p == target {
			return true
		}
	}
	return false
}
