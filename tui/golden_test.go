package tui_test

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/ansel1/tang/internal/testutil"
	"github.com/ansel1/tang/results"
)

// TestGolden_HierarchicalRendering validates the hierarchical TUI format for a
// finished run using the functional builder path.
func TestGolden_HierarchicalRendering(t *testing.T) {
	m := testutil.BuildModel(
		testutil.WithTermSize(80, 24),
		testutil.WithRunStatus(results.StatusPassed),
		testutil.WithPackage("github.com/test/pkg1",
			testutil.PkgStatus(results.StatusPassed),
			testutil.PkgElapsed(100*time.Millisecond),
			testutil.PkgOutput("ok\tgithub.com/test/pkg1\t0.10s"),
			testutil.WithTest("TestFoo",
				testutil.TStatus(results.StatusPassed),
				testutil.TElapsed(50*time.Millisecond),
				testutil.TSummaryLine("--- PASS: TestFoo (0.05s)"),
			),
		),
	)

	output := m.String()
	testutil.AssertGolden(t, "hierarchical_rendering", output)
}

// TestGolden_BleedProtection validates that ANSI resets are inserted to
// prevent color bleeding from test output, using the functional builder path.
func TestGolden_BleedProtection(t *testing.T) {
	m := testutil.BuildModel(
		testutil.WithTermSize(80, 20),
		testutil.WithRunStatus(results.StatusRunning),
		testutil.WithPackage("pkg1",
			testutil.PkgStatus(results.StatusRunning),
			testutil.WithTest("TestBleed",
				testutil.TStatus(results.StatusRunning),
				testutil.TSummaryLine("=== RUN   TestBleed"),
				testutil.TOutput("\033[31mThis is red text"),
			),
		),
	)

	output := m.String()
	testutil.AssertGolden(t, "bleed_protection", output)
}

// TestGolden_FixtureAlignment is an integration test that loads a .jsonl
// fixture through the real parser/collector pipeline and checks alignment.
func TestGolden_FixtureAlignment(t *testing.T) {
	m := testutil.LoadFixture(t, "alignment")
	output := m.String()
	testutil.AssertGolden(t, "fixture_alignment", output)

	// Property: no rendered line should exceed the terminal width.
	for _, line := range strings.Split(output, "\n") {
		w := lipgloss.Width(line)
		if w > 80 {
			t.Errorf("line exceeds terminal width (visual %d > 80): %q", w, line)
		}
	}
}
