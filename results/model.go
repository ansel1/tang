package results

import (
	"time"

	"github.com/ansel1/tang/parser"
)

// Status represents the state of a test, package, or run.  Some status are only
// applicable to certain types of entities.
type Status int

const (
	StatusUnknown Status = iota
	StatusPassed
	StatusFailed
	StatusRunning
	StatusSkipped
	StatusInterrupted
	StatusPaused
)

func (s Status) String() string {
	strs := []string{
		"unknown",
		"passed",
		"failed",
		"running",
		"skipped",
		"interrupted",
		"paused",
	}
	if s < 0 || s >= Status(len(strs)) {
		return "unknown"
	}
	return strs[s]
}

// Run represents a single discrete test execution.
//
// A run starts when any test or build event is received and there is no current run in progress.
// A run finishes when the number of running packages drops to 0.
type Run struct {
	ID             int                       // Sequential run ID (1, 2, 3...)
	Packages       map[string]*PackageResult // Package name -> PackageResult
	PackageOrder   []string                  // Chronological order of package starts
	TestResults    map[string]*TestResult    // "package/testname" -> TestResult
	FirstEventTime time.Time                 // When the run started
	WallStartTime  time.Time                 // When the run started (wall clock)
	LastEventTime  time.Time                 // When the run ended
	RunningPkgs    int                       // Number of currently running packages
	NonTestOutput  []string                  // Build errors, compilation output
	BuildEvents    []parser.BuildEvent       // Structured build events
	Counts         struct {
		Passed  int // Number of passed tests
		Failed  int // Number of failed tests
		Skipped int // Number of skipped tests
		Running int // Number of actively running tests (excludes paused)
		Paused  int // Number of paused tests
	}
	Status  Status
	Running bool
}

// GetBuildErrors returns all build events for the given import path
func (r *Run) GetBuildErrors(importPath string) []parser.BuildEvent {
	var errors []parser.BuildEvent
	for _, be := range r.BuildEvents {
		if be.ImportPath == importPath {
			errors = append(errors, be)
		}
	}
	return errors
}

// PackageResult represents the final result of a package's test run.
type PackageResult struct {
	Name          string
	Status        Status
	StartTime     time.Time // When the package testing started
	WallStartTime time.Time // When the package testing started (wall clock)
	Elapsed       time.Duration
	Counts        struct {
		Passed  int // Number of passed tests
		Failed  int // Number of failed tests
		Skipped int // Number of skipped tests
		Running int // Number of actively running tests (excludes paused)
		Paused  int // Number of paused tests
	}
	SummaryLine  string   // Final package result line (e.g. "ok\tpkg\t0.30s\tcoverage: 87.5%")
	OutputLines  []string // Package-level output that isn't the summary line or a bare PASS/FAIL
	TestOrder    []string // Chronological order of test starts
	DisplayOrder []string // Render order for TUI; reordered when paused tests resume
	FailedBuild  string   // ImportPath of failed build (if any)
	PanicTestKey string   // "package/test" key of the test carrying the timeout panic output
}

func (p *PackageResult) moveToEndOfDisplayOrder(name string) {
	for i, n := range p.DisplayOrder {
		if n == name {
			p.DisplayOrder = append(p.DisplayOrder[:i], append(p.DisplayOrder[i+1:], name)...)
			return
		}
	}
}

// TestExecution represents the result of a single execution of a test.
// When go test -count=N reruns a test, each iteration gets its own TestExecution.
type TestExecution struct {
	Status         Status    // "pass", "fail", "skip", "running"
	StartTime      time.Time // When the test started
	WallStartTime  time.Time // When the test started (wall clock)
	Elapsed        time.Duration
	Output         []string      // Failure/skip messages
	SummaryLine    string        // The "===" or "---" line
	Interrupted    bool          // True if the test was interrupted by a panic or runtime fatal
	ActiveDuration time.Duration // Accumulated time spent actively running (excludes paused time)
	LastResumeTime time.Time     // Wall clock time when the test last entered running state
}

// TestResult represents the result of a single test (possibly with multiple executions).
type TestResult struct {
	Package    string
	Name       string
	Executions []*TestExecution // One per iteration when -count=N is used
}

// Latest returns the most recent execution. Callers should ensure there's at least one.
func (t *TestResult) Latest() *TestExecution {
	if len(t.Executions) == 0 {
		return nil
	}
	return t.Executions[len(t.Executions)-1]
}

// Status returns the status of the latest execution.
func (t *TestResult) Status() Status {
	if latest := t.Latest(); latest != nil {
		return latest.Status
	}
	return StatusUnknown
}

// StartTime returns the start time of the latest execution.
func (t *TestResult) StartTime() time.Time {
	if latest := t.Latest(); latest != nil {
		return latest.StartTime
	}
	return time.Time{}
}

// WallStartTime returns the wall start time of the latest execution.
func (t *TestResult) WallStartTime() time.Time {
	if latest := t.Latest(); latest != nil {
		return latest.WallStartTime
	}
	return time.Time{}
}

// Elapsed returns the elapsed time of the latest execution.
func (t *TestResult) Elapsed() time.Duration {
	if latest := t.Latest(); latest != nil {
		return latest.Elapsed
	}
	return 0
}

// Output returns the output of the latest execution.
func (t *TestResult) Output() []string {
	if latest := t.Latest(); latest != nil {
		return latest.Output
	}
	return nil
}

// SummaryLine returns the summary line of the latest execution.
func (t *TestResult) SummaryLine() string {
	if latest := t.Latest(); latest != nil {
		return latest.SummaryLine
	}
	return ""
}

// Interrupted returns whether the latest execution was interrupted.
func (t *TestResult) Interrupted() bool {
	if latest := t.Latest(); latest != nil {
		return latest.Interrupted
	}
	return false
}

// ActiveDuration returns the active duration of the latest execution.
func (t *TestResult) ActiveDuration() time.Duration {
	if latest := t.Latest(); latest != nil {
		return latest.ActiveDuration
	}
	return 0
}

// LastResumeTime returns the last resume time of the latest execution.
func (t *TestResult) LastResumeTime() time.Time {
	if latest := t.Latest(); latest != nil {
		return latest.LastResumeTime
	}
	return time.Time{}
}

// Running returns whether the latest execution is currently running or paused.
func (t *TestResult) Running() bool {
	status := t.Status()
	return status == StatusRunning || status == StatusPaused
}

// NewTestResult creates a new TestResult with a single execution.
func NewTestResult(pkg, name string) *TestResult {
	return &TestResult{
		Package: pkg,
		Name:    name,
		Executions: []*TestExecution{
			{Status: StatusRunning},
		},
	}
}

// NewTestExecution creates a new TestExecution with Running status.
func NewTestExecution() *TestExecution {
	return &TestExecution{Status: StatusRunning}
}

// AppendExecution appends a new execution to the test result.
func (t *TestResult) AppendExecution() *TestExecution {
	exec := NewTestExecution()
	t.Executions = append(t.Executions, exec)
	return exec
}

// ExecutionDisplayName returns the display name for a test execution.
//
// When total <= 1, the plain test name is returned (no suffix).
// When total >= 2, every execution — including the first — receives a
// zero-padded "#NN" suffix (e.g. TestFoo#01, TestFoo#02, TestFoo#03) so
// multi-execution listings are visually uniform and unambiguous.
//
// For subtests, the suffix is anchored on the top-level test name and
// appears before the subtest path, e.g. TestFoo#02/sub or
// TestFoo#02/sub/nested.
func ExecutionDisplayName(name string, iteration, total int) string {
	if total <= 1 {
		return name
	}
	// Check if this is a subtest (contains /)
	// For subtests like TestFoo/sub/nested, we want TestFoo#02/sub/nested
	if idx := findSubtestSeparator(name); idx != -1 {
		parent := name[:idx]
		suffix := name[idx:]
		return parent + "#" + formatIteration(iteration) + suffix
	}
	return name + "#" + formatIteration(iteration)
}

// findSubtestSeparator finds the index of the first / that separates
// parent test from subtest, but only if there's a / after position 0.
func findSubtestSeparator(name string) int {
	for i := 1; i < len(name); i++ {
		if name[i] == '/' {
			return i
		}
	}
	return -1
}

func formatIteration(iteration int) string {
	if iteration < 10 {
		return "0" + string(rune('0'+iteration))
	}
	return string(rune('0'+iteration/10)) + string(rune('0'+iteration%10))
}

// State holds all runs and provides access to the current run.
type State struct {
	Runs       []*Run // All runs in chronological order
	CurrentRun *Run   // Currently active run (nil if no active run)
}

func (s *State) MostRecentRun() *Run {
	if len(s.Runs) == 0 {
		return nil
	}
	return s.Runs[len(s.Runs)-1]
}

// NewRun creates a new run.
func NewRun(id int) *Run {
	return &Run{
		ID:            id,
		Packages:      make(map[string]*PackageResult),
		PackageOrder:  make([]string, 0),
		TestResults:   make(map[string]*TestResult),
		WallStartTime: time.Now(),
		NonTestOutput: make([]string, 0),
	}
}

// NewState creates a new state.
func NewState() *State {
	return &State{
		Runs: make([]*Run, 0),
	}
}
