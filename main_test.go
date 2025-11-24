package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
