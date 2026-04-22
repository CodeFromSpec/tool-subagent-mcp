// spec: TEST/tech_design/server@v10
package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// binaryPath holds the path to the compiled subagent-mcp binary built once in TestMain.
// Spec ref: TEST/tech_design/server § "Context"
var binaryPath string

// TestMain builds the binary into a temp directory once before all tests run.
// On Windows the binary must have the .exe extension.
// Spec ref: TEST/tech_design/server § "Context"
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "subagent-mcp-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(dir)

	// Determine binary name — append .exe on Windows.
	// Spec ref: TEST/tech_design/server § "Context"
	binName := "subagent-mcp"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binaryPath = filepath.Join(dir, binName)

	// Build the binary from the module root (two levels up from cmd/subagent-mcp).
	// go test runs with cwd = package directory, so use the module path.
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Stdout = os.Stderr // build output → stderr so test output stays clean
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// runBinary executes the compiled binary with the given arguments and returns
// stdout, stderr, and the *exec.ExitError (nil if exit 0).
func runBinary(args ...string) (stdout, stderr string, exitErr *exec.ExitError) {
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var ok bool
		exitErr, ok = err.(*exec.ExitError)
		if !ok {
			panic("unexpected error type: " + err.Error())
		}
	}
	return outBuf.String(), errBuf.String(), exitErr
}

// --- Happy Path ---

// TestHelpFlagPrintsUsageToStdout runs the binary with --help and expects
// exit 0 and the usage message on stdout.
// Spec ref: TEST/tech_design/server § "Help flag prints usage to stdout"
func TestHelpFlagPrintsUsageToStdout(t *testing.T) {
	stdout, _, exitErr := runBinary("--help")
	if exitErr != nil {
		t.Fatalf("expected exit 0, got exit error: %v", exitErr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Usage: subagent-mcp")) {
		t.Errorf("expected stdout to contain usage message, got: %q", stdout)
	}
}

// TestHelpWordPrintsUsageToStdout runs the binary with "help" and expects
// exit 0 and the usage message on stdout.
// Spec ref: TEST/tech_design/server § "Help word prints usage to stdout"
func TestHelpWordPrintsUsageToStdout(t *testing.T) {
	stdout, _, exitErr := runBinary("help")
	if exitErr != nil {
		t.Fatalf("expected exit 0, got exit error: %v", exitErr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Usage: subagent-mcp")) {
		t.Errorf("expected stdout to contain usage message, got: %q", stdout)
	}
}

// TestShortHelpFlagPrintsUsageToStdout runs the binary with -h and expects
// exit 0 and the usage message on stdout.
// Spec ref: TEST/tech_design/server § "Short help flag prints usage to stdout"
func TestShortHelpFlagPrintsUsageToStdout(t *testing.T) {
	stdout, _, exitErr := runBinary("-h")
	if exitErr != nil {
		t.Fatalf("expected exit 0, got exit error: %v", exitErr)
	}
	if !bytes.Contains([]byte(stdout), []byte("Usage: subagent-mcp")) {
		t.Errorf("expected stdout to contain usage message, got: %q", stdout)
	}
}

// --- Failure Cases ---

// TestUnrecognizedArgumentPrintsUsageToStderr runs the binary with an unknown
// argument and expects exit 1 and the usage message on stderr.
// Spec ref: TEST/tech_design/server § "Unrecognized argument prints usage to stderr"
func TestUnrecognizedArgumentPrintsUsageToStderr(t *testing.T) {
	_, stderr, exitErr := runBinary("something")
	if exitErr == nil {
		t.Fatal("expected exit 1, got exit 0")
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
	if !bytes.Contains([]byte(stderr), []byte("Usage: subagent-mcp")) {
		t.Errorf("expected stderr to contain usage message, got: %q", stderr)
	}
}

// TestMultipleArgumentsPrintsUsageToStderr runs the binary with multiple
// arguments and expects exit 1 and the usage message on stderr.
// Spec ref: TEST/tech_design/server § "Multiple arguments prints usage to stderr"
func TestMultipleArgumentsPrintsUsageToStderr(t *testing.T) {
	_, stderr, exitErr := runBinary("foo", "bar")
	if exitErr == nil {
		t.Fatal("expected exit 1, got exit 0")
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
	if !bytes.Contains([]byte(stderr), []byte("Usage: subagent-mcp")) {
		t.Errorf("expected stderr to contain usage message, got: %q", stderr)
	}
}
