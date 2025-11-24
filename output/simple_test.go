package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleOutput_ProcessEvents_BasicTest(t *testing.T) {
	// Create summary collector and feed it events
	summaryCollector := tui.NewSummaryCollector()
	summaryEvents := make(chan engine.Event, 10)

	// Start summary collector in background
	go summaryCollector.ProcessEvents(summaryEvents)

	// Send test events to summary collector
	summaryEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestFoo",
		},
	}
	summaryEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "output",
			Package: "example.com/pkg",
			Test:    "TestFoo",
			Output:  "=== RUN   TestFoo\n",
		},
	}
	summaryEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "pass",
			Package: "example.com/pkg",
			Test:    "TestFoo",
			Elapsed: 0.5,
		},
	}
	summaryEvents <- engine.Event{Type: engine.EventComplete}
	close(summaryEvents)

	// Give summary collector time to process
	time.Sleep(10 * time.Millisecond)

	// Create simple output with the collector
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, summaryCollector)

	// Create events for simple output
	events := make(chan engine.Event, 10)
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "output",
			Package: "example.com/pkg",
			Test:    "TestFoo",
			Output:  "=== RUN   TestFoo\n",
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	// Check that output contains test output
	assert.Contains(t, output, "=== RUN   TestFoo")
	// Check that summary is displayed
	assert.Contains(t, output, "OVERALL RESULTS")
	assert.Contains(t, output, "Total tests:")
}

func TestSimpleOutput_ProcessEvents_FailedTest(t *testing.T) {
	// Create summary collector
	summaryCollector := tui.NewSummaryCollector()
	summaryEvents := make(chan engine.Event, 10)

	go summaryCollector.ProcessEvents(summaryEvents)

	summaryEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestFail",
		},
	}
	summaryEvents <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "fail",
			Package: "example.com/pkg",
			Test:    "TestFail",
			Elapsed: 0.5,
		},
	}
	summaryEvents <- engine.Event{Type: engine.EventComplete}
	close(summaryEvents)

	time.Sleep(10 * time.Millisecond)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, summaryCollector)

	events := make(chan engine.Event, 10)
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "output",
			Package: "example.com/pkg",
			Test:    "TestFail",
			Output:  "    test_fail.go:10: assertion failed\n",
		},
	}
	// Simple output needs to see the fail event to track failures
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "fail",
			Package: "example.com/pkg",
			Test:    "TestFail",
			Elapsed: 0.5,
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "assertion failed")
	assert.Contains(t, output, "OVERALL RESULTS")
	// Should have failures
	assert.True(t, simple.HasFailures())
}

func TestSimpleOutput_ProcessEvents_RawLines(t *testing.T) {
	summaryCollector := tui.NewSummaryCollector()
	summaryEvents := make(chan engine.Event, 10)

	go summaryCollector.ProcessEvents(summaryEvents)
	summaryEvents <- engine.Event{Type: engine.EventComplete}
	close(summaryEvents)

	time.Sleep(10 * time.Millisecond)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, summaryCollector)

	events := make(chan engine.Event, 10)
	events <- engine.Event{
		Type:    engine.EventRawLine,
		RawLine: []byte("This is a raw line"),
	}
	events <- engine.Event{
		Type:    engine.EventRawLine,
		RawLine: []byte("Another raw line"),
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "This is a raw line")
	assert.Contains(t, output, "Another raw line")
}

func TestSimpleOutput_HasFailures(t *testing.T) {
	summaryCollector := tui.NewSummaryCollector()

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, summaryCollector)

	// Initially no failures
	assert.False(t, simple.HasFailures())

	// Process a failure event
	events := make(chan engine.Event, 10)
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "fail",
			Package: "pkg",
			Test:    "Test1",
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	simple.ProcessEvents(events)

	// Now should have failures
	assert.True(t, simple.HasFailures())
}
