package tui

import (
	"strings"
	"testing"
)

func TestBleed(t *testing.T) {
	m := NewModel(false, 1.0)
	m.TerminalWidth = 80
	m.TerminalHeight = 20

	// Create a package and test
	pkg := NewPackageState("pkg1")
	m.Packages["pkg1"] = pkg
	m.PackageOrder = append(m.PackageOrder, "pkg1")

	test := NewTestState("TestBleed", "pkg1")
	test.Status = "running"
	test.SummaryLine = "=== RUN   TestBleed"
	// Add a line with an open color code (Red)
	test.AddOutputLine("\033[31mThis is red text")
	pkg.Tests["TestBleed"] = test
	pkg.TestOrder = append(pkg.TestOrder, "TestBleed")
	pkg.Running++
	m.Running++

	// Render
	output := m.View()

	// Check if the output line is followed by a reset sequence
	// The View() function renders lines. We expect the line containing "This is red text"
	// to be immediately followed by a reset sequence before the newline or next content.
	// However, since we are appending it in renderTest, let's check for the presence of the reset sequence.

	// In the current (buggy) implementation, the line is just "    \033[31mThis is red text\n"
	// In the fixed implementation, it should be "    \033[31mThis is red text\033[0m\n" (or similar)

	lines := strings.Split(output, "\n")
	var foundLine string
	for _, line := range lines {
		if strings.Contains(line, "This is red text") {
			foundLine = line
			break
		}
	}

	if foundLine == "" {
		t.Fatal("Could not find the test output line in View()")
	}

	if !strings.Contains(foundLine, "\033[0m") {
		t.Errorf("Expected line to contain reset sequence, but got: %q", foundLine)
	}
}
