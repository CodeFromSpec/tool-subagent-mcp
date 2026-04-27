// code-from-spec: ROOT/tech_design/internal/tools/load_chain@v59
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
	// for all file sections in this response to avoid collisions with file
	// content.
	delimID := uuid.New().String()

	// Step 6: Resolve the full chain.
	chain, err := chainresolver.ResolveChain(args.LogicalName)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain for %s: %v", args.LogicalName, err)), nil, nil
	}

	var buf strings.Builder

	// Step 7: Build the output.

	// --- Ancestors ---
	// For each ancestor, use ParseNode and extract the # Public section's
	// own body content followed by each subsection — WITHOUT the # Public
	// heading itself (per spec: "without the # Public heading itself").
	// If the public section is nil, or the reconstructed content is empty
	// (blank after trimming), skip this ancestor entirely.
	for _, item := range chain.Ancestors {
		nodeBody, parseErr := parsenode.ParseNode(item.LogicalName)
		if parseErr != nil {
			return toolError(fmt.Sprintf("error parsing node %s: %v", item.LogicalName, parseErr)), nil, nil
		}
		// reconstructPublicBody returns the body content and subsections,
		// but not the # Public heading line.
		sectionContent := reconstructPublicBody(nodeBody.Public)
		if strings.TrimSpace(sectionContent) == "" {
			// Skip ancestors with no meaningful public content.
			continue
		}
		writeSection(&buf, delimID, item.LogicalName, item.FilePath, sectionContent)
	}

	// --- Target ---
	// Include the target file with a reduced frontmatter containing only
	// version and implements. All other frontmatter fields are stripped to
	// save tokens and avoid confusing the subagent with internal details.
	targetContent, readErr := os.ReadFile(chain.Target.FilePath)
	if readErr != nil {
		return toolError(fmt.Sprintf("error reading file %s: %v", chain.Target.FilePath, readErr)), nil, nil
	}
	reducedContent := reduceTargetFrontmatter(string(targetContent), fm)
	writeSection(&buf, delimID, chain.Target.LogicalName, chain.Target.FilePath, reducedContent)

	// --- Dependencies ---
	// Group dependency items by FilePath, preserving first-occurrence order.
	// For each group, call ParseNode once using the base logical name (any
	// item in the group, qualifier stripped).
	//
	// Build the group's content as follows:
	//   - If any item in the group has Qualifier = nil, include the full
	//     # Public section's body content and subsections — WITHOUT the
	//     # Public heading itself.
	//   - Otherwise, for each item in the group (in order), find the
	//     ## <qualifier> subsection within # Public. If the subsection has
	//     no body content (blank after trimming), treat it as absent and
	//     contribute nothing. Otherwise, append the ## heading followed by
	//     the body content.
	//
	// If the consolidated content is empty (blank after trimming), skip the
	// group — do not emit a file section for it.

	// depOrder tracks the file paths in first-occurrence order.
	depOrder := make([]string, 0)
	// depGroups maps FilePath to the list of ChainItems in that group.
	depGroups := make(map[string][]chainresolver.ChainItem)

	for _, item := range chain.Dependencies {
		fp := item.FilePath
		if _, exists := depGroups[fp]; !exists {
			depOrder = append(depOrder, fp)
		}
		depGroups[fp] = append(depGroups[fp], item)
	}

	for _, fp := range depOrder {
		items := depGroups[fp]

		// Use the base logical name of the first item in the group (qualifier
		// stripped) to call ParseNode. All items in the group share the same
		// file path, so any item's base name resolves to the same node.
		baseName := stripQualifier(items[0].LogicalName)
		nodeBody, parseErr := parsenode.ParseNode(baseName)
		if parseErr != nil {
			return toolError(fmt.Sprintf("error parsing node %s: %v", baseName, parseErr)), nil, nil
		}

		// Determine what to include:
		// If any item has Qualifier == nil, the entire # Public section body
		// is needed (it already includes all subsections). The # Public
		// heading itself is NOT emitted per spec.
		hasFullPublic := false
		for _, item := range items {
			if item.Qualifier == nil {
				hasFullPublic = true
				break
			}
		}

		var groupContent string
		if hasFullPublic {
			// Include the full # Public body (content + subsections),
			// WITHOUT the # Public heading.
			groupContent = reconstructPublicBody(nodeBody.Public)
		} else {
			// Append each qualified subsection in order. Each subsection
			// includes its ## heading per spec: "including the ## heading".
			// If the subsection has no body content (blank after trimming),
			// treat it as absent and contribute nothing.
			var sb strings.Builder
			for _, item := range items {
				subContent := extractQualifiedSubsection(nodeBody.Public, *item.Qualifier)
				if subContent != "" {
					sb.WriteString(subContent)
				}
			}
			groupContent = sb.String()
		}

		// Skip groups with no meaningful content.
		if strings.TrimSpace(groupContent) == "" {
			continue
		}

		// Emit a single file section for the group. Per spec: "Use the base
		// logical name (qualifier stripped) as the node: header for the
		// emitted file section."
		writeSection(&buf, delimID, baseName, fp, groupContent)
	}

	// --- Code files ---
	// Include existing source files as-is, with only a path: header (no
	// node: header). These are the currently generated implementation files.
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

// reconstructPublicBody returns the markdown content of the # Public section's
// body and subsections, WITHOUT the "# Public" heading line itself.
//
// Per spec: ancestors and dependencies emit "the # Public section's own body
// content followed by each subsection reconstructed as markdown — without the
// # Public heading itself."
//
// Returns an empty string if pub is nil.
func reconstructPublicBody(pub *parsenode.Section) string {
	if pub == nil {
		return ""
	}

	var sb strings.Builder

	// Write the section's own content (between the # heading and first ##).
	// No # Public heading is emitted per spec.
	if pub.Content != "" {
		sb.WriteString(pub.Content)
		sb.WriteString("\n")
	}

	// Write each subsection as "## <heading>\n<content>".
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
// content as markdown including the ## heading.
//
// Per spec: "If the subsection has no body content (blank after trimming),
// treat it as absent and contribute nothing." So if the subsection's content
// is blank after trimming, this function returns an empty string — the ##
// heading is NOT emitted either.
//
// Returns an empty string if not found, if the public section is nil, or if
// the subsection body is blank after trimming.
func extractQualifiedSubsection(pub *parsenode.Section, qualifier string) string {
	if pub == nil {
		return ""
	}

	normalizedQualifier := normalizename.NormalizeName(qualifier)

	for _, sub := range pub.Subsections {
		if normalizename.NormalizeName(sub.Heading) == normalizedQualifier {
			// Per spec: if body content is blank after trimming, treat the
			// subsection as absent and contribute nothing.
			if strings.TrimSpace(sub.Content) == "" {
				return ""
			}
			var sb strings.Builder
			sb.WriteString("## ")
			sb.WriteString(sub.Heading)
			sb.WriteString("\n")
			sb.WriteString("\n")
			sb.WriteString(sub.Content)
			sb.WriteString("\n")
			return sb.String()
		}
	}

	return ""
}

// reduceTargetFrontmatter rewrites the frontmatter of the target file to
// contain only the version and implements fields. All other fields
// (parent_version, subject_version, depends_on, etc.) are stripped to save
// tokens and avoid confusing the subagent with internal details.
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
	// Split on newlines while preserving the line terminators.
	lines := strings.SplitAfter(content, "\n")

	// Find the opening "---".
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		i++
	}
	if i >= len(lines) {
		// No opening delimiter — return reduced frontmatter prepended to content.
		return reduced.String() + "\n" + content
	}
	// Preserve any content before the opening "---" (typically empty).
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
		// No closing delimiter — return original content unchanged.
		return content
	}

	// Reconstruct: before + reduced frontmatter + everything after closing "---".
	after := strings.Join(lines[j+1:], "")
	return before + reduced.String() + "\n" + after
}

// writeSection writes a spec/dependency file section with node: and path:
// headers, followed by a blank line and the file content.
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
// (no node: header), followed by a blank line and the file content.
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
// continues running after reporting the error to the caller.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
