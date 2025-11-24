package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
)

// SimpleOutput writes simple text output for -notty mode
// It accumulates output and summary data, then writes everything at completion
type SimpleOutput struct {
	writer io.Writer
	output []string

	// Summary statistics
	passed  int
	failed  int
	skipped int
	running int
}

// NewSimpleOutput creates a simple output writer
func NewSimpleOutput(w io.Writer) *SimpleOutput {
	return &SimpleOutput{
		writer: w,
		output: make([]string, 0),
	}
}

// ProcessEvents consumes events from channel and writes to output
func (s *SimpleOutput) ProcessEvents(events <-chan engine.Event) error {
	for evt := range events {
		switch evt.Type {
		case engine.EventRawLine:
			// Accumulate raw (non-JSON) lines for output
			s.output = append(s.output, string(evt.RawLine))

		case engine.EventTest:
			// Process test event to update summary and accumulate output
			s.handleTestEvent(evt.TestEvent)

		case engine.EventComplete:
			// Write all accumulated output
			return s.writeOutput()

		case engine.EventError:
			// Write error to stderr would be better, but we'll add to output
			s.output = append(s.output, fmt.Sprintf("Error: %v", evt.Error))
		}
	}
	return nil
}

// handleTestEvent processes a test event and updates state
func (s *SimpleOutput) handleTestEvent(evt parser.TestEvent) {
	// Track summary statistics
	switch evt.Action {
	case "run":
		if evt.Test != "" {
			s.running++
		}

	case "pass":
		if evt.Test != "" {
			s.passed++
			s.running--
		}

	case "fail":
		if evt.Test != "" {
			s.failed++
			s.running--
		}

	case "skip":
		if evt.Test != "" {
			s.skipped++
			s.running--
		}
	}

	// Accumulate output lines
	if evt.Output != "" {
		s.output = append(s.output, strings.TrimRight(evt.Output, "\n"))
	}
}

// writeOutput writes all accumulated output and summary
func (s *SimpleOutput) writeOutput() error {
	// Write summary first
	if err := s.writeSummary(); err != nil {
		return err
	}

	// Write all output lines
	for _, line := range s.output {
		if _, err := fmt.Fprintln(s.writer, line); err != nil {
			return err
		}
	}

	return nil
}

// HasFailures returns true if any tests failed
func (s *SimpleOutput) HasFailures() bool {
	return s.failed > 0
}

// writeSummary writes the final summary line
func (s *SimpleOutput) writeSummary() error {
	total := s.passed + s.failed + s.skipped
	statusWord := "PASSED"
	if s.failed > 0 {
		statusWord = "FAILED"
	}

	summary := fmt.Sprintf("%s: %d passed, %d failed, %d skipped, %d running, %d total",
		statusWord, s.passed, s.failed, s.skipped, s.running, total)

	_, err := fmt.Fprintln(s.writer, summary)
	return err
}
