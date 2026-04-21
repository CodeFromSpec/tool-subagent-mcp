// spec: TEST/tech_design/internal/modes/codegen/help_message@v1

// Package codegen — tests for the HelpMessage function.
//
// Spec: ROOT/tech_design/internal/modes/codegen/help_message
// The help message must convey:
//   - The usage line so the orchestrator knows how to invoke the mode.
//   - The names of the two MCP tools (load_context, write_file).
//   - An MCP configuration JSON example with "mcpServers" and "stdio".
//   - Guidance that the subagent must have no other tools available
//     (the confinement guarantee).
package codegen

import (
	"strings"
	"testing"
)

// TestHelpMessage_ContainsUsageLine verifies that HelpMessage returns a string
// that includes the canonical usage line so the orchestrator knows the correct
// CLI invocation for the codegen mode.
//
// Spec step: "Expect: result contains "Usage: subagent-mcp codegen"".
func TestHelpMessage_ContainsUsageLine(t *testing.T) {
	got := HelpMessage()
	const want = "Usage: subagent-mcp codegen"
	if !strings.Contains(got, want) {
		t.Errorf("HelpMessage() does not contain usage line %q\ngot:\n%s", want, got)
	}
}

// TestHelpMessage_ContainsToolDescriptions verifies that HelpMessage names both
// MCP tools exposed by the codegen mode. The orchestrator and any human reader
// must know which tools the subagent can call.
//
// Spec step: "Expect: result contains "load_context" and "write_file"".
func TestHelpMessage_ContainsToolDescriptions(t *testing.T) {
	got := HelpMessage()

	for _, want := range []string{"load_context", "write_file"} {
		if !strings.Contains(got, want) {
			t.Errorf("HelpMessage() does not contain tool name %q\ngot:\n%s", want, got)
		}
	}
}

// TestHelpMessage_ContainsMCPConfigurationExample verifies that HelpMessage
// includes a concrete MCP configuration snippet. The orchestrator uses this
// to wire the binary into an MCP-capable host. The snippet must reference the
// "mcpServers" key and the "stdio" transport type.
//
// Spec step: "Expect: result contains "mcpServers" and "stdio"".
func TestHelpMessage_ContainsMCPConfigurationExample(t *testing.T) {
	got := HelpMessage()

	for _, want := range []string{"mcpServers", "stdio"} {
		if !strings.Contains(got, want) {
			t.Errorf("HelpMessage() does not contain MCP config keyword %q\ngot:\n%s", want, got)
		}
	}
}

// TestHelpMessage_ContainsConfinementGuidance verifies that HelpMessage
// explicitly states that the subagent must have no other tools available.
// This is the confinement guarantee: the orchestrator must understand that
// giving the subagent additional tools undermines the security boundary.
//
// Spec step: "Expect: result contains "no other tools available"".
func TestHelpMessage_ContainsConfinementGuidance(t *testing.T) {
	got := HelpMessage()
	const want = "no other tools available"
	if !strings.Contains(got, want) {
		t.Errorf("HelpMessage() does not contain confinement guidance %q\ngot:\n%s", want, got)
	}
}
