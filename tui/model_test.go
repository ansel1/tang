package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
)

func TestPackageSummaryLastOutput(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	now := time.Now()

	events := []engine.Event{
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now,
				Action:  "start",
				Package: "github.com/test/pkg1",
			},
		},
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(100 * time.Millisecond),
				Action:  "output",
				Package: "github.com/test/pkg1",
				Output:  "ok  \tgithub.com/test/pkg1\t0.10s\n",
			},
		},
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(200 * time.Millisecond),
				Action:  "pass",
				Package: "github.com/test/pkg1",
				Elapsed: 0.10,
			},
		},
		{Type: engine.EventComplete},
	}

	for _, evt := range events {
		collector.Push(evt)
	}

	output := viewLatest(m)

	// The original summary line is "ok  \tgithub.com/test/pkg1\t0.10s". For
	// finished packages, the leading status word is replaced with a gutter
	// icon (✓/✗/∅), so the rendered tail is "github.com/test/pkg1    0.10s"
	// after tab expansion.
	expected := "github.com/test/pkg1    0.10s"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain last output line '%s'.\nGot:\n%s", expected, output)
	}
	if strings.Contains(output, "ok      github.com/test/pkg1") {
		t.Errorf("Finished package line should not include the 'ok' status word; gutter icon replaces it.\nGot:\n%s", output)
	}
}
