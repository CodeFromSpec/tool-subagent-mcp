// spec: ROOT/tech_design/internal/modes/codegen@v34

// Package codegen — shared types used by setup.go and the tool handlers.
//
// Target and WriteFileArgs are defined here (rather than in setup.go or the
// handler files) so every file in the package can reference them without
// creating import cycles.
package codegen

import "github.com/CodeFromSpec/tool-subagent-mcp/internal/frontmatter"

// Target holds the pre-resolved, pre-loaded state for a single codegen
// invocation.  It is built during Setup and captured by the tool handler
// closures so they can serve requests without additional I/O.
//
// Spec: ROOT/tech_design/internal/modes/codegen §Target type
type Target struct {
	// LogicalName is the validated logical name passed as the CLI argument.
	LogicalName string
	// FilePath is the spec file path resolved from LogicalName.
	FilePath string
	// Frontmatter is the parsed frontmatter of the target spec node.
	Frontmatter *frontmatter.Frontmatter
	// ChainContent is the fully concatenated chain built during Setup.
	// load_context returns this verbatim; no further I/O is needed.
	ChainContent string
}

// WriteFileArgs is the input schema for the write_file tool.
//
// Spec: ROOT/tech_design/internal/modes/codegen §WriteFileArgs type
type WriteFileArgs struct {
	// Path is the relative file path from the project root.
	Path string `json:"path" jsonschema:"Relative file path from project root."`
	// Content is the complete file content to write.
	Content string `json:"content" jsonschema:"Complete file content to write."`
}
