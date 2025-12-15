package tui

import (
	"strings"
)

// viewLatest helps tests render the last run even if it's finished (and thus m.View() returns empty)
func viewLatest(m *Model) string {
	state := m.collector.State()
	if len(state.Runs) == 0 {
		return ""
	}
	run := state.Runs[len(state.Runs)-1]
	out := m.renderRun(run)
	return strings.TrimRight(expandTabs(out, 8), "\n")
}
