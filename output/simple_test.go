package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/ansel1/tang/results"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleOutput_ProcessEvents_BasicTest(t *testing.T) {
	// Create collector and manually populate state for summary
	collector := results.NewCollector()
	state := collector.GetState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	// Populate run with some data so summary is generated
	pkg := &results.PackageResult{
		Name:    "example.com/pkg",
		Status:  results.StatusPassed,
		Elapsed: 100 * time.Millisecond,
	}
	pkg.Counts.Passed = 1
	run.Packages["example.com/pkg"] = pkg
	run.PackageOrder = append(run.PackageOrder, "example.com/pkg")

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector)

	// Create events for simple output
	events := make(chan results.Event, 10)
	events <- results.NewTestOutputEvent(1, "example.com/pkg", "TestFoo", "=== RUN   TestFoo\n")
	events <- results.NewRunFinishedEvent(1)
	close(events)

	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	// Check that output contains test output
	assert.Contains(t, output, "=== RUN   TestFoo")
	// Check that summary is displayed
	assert.Contains(t, output, "OVERALL RESULTS")
}

func TestSimpleOutput_ProcessEvents_FailedTest(t *testing.T) {
	collector := results.NewCollector()
	state := collector.GetState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	// Populate run with failure
	pkg := &results.PackageResult{
		Name:    "example.com/pkg",
		Status:  results.StatusFailed,
		Elapsed: 100 * time.Millisecond,
	}
	pkg.Counts.Failed = 1
	run.Packages["example.com/pkg"] = pkg
	run.PackageOrder = append(run.PackageOrder, "example.com/pkg")

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector)

	events := make(chan results.Event, 10)
	events <- results.NewTestOutputEvent(1, "example.com/pkg", "TestFail", "    test_fail.go:10: assertion failed\n")
	// We need to emit TestUpdatedEvent or similar?
	// SimpleOutput only cares about TestOutputEvent for printing.
	// But HasFailures checks collector state.
	events <- results.NewRunFinishedEvent(1)
	close(events)

	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "assertion failed")
	assert.Contains(t, output, "OVERALL RESULTS")
	// Should have failures (checked via collector state)
	assert.True(t, simple.HasFailures())
}

func TestSimpleOutput_ProcessEvents_RawLines(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector)

	events := make(chan results.Event, 10)
	events <- results.NewRawOutputEvent(1, []byte("This is a raw line"))
	events <- results.NewRawOutputEvent(1, []byte("Another raw line"))
	events <- results.NewRunFinishedEvent(1)
	close(events)

	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "This is a raw line")
	assert.Contains(t, output, "Another raw line")
}

func TestSimpleOutput_HasFailures(t *testing.T) {
	collector := results.NewCollector()
	state := collector.GetState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector)

	// Initially no failures
	assert.False(t, simple.HasFailures())

	// Manually add failure to collector state
	pkg := &results.PackageResult{
		Name:   "pkg",
		Status: results.StatusFailed,
	}
	pkg.Counts.Failed = 1
	run.Packages["pkg"] = pkg

	// Now should have failures
	assert.True(t, simple.HasFailures())
}
