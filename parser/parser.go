package parser

import (
	"encoding/json"
	"time"
)

// TestEvent represents a single event from `go test -json` output
type TestEvent struct {
	Time       time.Time `json:"Time"`
	Action     string    `json:"Action"`
	Package    string    `json:"Package"`
	Test       string    `json:"Test,omitempty"`
	Output     string    `json:"Output,omitempty"`
	Elapsed    float64   `json:"Elapsed,omitempty"`
	Source     string    `json:"Source,omitempty"`
	ImportPath string    `json:"ImportPath,omitempty"`
}

// ParseEvent parses a single line of JSON from `go test -json` output
func ParseEvent(line []byte) (TestEvent, error) {
	var event TestEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return event, err
	}
	return event, nil
}
