package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	return runTangCommandWithStdin(t, tangBinary, "", args...)
}

func runTangCommandWithStdin(t *testing.T, tangBinary string, stdin string, args ...string) (int, string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, tangBinary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
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

func TestTextfileFlag(t *testing.T) {
	tangBinary := buildTangBinary(t)
	tmpDir := t.TempDir()
	textfile := filepath.Join(tmpDir, "tang.txt")

	input := `{"Time":"2025-11-01T15:43:02.993511-05:00","Action":"start","Package":"github.com/example/test"}
{"Time":"2025-11-01T15:43:02.993565-05:00","Action":"run","Package":"github.com/example/test","Test":"TestExample"}
{"Time":"2025-11-01T15:43:02.993570-05:00","Action":"output","Package":"github.com/example/test","Test":"TestExample","Output":"=== RUN   TestExample\n"}
{"Time":"2025-11-01T15:43:02.993575-05:00","Action":"output","Package":"github.com/example/test","Test":"TestExample","Output":"    example_test.go:10: verbose log\n"}
{"Time":"2025-11-01T15:43:02.993579-05:00","Action":"pass","Package":"github.com/example/test","Test":"TestExample","Elapsed":0.001}
{"Time":"2025-11-01T15:43:02.993585-05:00","Action":"output","Package":"github.com/example/test","Output":"PASS\n"}
{"Time":"2025-11-01T15:43:02.993587-05:00","Action":"output","Package":"github.com/example/test","Output":"ok  \tgithub.com/example/test\t0.002s\n"}
{"Time":"2025-11-01T15:43:02.993590-05:00","Action":"pass","Package":"github.com/example/test","Elapsed":0.002}`

	exitCode, stdout, stderr := runTangCommandWithStdin(t, tangBinary, input, "-notty", "-textfile", textfile)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)

	content, err := os.ReadFile(textfile)
	require.NoError(t, err)
	require.Equal(t, stdout, string(content))
	require.Contains(t, stdout, "ok  \tgithub.com/example/test")
	require.NotContains(t, stdout, "verbose log")
}

func TestTextfileFlagVerbose(t *testing.T) {
	tangBinary := buildTangBinary(t)
	tmpDir := t.TempDir()
	textfile := filepath.Join(tmpDir, "tang-verbose.txt")

	input := `{"Time":"2025-11-01T15:43:02.993511-05:00","Action":"start","Package":"github.com/example/test"}
{"Time":"2025-11-01T15:43:02.993565-05:00","Action":"run","Package":"github.com/example/test","Test":"TestExample"}
{"Time":"2025-11-01T15:43:02.993570-05:00","Action":"output","Package":"github.com/example/test","Test":"TestExample","Output":"=== RUN   TestExample\n"}
{"Time":"2025-11-01T15:43:02.993575-05:00","Action":"output","Package":"github.com/example/test","Test":"TestExample","Output":"    example_test.go:10: verbose log\n"}
{"Time":"2025-11-01T15:43:02.993579-05:00","Action":"pass","Package":"github.com/example/test","Test":"TestExample","Elapsed":0.001}
{"Time":"2025-11-01T15:43:02.993585-05:00","Action":"output","Package":"github.com/example/test","Output":"PASS\n"}
{"Time":"2025-11-01T15:43:02.993587-05:00","Action":"output","Package":"github.com/example/test","Output":"ok  \tgithub.com/example/test\t0.002s\n"}
{"Time":"2025-11-01T15:43:02.993590-05:00","Action":"pass","Package":"github.com/example/test","Elapsed":0.002}`

	exitCode, stdout, stderr := runTangCommandWithStdin(t, tangBinary, input, "-notty", "-v", "-textfile", textfile)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)

	content, err := os.ReadFile(textfile)
	require.NoError(t, err)
	require.Equal(t, stdout, string(content))
	require.Contains(t, stdout, "=== RUN   TestExample")
	require.Contains(t, stdout, "verbose log")
}

func TestTextfileRejectsDuplicateOutputPath(t *testing.T) {
	tangBinary := buildTangBinary(t)
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "same.out")

	exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "-outfile", out, "-textfile", out)
	require.Equal(t, 1, exitCode)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "must write to different files")
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

func TestErrorRecordingWriter(t *testing.T) {
	t.Run("records first write error", func(t *testing.T) {
		boom := errors.New("boom")
		writer := &errorRecordingWriter{writer: failingWriter{err: boom}}

		n, err := writer.Write([]byte("hello"))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, boom)
		require.ErrorIs(t, writer.Err(), boom)

		n, err = writer.Write([]byte("again"))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, boom)
	})

	t.Run("records short write", func(t *testing.T) {
		writer := &errorRecordingWriter{writer: shortWriter{}}

		n, err := writer.Write([]byte("hello"))
		require.Equal(t, 4, n)
		require.ErrorIs(t, err, io.ErrShortWrite)
		require.ErrorIs(t, writer.Err(), io.ErrShortWrite)
	})
}

func TestTangTestSubcommand(t *testing.T) {
	tangBinary := buildTangBinary(t)

	t.Run("passes through notty test invocation", func(t *testing.T) {
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "test", "-count", "1", "-run", "TestOutfileFlag", ".")
		require.Equal(t, 0, exitCode)
		require.Empty(t, stderr)
		require.Contains(t, stdout, "github.com/ansel1/tang")
		require.Contains(t, stdout, "ok")
	})

	t.Run("supports verbose output", func(t *testing.T) {
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "test", "-v", "-count", "1", "-run", "TestOutfileFlag", ".")
		require.Equal(t, 0, exitCode)
		require.Empty(t, stderr)
		require.Contains(t, stdout, "github.com/ansel1/tang")
		require.Contains(t, stdout, "PASS")
	})

	t.Run("supports textfile output", func(t *testing.T) {
		textfile := filepath.Join(t.TempDir(), "test-output.txt")
		exitCode, stdout, stderr := runTangCommand(t, tangBinary, "-notty", "-textfile", textfile, "test", "-count", "1", "-run", "TestOutfileFlag", ".")
		require.Equal(t, 0, exitCode)
		require.Empty(t, stderr)

		content, err := os.ReadFile(textfile)
		require.NoError(t, err)
		require.Equal(t, stdout, string(content))
		require.Contains(t, stdout, "github.com/ansel1/tang")
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
		require.Contains(t, stderr, "error: -f is not compatible with 'test' subcommand")
	})
}

func TestTangPipeModeRegression(t *testing.T) {
	tangBinary := buildTangBinary(t)

	goTestCmd := exec.Command("go", "test", "-json", "-count", "1", "-run", "TestOutfileFlag", ".")
	tangCmd := exec.Command(tangBinary, "-notty")

	pipeReader, err := goTestCmd.StdoutPipe()
	require.NoError(t, err)
	tangCmd.Stdin = pipeReader

	// Each command needs its own buffer: os/exec copies stderr on a
	// per-command goroutine, and bytes.Buffer is not thread-safe.
	var tangStdout bytes.Buffer
	var tangStderr bytes.Buffer
	var goTestStderr bytes.Buffer
	goTestCmd.Stderr = &goTestStderr
	tangCmd.Stdout = &tangStdout
	tangCmd.Stderr = &tangStderr

	require.NoError(t, goTestCmd.Start())
	require.NoError(t, tangCmd.Start())
	require.NoError(t, goTestCmd.Wait())
	require.NoError(t, tangCmd.Wait())
	require.Contains(t, tangStdout.String(), "github.com/ansel1/tang")
	require.Contains(t, tangStdout.String(), "ok")
	require.Empty(t, tangStderr.String())
	require.Empty(t, goTestStderr.String())
}
