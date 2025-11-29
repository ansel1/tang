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

	// Setup collector processing
	engineEvents := make(chan engine.Event, 10)
	go collector.ProcessEvents(engineEvents)
	resultsEvents := collector.Subscribe()

	// Feed engine events
	now := time.Now()

	// 1. Start package
	engineEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now,
			Action:  "start", // or run
			Package: "github.com/test/pkg1",
		},
	}

	// 2. Output
	engineEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now.Add(100 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok  \tgithub.com/test/pkg1\t0.10s\n",
		},
	}

	// 3. Pass
	engineEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now.Add(200 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.10,
		},
	}

	// Close engine stream
	close(engineEvents)

	// Process results events in TUI
	// We need to wait for events to propagate
	for evt := range resultsEvents {
		m.Update(ResultsEventMsg(evt))
		if evt.Type == results.EventRunFinished {
			break
		}
	}

	output := m.View()

	// The output should contain the last output line (with tabs expanded)
	// The original line is "ok  \tgithub.com/test/pkg1\t0.10s"
	// After tab expansion, it becomes "ok      github.com/test/pkg1    0.10s"
	expected := "ok      github.com/test/pkg1    0.10s"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain last output line '%s'.\nGot:\n%s", expected, output)
	}
}
