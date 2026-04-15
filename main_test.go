package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOutfileFlag(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	outfile := filepath.Join(tmpDir, "test_output.json")

	// Sample test JSON input
	input := `{"Time":"2025-11-01T15:43:02.993511-05:00","Action":"start","Package":"github.com/example/test"}
{"Time":"2025-11-01T15:43:02.993565-05:00","Action":"run","Package":"github.com/example/test","Test":"TestExample"}
{"Time":"2025-11-01T15:43:02.993579-05:00","Action":"pass","Package":"github.com/example/test","Test":"TestExample","Elapsed":0.001}
{"Time":"2025-11-01T15:43:02.993590-05:00","Action":"pass","Package":"github.com/example/test","Elapsed":0.002}`

	// Build the tang binary
	tangBinary := filepath.Join(tmpDir, "tang")
	buildCmd := exec.Command("go", "build", "-o", tangBinary, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build tang binary")

	// Run tang with -outfile flag
	cmd := exec.Command(tangBinary, "-outfile", outfile)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	require.NoError(t, err, "Failed to run tang with -outfile")

	// Verify the output file was created
	require.FileExists(t, outfile, "Output file should be created")

	// Read the output file
	content, err := os.ReadFile(outfile)
	require.NoError(t, err, "Failed to read output file")

	// Verify the content matches the input
	require.Equal(t, input, strings.TrimRight(string(content), "\n"), "Output file should contain all input lines")
}

func TestOutfileWithInvalidPath(t *testing.T) {
	// Try to write to an invalid path
	input := `{"Time":"2025-11-01T15:43:02.993511-05:00","Action":"start","Package":"github.com/example/test"}`

	tmpDir := t.TempDir()
	tangBinary := filepath.Join(tmpDir, "tang")

	// Build the tang binary
	buildCmd := exec.Command("go", "build", "-o", tangBinary, ".")
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build tang binary")

	// Try to write to a directory that doesn't exist
	invalidPath := "/nonexistent/directory/output.json"
	cmd := exec.Command(tangBinary, "-outfile", invalidPath)
	cmd.Stdin = strings.NewReader(input)

	err = cmd.Run()
	require.Error(t, err, "Should fail when output file path is invalid")
}

func buildTangBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	tangBinary := filepath.Join(tmpDir, "tang")
	buildCmd := exec.Command("go", "build", "-o", tangBinary, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	require.NoError(t, buildCmd.Run(), "Failed to build tang binary")

	return tangBinary
}

func runTangCommand(t *testing.T, tangBinary string, args ...string) (int, string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, tangBinary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		require.FailNow(t, fmt.Sprintf("command timed out: %s %s\nstdout: %s\nstderr: %s", tangBinary, strings.Join(args, " "), stdout.String(), stderr.String()))
	}
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}

	var exitErr *exec.ExitError
	if strings.Contains(err.Error(), "executable file not found") {
		require.NoError(t, err)
	}
	require.ErrorAs(t, err, &exitErr, "Expected process exit error")
	return exitErr.ExitCode(), stdout.String(), stderr.String()
}

func TestTangTestSubcommand(t *testing.T) {
	tangBinary := buildTangBinary(t)

	t.Run("passes through notty test invocation", func(t *testing.T) {
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "test", "-count", "1", "-run", "TestOutfileFlag", ".")
		require.Equal(t, 0, exitCode)
		require.Empty(t, stderr)
		require.Contains(t, stdout, "github.com/ansel1/tang")
		require.Contains(t, stdout, "PASS")
	})

	t.Run("supports verbose output", func(t *testing.T) {
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "test", "-v", "-count", "1", "-run", "TestOutfileFlag", ".")
		require.Equal(t, 0, exitCode)
		require.Empty(t, stderr)
		require.Contains(t, stdout, "github.com/ansel1/tang")
		require.Contains(t, stdout, "PASS")
	})

	t.Run("invalid package exits non zero", func(t *testing.T) {
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "test", "./nonexistent-package-xyz")
		require.Equal(t, 1, exitCode)
		require.Contains(t, stdout, "nonexistent-package-xyz")
		require.True(t, strings.Contains(stdout, "FAIL") || strings.Contains(stderr, "FAIL"), fmt.Sprintf("expected FAIL marker in stdout or stderr\nstdout: %s\nstderr: %s", stdout, stderr))
	})

	t.Run("rejects incompatible file flag", func(t *testing.T) {
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-f", "somefile", "test", "./...")
		require.Equal(t, 1, exitCode)
		require.Empty(t, stdout)
		require.Contains(t, stderr, "Error: -f is not compatible with 'test' subcommand")
	})
}

func TestTangPipeModeRegression(t *testing.T) {
	tangBinary := buildTangBinary(t)

	goTestCmd := exec.Command("go", "test", "-json", "-count", "1", "-run", "TestOutfileFlag", ".")
	tangCmd := exec.Command(tangBinary, "-notty")

	pipeReader, err := goTestCmd.StdoutPipe()
	require.NoError(t, err)
	tangCmd.Stdin = pipeReader

	var tangStdout bytes.Buffer
	var tangStderr bytes.Buffer
	goTestCmd.Stderr = &tangStderr
	tangCmd.Stdout = &tangStdout
	tangCmd.Stderr = &tangStderr

	require.NoError(t, goTestCmd.Start())
	require.NoError(t, tangCmd.Start())
	require.NoError(t, goTestCmd.Wait())
	require.NoError(t, tangCmd.Wait())
	require.Contains(t, tangStdout.String(), "github.com/ansel1/tang")
	require.Contains(t, tangStdout.String(), "PASS")
	require.Empty(t, tangStderr.String())
}
