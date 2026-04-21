// spec: ROOT/tech_design/internal/modes/codegen/tools/load_context@v27

// Package codegen implements the codegen mode for the subagent MCP server.
// This file provides the load_context tool handler, which returns the
// pre-loaded chain content to the subagent in a single MCP text response.
//
// The handler performs no I/O — all chain content is assembled during setup
// and stored in Target.ChainContent. This design minimises per-call latency
// and keeps the handler trivially correct: there is nothing to go wrong at
// call time.
package codegen

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleLoadContext returns the closure registered as the load_context tool
// handler. It captures target so the closure can access the pre-loaded chain
// without any additional I/O.
//
// Per spec (ROOT/domain/modes/codegen): load_context must return the complete
// chain as a single text response. Partial results are never returned — if the
// chain could not be loaded during setup, the server would have already exited.
//
// Per spec (ROOT/tech_design/internal/modes/codegen/tools): the returned Go
// error is reserved for catastrophic server failures; all expected conditions
// use IsError on the result. Because this handler has no failure modes of its
// own (the content is already in memory), it always returns a success result.
func handleLoadContext(target *Target) func(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ struct{},
) (*mcp.CallToolResult, any, error) {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ struct{},
	) (*mcp.CallToolResult, any, error) {
		// Return the chain content that was assembled during Setup.
		// No I/O occurs here — the content is already in memory.
		// This satisfies the spec requirement that load_context returns
		// everything in one call (ROOT/domain/modes/codegen §Decisions).
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: target.ChainContent}},
		}, nil, nil
	}
}
