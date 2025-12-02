package results

import (
	"strings"
	"sync"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
)

// Collector processes engine events, updates the state model, and emits high-level events.
//
// The Collector is the single consumer of engine.Event and the single source of truth
// for test run state. It detects run boundaries using the heuristic:
// - Run starts: Any test event when no current run exists
// - Run finishes: Running package count drops to 0
type Collector struct {
	state               *State
	mu                  sync.RWMutex
	subscribers         []chan Event
	subMu               sync.Mutex
	lastEventTime       time.Time
	isReplay            bool
	replayRate          float64
	currentRunWallStart time.Time
}

// NewCollector creates a new result collector.
func NewCollector() *Collector {
	return &Collector{
		state:       NewState(),
		subscribers: make([]chan Event, 0),
	}
}

// SetReplay configures whether the collector is running in replay mode and the rate.
func (c *Collector) SetReplay(replay bool, rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isReplay = replay
	c.replayRate = rate
}

// Subscribe returns a channel that will receive result events.
// The caller should read from this channel until it is closed.
func (c *Collector) Subscribe() <-chan Event {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	ch := make(chan Event, 100)
	c.subscribers = append(c.subscribers, ch)
	return ch
}

// emit sends an event to all subscribers.
func (c *Collector) emit(evt Event) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, sub := range c.subscribers {
		sub <- evt
	}
}

// closeSubscribers closes all subscriber channels.
func (c *Collector) closeSubscribers() {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, sub := range c.subscribers {
		close(sub)
	}
}

// ProcessEvents consumes engine events and updates state.
// This method should be called as a goroutine.
func (c *Collector) ProcessEvents(events <-chan engine.Event) {
	for evt := range events {
		switch evt.Type {
		case engine.EventRawLine:
			// Pass through raw output
			c.mu.RLock()
			runID := 0
			if c.state.CurrentRun != nil {
				runID = c.state.CurrentRun.ID
			}
			c.mu.RUnlock()
			c.emit(NewRawOutputEvent(runID, evt.RawLine))

		case engine.EventTest:
			// Handle test event and emit events after lock is released
			eventsToEmit := c.handleTestEvent(evt.TestEvent)
			for _, e := range eventsToEmit {
				c.emit(e)
			}

		case engine.EventComplete:
			// Finish current run if any
			c.Finish()
			c.closeSubscribers()
			return

		case engine.EventError:
			// Log error but continue processing
			_ = evt.Error
		}
	}
}

// handleTestEvent processes a test event and updates the state.
// Returns events to emit after the lock is released.
func (c *Collector) handleTestEvent(event parser.TestEvent) []Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	eventsToEmit := make([]Event, 0)

	// Update last event time
	c.lastEventTime = event.Time

	// Start a new run if needed
	if c.state.CurrentRun == nil {
		evt := c.startNewRun(event.Time)
		eventsToEmit = append(eventsToEmit, evt)
	}

	run := c.state.CurrentRun

	// Handle build-output and other non-package events
	if event.Package == "" {
		switch event.Action {
		case "build-output", "build-fail", "build-pass":
			if event.Output != "" {
				output := strings.TrimRight(event.Output, "\n")
				run.NonTestOutput = append(run.NonTestOutput, output)
				eventsToEmit = append(eventsToEmit, NewNonTestOutputEvent(run.ID, output))
			}
		}
		return eventsToEmit
	}

	// Get or create package result
	pkgResult, exists := run.Packages[event.Package]
	if !exists {
		pkgResult = &PackageResult{
			Name:      event.Package,
			StartTime: event.Time,
			TestOrder: make([]string, 0),
		}
		run.Packages[event.Package] = pkgResult
		run.PackageOrder = append(run.PackageOrder, event.Package)
		run.RunningPkgs++
	}

	// Handle package-level events
	if event.Test == "" {
		pkgEvents := c.handlePackageEvent(run, pkgResult, event)
		eventsToEmit = append(eventsToEmit, pkgEvents...)
		return eventsToEmit
	}

	// Handle test-level events
	testEvents := c.handleTestLevelEvent(run, pkgResult, event)
	eventsToEmit = append(eventsToEmit, testEvents...)
	return eventsToEmit
}

// handlePackageEvent handles package-level events.
// Returns events to emit after the lock is released.
func (c *Collector) handlePackageEvent(run *Run, pkg *PackageResult, event parser.TestEvent) []Event {
	eventsToEmit := make([]Event, 0, 2)

	switch event.Action {
	case "output":
		if event.Output != "" {
			output := event.Output
			if len(output) > 0 && output[len(output)-1] == '\n' {
				output = output[:len(output)-1]
			}
			if output != "" {
				pkg.Output = output
				eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, pkg.Name))
			}
		}

	case "pass":
		pkg.Status = "ok"
		pkg.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		run.RunningPkgs--
		eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, pkg.Name))
		if evt := c.checkRunFinished(run); evt != nil {
			eventsToEmit = append(eventsToEmit, *evt)
		}

	case "fail":
		pkg.Status = "FAIL"
		pkg.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		run.RunningPkgs--
		eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, pkg.Name))
		if evt := c.checkRunFinished(run); evt != nil {
			eventsToEmit = append(eventsToEmit, *evt)
		}

	case "skip":
		pkg.Status = "?"
		pkg.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		run.RunningPkgs--
		eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, pkg.Name))
		if evt := c.checkRunFinished(run); evt != nil {
			eventsToEmit = append(eventsToEmit, *evt)
		}
	}

	return eventsToEmit
}

// handleTestLevelEvent handles test-level events.
// Returns events to emit after the lock is released.
func (c *Collector) handleTestLevelEvent(run *Run, pkg *PackageResult, event parser.TestEvent) []Event {
	eventsToEmit := make([]Event, 0, 2)
	testKey := event.Package + "/" + event.Test

	testResult, exists := run.TestResults[testKey]
	if !exists {
		testResult = &TestResult{
			Package: event.Package,
			Name:    event.Test,
			Status:  "running",
			Output:  make([]string, 0),
		}
		run.TestResults[testKey] = testResult
		pkg.TestOrder = append(pkg.TestOrder, event.Test)
	}

	switch event.Action {
	case "run":
		testResult.Status = "running"
		eventsToEmit = append(eventsToEmit, NewTestUpdatedEvent(run.ID, event.Package, event.Test))

	case "output":
		if event.Output != "" {
			output := strings.TrimRight(event.Output, "\n")

			// Extract summary line (lines starting with "===" or "---")
			if strings.HasPrefix(output, "===") || strings.HasPrefix(output, "---") {
				testResult.SummaryLine = output
				eventsToEmit = append(eventsToEmit, NewTestUpdatedEvent(run.ID, event.Package, event.Test))
			} else {
				testResult.Output = append(testResult.Output, output)
				eventsToEmit = append(eventsToEmit, NewTestOutputEvent(run.ID, event.Package, event.Test, output))
				eventsToEmit = append(eventsToEmit, NewTestUpdatedEvent(run.ID, event.Package, event.Test))
			}
		}

	case "pass":
		testResult.Status = "pass"
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		pkg.PassedTests++
		eventsToEmit = append(eventsToEmit, NewTestUpdatedEvent(run.ID, event.Package, event.Test))
		eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, event.Package))

	case "fail":
		testResult.Status = "fail"
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		pkg.FailedTests++
		eventsToEmit = append(eventsToEmit, NewTestUpdatedEvent(run.ID, event.Package, event.Test))
		eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, event.Package))

	case "skip":
		testResult.Status = "skip"
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
		pkg.SkippedTests++
		eventsToEmit = append(eventsToEmit, NewTestUpdatedEvent(run.ID, event.Package, event.Test))
		eventsToEmit = append(eventsToEmit, NewPackageUpdatedEvent(run.ID, event.Package))
	}

	return eventsToEmit
}

// startNewRun creates a new run and returns RunStarted event.
// The caller should emit the event after releasing the lock.
func (c *Collector) startNewRun(startTime time.Time) Event {
	runID := len(c.state.Runs) + 1
	run := NewRun(runID)
	run.StartTime = startTime
	c.state.Runs = append(c.state.Runs, run)
	c.state.CurrentRun = run
	c.currentRunWallStart = time.Now()
	return NewRunStartedEvent(runID)
}

// checkRunFinished checks if the current run has finished and returns RunFinished event.
// Returns nil if the run is not finished.
// The caller should emit the event after releasing the lock.
func (c *Collector) checkRunFinished(run *Run) *Event {
	if run.RunningPkgs == 0 {
		run.EndTime = time.Now()
		c.state.CurrentRun = nil
		evt := NewRunFinishedEvent(run.ID)
		return &evt
	}
	return nil
}

// Finish finishes the current run if any.
// This should be called when processing is complete or interrupted.
func (c *Collector) Finish() {
	c.mu.Lock()
	var eventToEmit *Event
	if c.state.CurrentRun != nil {
		run := c.state.CurrentRun

		// Determine end time: use last event time if available AND in replay mode, otherwise now
		endTime := time.Now()
		if c.isReplay && !c.lastEventTime.IsZero() {
			// Calculate simulated end time based on wall clock duration and replay rate
			// This ensures that the summary matches the TUI's "perceived" time
			wallDuration := time.Since(c.currentRunWallStart)

			// Apply rate (rate is inverse speed, e.g. 0.5 means 2x speed)
			// If rate is 0 (instant), we fall back to lastEventTime
			if c.replayRate > 0 {
				simulatedDuration := time.Duration(float64(wallDuration) / c.replayRate)
				endTime = run.StartTime.Add(simulatedDuration)
			} else {
				endTime = c.lastEventTime
			}
		}
		run.EndTime = endTime

		// Mark any still-running packages as interrupted and compute their elapsed time
		for _, pkg := range run.Packages {
			if pkg.Status == "" {
				pkg.Status = "interrupted"

				if c.isReplay && c.replayRate > 0 {
					// Calculate simulated elapsed time based on run duration and package start offset
					// This ensures consistency with TUI even if ReplayReader doesn't sleep exactly as expected
					wallRunDuration := time.Since(c.currentRunWallStart)
					simulatedRunDuration := time.Duration(float64(wallRunDuration) / c.replayRate)
					pkgOffset := pkg.StartTime.Sub(run.StartTime)

					pkg.Elapsed = simulatedRunDuration - pkgOffset
					if pkg.Elapsed < 0 {
						pkg.Elapsed = 0
					}
				} else {
					// Fallback for live runs
					pkg.Elapsed = endTime.Sub(pkg.StartTime)
				}
			}
		}

		runID := run.ID
		c.state.CurrentRun = nil
		evt := NewRunFinishedEvent(runID)
		eventToEmit = &evt
	}
	c.mu.Unlock()

	// Emit after releasing the lock
	if eventToEmit != nil {
		c.emit(*eventToEmit)
	}
}

// GetState returns a thread-safe snapshot of the current state.
// DEPRECATED: This method only holds the lock while returning the pointer.
// Accessing nested maps/slices after this method returns is NOT safe.
// Use WithState() instead to ensure proper synchronization during access.
func (c *Collector) GetState() *State {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a shallow copy of the state
	// The maps and slices are still shared, but that's okay since
	// the collector is the only writer
	return c.state
}

// GetRun returns a specific run by ID (1-indexed).
// DEPRECATED: This method only holds the lock while returning the pointer.
// Accessing nested maps/slices after this method returns is NOT safe.
// Use WithRun() instead to ensure proper synchronization during access.
func (c *Collector) GetRun(runID int) *Run {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if runID < 1 || runID > len(c.state.Runs) {
		return nil
	}
	return c.state.Runs[runID-1]
}

// GetCurrentRun returns the currently active run, or nil if none.
// DEPRECATED: This method only holds the lock while returning the pointer.
// Accessing nested maps/slices after this method returns is NOT safe.
// Use WithCurrentRun() instead to ensure proper synchronization during access.
func (c *Collector) GetCurrentRun() *Run {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.state.CurrentRun
}

// WithRun executes fn with the specified run while holding RLock.
// This ensures thread-safe access to the run and all nested structures
// (maps, slices, etc.) for the entire duration of the callback.
// The callback is not executed if the run does not exist.
func (c *Collector) WithRun(runID int, fn func(*Run)) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if runID < 1 || runID > len(c.state.Runs) {
		return
	}
	fn(c.state.Runs[runID-1])
}

// WithState executes fn with the state while holding RLock.
// This ensures thread-safe access to the state and all nested structures
// (maps, slices, etc.) for the entire duration of the callback.
func (c *Collector) WithState(fn func(*State)) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fn(c.state)
}

// WithCurrentRun executes fn with the current run while holding RLock.
// This ensures thread-safe access to the run and all nested structures
// (maps, slices, etc.) for the entire duration of the callback.
// The callback is not executed if there is no current run.
func (c *Collector) WithCurrentRun(fn func(*Run)) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.state.CurrentRun != nil {
		fn(c.state.CurrentRun)
	}
}
