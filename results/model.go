package results

import (
	"time"
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
	ID            int                       // Sequential run ID (1, 2, 3...)
	Packages      map[string]*PackageResult // Package name -> PackageResult
	PackageOrder  []string                  // Chronological order of package starts
	TestResults   map[string]*TestResult    // "package/testname" -> TestResult
	StartTime     time.Time                 // When the run started
	WallStartTime time.Time                 // When the run started (wall clock)
	EndTime       time.Time                 // When the run ended
	RunningPkgs   int                       // Number of currently running packages
	NonTestOutput []string                  // Build errors, compilation output
	Counts        struct {
		Passed  int // Number of passed tests
		Failed  int // Number of failed tests
		Skipped int // Number of skipped tests
		Running int // Number of running tests
	}
	Status  Status
	Running bool
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
		Running int // Number of running tests
	}
	Output    string   // Final output line (e.g., coverage information)
	TestOrder []string // Chronological order of test starts
}

// TestResult represents the result of a single test.
type TestResult struct {
	Package       string
	Name          string
	Status        Status    // "pass", "fail", "skip", "running"
	StartTime     time.Time // When the test started
	WallStartTime time.Time // When the test started (wall clock)
	Elapsed       time.Duration
	Output        []string // Failure/skip messages
	SummaryLine   string   // The "===" or "---" line
}

func (t *TestResult) Running() bool {
	return t.Status == StatusRunning
}

// State holds all runs and provides access to the current run.
type State struct {
	Runs       []*Run // All runs in chronological order
	CurrentRun *Run   // Currently active run (nil if no active run)
}

// NewRun creates a new run.
func NewRun(id int) *Run {
	return &Run{
		ID:            id,
		Packages:      make(map[string]*PackageResult),
		PackageOrder:  make([]string, 0),
		TestResults:   make(map[string]*TestResult),
		StartTime:     time.Now(),
		NonTestOutput: make([]string, 0),
	}
}

// NewState creates a new state.
func NewState() *State {
	return &State{
		Runs: make([]*Run, 0),
	}
}
