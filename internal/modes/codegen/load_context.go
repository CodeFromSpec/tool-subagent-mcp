// spec: ROOT/tech_design/internal/modes/codegen/tools/load_context@v29

package codegen

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CodeFromSpec/tool-subagent-mcp/internal/chainresolver"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/logicalnames"
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/pathvalidation"
)

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func isValidCodegenTarget(name string) bool {
	return name == "ROOT" || name == "TEST" ||
		strings.HasPrefix(name, "ROOT/") || strings.HasPrefix(name, "TEST/")
}

func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// Version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func buildChainContent(chain *chainresolver.Chain, uuid string) (string, error) {
	var sb strings.Builder

	writeItem := func(item chainresolver.ChainItem) error {
		for _, fp := range item.FilePaths {
			data, err := os.ReadFile(fp)
			if err != nil {
				return fmt.Errorf("reading %s: %w", fp, err)
			}
			fmt.Fprintf(&sb, "<<<FILE_%s>>>\nnode: %s\npath: %s\n\n%s\n<<<END_FILE_%s>>>\n\n",
				uuid, item.LogicalName, fp, data, uuid)
		}
		return nil
	}

	for _, item := range chain.Ancestors {
		if err := writeItem(item); err != nil {
			return "", err
		}
	}
	if err := writeItem(chain.Target); err != nil {
		return "", err
	}
	for _, item := range chain.Dependencies {
		if err := writeItem(item); err != nil {
			return "", err
		}
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

func handleLoadContext(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args LoadContextArgs,
) (*mcp.CallToolResult, any, error) {
	// Step 1
	if !isValidCodegenTarget(args.LogicalName) {
		return toolError(fmt.Sprintf("codegen target must be a ROOT/ or TEST/ logical name: %s", args.LogicalName)), nil, nil
	}

	// Step 2
	filePath, ok := logicalnames.PathFromLogicalName(args.LogicalName)
	if !ok {
		return toolError(fmt.Sprintf("invalid logical name: %s", args.LogicalName)), nil, nil
	}

	// Step 3
	fm, err := frontmatter.ParseFrontmatter(filePath)
	if err != nil {
		return toolError(fmt.Sprintf("error loading frontmatter: %v", err)), nil, nil
	}

	// Step 4
	if len(fm.Implements) == 0 {
		return toolError(fmt.Sprintf("node %s has no implements", args.LogicalName)), nil, nil
	}
	projectRoot, err := os.Getwd()
	if err != nil {
		return toolError(fmt.Sprintf("error getting working directory: %v", err)), nil, nil
	}
	for _, impl := range fm.Implements {
		if err := pathvalidation.ValidatePath(impl, projectRoot); err != nil {
			return toolError(fmt.Sprintf("invalid implements path %s: %v", impl, err)), nil, nil
		}
	}

	// Step 5
	uuid, err := generateUUID()
	if err != nil {
		return toolError(fmt.Sprintf("error generating UUID: %v", err)), nil, nil
	}
	chain, err := chainresolver.ResolveChain(args.LogicalName)
	if err != nil {
		return toolError(fmt.Sprintf("error resolving chain: %v", err)), nil, nil
	}
	chainContent, err := buildChainContent(chain, uuid)
	if err != nil {
		return toolError(fmt.Sprintf("error building chain content: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: chainContent}},
	}, nil, nil
}
