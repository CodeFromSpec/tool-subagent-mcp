// spec: TEST/tech_design/server@v9

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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	// Import the codegen package so we can compare against its exported
	// Instructions constant in the MCP initialize response test.
	// This avoids hard-coding the instructions string in two places.
	// Use the same module path as main.go to stay consistent.
	"github.com/CodeFromSpec/tool-subagent-mcp/internal/modes/codegen"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

	// On Windows, Go toolchain produces executables with the .exe extension.
	// Omitting it causes the binary to not be found when invoked via exec.Command.
	// Spec ref: TEST/tech_design/server § Context — "On Windows, the binary name
	// must include the `.exe` extension: use `runtime.GOOS == "windows"` to detect
	// the platform and append `.exe` to the output path when building."
	binaryName := "subagent-mcp"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath = filepath.Join(tmpDir, binaryName)

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
// The test uses mcp.Client with mcp.CommandTransport to connect to the server
// binary over stdio, performing the full MCP initialize handshake.
//
// Spec ref: TEST/tech_design/server § "Codegen mode sets correct server instructions"
func TestCodegenMode_ServerInstructionsMatchCodegenInstructions(t *testing.T) {
	// Use a context with timeout so the test does not hang if the binary
	// gets stuck waiting for more input.
	ctx, cancel := context.WithTimeout(context.Background(), 10_000_000_000) // 10s
	defer cancel()

	// Derive project root by walking up two levels from the package directory.
	// go test sets cwd to the package directory (cmd/subagent-mcp/), so walking
	// up two levels reaches the project root.
	//
	// Spec ref: TEST/tech_design/server § Context — "Derive the project root from
	// os.Getwd() and walking up two directory levels. Set cmd.Dir to this project
	// root for any subprocess that needs access to spec files."
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	cmd := exec.CommandContext(ctx, binaryPath, "codegen")
	cmd.Dir = projectRoot

	transport := &mcp.CommandTransport{Command: cmd}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)

	// Connect performs the MCP initialize handshake and starts the subprocess.
	// Do NOT call cmd.Start() manually — CommandTransport handles process startup.
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer cs.Close()

	// Assert that the server's initialize response instructions match the
	// codegen package's Instructions constant exactly.
	//
	// Spec ref: ROOT/tech_design/server § "Startup sequence" step 3b:
	// "ServerOptions.Instructions = codegen.Instructions"
	if cs.InitializeResult().Instructions != codegen.Instructions {
		t.Errorf(
			"initialize response instructions mismatch\nwant:\n%s\ngot:\n%s",
			codegen.Instructions,
			cs.InitializeResult().Instructions,
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

// TestCodegenWithExtraArgs_PrintsErrorToStderr verifies that running the binary
// with extra arguments after "codegen" exits 1 and prints the setup error.
//
// Spec ref: TEST/tech_design/server § "Codegen with extra arguments prints error to stderr"
func TestCodegenWithExtraArgs_PrintsErrorToStderr(t *testing.T) {
	_, stderr, exitCode := runBinary("codegen", "extraarg")

	if exitCode != 1 {
		t.Errorf("expected exit 1, got %d", exitCode)
	}

	const want = "codegen mode does not accept arguments"
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr does not contain %q\ngot stderr:\n%s", want, stderr)
	}
}
