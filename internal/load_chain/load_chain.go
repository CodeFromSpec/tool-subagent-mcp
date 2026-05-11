// code-from-spec: ROOT/tech_design/internal/tools/load_chain@v67
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

// LoadChainArgs defines the input parameters for the load_chain MCP tool.
// The logical_name field identifies the target spec node to load the chain for.
type LoadChainArgs struct {
	LogicalName string `json:"logical_name" jsonschema:"Logical name of the node to generate code for."`
}

// HandleLoadChain is the MCP tool handler for the load_chain tool. It:
//  1. Validates the logical name (must start with ROOT/ or TEST/).
//  2. Resolves the logical name to a file path.
//  3. Parses and validates the frontmatter (implements must be non-empty, paths valid).
//  4. Generates a UUID for heredoc-style delimiters.
//  5. Resolves the full spec chain (ancestors, target, dependencies, code files).
//  6. Serializes the chain into a single text response using the delimiter format.
//
// All expected error conditions are returned as MCP tool errors (IsError: true)
// so the server continues running after any error.
func HandleLoadChain(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args LoadChainArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1: Validate that the logical name is a ROOT or TEST name.
	// Valid prefixes: "ROOT", "ROOT/...", "TEST", "TEST/..."
	if !isValidTargetPrefix(args.LogicalName) {
		return toolError("target must be a ROOT/ or TEST/ logical name: " + args.LogicalName), nil, nil
	}

	// Step 2: Resolve the logical name to a file path relative to the project root.
	// PathFromLogicalName strips any qualifier before resolving.
	nodePath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError("invalid logical name: " + args.LogicalName), nil, nil
	}

	// Step 3: Parse the frontmatter of the target node.
	// This provides the version, implements list, and other metadata.
	fm, err := frontmatter.ParseFrontmatter(nodePath)
	if err != nil {
		return toolError(fmt.Sprintf("error parsing frontmatter for %s: %v", args.LogicalName, err)), nil, nil
	}

	// Step 4a: The implements list must not be empty. A node with no implements
	// cannot be used for code generation.
	if len(fm.Implements) == 0 {
		return toolError("node " + args.LogicalName + " has no implements"), nil, nil
	}

	// Step 4b: Validate each path in implements against the working directory.
	// This prevents path traversal attacks or accidental writes outside the project.
	wd, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("error getting working directory: %v", err)), nil, nil
	}
	for _, implPath := range fm.Implements {
		if validateErr := pathvalidation.ValidatePath(implPath, wd); validateErr != nil {
			return toolError(fmt.Sprintf("invalid implements path %q: %v", implPath, validateErr)), nil, nil
		}
	}

	// Step 5: Generate a UUID for the heredoc-style delimiters.
	// The same UUID is reused for all file sections in this call to maintain
	// consistency within the response. It is generated per-call to avoid
	// collisions with file content that might contain delimiter strings.
	delimID := uuid.New().String()

	// Step 6: Resolve the full chain (ancestors, target, dependencies, code).
	chain, err := chainresolver.ResolveChain(args.LogicalName)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain for %s: %v", args.LogicalName, err)), nil, nil
	}

	// Step 7: Build the serialized chain output.
	var buf strings.Builder

	// --- Ancestors ---
	// For each ancestor node, include its # Public section content (body and
	// subsections), but NOT the "# Public" heading line itself.
	// Ancestors with no public section, or with an empty public section
	// (blank after trimming), are skipped entirely — no file section is emitted.
	for _, item := range chain.Ancestors {
		nodeBody, parseErr := parsenode.ParseNode(item.LogicalName)
		if parseErr != nil {
			return toolError(fmt.Sprintf("error parsing node %s: %v", item.LogicalName, parseErr)), nil, nil
		}

		// Reconstruct the # Public section's body (without the heading).
		sectionContent := reconstructPublicBody(nodeBody.Public)
		if strings.TrimSpace(sectionContent) == "" {
			// Skip ancestors with no meaningful public content.
			continue
		}

		writeSection(&buf, delimID, item.LogicalName, item.FilePath, sectionContent)
	}

	// --- Target ---
	// Read the target file and include it with a reduced frontmatter.
	// The reduced frontmatter contains only "version" and "implements" fields;
	// all other fields (parent_version, subject_version, depends_on) are
	// stripped to save tokens and avoid confusing the subagent.
	targetContent, readErr := os.ReadFile(chain.Target.FilePath)
	if readErr != nil {
		return toolError(fmt.Sprintf("error reading file %s: %v", chain.Target.FilePath, readErr)), nil, nil
	}
	reducedContent := reduceTargetFrontmatter(string(targetContent), fm)
	writeSection(&buf, delimID, chain.Target.LogicalName, chain.Target.FilePath, reducedContent)

	// --- Dependencies ---
	// Group dependency items by FilePath, preserving first-occurrence order.
	// This ensures each dependency file is emitted as a single, consolidated
	// section even when referenced multiple times with different qualifiers.
	depOrder := make([]string, 0)                           // file paths in first-occurrence order
	depGroups := make(map[string][]chainresolver.ChainItem) // FilePath → items

	for _, item := range chain.Dependencies {
		fp := item.FilePath
		if _, exists := depGroups[fp]; !exists {
			depOrder = append(depOrder, fp)
		}
		depGroups[fp] = append(depGroups[fp], item)
	}

	for _, fp := range depOrder {
		items := depGroups[fp]

		// Use the base logical name (qualifier stripped) of the first item to
		// call ParseNode. All items in the group share the same FilePath, so
		// any item's base name resolves to the same node.
		baseName := stripQualifier(items[0].LogicalName)
		nodeBody, parseErr := parsenode.ParseNode(baseName)
		if parseErr != nil {
			return toolError(fmt.Sprintf("error parsing node %s: %v", baseName, parseErr)), nil, nil
		}

		// Determine whether to include the full # Public section or only
		// specific qualified subsections.
		//
		// If any item in the group has Qualifier == nil, it means the caller
		// wants the entire # Public section (all body content and subsections),
		// without the # Public heading.
		hasFullPublic := false
		for _, item := range items {
			if item.Qualifier == nil {
				hasFullPublic = true
				break
			}
		}

		var groupContent string
		if hasFullPublic {
			// Include the full # Public body content and subsections,
			// WITHOUT the "# Public" heading line itself.
			groupContent = reconstructPublicBody(nodeBody.Public)
		} else {
			// Include only the specified qualified subsections, in order.
			// Each qualifying subsection is appended as "## <heading>\n<content>".
			// Subsections with blank body content after trimming are skipped.
			var sb strings.Builder
			for _, item := range items {
				subContent := extractQualifiedSubsection(nodeBody.Public, *item.Qualifier)
				if subContent != "" {
					sb.WriteString(subContent)
				}
			}
			groupContent = sb.String()
		}

		// Skip groups that produce no meaningful content.
		if strings.TrimSpace(groupContent) == "" {
			continue
		}

		// Emit a single file section for the group.
		// Per spec: use the base logical name (qualifier stripped) as the node: header.
		writeSection(&buf, delimID, baseName, fp, groupContent)
	}

	// --- Code files ---
	// Include existing source files (the previously generated implementation
	// files) as-is. These sections use only a path: header, no node: header,
	// since they are generated source files rather than spec files.
	for _, fp := range chain.Code {
		codeContent, readErr := os.ReadFile(fp)
		if readErr != nil {
			return toolError(fmt.Sprintf("error reading file %s: %v", fp, readErr)), nil, nil
		}
		writeCodeSection(&buf, delimID, fp, string(codeContent))
	}

	// Step 8: Return the fully serialized chain as a single MCP text response.
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: buf.String()}},
	}, nil, nil
}

// isValidTargetPrefix reports whether the logical name starts with a valid
// target prefix: "ROOT", "ROOT/...", "TEST", or "TEST/...".
func isValidTargetPrefix(name string) bool {
	return name == "ROOT" || name == "TEST" ||
		strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/")
}

// stripQualifier removes the parenthetical qualifier from a logical name.
// For example, "ROOT/x/y(z)" becomes "ROOT/x/y". If there is no qualifier,
// the name is returned unchanged.
func stripQualifier(name string) string {
	if idx := strings.IndexByte(name, '('); idx >= 0 {
		return name[:idx]
	}
	return name
}

// reconstructPublicBody serializes the # Public section's content and all of
// its subsections into a markdown string, WITHOUT the "# Public" heading line.
//
// Per spec: ancestors and dependencies emit "the # Public section's own body
// content followed by each subsection reconstructed as markdown — without the
// # Public heading itself."
//
// Returns an empty string if pub is nil (i.e., no # Public section exists).
func reconstructPublicBody(pub *parsenode.Section) string {
	if pub == nil {
		return ""
	}

	var sb strings.Builder

	// Write the section's direct body content (the text between the # Public
	// heading and the first ## subsection heading, if any).
	if pub.Content != "" {
		sb.WriteString(pub.Content)
		// Ensure the content ends with a newline before any subsections.
		if !strings.HasSuffix(pub.Content, "\n") {
			sb.WriteString("\n")
		}
	}

	// Write each subsection as "## <heading>\n\n<content>".
	for _, sub := range pub.Subsections {
		sb.WriteString("\n## ")
		sb.WriteString(sub.Heading)
		sb.WriteString("\n")
		if sub.Content != "" {
			sb.WriteString("\n")
			sb.WriteString(sub.Content)
			// Ensure the subsection content ends with a newline.
			if !strings.HasSuffix(sub.Content, "\n") {
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// extractQualifiedSubsection finds the ## subsection within the # Public
// section whose normalized heading matches the normalized qualifier, and
// returns its content formatted as markdown (including the ## heading line).
//
// Per spec: "If the subsection has no body content (blank after trimming),
// treat it as absent and contribute nothing." In that case, this function
// returns an empty string — the ## heading is NOT emitted either.
//
// Returns an empty string if:
//   - pub is nil (no # Public section).
//   - No subsection matches the qualifier.
//   - The matching subsection's body content is blank after trimming.
func extractQualifiedSubsection(pub *parsenode.Section, qualifier string) string {
	if pub == nil {
		return ""
	}

	// Normalize the qualifier for case-insensitive, whitespace-collapsed comparison.
	normalizedQualifier := normalizename.NormalizeName(qualifier)

	for _, sub := range pub.Subsections {
		if normalizename.NormalizeName(sub.Heading) == normalizedQualifier {
			// Per spec: blank body content after trimming means the subsection
			// is treated as absent — contribute nothing (not even the heading).
			if strings.TrimSpace(sub.Content) == "" {
				return ""
			}
			var sb strings.Builder
			sb.WriteString("## ")
			sb.WriteString(sub.Heading)
			sb.WriteString("\n\n")
			sb.WriteString(sub.Content)
			// Ensure the content ends with a newline.
			if !strings.HasSuffix(sub.Content, "\n") {
				sb.WriteString("\n")
			}
			return sb.String()
		}
	}

	// No matching subsection found.
	return ""
}

// reduceTargetFrontmatter rewrites the YAML frontmatter block of the target
// file to contain only the "version" and "implements" fields. All other fields
// (parent_version, subject_version, depends_on, etc.) are stripped.
//
// This reduces the token count in the response and avoids exposing internal
// dependency tracking fields that would confuse the code-generation subagent.
//
// The rest of the file content (after the closing ---) is preserved as-is.
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

	// Split the file content into lines (preserving line endings) so we can
	// locate the opening and closing frontmatter delimiters.
	lines := strings.SplitAfter(content, "\n")

	// Find the opening "---" delimiter.
	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "---" {
			break
		}
		i++
	}
	if i >= len(lines) {
		// No opening delimiter found — prepend the reduced frontmatter.
		return reduced.String() + "\n" + content
	}

	// Preserve any content before the opening "---" (typically none).
	before := strings.Join(lines[:i], "")

	// Find the closing "---" delimiter.
	j := i + 1
	for j < len(lines) {
		if strings.TrimSpace(lines[j]) == "---" {
			break
		}
		j++
	}
	if j >= len(lines) {
		// No closing delimiter found — return the original content unchanged.
		return content
	}

	// Reconstruct: content before opening --- + reduced frontmatter + content after closing ---.
	after := strings.Join(lines[j+1:], "")
	return before + reduced.String() + "\n" + after
}

// writeSection writes a spec/dependency file section to buf using the
// heredoc-style delimiter format with node: and path: headers.
//
// Format:
//
//	<<<FILE_<uuid>>>
//	node: <logicalName>
//	path: <filePath>
//
//	<content>
//	<<<END_FILE_<uuid>>>
func writeSection(buf *strings.Builder, delimID, logicalName, filePath, content string) {
	fmt.Fprintf(buf, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(buf, "node: %s\n", logicalName)
	fmt.Fprintf(buf, "path: %s\n", filePath)
	fmt.Fprintf(buf, "\n%s\n", content)
	fmt.Fprintf(buf, "<<<END_FILE_%s>>>\n", delimID)
}

// writeCodeSection writes a code file section to buf using the heredoc-style
// delimiter format. Code sections have only a path: header (no node: header),
// since they are generated source files rather than spec files.
//
// Format:
//
//	<<<FILE_<uuid>>>
//	path: <filePath>
//
//	<content>
//	<<<END_FILE_<uuid>>>
func writeCodeSection(buf *strings.Builder, delimID, filePath, content string) {
	fmt.Fprintf(buf, "<<<FILE_%s>>>\n", delimID)
	fmt.Fprintf(buf, "path: %s\n", filePath)
	fmt.Fprintf(buf, "\n%s\n", content)
	fmt.Fprintf(buf, "<<<END_FILE_%s>>>\n", delimID)
}

// toolError creates an MCP tool error result with the given message.
// Using IsError: true allows the MCP server to continue running after
// reporting the error to the calling agent.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
