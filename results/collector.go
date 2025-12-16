package results

import (
	"strings"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
)

// Collector processes engine events and updates the state model.
//
// The Collector is a passive component that tracks the state of test runs.
// It is NOT thread-safe. Synchronization is the responsibility of the caller.
// It detects run boundaries using the heuristic:
// - Run starts: Any test event when no current run exists
// - Run finishes: Running package count drops to 0
type Collector struct {
	state         *State
	lastEventTime time.Time
	isReplay      bool
	replayRate    float64
}

// NewCollector creates a new result collector.
func NewCollector() *Collector {
	return &Collector{
		state: NewState(),
	}
}

// SetReplay configures whether the collector is running in replay mode and the rate.
func (c *Collector) SetReplay(replay bool, rate float64) {
	c.isReplay = replay
	c.replayRate = rate
}

// State returns the current state.
// Note: The returned pointer provides direct access to the internal state.
// It is NOT thread-safe.
func (c *Collector) State() *State {
	return c.state
}

// Push updates the collector state with a new event.
func (c *Collector) Push(evt engine.Event) {
	switch evt.Type {
	case engine.EventTest:
		c.handleTestEvent(evt.TestEvent)

	case engine.EventComplete:
		// Finish current run if any
		c.Finish()

	case engine.EventError:
		// Log error but continue processing
		_ = evt.Error
	}
}

// handleTestEvent processes a test event and updates the state.
func (c *Collector) handleTestEvent(event parser.TestEvent) {
	// Update last event time
	c.lastEventTime = event.Time

	// Start a new run if needed
	if c.state.CurrentRun == nil {
		c.startNewRun()
	}

	run := c.state.CurrentRun

	if !event.Time.IsZero() {
		if run.FirstEventTime.IsZero() {
			run.FirstEventTime = event.Time
		}
		run.LastEventTime = event.Time
	}

	// Handle build-output and other non-package events
	if event.Package == "" {
		switch event.Action {
		case "build-output", "build-fail", "build-pass":
			if event.Output != "" {
				output := strings.TrimRight(event.Output, "\n")
				run.NonTestOutput = append(run.NonTestOutput, output)
			}
		}
		return
	}

	// Get or create package result
	pkgResult, exists := run.Packages[event.Package]
	if !exists {
		pkgResult = &PackageResult{
			Name:          event.Package,
			StartTime:     event.Time,
			WallStartTime: time.Now(),
			TestOrder:     make([]string, 0),
			Status:        StatusRunning,
		}
		run.Packages[event.Package] = pkgResult
		run.PackageOrder = append(run.PackageOrder, event.Package)
		run.RunningPkgs++
	}

	// Handle package-level events
	if event.Test == "" {
		c.handlePackageEvent(run, pkgResult, event)
		return
	}

	// Handle test-level events
	c.handleTestLevelEvent(run, pkgResult, event)
}

// handlePackageEvent handles package-level events.
func (c *Collector) handlePackageEvent(run *Run, pkg *PackageResult, event parser.TestEvent) {
	switch event.Action {
	case "output":
		if event.Output != "" {
			output := event.Output
			if len(output) > 0 && output[len(output)-1] == '\n' {
				output = output[:len(output)-1]
			}
			if output != "" {
				pkg.Output = output
			}
		}

	case "pass":
		pkg.Status = StatusPassed
		pkg.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		run.RunningPkgs--
		c.checkRunFinished(run)

	case "fail":
		pkg.Status = StatusFailed
		pkg.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		run.RunningPkgs--
		c.checkRunFinished(run)

	case "skip":
		pkg.Status = StatusSkipped
		pkg.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		run.RunningPkgs--
		c.checkRunFinished(run)
	}
}

// handleTestLevelEvent handles test-level events.
func (c *Collector) handleTestLevelEvent(run *Run, pkg *PackageResult, event parser.TestEvent) {
	testKey := event.Package + "/" + event.Test

	testResult, exists := run.TestResults[testKey]
	if !exists {
		testResult = &TestResult{
			Package:       event.Package,
			Name:          event.Test,
			Status:        StatusRunning,
			Output:        make([]string, 0),
			StartTime:     event.Time,
			WallStartTime: time.Now(),
		}
		run.TestResults[testKey] = testResult
		pkg.TestOrder = append(pkg.TestOrder, event.Test)
		pkg.Counts.Running++
		run.Counts.Running++
	}

	switch event.Action {
	case "run":
		testResult.Status = StatusRunning

	case "output":
		if event.Output != "" {
			output := strings.TrimRight(event.Output, "\n")

			// Extract summary line (lines starting with "===" or "---")
			if strings.HasPrefix(output, "===") || strings.HasPrefix(output, "---") {
				testResult.SummaryLine = output
			} else {
				testResult.Output = append(testResult.Output, output)
			}
		}

	case "pass":
		testResult.Status = StatusPassed
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		pkg.Counts.Passed++
		pkg.Counts.Running--
		run.Counts.Passed++
		run.Counts.Running--

	case "fail":
		testResult.Status = StatusFailed
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		pkg.Counts.Failed++
		pkg.Counts.Running--
		run.Counts.Failed++
		run.Counts.Running--

	case "skip":
		testResult.Status = StatusSkipped
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		pkg.Counts.Skipped++
		pkg.Counts.Running--
		run.Counts.Skipped++
		run.Counts.Running--

	case "pause":
		testResult.Status = StatusPaused

	case "cont":
		testResult.Status = StatusRunning
	}
}

// startNewRun creates a new run.
func (c *Collector) startNewRun() {
	runID := len(c.state.Runs) + 1
	run := NewRun(runID)
	run.Status = StatusRunning
	c.state.Runs = append(c.state.Runs, run)
	c.state.CurrentRun = run
}

// checkRunFinished checks if the current run has finished.
func (c *Collector) checkRunFinished(run *Run) {
	if run.RunningPkgs == 0 {
		c.Finish()
	}
}

// Finish finishes the current run if any.
// This should be called when processing is complete or interrupted.
func (c *Collector) Finish() {
	if c.state.CurrentRun != nil {
		run := c.state.CurrentRun

		// Determine end time: use last event time if available, otherwise now
		endTime := c.lastEventTime
		if c.lastEventTime.IsZero() {
			endTime = time.Now()
			if c.isReplay {
				// Calculate simulated end time based on wall clock duration and replay rate
				// This ensures that the summary matches the TUI's "perceived" time
				wallDuration := time.Since(run.WallStartTime)

				// Apply rate (rate is inverse speed, e.g. 0.5 means 2x speed)
				// If rate is 0 (instant), we fall back to lastEventTime
				if c.replayRate > 0 {
					wallDuration = time.Duration(float64(wallDuration) / c.replayRate)
				}
				endTime = run.FirstEventTime.Add(wallDuration)
			}
		}
		run.LastEventTime = endTime

		var interrupted bool

		// Mark any still-running packages as interrupted and compute their elapsed time
		for _, pkg := range run.Packages {
			if pkg.Status == StatusRunning {
				interrupted = true
				pkg.Status = StatusInterrupted

				// Calculate elapsed time based on run duration and package start offset
				// This ensures consistency with TUI even if ReplayReader doesn't sleep exactly as expected
				wallRunDuration := time.Since(pkg.WallStartTime)
				if c.isReplay && c.replayRate > 0 {
					wallRunDuration = time.Duration(float64(wallRunDuration) / c.replayRate)
				}
				pkg.Elapsed = wallRunDuration
			}
		}

		if interrupted {
			run.Status = StatusInterrupted
		} else if run.Counts.Failed > 0 {
			run.Status = StatusFailed
		} else {
			run.Status = StatusPassed
		}

		c.state.CurrentRun = nil
	}
}
