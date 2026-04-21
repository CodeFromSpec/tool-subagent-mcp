// spec: ROOT/tech_design/internal/modes/codegen/tools/write_file@v25

// Package codegen implements the codegen mode for the subagent MCP server.
// This file provides the write_file tool handler, which validates the requested
// output path and writes the generated content to disk.
//
// Security model (from spec ROOT/tech_design/internal/modes/codegen §Path validation):
//   - ValidatePath is called first to enforce filesystem containment (defense 1).
//   - The path must appear verbatim in target.Frontmatter.Implements (defense 2).
//   - Both checks must pass before any bytes are written to disk.
//
// Error handling (from spec ROOT/tech_design/internal/modes/codegen/tools):
//   - All expected error conditions use IsError:true on the result.
//   - The returned Go error is reserved for catastrophic server failures.
//   - The server must continue running after a tool error.
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

// handleWriteFile returns the closure registered as the write_file tool handler.
// It captures target so the closure can access the implements list and project
// root without additional I/O.
//
// The returned closure enforces two layers of path validation before writing:
//  1. ValidatePath — ensures the path cannot escape the project root.
//  2. implements check — ensures the path was declared as an output of this node.
//
// This implements the algorithm from spec §Contracts/Algorithm:
//  1. ValidatePath on args.Path vs. working directory.
//  2. Exact string match against target.Frontmatter.Implements.
//  3. Create missing intermediate directories.
//  4. Write args.Content, overwriting if the file exists.
//  5. Return success with "wrote <path>".
func handleWriteFile(target *Target) func(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args WriteFileArgs,
) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		args WriteFileArgs,
	) (*mcp.CallToolResult, any, error) {
		// Build the list of allowed paths once so we can include it in error
		// messages. This gives the subagent actionable feedback.
		allowedPaths := strings.Join(target.Frontmatter.Implements, ", ")

		// Step 1 — path traversal / containment validation.
		//
		// ValidatePath resolves the path against the project working directory
		// and rejects any path that would escape the project root (via "..",
		// absolute references, symlinks, or OS-specific separator tricks).
		//
		// The working directory is the project root (see spec
		// ROOT/tech_design §Project root): the binary is always launched from
		// there, so passing "." gives us the canonical project root.
		cwd, err := os.Getwd()
		if err != nil {
			// Unable to determine working directory — this is a server-level
			// problem. Return a tool error so the agent can report it.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("failed to determine working directory: %v", err),
				}},
				IsError: true,
			}, nil, nil
		}

		if err := pathvalidation.ValidatePath(args.Path, cwd); err != nil {
			// Spec §Error handling: path validation failure → tool error with
			// the violation and the list of allowed paths.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf(
						"path validation failed: %v. allowed paths: %s",
						err, allowedPaths,
					),
				}},
				IsError: true,
			}, nil, nil
		}

		// Step 2 — implements allow-list check.
		//
		// Only paths declared in the node's implements list may be written.
		// This is the primary security boundary of write_file
		// (spec ROOT/domain/modes/codegen §Constraints).
		// An exact string match is required — no glob or prefix matching.
		if !containsPath(target.Frontmatter.Implements, args.Path) {
			// Spec §Error handling: path not in implements → tool error with
			// the path and list of allowed paths.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf(
						"path not allowed: %s. allowed paths: %s",
						args.Path, allowedPaths,
					),
				}},
				IsError: true,
			}, nil, nil
		}

		// Step 3 — create any missing intermediate directories.
		//
		// filepath.Dir extracts the directory portion of the path. MkdirAll is
		// a no-op when the directory already exists, so this is always safe.
		dir := filepath.Dir(args.Path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			// Spec §Error handling: directory creation failure.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf(
						"failed to create directories for %s: %v",
						args.Path, err,
					),
				}},
				IsError: true,
			}, nil, nil
		}

		// Step 4 — write the content, overwriting any existing file.
		//
		// os.WriteFile creates the file if it does not exist and truncates it
		// if it does. Permissions 0o644 are standard for source files.
		if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
			// Spec §Error handling: write failure.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("failed to write %s: %v", args.Path, err),
				}},
				IsError: true,
			}, nil, nil
		}

		// Step 5 — success. Spec §Algorithm: return "wrote <path>".
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "wrote " + args.Path}},
		}, nil, nil
	}
}

// containsPath reports whether the slice s contains the exact string target.
// The comparison is case-sensitive and requires a verbatim match — no glob,
// no normalization — because the implements list stores exact relative paths
// as authored in the spec frontmatter.
func containsPath(s []string, target string) bool {
	for _, p := range s {
		if p == target {
			return true
		}
	}
	return false
}
