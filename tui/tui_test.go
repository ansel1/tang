package tui

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
	teatest "github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"
)

// ScriptBlock represents a test block with input, expected output, and match type
type ScriptBlock struct {
	Input     string
	Expected  string
	MatchType string // "contains" for ">>>", "equals" for "==="
}

// parseScriptFile reads a file and parses it into a []ScriptBlock slice.
// The file format consists of blocks separated by "###" lines.
// Each block contains two strings separated by either "===" or ">>>" line.
// "===" indicates exact match, ">>>" indicates contains match.
func parseScriptFile(filename string) ([]ScriptBlock, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var result []ScriptBlock
	var currentInput []string
	var currentExpected []string
	var currentMatchType string
	mode := "input" // "input" or "expected"

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch trimmed {
		case "###":
			// Start of a new block - save previous block if it exists
			if len(currentInput) > 0 && len(currentExpected) > 0 {
				result = append(result, ScriptBlock{
					Input:     strings.Join(currentInput, "\n"),
					Expected:  strings.Join(currentExpected, "\n"),
					MatchType: currentMatchType,
				})
			}
			currentInput = nil
			currentExpected = nil
			currentMatchType = ""
			mode = "input"
		case "===":
			// Separator for exact match
			mode = "expected"
			currentMatchType = "equals"
		case ">>>":
			// Separator for contains match
			mode = "expected"
			currentMatchType = "contains"
		default:
			// Regular line - add to current section
			if mode == "input" {
				currentInput = append(currentInput, line)
			} else if mode == "expected" {
				currentExpected = append(currentExpected, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Don't forget the last block if file doesn't end with ###
	if len(currentInput) > 0 && len(currentExpected) > 0 {
		result = append(result, ScriptBlock{
			Input:     strings.Join(currentInput, "\n"),
			Expected:  strings.Join(currentExpected, "\n"),
			MatchType: currentMatchType,
		})
	}

	return result, nil
}

// TestHierarchicalRendering validates the new hierarchical TUI format
func TestHierarchicalRendering(t *testing.T) {
	collector := NewSummaryCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	// Simulate test events for a complete test run
	events := []parser.TestEvent{
		{
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Output:  "=== RUN   TestFoo\n",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Output:  "    foo_test.go:10: Test output\n",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Output:  "--- PASS: TestFoo (0.05s)\n",
		},
		{
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Elapsed: 0.05,
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok\tgithub.com/test/pkg1\t0.10s\n",
		},
		{
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.10,
		},
	}

	for _, evt := range events {
		_, _ = m.Update(evt)
	}

	// Mark as finished to get final summary instead of running status
	m.Finished = true

	output := m.String()

	// Verify key aspects of the output
	// Note: With the new collapsed view, completed packages only show their summary line
	// Individual tests and their output are NOT displayed for completed packages
	tests := []struct {
		name     string
		contains string
	}{
		{"Package name", "github.com/test/pkg1"},
		// {"Test name", "TestFoo"},
		// {"Test output", "foo_test.go:10"},
		// {"Status indicator", "✓"}, // Status indicator removed in favor of tabular layout
		{"Pass count", "✓ 1"},
		{"Separator line", "--------"},
	}

	for _, test := range tests {
		if !strings.Contains(output, test.contains) {
			t.Errorf("Expected output to contain '%s' (%s).\nGot:\n%s", test.contains, test.name, output)
		}
	}

	// Verify that test details are NOT shown for completed packages
	if strings.Contains(output, "foo_test.go:10") {
		t.Errorf("Test output should NOT be shown for completed packages.\nGot:\n%s", output)
	}
	if strings.Contains(output, "TestFoo") {
		t.Errorf("Test names should NOT be shown for completed packages.\nGot:\n%s", output)
	}
}

// TestRunningPackagesShowTests verifies that running packages display their individual tests
func TestRunningPackagesShowTests(t *testing.T) {
	collector := NewSummaryCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	// Simulate a running test (not yet completed)
	events := []parser.TestEvent{
		{
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestBar",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestBar",
			Output:  "=== RUN   TestBar\n",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestBar",
			Output:  "    bar_test.go:20: Running long test\n",
		},
		// Note: No "pass" action - test is still running
	}

	for _, evt := range events {
		_, _ = m.Update(evt)
	}

	output := m.String()

	// For running packages, we SHOULD see test details
	tests := []struct {
		name     string
		contains string
	}{
		{"Package name", "github.com/test/pkg1"},
		{"Test name", "TestBar"},
		{"Test output", "bar_test.go:20"},
		{"Running indicator", "RUNNING:"},
		{"Running count", "1 running"},
	}

	for _, test := range tests {
		if !strings.Contains(output, test.contains) {
			t.Errorf("Expected output to contain '%s' (%s) for running package.\nGot:\n%s", test.contains, test.name, output)
		}
	}
}

func TestParseScriptFile(t *testing.T) {
	result, err := parseScriptFile("testdata/script1")
	if err != nil {
		t.Fatalf("Failed to parse script file: %v", err)
	}

	// Check that we got the expected number of blocks
	expectedBlocks := 4
	if len(result) != expectedBlocks {
		t.Errorf("Expected %d blocks, got %d", expectedBlocks, len(result))
	}

	// Verify each block has required fields
	for i, block := range result {
		if block.Input == "" {
			t.Errorf("Block %d: input is empty", i)
		}
		if block.Expected == "" {
			t.Errorf("Block %d: expected is empty", i)
		}
		if block.MatchType == "" {
			t.Errorf("Block %d: matchType is empty", i)
		}
		// script1 should use "contains" match type
		if block.MatchType != "contains" {
			t.Errorf("Block %d: expected matchType 'contains', got '%s'", i, block.MatchType)
		}
	}

	// Test a specific block to verify parsing correctness
	if len(result) > 0 {
		firstInput := result[0].Input
		firstOutput := result[0].Expected

		// The first block should contain the build-output JSON
		if !strings.Contains(firstInput, `"Action":"build-output"`) {
			t.Errorf("First block input doesn't contain expected JSON content")
		}
		if !strings.Contains(firstOutput, "# github.com/ansel1/tang/tui") {
			t.Errorf("First block output doesn't contain expected content")
		}
	}
}

func TestRunScripts(t *testing.T) {
	t.Skip("Skipping old flat-format tests - replaced with new hierarchical format per spec")
	// The old test scripts were designed for the flat output format.
	// The new enhanced TUI uses hierarchical output format grouped by package.
	// See TestHierarchicalRendering for validation of the new format.
}

// TestRunScript1WithTeatest demonstrates using the teatest package
// to test the TUI model similar to TestRunScripts/script1
func TestRunScript1WithTeatest(t *testing.T) {
	t.Skip("not working: the output from the tea program seems to have wierd whitespace issues.")
	// Create a new model
	collector := NewSummaryCollector()
	m := NewModel(false, 1.0, collector)

	// Create a teatest model that wraps our model
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Parse the first block from script1
	blocks, err := parseScriptFile("testdata/script1")
	require.NoError(t, err)
	require.NotEmpty(t, blocks)

	// Get the first block
	block := blocks[0]

	// Parse and send each test event from the input
	lines := strings.Split(block.Input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		testEvent := parser.TestEvent{}
		err := json.Unmarshal([]byte(line), &testEvent)
		require.NoError(t, err)

		// Send the event to the model using teatest's Send method
		tm.Send(testEvent)
	}

	// Wait for the output to contain the expected string
	// Note: We check each line individually because the output includes ANSI codes
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			output := string(bts)
			return strings.Contains(output, block.Expected)

			// Check if output contains all expected lines
			// expectedLines := strings.Split(strings.TrimSpace(block.Expected), "\n")
			// for _, line := range expectedLines {
			// 	if !strings.Contains(output, line) {
			// 		return false
			// 	}
			// }
			// return true
		},
		teatest.WithDuration(1*time.Second),
		teatest.WithCheckInterval(50*time.Millisecond),
	)

	// Quit the program
	err = tm.Quit()
	require.NoError(t, err)
}

// TestRunScriptsWithTeatest re-implements TestRunScripts using the teatest package
// It tests all script files in testdata directory using a real Bubbletea program
func TestRunScriptsWithTeatest(t *testing.T) {
	t.Skip("not working")
	// Read all files in testdata directory
	entries, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		t.Run(filename, func(t *testing.T) {
			blocks, err := parseScriptFile("testdata/" + filename)
			if err != nil {
				t.Fatalf("Failed to parse script file: %v", err)
			}

			collector := NewSummaryCollector()
			m := NewModel(false, 1.0, collector)
			// Create a teatest model that wraps our model
			tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
			t.Cleanup(func() {
				// Quit the program

				if err := tm.Quit(); err != nil {
					t.Fatal(err)
				}
			})
			for _, block := range blocks {
				input := block.Input
				expected := block.Expected
				matchType := block.MatchType

				// Parse and send each test event from the input
				lines := strings.Split(input, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					testEvent := parser.TestEvent{}
					err := json.Unmarshal([]byte(line), &testEvent)
					require.NoError(t, err)

					// Send the event to the model using teatest's Send method
					tm.Send(testEvent)
				}

				// Wait for and verify the output based on match type
				teatest.WaitFor(
					t,
					tm.Output(),
					func(bts []byte) bool {
						output := string(bts)
						switch matchType {
						case "contains":
							return strings.Contains(output, expected)
						case "equals":
							return strings.TrimSpace(output) == expected
						default:
							t.Fatalf("Unknown match type: %s", matchType)
							return false
						}
					},
					teatest.WithDuration(1*time.Second),
					teatest.WithCheckInterval(50*time.Millisecond),
				)
			}

		})
	}
}
