// spec: TEST/tech_design/server@v14

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

// usageSnippet is a substring of the usage message used to verify
// that the binary printed the expected output.
const usageSnippet = "Usage: subagent-mcp"

// TestMain builds the binary into a temp directory so all tests
// can invoke it as a subprocess. On Windows the binary needs
// the .exe extension.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "subagent-mcp-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	// Determine binary name, appending .exe on Windows.
	name := "subagent-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binaryPath = filepath.Join(tmp, name)

	// Build the binary from the current package directory.
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// --- Happy Path ---

// TestHelpFlag verifies that --help prints usage to stdout and exits 0.
func TestHelpFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}
	if !strings.Contains(string(out), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\ngot: %s", string(out))
	}
}

// TestHelpWord verifies that "help" prints usage to stdout and exits 0.
func TestHelpWord(t *testing.T) {
	cmd := exec.Command(binaryPath, "help")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}
	if !strings.Contains(string(out), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\ngot: %s", string(out))
	}
}

// TestShortHelpFlag verifies that -h prints usage to stdout and exits 0.
func TestShortHelpFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "-h")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}
	if !strings.Contains(string(out), usageSnippet) {
		t.Errorf("stdout does not contain usage message.\ngot: %s", string(out))
	}
}

// --- Failure Cases ---

// TestUnrecognizedArgument verifies that an unknown argument prints
// usage to stderr and exits 1.
func TestUnrecognizedArgument(t *testing.T) {
	cmd := exec.Command(binaryPath, "something")
	// CombinedOutput is not used — we need to check stderr specifically.
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1, got exit 0")
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
	if !strings.Contains(stderrBuf.String(), usageSnippet) {
		t.Errorf("stderr does not contain usage message.\ngot: %s", stderrBuf.String())
	}
}

// TestMultipleArguments verifies that multiple arguments print
// usage to stderr and exit 1.
func TestMultipleArguments(t *testing.T) {
	cmd := exec.Command(binaryPath, "foo", "bar")
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1, got exit 0")
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
	if !strings.Contains(stderrBuf.String(), usageSnippet) {
		t.Errorf("stderr does not contain usage message.\ngot: %s", stderrBuf.String())
	}
}
