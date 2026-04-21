// spec: TEST/tech_design/internal/modes/codegen/tools/load_context@v4

// Package codegen contains tests for the codegen mode tool handlers.
// This file tests handleLoadContext, which wraps pre-loaded chain content
// and returns it as a single MCP text response.
//
// Spec reference: ROOT/tech_design/internal/modes/codegen/tools/load_context
// The handler performs no I/O — content was loaded during Setup.
// Tests simply verify that whatever is in Target.ChainContent is echoed back
// verbatim as a success MCP result.
package codegen

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestHandleLoadContext_ReturnsChainContent verifies the happy path:
// a Target with a non-empty ChainContent produces a success MCP result
// whose text equals ChainContent exactly.
//
// Spec step: "Returns pre-loaded chain content"
func TestHandleLoadContext_ReturnsChainContent(t *testing.T) {
	// Arrange: build a Target with a known chain content string.
	// The handler should return this verbatim without any modification.
	target := &Target{
		ChainContent: "<<<FILE_abc>>>\nnode: ROOT\npath: code-from-spec/spec/_node.md\n\nhello world\n<<<END_FILE_abc>>>",
	}

	// Act: obtain the closure from handleLoadContext and invoke it.
	// The closure captures target; ctx and req are unused by this handler.
	handler := handleLoadContext(target)
	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, struct{}{})

	// Assert: no Go-level error (catastrophic failures only use that path).
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	// Assert: result must not be nil and must not be an MCP-level error.
	if result == nil {
		t.Fatal("expected non-nil CallToolResult")
	}
	if result.IsError {
		t.Fatalf("expected success result, got IsError=true with content: %v", result.Content)
	}

	// Assert: exactly one content entry of type *mcp.TextContent.
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content entry, got %d", len(result.Content))
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}

	// Assert: the text equals the original ChainContent without alteration.
	if textContent.Text != target.ChainContent {
		t.Errorf("text mismatch:\n  got:  %q\n  want: %q", textContent.Text, target.ChainContent)
	}
}

// TestHandleLoadContext_EmptyChainContent verifies that an empty ChainContent
// is returned as a success result with an empty text string — not an error.
//
// Spec step: "Empty chain content"
// Rationale: the handler's contract is to echo whatever is in the Target;
// empty is a valid (if unusual) value and must not be treated as an error.
func TestHandleLoadContext_EmptyChainContent(t *testing.T) {
	// Arrange: Target with an explicitly empty chain.
	target := &Target{
		ChainContent: "",
	}

	// Act: invoke the closure.
	handler := handleLoadContext(target)
	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, struct{}{})

	// Assert: no Go-level error.
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	// Assert: result present and not an MCP-level error.
	if result == nil {
		t.Fatal("expected non-nil CallToolResult")
	}
	if result.IsError {
		t.Fatalf("expected success result for empty content, got IsError=true")
	}

	// Assert: exactly one content entry.
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content entry, got %d", len(result.Content))
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}

	// Assert: empty string returned as-is.
	if textContent.Text != "" {
		t.Errorf("expected empty text, got %q", textContent.Text)
	}
}
