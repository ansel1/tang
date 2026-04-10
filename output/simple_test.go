package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var baseTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func passingPackageEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkg, Test: "TestFoo"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestFoo", Output: "=== RUN   TestFoo\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestFoo", Output: "--- PASS: TestFoo (0.00s)\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "pass", Package: pkg, Test: "TestFoo", Elapsed: 0.001}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "PASS\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "ok  \t" + pkg + "\t0.100s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "pass", Package: pkg, Elapsed: 0.1}},
	}
}

func failingPackageEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkg, Test: "TestFail"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestFail", Output: "=== RUN   TestFail\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestFail", Output: "    test_fail.go:10: assertion failed\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestFail", Output: "--- FAIL: TestFail (0.00s)\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkg, Test: "TestFail", Elapsed: 0.001}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "FAIL\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "FAIL\t" + pkg + "\t0.100s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkg, Elapsed: 0.1}},
	}
}

func sendEvents(events []engine.Event) <-chan engine.Event {
	ch := make(chan engine.Event, len(events)+1)
	for _, evt := range events {
		ch <- evt
	}
	ch <- engine.Event{Type: engine.EventComplete}
	close(ch)
	return ch
}

func TestSimpleOutput_Verbose_PassingTest(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, true, 80, false)

	err := simple.ProcessEvents(sendEvents(passingPackageEvents("example.com/pkg")))
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "=== RUN   TestFoo")
	assert.Contains(t, output, "--- PASS: TestFoo")
	assert.Contains(t, output, "(1 packages)")
	assert.False(t, simple.HasFailures())
}

func TestSimpleOutput_Verbose_FailedTest(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, true, 80, false)

	err := simple.ProcessEvents(sendEvents(failingPackageEvents("example.com/pkg")))
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "=== RUN   TestFail")
	assert.Contains(t, output, "assertion failed")
	assert.Contains(t, output, "--- FAIL: TestFail")
	assert.Contains(t, output, "(1 packages)")
	assert.True(t, simple.HasFailures())
}

func TestSimpleOutput_NonVerbose_PassingTest(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	err := simple.ProcessEvents(sendEvents(passingPackageEvents("example.com/pkg")))
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "=== RUN")
	assert.NotContains(t, output, "--- PASS")
	assert.Contains(t, output, "ok  \texample.com/pkg")
	assert.Contains(t, output, "(1 packages)")
}

func TestSimpleOutput_NonVerbose_FailedTest(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	err := simple.ProcessEvents(sendEvents(failingPackageEvents("example.com/pkg")))
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "=== RUN")
	assert.Contains(t, output, "assertion failed")
	assert.Contains(t, output, "--- FAIL: TestFail (0.00s)")
	assert.Contains(t, output, "FAIL\texample.com/pkg")
	assert.Contains(t, output, "(1 packages)")
	assert.True(t, simple.HasFailures())

	failIdx := strings.Index(output, "--- FAIL: TestFail")
	logIdx := strings.Index(output, "assertion failed")
	assert.Greater(t, logIdx, failIdx, "log output should come after --- FAIL line")
}

func TestSimpleOutput_NonVerbose_BuildError(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	events := []engine.Event{
		{Type: engine.EventBuild, BuildEvent: parser.BuildEvent{ImportPath: "example.com/broken", Action: "build-output", Output: "# example.com/broken\n"}},
		{Type: engine.EventBuild, BuildEvent: parser.BuildEvent{ImportPath: "example.com/broken", Action: "build-output", Output: "broken.go:7:1: syntax error\n"}},
		{Type: engine.EventBuild, BuildEvent: parser.BuildEvent{ImportPath: "example.com/broken", Action: "build-fail"}},
	}
	events = append(events, passingPackageEvents("example.com/ok")...)

	err := simple.ProcessEvents(sendEvents(events))
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "# example.com/broken")
	assert.Contains(t, output, "syntax error")
	assert.Contains(t, output, "ok  \texample.com/ok")
}

func TestSimpleOutput_RawLines(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	events := []engine.Event{
		{Type: engine.EventRawLine, RawLine: []byte("This is a raw line")},
		{Type: engine.EventRawLine, RawLine: []byte("Another raw line")},
	}

	err := simple.ProcessEvents(sendEvents(events))
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "This is a raw line")
	assert.Contains(t, output, "Another raw line")
}

func TestSimpleOutput_HasFailures(t *testing.T) {
	collector := results.NewCollector()
	state := collector.State()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	assert.False(t, simple.HasFailures())

	pkg := &results.PackageResult{
		Name:   "pkg",
		Status: results.StatusFailed,
	}
	pkg.Counts.Failed = 1
	run.Packages["pkg"] = pkg
	run.Counts.Failed = 1

	assert.True(t, simple.HasFailures())
}
