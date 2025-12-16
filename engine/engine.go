package engine

import (
	"bufio"
	"io"

	"github.com/ansel1/tang/parser"
)

// EventType identifies the type of event emitted by the engine
type EventType string

const (
	EventRawLine  EventType = "raw"      // Non-JSON line from input
	EventTest     EventType = "test"     // Parsed test event from go test -json
	EventBuild    EventType = "build"    // Parsed build event from go test -json
	EventError    EventType = "error"    // Error occurred during processing
	EventComplete EventType = "complete" // Input stream finished
)

// Event represents a single event emitted by the engine
type Event struct {
	Type       EventType
	RawLine    []byte            // Populated for EventRawLine
	TestEvent  parser.TestEvent  // Populated for EventTest
	BuildEvent parser.BuildEvent // Populated for EventBuild
	Error      error             // Populated for EventError
}

// Engine processes raw input and broadcasts events
// It maintains no state about tests - just parses and streams events
type Engine struct {
	// Output writers for pass-through file writing
	rawWriter  io.Writer
	jsonWriter io.Writer
}

// Option configures the engine
type Option func(*Engine)

// WithRawOutput configures engine to write all raw lines to a file
func WithRawOutput(w io.Writer) Option {
	return func(e *Engine) {
		e.rawWriter = w
	}
}

// WithJSONOutput configures engine to write parsed JSON events to a file
func WithJSONOutput(w io.Writer) Option {
	return func(e *Engine) {
		e.jsonWriter = w
	}
}

// NewEngine creates a new event processing engine
func NewEngine(opts ...Option) *Engine {
	e := &Engine{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Stream reads from input, parses lines, and emits events via channel
// The channel is closed when input is exhausted or an error occurs
func (e *Engine) Stream(input io.Reader) <-chan Event {
	events := make(chan Event, 100) // buffered channel for better throughput

	go func() {
		defer close(events)

		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			line := scanner.Bytes()

			// Always write raw output to file if configured
			if e.rawWriter != nil {
				e.rawWriter.Write(line)
				e.rawWriter.Write([]byte("\n"))
			}

			// Try to parse as JSON event (build or test)
			parsedEvent, err := parser.ParseEvent(line)
			if err != nil {
				// Not a JSON event - emit raw line
				// Make a copy of the line since scanner reuses the buffer
				lineCopy := make([]byte, len(line))
				copy(lineCopy, line)
				events <- Event{
					Type:    EventRawLine,
					RawLine: lineCopy,
				}
				continue
			}

			// Successfully parsed - write to JSON output file if configured
			if e.jsonWriter != nil {
				e.jsonWriter.Write(line)
				e.jsonWriter.Write([]byte("\n"))
			}

			// Determine event type and emit
			if parsedEvent.IsBuildEvent() {
				events <- Event{
					Type:       EventBuild,
					BuildEvent: parsedEvent.ToBuildEvent(),
				}
			} else if parsedEvent.IsTestEvent() {
				events <- Event{
					Type:      EventTest,
					TestEvent: parsedEvent.ToTestEvent(),
				}
			}
			// else: ignore unknown event types
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			events <- Event{
				Type:  EventError,
				Error: err,
			}
		}

		// Signal completion
		events <- Event{
			Type: EventComplete,
		}
	}()

	return events
}
