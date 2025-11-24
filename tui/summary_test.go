package tui

import (
	"strings"
	"testing"

	"github.com/ansel1/tang/parser"
)

func TestPackageSummaryLastOutput(t *testing.T) {
	m := NewModel(false, 1.0)
	m.TerminalWidth = 80

	events := []parser.TestEvent{
		{
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok  \tgithub.com/test/pkg1\t0.10s\n",
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

	output := m.String()

	// The output should contain the last output line (with tabs expanded)
	// The original line is "ok  \tgithub.com/test/pkg1\t0.10s"
	// After tab expansion, it becomes "ok      github.com/test/pkg1    0.10s"
	expected := "ok      github.com/test/pkg1    0.10s"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain last output line '%s'.\nGot:\n%s", expected, output)
	}

	// It should NOT contain the package name as the left part (although the output line contains it,
	// we want to ensure the *summary line* is using the output line.
	// Since the output line contains the package name, checking for the output line is sufficient
	// to prove we are using it, provided the package name alone wouldn't match the full line check).
}
