package parser

import (
	"encoding/json"
	"time"
)

// Event represents either a build event or test event from go test -json output.
// Build events have ImportPath (no Time), test events have Time and Package.
type Event struct {
	// Common fields
	Action string `json:"Action"`
	Output string `json:"Output,omitempty"`

	// Build event fields (mutually exclusive with test event fields)
	ImportPath string `json:"ImportPath,omitempty"`

	// Test event fields
	Time        time.Time `json:"Time,omitempty"`
	Package     string    `json:"Package,omitempty"`
	Test        string    `json:"Test,omitempty"`
	Elapsed     float64   `json:"Elapsed,omitempty"`
	FailedBuild string    `json:"FailedBuild,omitempty"`
}

// IsBuildEvent returns true if this is a build event (has ImportPath, no Time)
func (e *Event) IsBuildEvent() bool {
	return e.ImportPath != "" && e.Time.IsZero()
}

// IsTestEvent returns true if this is a test event (has Time and Package)
func (e *Event) IsTestEvent() bool {
	return !e.Time.IsZero() || e.Package != ""
}

// ToBuildEvent converts to a BuildEvent (only call if IsBuildEvent() is true)
func (e *Event) ToBuildEvent() BuildEvent {
	return BuildEvent{
		ImportPath: e.ImportPath,
		Action:     e.Action,
		Output:     e.Output,
	}
}

// ToTestEvent converts to a TestEvent (only call if IsTestEvent() is true)
func (e *Event) ToTestEvent() TestEvent {
	return TestEvent{
		Time:        e.Time,
		Action:      e.Action,
		Package:     e.Package,
		Test:        e.Test,
		Output:      e.Output,
		Elapsed:     e.Elapsed,
		FailedBuild: e.FailedBuild,
	}
}

// BuildEvent represents a build event
type BuildEvent struct {
	ImportPath string
	Action     string // "build-output", "build-fail", "build-pass"
	Output     string
}

// TestEvent represents a test event from `go test -json` output
type TestEvent struct {
	Time        time.Time `json:"Time"`
	Action      string    `json:"Action"`
	Package     string    `json:"Package"`
	Test        string    `json:"Test,omitempty"`
	Output      string    `json:"Output,omitempty"`
	Elapsed     float64   `json:"Elapsed,omitempty"`
	Source      string    `json:"Source,omitempty"`
	ImportPath  string    `json:"ImportPath,omitempty"`
	FailedBuild string    `json:"FailedBuild,omitempty"`
}

// ParseEvent parses a single line of JSON from `go test -json` output
// Returns a union Event that can be either a build or test event
func ParseEvent(line []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(line, &event); err != nil {
		return event, err
	}
	return event, nil
}
