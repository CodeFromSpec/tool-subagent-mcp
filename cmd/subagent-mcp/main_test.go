// spec: TEST/tech_design/server@v19

package main

import (
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
