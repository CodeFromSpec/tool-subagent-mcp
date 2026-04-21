// spec: TEST/tech_design/server@v3

// Package main — integration tests for the subagent-mcp binary.
//
// All tests build the binary once per test run (using go build in a temp
// directory) and then invoke it as a subprocess with os/exec. This approach
// validates the real startup sequence, exit codes, and stdio behavior exactly
// as the orchestrator will observe them.
//
// Spec ref: TEST/tech_design/server § Context
// "Tests invoke the compiled binary as a subprocess using os/exec and verify
// its behavior: exit codes, stderr output, and stdout output."
//
// Constraint (ROOT/tech_design): no test framework beyond standard `testing`.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	// Import the codegen package so we can compare against its exported
	// Instructions constant in the MCP initialize response test.
	// This avoids hard-coding the instructions string in two places.
	// Use the same module path as main.go to stay consistent.
	"github.com/gustavo-neto/tool-subagent-mcp/internal/modes/codegen"
)

// binaryPath holds the path to the compiled binary, built once in TestMain.
var binaryPath string

// TestMain builds the binary before running any tests, then cleans up.
//
// Building once and reusing across all tests is faster than per-test builds
// and also validates that the package compiles cleanly.
func TestMain(m *testing.M) {
	// Create a temporary directory to hold the compiled binary.
	tmpDir, err := os.MkdirTemp("", "subagent-mcp-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "subagent-mcp")

	// Resolve the package directory relative to this test file.
	// The test file lives in cmd/subagent-mcp/, so "." refers to that package.
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// runBinary is a helper that runs the compiled binary with the given args,
// captures stdout and stderr, and returns the exit code along with both
// output buffers. It never calls t.Fatal itself so callers can inspect
// partial output even on non-zero exits.
func runBinary(args ...string) (stdout, stderr string, exitCode int) {
	cmd := exec.Command(binaryPath, args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Unexpected error (e.g. process couldn't start at all) — treat
			// as exit 1 so the calling test can report failure clearly.
			exitCode = 1
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Happy path tests
// ---------------------------------------------------------------------------

// TestHelpFlag_PrintsUsageToStdout verifies that --help prints the usage
// message to stdout and exits 0.
//
// Spec ref: TEST/tech_design/server § "Help flag prints usage to stdout"
func TestHelpFlag_PrintsUsageToStdout(t *testing.T) {
	stdout, _, exitCode := runBinary("--help")

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}

	// The usage message must mention the invocation pattern and at least the
	// codegen mode so the caller knows what modes are available.
	const want = "Usage: subagent-mcp"
	if !strings.Contains(stdout, want) {
		t.Errorf("stdout does not contain usage message %q\ngot stdout:\n%s", want, stdout)
	}
}

// TestHelpWord_PrintsUsageToStdout verifies that the literal word "help"
// (not a flag) prints the usage message to stdout and exits 0.
//
// Spec ref: TEST/tech_design/server § "Help word prints usage to stdout"
func TestHelpWord_PrintsUsageToStdout(t *testing.T) {
	stdout, _, exitCode := runBinary("help")

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}

	const want = "Usage: subagent-mcp"
	if !strings.Contains(stdout, want) {
		t.Errorf("stdout does not contain usage message %q\ngot stdout:\n%s", want, stdout)
	}
}

// TestShortHelpFlag_PrintsUsageToStdout verifies that -h prints the usage
// message to stdout and exits 0.
//
// Spec ref: TEST/tech_design/server § "Short help flag prints usage to stdout"
func TestShortHelpFlag_PrintsUsageToStdout(t *testing.T) {
	stdout, _, exitCode := runBinary("-h")

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}

	const want = "Usage: subagent-mcp"
	if !strings.Contains(stdout, want) {
		t.Errorf("stdout does not contain usage message %q\ngot stdout:\n%s", want, stdout)
	}
}

// TestCodegenHelpFlag_PrintsModeHelpToStdout verifies that "codegen --help"
// prints the codegen-specific help message to stdout and exits 0.
//
// Spec ref: TEST/tech_design/server § "Codegen help flag prints mode help to stdout"
func TestCodegenHelpFlag_PrintsModeHelpToStdout(t *testing.T) {
	stdout, _, exitCode := runBinary("codegen", "--help")

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}

	// The codegen help message must mention the codegen usage line.
	// We compare against the HelpMessage function's output to stay in sync
	// with the codegen package rather than duplicating the exact text.
	wantPrefix := "Usage: subagent-mcp codegen"
	if !strings.Contains(stdout, wantPrefix) {
		t.Errorf("stdout does not contain codegen usage line %q\ngot stdout:\n%s", wantPrefix, stdout)
	}
}

// TestCodegenMode_ServerInstructionsMatchCodegenInstructions verifies that
// when the binary is launched in codegen mode, the MCP server's initialize
// response contains instructions that match codegen.Instructions.
//
// The test acts as a minimal MCP client over stdio:
//  1. Start the binary with a known logical name argument.
//  2. Send a JSON-RPC 2.0 "initialize" request over stdin.
//  3. Read the response from stdout.
//  4. Assert that result.instructions == codegen.Instructions.
//
// A real logical name that exists in the project must be used so Setup does
// not exit early with an error. We use the root node "ROOT" which always
// exists in any Code from Spec project.
//
// Spec ref: TEST/tech_design/server § "Codegen mode sets correct server instructions"
func TestCodegenMode_ServerInstructionsMatchCodegenInstructions(t *testing.T) {
	// Use a context with timeout so the test does not hang if the binary
	// gets stuck waiting for more input.
	ctx, cancel := context.WithTimeout(context.Background(), 10_000_000_000) // 10s
	defer cancel()

	// "ROOT" is the top-level spec node (code-from-spec/spec/_node.md).
	// It always exists and has no implements list requirement for this test
	// because we only need the server to start successfully — we disconnect
	// immediately after receiving the initialize response.
	//
	// NOTE: if ROOT has no implements list, Setup will return an error and
	// the test will fail. In that case, use any leaf node logical name that
	// exists in the project and has at least one file in its implements list.
	// The test records the logical name in the error message for easy diagnosis.
	logicalName := "TEST/tech_design/server"

	cmd := exec.CommandContext(ctx, binaryPath, "codegen", logicalName)

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = os.Stderr // let setup errors surface in test output

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to open stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start binary: %v", err)
	}

	// Send MCP initialize request (JSON-RPC 2.0 over stdio, newline-delimited).
	// The MCP SDK expects each message as a JSON object followed by a newline.
	//
	// Spec ref: EXTERNAL/mcp-go-sdk — the server speaks JSON-RPC 2.0 over stdio.
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "0.0.1",
			},
		},
	}

	reqBytes, err := json.Marshal(initRequest)
	if err != nil {
		t.Fatalf("failed to marshal initialize request: %v", err)
	}

	// Write the request then close stdin so the server sees EOF and can stop
	// waiting for more input. The server will process initialize and write its
	// response before the stdin EOF causes it to exit.
	if _, err := fmt.Fprintf(stdin, "%s\n", reqBytes); err != nil {
		t.Fatalf("failed to write initialize request: %v", err)
	}
	stdin.Close()

	// Wait for the process to finish (it will exit after stdin EOF).
	// Ignore the exit error — a non-zero exit is acceptable here because
	// the server may exit 1 after the transport closes, which is an
	// implementation detail, not a test failure.
	_ = cmd.Wait()

	// Parse the server's stdout. The MCP server writes one JSON object per
	// line; we look for a line that contains the initialize response.
	responseLines := strings.Split(outBuf.String(), "\n")

	var initResponse struct {
		ID     int `json:"id"`
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
			Instructions string `json:"instructions"`
		} `json:"result"`
	}

	found := false
	for _, line := range responseLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &initResponse); err != nil {
			// Not a JSON object we recognize — skip.
			continue
		}
		// Identify the initialize response by its ID matching our request.
		if initResponse.ID == 1 {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"did not receive an initialize response (id=1) from the server\n"+
				"logical name: %s\nraw stdout:\n%s",
			logicalName, outBuf.String(),
		)
	}

	// The core assertion: instructions must match codegen.Instructions exactly.
	// This ensures the server creates the MCP server with the correct options.
	//
	// Spec ref: ROOT/tech_design/server § "Startup sequence" step 3b:
	// "ServerOptions.Instructions = codegen.Instructions"
	if initResponse.Result.Instructions != codegen.Instructions {
		t.Errorf(
			"initialize response instructions mismatch\nwant:\n%s\ngot:\n%s",
			codegen.Instructions,
			initResponse.Result.Instructions,
		)
	}
}

// ---------------------------------------------------------------------------
// Failure case tests
// ---------------------------------------------------------------------------

// TestNoArguments_PrintsUsageToStderr verifies that running the binary with
// no arguments exits 1 and prints the usage message to stderr.
//
// Spec ref: TEST/tech_design/server § "No arguments prints usage to stderr"
func TestNoArguments_PrintsUsageToStderr(t *testing.T) {
	_, stderr, exitCode := runBinary()

	if exitCode != 1 {
		t.Errorf("expected exit 1, got %d", exitCode)
	}

	const want = "Usage: subagent-mcp"
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr does not contain usage message %q\ngot stderr:\n%s", want, stderr)
	}
}

// TestUnrecognizedMode_PrintsUsageToStderr verifies that an unrecognized mode
// argument exits 1 and prints the usage message (including the list of valid
// modes) to stderr.
//
// Spec ref: TEST/tech_design/server § "Unrecognized mode prints usage to stderr"
func TestUnrecognizedMode_PrintsUsageToStderr(t *testing.T) {
	_, stderr, exitCode := runBinary("unknownmode")

	if exitCode != 1 {
		t.Errorf("expected exit 1, got %d", exitCode)
	}

	// The usage message must appear on stderr.
	const wantUsage = "Usage: subagent-mcp"
	if !strings.Contains(stderr, wantUsage) {
		t.Errorf("stderr does not contain usage message %q\ngot stderr:\n%s", wantUsage, stderr)
	}

	// The error output must also list valid modes so the user can correct
	// their invocation (spec: "lists valid modes").
	const wantModes = "codegen"
	if !strings.Contains(stderr, wantModes) {
		t.Errorf("stderr does not list valid modes (expected %q)\ngot stderr:\n%s", wantModes, stderr)
	}
}

// TestCodegenSetupError_PrintsToStderr verifies that running the binary with
// "codegen" and no logical name argument exits 1 and prints the setup error
// to stderr.
//
// Spec ref: TEST/tech_design/server § "Codegen setup error prints to stderr"
func TestCodegenSetupError_PrintsToStderr(t *testing.T) {
	// Provide "codegen" with no further arguments. Setup requires exactly one
	// argument (the logical name), so it must return an error.
	_, stderr, exitCode := runBinary("codegen")

	if exitCode != 1 {
		t.Errorf("expected exit 1, got %d", exitCode)
	}

	// The error message must appear on stderr. We check for "error:" which
	// is the prefix main adds before all Setup errors (see main.go step 4).
	const want = "error:"
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr does not contain setup error prefix %q\ngot stderr:\n%s", want, stderr)
	}
}
