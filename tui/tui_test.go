package tui

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ScriptBlock represents a test block with input, expected output, and match type
type ScriptBlock struct {
	Input     string
	Expected  string
	MatchType string // "contains" for ">>>", "equals" for "==="
}

// parseScriptFile reads a file and parses it into a []ScriptBlock slice.
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
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	// Simulate test events for a complete test run
	now := time.Now()
	events := []parser.TestEvent{
		{
			Time:    now,
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Time:    now.Add(10 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
		},
		{
			Time:    now.Add(20 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Output:  "=== RUN   TestFoo\n",
		},
		{
			Time:    now.Add(30 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Output:  "    foo_test.go:10: Test output\n",
		},
		{
			Time:    now.Add(40 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Output:  "--- PASS: TestFoo (0.05s)\n",
		},
		{
			Time:    now.Add(50 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestFoo",
			Elapsed: 0.05,
		},
		{
			Time:    now.Add(60 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok\tgithub.com/test/pkg1\t0.10s\n",
		},
		{
			Time:    now.Add(70 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.10,
		},
	}

	for _, evt := range events {
		m.Update(EngineEventMsg(engine.Event{Type: engine.EventTest, TestEvent: evt}))
	}

	output := viewLatest(m)

	// Verify key aspects of the output
	tests := []struct {
		name     string
		contains string
	}{
		{"Package name", "github.com/test/pkg1"},
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
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	// Simulate a running test (not yet completed)
	now := time.Now()
	events := []parser.TestEvent{
		{
			Time:    now,
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Time:    now.Add(10 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestBar",
		},
		{
			Time:    now.Add(20 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestBar",
			Output:  "=== RUN   TestBar\n",
		},
		{
			Time:    now.Add(30 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestBar",
			Output:  "    bar_test.go:20: Running long test\n",
		},
		// Note: No "pass" action - test is still running
	}

	for _, evt := range events {
		m.Update(EngineEventMsg(engine.Event{Type: engine.EventTest, TestEvent: evt}))
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
}

func TestRunScript1WithTeatest(t *testing.T) {
	t.Skip("not working: the output from the tea program seems to have wierd whitespace issues.")
}

func TestRunScriptsWithTeatest(t *testing.T) {
	t.Skip("not working")
}

// TestBoldRunningEntities verifies that running entities are bolded
func TestBoldRunningEntities(t *testing.T) {
	// Force lipgloss to emit ansi escape codes
	lipgloss.SetColorProfile(termenv.ANSI)

	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	// Simulate a running test
	now := time.Now()
	events := []parser.TestEvent{
		{
			Time:    now,
			Action:  "start",
			Package: "github.com/test/running-pkg",
		},
		{
			Time:    now.Add(10 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/running-pkg",
			Test:    "TestRunning",
		},
		{
			Time:    now.Add(20 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/running-pkg",
			Test:    "TestRunning",
			Output:  "=== RUN   TestRunning\n",
		},
	}

	for _, evt := range events {
		m.Update(EngineEventMsg(engine.Event{Type: engine.EventTest, TestEvent: evt}))
	}

	output := m.String()

	// Define ANSI bold code
	const bold = "\x1b[1m"

	// Check for bolded elements
	// 1. Package name should be bolded
	if !strings.Contains(output, bold+"github.com/test/running-pkg") {
		t.Errorf("Expected running package name to be bolded.\nGot:\n%s", output)
	}

	// 2. Test name should be bolded
	if !strings.Contains(output, bold+"  === RUN   TestRunning") {
		t.Errorf("Expected running test summary to be bolded.\nGot:\n%s", output)
	}

	// 3. Summary line should be bolded (RUNNING status)
	// The summary format is "RUNNING: X passed, ..."
	if !strings.Contains(output, bold+"RUNNING:") {
		t.Errorf("Expected summary line to be bolded.\nGot:\n%s", output)
	}
}
