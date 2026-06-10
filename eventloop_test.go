package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var eventBaseTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// fakeTUI is a tuiController that exits when sent a QuitMsg, emulating the
// live TUI program without a terminal.
type fakeTUI struct {
	done chan struct{}
}

func newFakeTUI() *fakeTUI {
	return &fakeTUI{done: make(chan struct{})}
}

func (f *fakeTUI) Send(msg tea.Msg) {
	if _, ok := msg.(tui.QuitMsg); ok {
		select {
		case <-f.done:
		default:
			close(f.done)
		}
	}
}

func (f *fakeTUI) Done() <-chan struct{} { return f.done }

func passingPkgEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "run", Package: pkg, Test: "TestFoo"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Test: "TestFoo", Output: "=== RUN   TestFoo\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Test: "TestFoo", Output: "--- PASS: TestFoo (0.00s)\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "pass", Package: pkg, Test: "TestFoo", Elapsed: 0.001}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Output: "PASS\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Output: "ok  \t" + pkg + "\t0.100s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "pass", Package: pkg, Elapsed: 0.1}},
	}
}

func failingPkgEvents(pkg string) []engine.Event {
	return []engine.Event{
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "start", Package: pkg}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "run", Package: pkg, Test: "TestFail"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Test: "TestFail", Output: "=== RUN   TestFail\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Test: "TestFail", Output: "    test_fail.go:10: assertion failed\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Test: "TestFail", Output: "--- FAIL: TestFail (0.00s)\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "fail", Package: pkg, Test: "TestFail", Elapsed: 0.001}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Output: "FAIL\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "output", Package: pkg, Output: "FAIL\t" + pkg + "\t0.100s\n"}},
		{Type: engine.EventTest, TestEvent: parser.TestEvent{Time: eventBaseTime, Action: "fail", Package: pkg, Elapsed: 0.1}},
	}
}

func rawLine(line string) engine.Event {
	return engine.Event{Type: engine.EventRawLine, RawLine: []byte(line)}
}

// driveEventLoop runs runEventLoop over the given events (with a trailing
// EventComplete, like a real engine stream) and returns what was written to
// stdout. startTUI may be nil (notty mode) or a fake TUI factory (live mode).
func driveEventLoop(events []engine.Event, verbose bool, startTUI func() tuiController) string {
	ch := make(chan engine.Event, len(events)+1)
	for _, evt := range events {
		ch <- evt
	}
	ch <- engine.Event{Type: engine.EventComplete}
	close(ch)

	collector := results.NewCollector()
	var buf bytes.Buffer
	var stdout bytes.Buffer
	simple := output.NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, verbose, 80, true)

	runEventLoop(ch, collector, simple, &buf, &stdout, startTUI)
	return stdout.String()
}

// TestRunEventLoop_NottyLiveEquivalence verifies the core promise of the
// unified event loop: after tang exits, the terminal contents are identical
// whether or not the live TUI was used.
func TestRunEventLoop_NottyLiveEquivalence(t *testing.T) {
	multiRun := append(passingPkgEvents("example.com/pkg1"),
		rawLine("--- run boundary ---"))
	multiRun = append(multiRun, failingPkgEvents("example.com/pkg2")...)

	fixtures := map[string][]engine.Event{
		"passing":  passingPkgEvents("example.com/pkg"),
		"failing":  failingPkgEvents("example.com/pkg"),
		"multiRun": multiRun,
	}

	for name, events := range fixtures {
		for _, verbose := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_verbose=%v", name, verbose), func(t *testing.T) {
				notty := driveEventLoop(events, verbose, nil)
				live := driveEventLoop(events, verbose, func() tuiController { return newFakeTUI() })
				assert.Equal(t, notty, live, "notty and live modes should leave identical output")
				assert.NotEmpty(t, notty)
			})
		}
	}
}

// TestRunEventLoop_MultiRunSummaries verifies that a summary block is
// written after every run, with raw run-separator lines appearing between
// the previous run's summary and the next run's output.
func TestRunEventLoop_MultiRunSummaries(t *testing.T) {
	events := append(passingPkgEvents("example.com/pkg1"),
		rawLine("--- run boundary ---"))
	events = append(events, passingPkgEvents("example.com/pkg2")...)

	out := driveEventLoop(events, false, nil)

	assert.Equal(t, 2, strings.Count(out, "(1 packages)"),
		"expected one summary block per run:\n%s", out)

	pkg1 := strings.Index(out, "ok  \texample.com/pkg1")
	summary1 := strings.Index(out, "(1 packages)")
	boundary := strings.Index(out, "--- run boundary ---")
	pkg2 := strings.Index(out, "ok  \texample.com/pkg2")
	summary2 := strings.LastIndex(out, "(1 packages)")

	require.NotEqual(t, -1, pkg1)
	require.NotEqual(t, -1, summary1)
	require.NotEqual(t, -1, boundary)
	require.NotEqual(t, -1, pkg2)

	assert.Less(t, pkg1, summary1, "run 1 output should precede run 1 summary:\n%s", out)
	assert.Less(t, summary1, boundary, "run 1 summary should precede the raw boundary line:\n%s", out)
	assert.Less(t, boundary, pkg2, "raw boundary line should precede run 2 output:\n%s", out)
	assert.Less(t, pkg2, summary2, "run 2 output should precede run 2 summary:\n%s", out)
}

// TestRunEventLoop_RawLinesInline verifies raw lines are rendered in stream
// order in both modes (live mode previously batched them after the run).
func TestRunEventLoop_RawLinesInline(t *testing.T) {
	events := []engine.Event{
		rawLine("before any run"),
		rawLine("another preamble line"),
	}

	for _, mode := range []struct {
		name     string
		startTUI func() tuiController
	}{
		{"notty", nil},
		{"live", func() tuiController { return newFakeTUI() }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			out := driveEventLoop(events, false, mode.startTUI)
			assert.Contains(t, out, "before any run\n")
			assert.Contains(t, out, "another preamble line\n")
		})
	}
}

// TestRunEventLoop_UserQuitSummarizesPartialRun verifies break-and-summarize:
// when the TUI exits on its own (user quit), the in-progress run is finished
// and summarized, and remaining events are not processed.
func TestRunEventLoop_UserQuitSummarizesPartialRun(t *testing.T) {
	events := passingPkgEvents("example.com/pkg1")
	// Events after the user quits; should never be rendered.
	events = append(events, failingPkgEvents("example.com/pkg2")...)

	quitTUI := newFakeTUI()
	started := false

	ch := make(chan engine.Event, len(events)+1)
	for _, evt := range events {
		ch <- evt
	}
	close(ch)

	collector := results.NewCollector()
	var buf bytes.Buffer
	var stdout bytes.Buffer
	simple := output.NewSimpleOutput(&buf, collector, 10*time.Second, format.SummaryOptions{}, false, 80, true)

	runEventLoop(ch, collector, simple, &buf, &stdout, func() tuiController {
		if started {
			t.Fatal("TUI should only be started once in this scenario")
		}
		started = true
		// Simulate the user quitting immediately after the TUI starts.
		close(quitTUI.done)
		return quitTUI
	})

	out := stdout.String()
	assert.True(t, started, "TUI should have been started")
	assert.Contains(t, out, "(1 packages)", "partial run should be summarized")
	assert.NotContains(t, out, "example.com/pkg2", "events after quit should not be rendered")
}
