package results

// EventType identifies the type of event emitted by the Collector.
type EventType string

const (
	EventRunStarted     EventType = "run_started"     // A new test run has started
	EventRunFinished    EventType = "run_finished"    // A test run has finished
	EventPackageUpdated EventType = "package_updated" // A package's state changed
	EventTestUpdated    EventType = "test_updated"    // A test's state changed
	EventTestOutput     EventType = "test_output"     // A test produced output
	EventRawOutput      EventType = "raw_output"      // Raw non-test output
	EventNonTestOutput  EventType = "non_test_output" // Build errors, compilation output
)

// Event represents a high-level event emitted by the Collector.
type Event struct {
	Type        EventType
	RunID       int    // Which run this event belongs to
	PackageName string // For EventPackageUpdated, EventTestUpdated
	TestName    string // For EventTestUpdated
	RawLine     []byte // For EventRawOutput
	Output      string // For EventNonTestOutput
}

// NewRunStartedEvent creates a new RunStarted event.
func NewRunStartedEvent(runID int) Event {
	return Event{
		Type:  EventRunStarted,
		RunID: runID,
	}
}

// NewRunFinishedEvent creates a new RunFinished event.
func NewRunFinishedEvent(runID int) Event {
	return Event{
		Type:  EventRunFinished,
		RunID: runID,
	}
}

// NewPackageUpdatedEvent creates a new PackageUpdated event.
func NewPackageUpdatedEvent(runID int, packageName string) Event {
	return Event{
		Type:        EventPackageUpdated,
		RunID:       runID,
		PackageName: packageName,
	}
}

// NewTestUpdatedEvent creates a new TestUpdated event.
func NewTestUpdatedEvent(runID int, pkgName, testName string) Event {
	return Event{
		Type:        EventTestUpdated,
		RunID:       runID,
		PackageName: pkgName,
		TestName:    testName,
	}
}

func NewTestOutputEvent(runID int, pkgName, testName, output string) Event {
	return Event{
		Type:        EventTestOutput,
		RunID:       runID,
		PackageName: pkgName,
		TestName:    testName,
		Output:      output,
	}
}

// NewRawOutputEvent creates a new RawOutput event.
func NewRawOutputEvent(runID int, line []byte) Event {
	return Event{
		Type:    EventRawOutput,
		RunID:   runID,
		RawLine: line,
	}
}

// NewNonTestOutputEvent creates a new NonTestOutput event.
func NewNonTestOutputEvent(runID int, output string) Event {
	return Event{
		Type:   EventNonTestOutput,
		RunID:  runID,
		Output: output,
	}
}
