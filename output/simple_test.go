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

// processEvents drives a SimpleOutput the same way the main event loop
// does: push each event to the collector, render it, then flush and write
// run summaries at the end of the stream. An EventComplete is appended to
// finish the run.
func processEvents(simple *SimpleOutput, collector *results.Collector, events []engine.Event) {
	simple.Init()

	events = append(events, engine.Event{Type: engine.EventComplete})
	for _, evt := range events {
		collector.Push(evt)
		simple.ProcessEvent(evt)
	}

	simple.Flush()
	simple.WriteRunSummaries()
}

func TestSimpleOutput_Verbose_PassingTest(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, true, 80, false)

	processEvents(simple, collector, passingPackageEvents("example.com/pkg"))

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

	processEvents(simple, collector, failingPackageEvents("example.com/pkg"))

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
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, true)

	processEvents(simple, collector, passingPackageEvents("example.com/pkg"))

	output := buf.String()
	assert.NotContains(t, output, "=== RUN")
	assert.NotContains(t, output, "--- PASS")
	assert.Contains(t, output, "ok  \texample.com/pkg\t0.100s\n\n─── Summary ")
	assert.Contains(t, output, "(1 packages)")
}

func TestSimpleOutput_NonVerbose_FailedTest(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	processEvents(simple, collector, failingPackageEvents("example.com/pkg"))

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

	processEvents(simple, collector, events)

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

	processEvents(simple, collector, events)

	output := buf.String()
	assert.Contains(t, output, "This is a raw line")
	assert.Contains(t, output, "Another raw line")
}

// badFlagEvents simulates `go test -json --badflag`: the test binary starts but
// immediately fails with a flag error. No individual tests ever run.
func badFlagEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "flag provided but not defined: -badflag\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "Usage of /tmp/test.binary:\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "exit status 2\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "FAIL\t" + pkg + "\t0.100s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkg, Elapsed: 0.1}},
	}
}

// testMainPanicEvents simulates `go test -json` when TestMain panics before
// any tests run.
func testMainPanicEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "panic: TestMain setup failed\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "goroutine 1 [running]:\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "main.main()\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "\t/tmp/TestMain_test.go:10 +0x38\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "exit status 2\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "FAIL\t" + pkg + "\t0.050s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkg, Elapsed: 0.05}},
	}
}

// parallelCrashEvents simulates `go test -json` when a parallel subtest
// triggers an unrecoverable runtime fatal (e.g. concurrent map writes). The
// crash kills the binary: the fatal output is attributed to the running
// subtest as test-level output, but no test ever receives a "fail" event —
// only the package does. Other parallel tests are left interrupted.
func parallelCrashEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkg, Test: "TestParallel"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel", Output: "=== RUN   TestParallel\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkg, Test: "TestParallel/crash"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/crash", Output: "=== RUN   TestParallel/crash\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkg, Test: "TestParallel/sibling"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/sibling", Output: "=== RUN   TestParallel/sibling\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/crash", Output: "    crash_test.go:10: triggering concurrent map writes...\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/crash", Output: "fatal error: concurrent map writes\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/crash", Output: "\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/crash", Output: "goroutine 6 [running]:\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Test: "TestParallel/crash", Output: "\tcrash.go:40 +0x50\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkg, Output: "FAIL\t" + pkg + "\t0.100s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkg, Elapsed: 0.1}},
	}
}

// TestSimpleOutput_NonVerbose_ParallelCrash verifies that the crash output of
// a test interrupted by a runtime fatal is streamed live, even though the test
// never receives its own "fail" event. Without this, the fatal block would
// only appear in the summary, not the realtime output (matching `go test`).
func TestSimpleOutput_NonVerbose_ParallelCrash(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	processEvents(simple, collector, parallelCrashEvents("example.com/pkg"))

	output := buf.String()

	// The realtime portion is everything before the summary block.
	realtime, _, _ := strings.Cut(output, "=== example.com/pkg")

	assert.Contains(t, realtime, "fatal error: concurrent map writes",
		"crash output should stream live, not just in the summary")
	assert.Contains(t, realtime, "goroutine 6 [running]:")
	assert.Contains(t, realtime, "crash.go:40 +0x50")

	// The fatal block must appear exactly once in the realtime stream (no
	// double-emit), and the crash output must precede the package summary line.
	assert.Equal(t, 1, strings.Count(realtime, "fatal error: concurrent map writes"))
	fatalIdx := strings.Index(realtime, "fatal error: concurrent map writes")
	summaryIdx := strings.Index(realtime, "FAIL\texample.com/pkg")
	assert.GreaterOrEqual(t, summaryIdx, 0, "package summary line should be in realtime output")
	assert.Less(t, fatalIdx, summaryIdx, "crash output should come before the package summary line")
	assert.True(t, simple.HasFailures())
}

func TestSimpleOutput_NonVerbose_BadFlag(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	processEvents(simple, collector, badFlagEvents("example.com/pkg"))

	output := buf.String()
	assert.Contains(t, output, "flag provided but not defined: -badflag")
	assert.Contains(t, output, "exit status 2")
	assert.Contains(t, output, "FAIL\texample.com/pkg")
	assert.True(t, simple.HasFailures())
}

func TestSimpleOutput_NonVerbose_TestMainPanic(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	processEvents(simple, collector, testMainPanicEvents("example.com/pkg"))

	output := buf.String()
	assert.Contains(t, output, "panic: TestMain setup failed")
	assert.Contains(t, output, "goroutine 1 [running]:")
	assert.Contains(t, output, "FAIL\texample.com/pkg")
	assert.True(t, simple.HasFailures())
}

// TestSimpleOutput_NonVerbose_SummaryLineTrailingNewline guards the refactor
// that sources the package summary line from the collector. The collector
// stores PackageResult.SummaryLine with its trailing newline stripped, so
// pkgSummaryText must re-add it; this asserts the exact terminated line (the
// other tests only check a prefix and would miss a dropped newline).
func TestSimpleOutput_NonVerbose_SummaryLineTrailingNewline(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	processEvents(simple, collector, passingPackageEvents("example.com/pkg"))

	assert.Contains(t, buf.String(), "ok  \texample.com/pkg\t0.100s\n")
}

// TestSimpleOutput_NonVerbose_TwoPackageFlush drives two failing packages so
// that one streams while the other buffers and is later dumped via
// flushPackage. It pins that each package's summary line (sourced from the
// collector) appears exactly once, with its trailing newline.
func TestSimpleOutput_NonVerbose_TwoPackageFlush(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, false)

	events := append(failingPackageEvents("example.com/a"), failingPackageEvents("example.com/b")...)
	processEvents(simple, collector, events)

	output := buf.String()
	for _, pkg := range []string{"example.com/a", "example.com/b"} {
		summary := "FAIL\t" + pkg + "\t0.100s\n"
		assert.Equal(t, 1, strings.Count(output, summary),
			"summary line for %s should appear exactly once with its trailing newline", pkg)
	}
}

// TestRenderFailedTest pins the shared failure-rendering helper used by both
// the normal per-fail-event path (header + depth indent) and the interrupted
// crash path (raw output, no header, no indent).
func TestRenderFailedTest(t *testing.T) {
	tr := results.NewTestResult("pkg", "TestX")
	tr.Latest().SummaryLine = "--- FAIL: TestX (0.00s)"
	tr.Latest().Output = []string{"x_test.go:5: boom", "more"}

	assert.Equal(t,
		[]string{"    --- FAIL: TestX (0.00s)\n", "    x_test.go:5: boom\n", "    more\n"},
		renderFailedTest(tr, "    ", true))

	assert.Equal(t,
		[]string{"x_test.go:5: boom\n", "more\n"},
		renderFailedTest(tr, "", false))

	assert.Nil(t, renderFailedTest(nil, "", false))
}

func TestSimpleOutput_Verbose_BadFlag(t *testing.T) {
	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, true, 80, false)

	processEvents(simple, collector, badFlagEvents("example.com/pkg"))

	output := buf.String()
	assert.Contains(t, output, "flag provided but not defined: -badflag")
	assert.Contains(t, output, "exit status 2")
	assert.True(t, simple.HasFailures())
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

// TestSimpleOutput_MultiExecutionBothFail tests -count=2 where both executions fail
// This verifies that both executions' summary/output blocks are emitted in the stream
func TestSimpleOutput_MultiExecutionBothFail(t *testing.T) {
	pkgName := "example.com/pkg"

	// Events for -count=2 where both executions fail
	events := []engine.Event{
		// Package start
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "start", Package: pkgName}},
		// First execution
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkgName, Test: "TestFoo"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Test: "TestFoo", Output: "=== RUN   TestFoo\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Test: "TestFoo", Output: "    first_fail.go:5: first failure\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Test: "TestFoo", Output: "--- FAIL: TestFoo (0.10s)\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkgName, Test: "TestFoo", Elapsed: 0.1}},
		// Second execution (detected because first failed)
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "run", Package: pkgName, Test: "TestFoo"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Test: "TestFoo", Output: "=== RUN   TestFoo\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Test: "TestFoo", Output: "    second_fail.go:7: second failure\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Test: "TestFoo", Output: "--- FAIL: TestFoo (0.15s)\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkgName, Test: "TestFoo", Elapsed: 0.15}},
		// Package finishes
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Output: "FAIL\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "output", Package: pkgName, Output: "FAIL\t" + pkgName + "\t0.250s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: baseTime, Action: "fail", Package: pkgName, Elapsed: 0.25}},
	}

	collector := results.NewCollector()
	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, true, 80, false)

	processEvents(simple, collector, events)

	output := buf.String()

	// First execution failure output should appear
	assert.True(t, strings.Contains(output, "first failure"),
		"Expected first failure output to be emitted")

	// Second execution failure output should appear
	assert.True(t, strings.Contains(output, "second failure"),
		"Expected second failure output to be emitted")

	// Both FAIL lines should appear
	failCount := strings.Count(output, "--- FAIL: TestFoo")
	assert.Equal(t, 2, failCount,
		"Expected 2 FAIL summary lines (one per execution)")

	// Verify the test result has 2 executions
	run := collector.State().MostRecentRun()
	tr := run.TestResults[pkgName+"/TestFoo"]
	assert.NotNil(t, tr, "TestFoo should exist in results")
	assert.Len(t, tr.Executions, 2, "Should have 2 executions")

	// Verify both executions are failed
	assert.Equal(t, results.StatusFailed, tr.Executions[0].Status,
		"First execution should be failed")
	assert.Equal(t, results.StatusFailed, tr.Executions[1].Status,
		"Second execution should be failed")

	// Verify both outputs are preserved
	assert.Contains(t, tr.Executions[0].Output[0], "first failure",
		"First execution output should be preserved")
	assert.Contains(t, tr.Executions[1].Output[0], "second failure",
		"Second execution output should be preserved")

	// Verify package counts are correct
	pkg := run.Packages[pkgName]
	assert.Equal(t, 2, pkg.Counts.Failed, "Package should have 2 failed tests")
	assert.Equal(t, 0, run.Counts.Running, "Running count should be 0")

	// Verify HasFailures returns true
	assert.True(t, simple.HasFailures(), "HasFailures should return true")
}
