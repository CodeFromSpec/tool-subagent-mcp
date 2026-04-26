// spec: ROOT/tech_design/internal/tools/load_chain@v51
package load_chain

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/chainresolver"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/normalizename"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/parsenode"
	"github.com/CodeFromSpec/tool-subagent-mcp/v2/internal/pathvalidation"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoadChainArgs defines the input parameters for the load_chain tool.
type LoadChainArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

// HandleLoadChain is the MCP tool handler for load_chain. It validates the
// logical name, loads the spec chain, and returns the concatenated chain
// content as a single MCP text response.
func HandleLoadChain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args LoadChainArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name starts with ROOT/ or TEST/
	// (or equals ROOT or TEST exactly).
	if !isValidTargetPrefix(args.LogicalName) {
		return toolError("target must be a ROOT/ or TEST/ logical name: " + args.LogicalName), nil, nil
	}

	// Step 2: Resolve the logical name to a file path.
	nodePath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError("invalid logical name: " + args.LogicalName), nil, nil
	}

	// Step 3: Parse frontmatter from the resolved path.
	fm, err := frontmatter.ParseFrontmatter(nodePath)
	if err != nil {
		return toolError(fmt.Sprintf("error parsing frontmatter for %s: %v", args.LogicalName, err)), nil, nil
	}

	// Step 4a: Implements must not be empty.
	if len(fm.Implements) == 0 {
		return toolError("node " + args.LogicalName + " has no implements"), nil, nil
	}

	// Step 4b: Validate each implements path against the working directory.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("error getting working directory: %v", err)), nil, nil
	}
	for _, implPath := range fm.Implements {
		if validateErr := pathvalidation.ValidatePath(implPath, wd); validateErr != nil {
			return toolError(fmt.Sprintf("invalid implements path %q: %v", implPath, validateErr)), nil, nil
		}
	}

	// Step 5: Generate a UUID for heredoc delimiters. The same UUID is used
	// for all file sections in this response.
	delimID := uuid.New().String()

	// Step 6: Resolve the full chain.
	chain, err := chainresolver.ResolveChain(args.LogicalName)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain for %s: %v", args.LogicalName, err)), nil, nil
	}

	var buf strings.Builder

	// Step 7: Build the output.

	// --- Ancestors ---
	// For each ancestor, use ParseNode and extract the # Public section.
	// If Public is nil, include an empty section body.
	// The content is the "# Public" heading followed by its content and all
	// subsections, reconstructed as markdown.
	for _, item := range chain.Ancestors {
		nodeBody, parseErr := parsenode.ParseNode(item.LogicalName)
		if parseErr != nil {
			return toolError(fmt.Sprintf("error parsing node %s: %v", item.LogicalName, parseErr)), nil, nil
		}
		sectionContent := reconstructPublicSection(nodeBody.Public)
		writeSection(&buf, delimID, item.LogicalName, item.FilePath, sectionContent)
	}

	// --- Target ---
	// Include the target file with a reduced frontmatter containing only
	// version and implements. All other frontmatter fields are stripped.
	targetContent, readErr := os.ReadFile(chain.Target.FilePath)
	if readErr != nil {
		return toolError(fmt.Sprintf("error reading file %s: %v", chain.Target.FilePath, readErr)), nil, nil
	}
	reducedContent := reduceTargetFrontmatter(string(targetContent), fm)
	writeSection(&buf, delimID, chain.Target.LogicalName, chain.Target.FilePath, reducedContent)

	// --- Dependencies ---
	// For each dependency:
	//   - If Qualifier is nil: extract and include the full # Public section.
	//   - If Qualifier is non-nil: find the ## <qualifier> subsection within
	//     # Public and include only that subsection's content.
	for _, item := range chain.Dependencies {
		// The base logical name (without qualifier) is used for ParseNode.
		// logicalnames.PathFromLogicalName already strips the qualifier, and
		// the ChainItem.LogicalName from chainresolver is the original entry.
		// We need to strip any qualifier from the name before calling ParseNode.
		baseName := stripQualifier(item.LogicalName)

		nodeBody, parseErr := parsenode.ParseNode(baseName)
		if parseErr != nil {
			return toolError(fmt.Sprintf("error parsing node %s: %v", baseName, parseErr)), nil, nil
		}

		var sectionContent string
		if item.Qualifier == nil {
			// No qualifier: include the full # Public section.
			sectionContent = reconstructPublicSection(nodeBody.Public)
		} else {
			// Qualifier present: find and include only the matching ## subsection.
			sectionContent = extractQualifiedSubsection(nodeBody.Public, *item.Qualifier)
			if sectionContent == "" {
				// Subsection not found; include an empty body to signal the gap.
				sectionContent = fmt.Sprintf("(subsection %q not found in # Public)", *item.Qualifier)
			}
		}
		writeSection(&buf, delimID, item.LogicalName, item.FilePath, sectionContent)
	}

	// --- Code files ---
	// Include existing source files as-is, with only path: header (no node:).
	for _, fp := range chain.Code {
		codeContent, readErr := os.ReadFile(fp)
		if readErr != nil {
			return toolError(fmt.Sprintf("error reading file %s: %v", fp, readErr)), nil, nil
		}
		writeCodeSection(&buf, delimID, fp, string(codeContent))
	}

	// Step 8: Return the chain content as a success result.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: buf.String()}},
	}, nil, nil
}

// isValidTargetPrefix checks that the logical name starts with ROOT/ or TEST/,
// or equals ROOT or TEST exactly.
func isValidTargetPrefix(name string) bool {
	return name == "ROOT" || name == "TEST" ||
		strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/")
}

// stripQualifier removes the parenthetical qualifier from a logical name, if
// present. For example, "ROOT/x/y(z)" becomes "ROOT/x/y".
func stripQualifier(name string) string {
	if idx := strings.IndexByte(name, '('); idx >= 0 {
		return name[:idx]
	}
	return name
}

// reconstructPublicSection reconstructs the markdown for the # Public section
// from a parsed Section. If pub is nil (no public section exists), an empty
// string is returned.
//
// The reconstructed content includes:
//   - The "# Public" heading line.
//   - The section's own Content (content between the heading and first ##).
//   - Each subsection as "## <heading>\n<content>".
func reconstructPublicSection(pub *parsenode.Section) string {
	if pub == nil {
		return ""
	}

	var sb strings.Builder

	// Write the # Public heading using the original heading text.
	sb.WriteString("# ")
	sb.WriteString(pub.Heading)
	sb.WriteString("\n")

	// Write the section's own content (between the # heading and first ##).
	if pub.Content != "" {
		sb.WriteString("\n")
		sb.WriteString(pub.Content)
		sb.WriteString("\n")
	}

	// Write each subsection.
	for _, sub := range pub.Subsections {
		sb.WriteString("\n## ")
		sb.WriteString(sub.Heading)
		sb.WriteString("\n")
		if sub.Content != "" {
			sb.WriteString("\n")
			sb.WriteString(sub.Content)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// extractQualifiedSubsection finds a ## subsection within the # Public section
// whose normalized heading matches the normalized qualifier, and returns its
// content as markdown (including the ## heading). Returns an empty string if
// not found or if the public section is nil.
func extractQualifiedSubsection(pub *parsenode.Section, qualifier string) string {
	if pub == nil {
		return ""
	}

	normalizedQualifier := normalizename.NormalizeName(qualifier)

	for _, sub := range pub.Subsections {
		if normalizename.NormalizeName(sub.Heading) == normalizedQualifier {
			var sb strings.Builder
			sb.WriteString("## ")
			sb.WriteString(sub.Heading)
			sb.WriteString("\n")
			if sub.Content != "" {
				sb.WriteString("\n")
				sb.WriteString(sub.Content)
				sb.WriteString("\n")
			}
			return sb.String()
		}
	}

	return ""
}

// reduceTargetFrontmatter rewrites the frontmatter of the target file to
// contain only the version and implements fields. All other fields
// (parent_version, subject_version, depends_on, etc.) are stripped.
//
// This reduces token usage and avoids exposing internal dependency version
// information to the subagent.
func reduceTargetFrontmatter(content string, fm *frontmatter.Frontmatter) string {
	// Build the reduced frontmatter block.
	var reduced strings.Builder
	reduced.WriteString("---\n")
	reduced.WriteString(fmt.Sprintf("version: %d\n", fm.Version))
	if len(fm.Implements) > 0 {
		reduced.WriteString("implements:\n")
		for _, impl := range fm.Implements {
			reduced.WriteString(fmt.Sprintf("  - %s\n", impl))
		}
	}
	reduced.WriteString("---")

	// Find and replace the frontmatter block in the original content.
	// Locate the opening "---".
	lines := strings.SplitAfter(content, "\n")
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		i++
	}
	if i >= len(lines) {
		// No opening delimiter — return original content with reduced frontmatter prepended.
		return reduced.String() + "\n" + content
	}
	// Preserve content before the opening "---" (typically empty).
	before := strings.Join(lines[:i], "")

	// Find the closing "---".
	j := i + 1
	for j < len(lines) {
		if strings.TrimSpace(lines[j]) == "---" {
			break
		}
		j++
	}
	if j >= len(lines) {
		// No closing delimiter — return original.
		return content
	}

	// Reconstruct: before + reduced frontmatter + everything after closing "---".
	after := strings.Join(lines[j+1:], "")
	return before + reduced.String() + "\n" + after
}

// writeSection writes a spec/dependency file section with node: and path:
// headers, separated from the content by a blank line.
//
// Format:
//
//	<<<FILE_<uuid>>>>
//	node: <logicalName>
//	path: <filePath>
//
//	<content>
//	<<<END_FILE_<uuid>>>>
func writeSection(buf *strings.Builder, delimID, logicalName, filePath, content string) {
	fmt.Fprintf(buf, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(buf, "node: %s\n", logicalName)
	fmt.Fprintf(buf, "path: %s\n", filePath)
	fmt.Fprintf(buf, "\n%s\n", content)
	fmt.Fprintf(buf, "<<<END_FILE_%s>>>\n", delimID)
}

// writeCodeSection writes a code file section with only a path: header
// (no node: header), separated from the content by a blank line.
//
// Format:
//
//	<<<FILE_<uuid>>>>
//	path: <filePath>
//
//	<content>
//	<<<END_FILE_<uuid>>>>
func writeCodeSection(buf *strings.Builder, delimID, filePath, content string) {
	fmt.Fprintf(buf, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(buf, "path: %s\n", filePath)
	fmt.Fprintf(buf, "\n%s\n", content)
	fmt.Fprintf(buf, "<<<END_FILE_%s>>>\n", delimID)
}

// toolError creates an MCP tool error result with the given message.
// Tool errors are returned as MCP-level errors (IsError: true) so the server
// continues running after reporting the error.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
