package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
	"github.com/stretchr/testify/require"
)

// LoadFixture reads a .jsonl fixture file from testdata/fixtures/<name>.jsonl
// relative to the caller's package directory, feeds each line through the
// parser and collector, and returns a *tui.Model ready to render.
func LoadFixture(t *testing.T, name string) *tui.Model {
	t.Helper()

	_, callerFile, _, ok := runtime.Caller(1)
	require.True(t, ok, "runtime.Caller failed")

	fixturePath := filepath.Join(filepath.Dir(callerFile), "testdata", "fixtures", name+".jsonl")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "fixture file missing: %s", fixturePath)

	collector := results.NewCollector()

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		evt, err := parser.ParseEvent([]byte(line))
		require.NoError(t, err, "bad JSON in fixture %s: %s", name, line)

		if evt.IsBuildEvent() {
			collector.Push(engine.Event{
				Type:       engine.EventBuild,
				BuildEvent: evt.ToBuildEvent(),
			})
		} else if evt.IsTestEvent() {
			collector.Push(engine.Event{
				Type:      engine.EventTest,
				TestEvent: evt.ToTestEvent(),
			})
		}
	}

	// Push EventComplete to cleanly finish the run
	collector.Push(engine.Event{Type: engine.EventComplete})

	m := tui.NewModel(false, 1.0, collector)
	m.TerminalWidth = 80
	m.TerminalHeight = 24

	return m
}
