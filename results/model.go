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
// A run starts when any test event is received and there is no current run in progress.
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
	Output       string   // Final output line (e.g., coverage information)
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

// TestResult represents the result of a single test.
type TestResult struct {
	Package        string
	Name           string
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

func (t *TestResult) Running() bool {
	return t.Status == StatusRunning || t.Status == StatusPaused
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
