package results

import (
	"time"
)

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
	EndTime       time.Time                 // When the run ended
	RunningPkgs   int                       // Number of currently running packages
	NonTestOutput []string                  // Build errors, compilation output
}

// PackageResult represents the final result of a package's test run.
type PackageResult struct {
	Name         string
	Status       string // "ok", "FAIL", "?" (incomplete)
	Elapsed      time.Duration
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Output       string   // Final output line (e.g., coverage information)
	TestOrder    []string // Chronological order of test starts
}

// TestResult represents the result of a single test.
type TestResult struct {
	Package     string
	Name        string
	Status      string // "pass", "fail", "skip", "running"
	Elapsed     time.Duration
	Output      []string // Failure/skip messages
	SummaryLine string   // The "===" or "---" line
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
