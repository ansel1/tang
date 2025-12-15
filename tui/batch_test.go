package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
)

func TestEventBatching(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	now := time.Now()

	// Create a batch of events
	events := []engine.Event{
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now,
				Action:  "start",
				Package: "github.com/test/batch",
			},
		},
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(10 * time.Millisecond),
				Action:  "run",
				Package: "github.com/test/batch",
				Test:    "TestBatch1",
			},
		},
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(20 * time.Millisecond),
				Action:  "output",
				Package: "github.com/test/batch",
				Test:    "TestBatch1",
				Output:  "running batch 1\n",
			},
		},
		{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(30 * time.Millisecond),
				Action:  "pass",
				Package: "github.com/test/batch",
				Test:    "TestBatch1",
			},
		},
	}

	// Send as a single batch message
	m.Update(EngineEventBatchMsg(events))

	// Verify state
	output := viewLatest(m)

	// Output should contain the package and the test result
	if !strings.Contains(output, "github.com/test/batch") {
		t.Error("Expected output to contain package name")
	}
	if !strings.Contains(output, "TestBatch1") {
		t.Error("Expected output to contain test name")
	}

	// Check internal state directly via collector to ensure all events were processed
	state := collector.State()
	if len(state.Runs) == 0 {
		t.Fatal("Expected runs in collector state")
	}
	run := state.Runs[len(state.Runs)-1]
	pkg := run.Packages["github.com/test/batch"]
	if pkg == nil {
		t.Fatal("Expected package to be tracked")
	}
	if pkg.Counts.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.Counts.Passed)
	}
}
