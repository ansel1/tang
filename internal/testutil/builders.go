package testutil

import (
	"time"

	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
)

// --- Model options ---

// ModelOpt configures the model returned by BuildModel.
type ModelOpt func(m *tui.Model, run *results.Run)

// WithTermSize sets the terminal dimensions.
func WithTermSize(w, h int) ModelOpt {
	return func(m *tui.Model, _ *results.Run) {
		m.TerminalWidth = w
		m.TerminalHeight = h
	}
}

// WithRunStatus sets the run-level status.
func WithRunStatus(s results.Status) ModelOpt {
	return func(_ *tui.Model, run *results.Run) {
		run.Status = s
	}
}

// --- Test options ---

// TestOpt configures a test added by WithTest.
type TestOpt func(t *results.TestResult)

// TIterations adds multiple executions to a test result.
// This is useful for creating test fixtures with -count=N scenarios.
func TIterations(iterations int) TestOpt {
	return func(t *results.TestResult) {
		for i := 1; i < iterations; i++ {
			t.AppendExecution()
		}
	}
}

// TAddExecution adds an additional execution to a test result.
// This is useful for creating test fixtures with -count=N scenarios.
func TAddExecution() TestOpt {
	return func(t *results.TestResult) {
		t.AppendExecution()
	}
}

// TStatus sets the test status.
func TStatus(s results.Status) TestOpt {
	return func(t *results.TestResult) {
		t.Latest().Status = s
	}
}

// TElapsed sets the test elapsed duration.
func TElapsed(d time.Duration) TestOpt {
	return func(t *results.TestResult) {
		t.Latest().Elapsed = d
	}
}

// TSummaryLine sets the summary line (e.g. "--- PASS: TestFoo (0.05s)").
func TSummaryLine(s string) TestOpt {
	return func(t *results.TestResult) {
		t.Latest().SummaryLine = s
	}
}

// TOutput sets the test output lines.
func TOutput(lines ...string) TestOpt {
	return func(t *results.TestResult) {
		t.Latest().Output = lines
	}
}

// testSpec holds a deferred test configuration to be applied after
// the package is inserted into the run.
type testSpec struct {
	name string
	opts []TestOpt
}

// pkgSpec holds all configuration for a package including its tests.
type pkgSpec struct {
	name  string
	tests []testSpec
	opts  []func(pkg *results.PackageResult)
}

// PkgOpt configures a package added by WithPackage.
type PkgOpt func(ps *pkgSpec)

// PkgStatus sets the package status.
func PkgStatus(s results.Status) PkgOpt {
	return func(ps *pkgSpec) {
		ps.opts = append(ps.opts, func(pkg *results.PackageResult) {
			pkg.Status = s
		})
	}
}

// PkgElapsed sets the package elapsed duration.
func PkgElapsed(d time.Duration) PkgOpt {
	return func(ps *pkgSpec) {
		ps.opts = append(ps.opts, func(pkg *results.PackageResult) {
			pkg.Elapsed = d
		})
	}
}

// PkgOutput sets the final summary line for the package (e.g. the "ok ..." line).
func PkgOutput(s string) PkgOpt {
	return func(ps *pkgSpec) {
		ps.opts = append(ps.opts, func(pkg *results.PackageResult) {
			pkg.SummaryLine = s
		})
	}
}

// PkgOutputLines appends arbitrary package-level output lines (panics, flag
// errors, coverage, etc.) to the package result.
func PkgOutputLines(lines ...string) PkgOpt {
	return func(ps *pkgSpec) {
		ps.opts = append(ps.opts, func(pkg *results.PackageResult) {
			pkg.OutputLines = append(pkg.OutputLines, lines...)
		})
	}
}

// WithTest adds a test to the package.
func WithTest(name string, opts ...TestOpt) PkgOpt {
	return func(ps *pkgSpec) {
		ps.tests = append(ps.tests, testSpec{name: name, opts: opts})
	}
}

// WithPackage adds a package to the run with the given options.
func WithPackage(name string, opts ...PkgOpt) ModelOpt {
	return func(_ *tui.Model, run *results.Run) {
		now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		// Collect all specs first.
		ps := &pkgSpec{name: name}
		for _, o := range opts {
			o(ps)
		}

		pkg := &results.PackageResult{
			Name:          name,
			Status:        results.StatusRunning, // default
			StartTime:     now,
			WallStartTime: now,
			TestOrder:     make([]string, 0),
			DisplayOrder:  make([]string, 0),
		}

		// Apply package-level options.
		for _, o := range ps.opts {
			o(pkg)
		}

		run.Packages[name] = pkg
		run.PackageOrder = append(run.PackageOrder, name)

		if pkg.Status == results.StatusRunning {
			run.RunningPkgs++
		}

		// Wire up tests.
		for _, ts := range ps.tests {
			tr := results.NewTestResult(name, ts.name)
			tr.Latest().StartTime = now
			tr.Latest().WallStartTime = now
			tr.Latest().LastResumeTime = now

			for _, to := range ts.opts {
				to(tr)
			}

			pkg.TestOrder = append(pkg.TestOrder, ts.name)
			pkg.DisplayOrder = append(pkg.DisplayOrder, ts.name)
			testKey := name + "/" + ts.name
			run.TestResults[testKey] = tr

			// Update counts.
			switch tr.Status() {
			case results.StatusPassed:
				pkg.Counts.Passed++
				run.Counts.Passed++
			case results.StatusFailed:
				pkg.Counts.Failed++
				run.Counts.Failed++
			case results.StatusSkipped:
				pkg.Counts.Skipped++
				run.Counts.Skipped++
			case results.StatusRunning, results.StatusPaused:
				pkg.Counts.Running++
				run.Counts.Running++
			}
		}
	}
}

// BuildModel constructs a *tui.Model with a single run, ready for rendering.
func BuildModel(opts ...ModelOpt) *tui.Model {
	collector := results.NewCollector()
	m := tui.NewModel(false, 1.0, collector)

	// Defaults.
	m.TerminalWidth = 80
	m.TerminalHeight = 24

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	run := results.NewRun(1)
	run.Status = results.StatusRunning
	run.FirstEventTime = now
	run.WallStartTime = now
	run.LastEventTime = now

	for _, o := range opts {
		o(m, run)
	}

	// If run is finished, clear CurrentRun.
	state := collector.State()
	state.Runs = append(state.Runs, run)
	if run.Status != results.StatusRunning {
		state.CurrentRun = nil
	} else {
		state.CurrentRun = run
	}

	return m
}
