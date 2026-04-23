// spec: TEST/tech_design/server@v17

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath holds the path to the compiled binary, built once in TestMain.
var binaryPath string

// TestMain builds the binary into a temp directory before running tests.
// On Windows, the binary name includes the .exe extension.
func TestMain(m *testing.M) {
	// Create a temp directory for the test binary.
	tmpDir, err := os.MkdirTemp("", "subagent-mcp-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	// Determine binary name with platform-appropriate extension.
	binaryName := "subagent-mcp"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath = filepath.Join(tmpDir, binaryName)

	// Build the binary from the current package directory.
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// usageSnippet is a substring of the usage message that we check for
// in stdout or stderr to confirm the usage message was printed.
const usageSnippet = "Usage: subagent-mcp"

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
		t.Errorf("stdout does not contain usage message.\nstdout: %s", stdout)
	}
}

// TestHelpWord verifies that "help" prints usage to stdout and exits 0.
func TestHelpWord(t *testing.T) {
	cmd := exec.Command(binaryPath, "help")
	stdout, err := cmd.Output()

	// "help" should exit 0, so err should be nil.
	if err != nil {
		t.Fatalf("expected exit 0 for help, got error: %v", err)
	}

	if !strings.Contains(string(stdout), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\nstdout: %s", stdout)
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
		t.Errorf("stdout does not contain usage message.\nstdout: %s", stdout)
	}
}

// --- Failure Cases ---

// TestUnrecognizedArgument verifies that an unrecognized argument
// prints usage to stderr and exits 1.
func TestUnrecognizedArgument(t *testing.T) {
	cmd := exec.Command(binaryPath, "something")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Expect a non-zero exit code.
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if !strings.Contains(stderr.String(), usageSnippet) {
		t.Errorf("stderr does not contain usage message.\nstderr: %s", stderr.String())
	}
}

// TestMultipleArguments verifies that multiple arguments print usage
// to stderr and exit 1.
func TestMultipleArguments(t *testing.T) {
	cmd := exec.Command(binaryPath, "foo", "bar")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Expect a non-zero exit code.
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if !strings.Contains(stderr.String(), usageSnippet) {
		t.Errorf("stderr does not contain usage message.\nstderr: %s", stderr.String())
	}
}
