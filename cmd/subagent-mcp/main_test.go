// code-from-spec: TEST/tech_design/server@v23

package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// binaryPath holds the path to the compiled binary, built once in TestMain.
var binaryPath string

// usageSnippet is a substring expected to appear in the usage message output.
// Taken from the usage message defined in ROOT/tech_design/server.
const usageSnippet = "Usage: subagent-mcp"

// TestMain builds the binary once into a temp directory. All tests in this
// file invoke the compiled binary as a subprocess.
func TestMain(m *testing.M) {
	// Create a temp directory for the built binary.
	tmpDir, err := os.MkdirTemp("", "subagent-mcp-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	// Determine the binary output path. On Windows, the binary must have
	// the .exe extension.
	binaryPath = tmpDir + "/subagent-mcp"
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Build the binary.
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// --- Happy Path ---

// TestHelpFlag verifies that --help prints usage to stdout and exits 0.
func TestHelpFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	stdout, err := cmd.Output()

	// --help should exit 0, so err should be nil.
	if err != nil {
		t.Fatalf("expected exit 0 for --help, got error: %v", err)
	}

	if !strings.Contains(string(stdout), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\nstdout: %s", string(stdout))
	}
}

// TestHelpWord verifies that the bare word "help" prints usage to stdout
// and exits 0.
func TestHelpWord(t *testing.T) {
	cmd := exec.Command(binaryPath, "help")
	stdout, err := cmd.Output()

	// "help" should exit 0, so err should be nil.
	if err != nil {
		t.Fatalf("expected exit 0 for help, got error: %v", err)
	}

	if !strings.Contains(string(stdout), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\nstdout: %s", string(stdout))
	}
}

// TestShortHelpFlag verifies that -h prints usage to stdout and exits 0.
func TestShortHelpFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "-h")
	stdout, err := cmd.Output()

	// -h should exit 0, so err should be nil.
	if err != nil {
		t.Fatalf("expected exit 0 for -h, got error: %v", err)
	}

	if !strings.Contains(string(stdout), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\nstdout: %s", string(stdout))
	}
}

// --- Failure Cases ---

// TestUnrecognizedArgument verifies that an unrecognized argument prints
// usage to stderr and exits 1.
func TestUnrecognizedArgument(t *testing.T) {
	cmd := exec.Command(binaryPath, "something")
	// Output() captures stdout and returns an error if exit code != 0.
	// We need stderr, so use CombinedOutput or capture separately.
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	// Expect a non-zero exit code.
	if err == nil {
		t.Fatal("expected non-zero exit code for unrecognized argument, got exit 0")
	}

	// Verify exit code is 1.
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	// Verify stderr contains the usage message.
	stderr := stderrBuf.String()
	if !strings.Contains(stderr, usageSnippet) {
		t.Errorf("stderr does not contain usage message.\nstderr: %s", stderr)
	}
}

// TestMultipleArguments verifies that multiple arguments print usage to
// stderr and exit 1.
func TestMultipleArguments(t *testing.T) {
	cmd := exec.Command(binaryPath, "foo", "bar")
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	// Expect a non-zero exit code.
	if err == nil {
		t.Fatal("expected non-zero exit code for multiple arguments, got exit 0")
	}

	// Verify exit code is 1.
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	// Verify stderr contains the usage message.
	stderr := stderrBuf.String()
	if !strings.Contains(stderr, usageSnippet) {
		t.Errorf("stderr does not contain usage message.\nstderr: %s", stderr)
	}
}

// --- MCP Protocol ---

// testStartMCPSubprocess starts the binary as a subprocess with stdin/stdout
// pipes, sends an initialize request and initialized notification, then
// sends a tools/list request. Returns the parsed tools array and a cleanup
// function that closes stdin and waits for the process.
//
// Callers must invoke the returned cleanup function when done.
func testStartMCPSubprocess(t *testing.T) ([]interface{}, func()) {
	t.Helper()

	// Start the binary as a subprocess with stdin/stdout pipes.
	cmd := exec.Command(binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start binary: %v", err)
	}

	// Use a buffered reader to read newline-delimited JSON-RPC responses.
	reader := bufio.NewReader(stdout)

	// Step 1: Send the initialize request (JSON-RPC 2.0).
	initializeReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test-client","version":"1.0.0"}}}` + "\n"
	if _, err := io.WriteString(stdin, initializeReq); err != nil {
		t.Fatalf("failed to write initialize request: %v", err)
	}

	// Read the initialize response.
	initResp, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("failed to read initialize response: %v", err)
	}

	// Verify that the initialize response is valid JSON-RPC with id 1.
	var initResult map[string]interface{}
	if err := json.Unmarshal(initResp, &initResult); err != nil {
		t.Fatalf("failed to parse initialize response: %v\nraw: %s", err, string(initResp))
	}
	if initResult["error"] != nil {
		t.Fatalf("initialize returned an error: %v", initResult["error"])
	}

	// Step 2: Send the initialized notification (no id — it's a notification).
	initializedNotif := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	if _, err := io.WriteString(stdin, initializedNotif); err != nil {
		t.Fatalf("failed to write initialized notification: %v", err)
	}

	// Step 3: Send the tools/list request.
	toolsListReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"
	if _, err := io.WriteString(stdin, toolsListReq); err != nil {
		t.Fatalf("failed to write tools/list request: %v", err)
	}

	// Read the tools/list response.
	toolsResp, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("failed to read tools/list response: %v", err)
	}

	// Parse the tools/list response.
	var toolsResult map[string]interface{}
	if err := json.Unmarshal(toolsResp, &toolsResult); err != nil {
		t.Fatalf("failed to parse tools/list response: %v\nraw: %s", err, string(toolsResp))
	}
	if toolsResult["error"] != nil {
		t.Fatalf("tools/list returned an error: %v", toolsResult["error"])
	}

	// Extract the result.tools array.
	result, ok := toolsResult["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("tools/list response missing 'result' object.\nraw: %s", string(toolsResp))
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("tools/list result missing 'tools' array.\nraw: %s", string(toolsResp))
	}

	// Cleanup: close stdin to signal the subprocess to shut down, then wait.
	cleanup := func() {
		stdin.Close()
		// Ignore the wait error — the process may exit with a non-zero code
		// when stdin is closed, which is acceptable for these tests.
		_ = cmd.Wait()
	}

	return tools, cleanup
}

// TestToolsListAdvertisesMaxResultSizeChars starts the binary as a subprocess,
// sends an MCP initialize request followed by a tools/list request over stdin
// (JSON-RPC 2.0, newline-delimited), and verifies that the load_chain tool
// has _meta["anthropic/maxResultSizeChars"] equal to 500000.
func TestToolsListAdvertisesMaxResultSizeChars(t *testing.T) {
	tools, cleanup := testStartMCPSubprocess(t)
	defer cleanup()

	// Find the load_chain tool and check its _meta field.
	var found bool
	for _, toolRaw := range tools {
		tool, ok := toolRaw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tool["name"].(string)
		if name != "load_chain" {
			continue
		}
		found = true

		// The _meta field is at the top level of the tool object.
		meta, ok := tool["_meta"].(map[string]interface{})
		if !ok {
			t.Fatalf("load_chain tool missing '_meta' field.\ntool: %v", tool)
		}

		// Check anthropic/maxResultSizeChars equals 500000.
		maxSize, ok := meta["anthropic/maxResultSizeChars"]
		if !ok {
			t.Fatalf("load_chain _meta missing 'anthropic/maxResultSizeChars'.\n_meta: %v", meta)
		}

		// JSON numbers are parsed as float64 by default.
		maxSizeFloat, ok := maxSize.(float64)
		if !ok {
			t.Fatalf("expected anthropic/maxResultSizeChars to be a number, got %T: %v", maxSize, maxSize)
		}
		if int(maxSizeFloat) != 500000 {
			t.Errorf("expected anthropic/maxResultSizeChars = 500000, got %v", maxSize)
		}
		break
	}

	if !found {
		t.Fatalf("load_chain tool not found in tools/list response.\ntools: %v", tools)
	}
}

// TestToolsListAdvertisesAllThreeTools starts the binary as a subprocess,
// sends an MCP initialize request followed by a tools/list request over stdin
// (JSON-RPC 2.0, newline-delimited), and verifies that all three expected
// tools — load_chain, write_file, and patch_file — are present in the
// response.
func TestToolsListAdvertisesAllThreeTools(t *testing.T) {
	tools, cleanup := testStartMCPSubprocess(t)
	defer cleanup()

	// Build a set of advertised tool names for easy lookup.
	advertised := make(map[string]bool)
	for _, toolRaw := range tools {
		tool, ok := toolRaw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tool["name"].(string)
		if name != "" {
			advertised[name] = true
		}
	}

	// Verify each required tool is present.
	required := []string{"load_chain", "write_file", "patch_file"}
	for _, name := range required {
		if !advertised[name] {
			t.Errorf("expected tool %q not found in tools/list response.\nadvertised: %v", name, advertised)
		}
	}
}
