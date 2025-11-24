package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleOutput_ProcessEvents_BasicTest(t *testing.T) {
	events := make(chan engine.Event, 10)

	// Send test events
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestFoo",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "output",
			Package: "example.com/pkg",
			Test:    "TestFoo",
			Output:  "=== RUN   TestFoo\n",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    time.Now(),
			Action:  "pass",
			Package: "example.com/pkg",
			Test:    "TestFoo",
			Elapsed: 0.5,
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "PASSED: 1 passed, 0 failed, 0 skipped, 0 running, 1 total")
	assert.Contains(t, output, "=== RUN   TestFoo")
}

func TestSimpleOutput_ProcessEvents_FailedTest(t *testing.T) {
	events := make(chan engine.Event, 10)

	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestFail",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "output",
			Package: "example.com/pkg",
			Test:    "TestFail",
			Output:  "=== RUN   TestFail\n",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "output",
			Package: "example.com/pkg",
			Test:    "TestFail",
			Output:  "    test_fail.go:10: assertion failed\n",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "fail",
			Package: "example.com/pkg",
			Test:    "TestFail",
			Elapsed: 0.5,
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "FAILED: 0 passed, 1 failed, 0 skipped, 0 running, 1 total")
	assert.Contains(t, output, "assertion failed")
}

func TestSimpleOutput_ProcessEvents_SkippedTest(t *testing.T) {
	events := make(chan engine.Event, 10)

	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestSkip",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "skip",
			Package: "example.com/pkg",
			Test:    "TestSkip",
			Elapsed: 0.01,
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "PASSED: 0 passed, 0 failed, 1 skipped, 0 running, 1 total")
}

func TestSimpleOutput_ProcessEvents_MultipleTests(t *testing.T) {
	events := make(chan engine.Event, 20)

	// Test 1: pass
	events <- engine.Event{
		Type:      engine.EventTest,
		TestEvent: parser.TestEvent{Action: "run", Package: "pkg", Test: "Test1"},
	}
	events <- engine.Event{
		Type:      engine.EventTest,
		TestEvent: parser.TestEvent{Action: "pass", Package: "pkg", Test: "Test1", Elapsed: 0.1},
	}

	// Test 2: fail
	events <- engine.Event{
		Type:      engine.EventTest,
		TestEvent: parser.TestEvent{Action: "run", Package: "pkg", Test: "Test2"},
	}
	events <- engine.Event{
		Type:      engine.EventTest,
		TestEvent: parser.TestEvent{Action: "fail", Package: "pkg", Test: "Test2", Elapsed: 0.2},
	}

	// Test 3: skip
	events <- engine.Event{
		Type:      engine.EventTest,
		TestEvent: parser.TestEvent{Action: "run", Package: "pkg", Test: "Test3"},
	}
	events <- engine.Event{
		Type:      engine.EventTest,
		TestEvent: parser.TestEvent{Action: "skip", Package: "pkg", Test: "Test3", Elapsed: 0.01},
	}

	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "FAILED: 1 passed, 1 failed, 1 skipped, 0 running, 3 total")
}

func TestSimpleOutput_ProcessEvents_RawLines(t *testing.T) {
	events := make(chan engine.Event, 10)

	events <- engine.Event{
		Type:    engine.EventRawLine,
		RawLine: []byte("This is a raw line"),
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "run",
			Package: "pkg",
			Test:    "Test1",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "pass",
			Package: "pkg",
			Test:    "Test1",
		},
	}
	events <- engine.Event{
		Type:    engine.EventRawLine,
		RawLine: []byte("Another raw line"),
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "This is a raw line")
	assert.Contains(t, output, "Another raw line")
	assert.Contains(t, output, "PASSED: 1 passed, 0 failed, 0 skipped, 0 running, 1 total")
}

func TestSimpleOutput_ProcessEvents_WithError(t *testing.T) {
	events := make(chan engine.Event, 10)

	events <- engine.Event{
		Type:  engine.EventError,
		Error: assert.AnError,
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Error:")
}

func TestSimpleOutput_SummaryFormat(t *testing.T) {
	tests := []struct {
		name     string
		passed   int
		failed   int
		skipped  int
		running  int
		expected string
	}{
		{
			name:     "all passed",
			passed:   5,
			failed:   0,
			skipped:  0,
			running:  0,
			expected: "PASSED: 5 passed, 0 failed, 0 skipped, 0 running, 5 total",
		},
		{
			name:     "some failed",
			passed:   3,
			failed:   2,
			skipped:  0,
			running:  0,
			expected: "FAILED: 3 passed, 2 failed, 0 skipped, 0 running, 5 total",
		},
		{
			name:     "with skipped",
			passed:   2,
			failed:   0,
			skipped:  1,
			running:  0,
			expected: "PASSED: 2 passed, 0 failed, 1 skipped, 0 running, 3 total",
		},
		{
			name:     "with running",
			passed:   1,
			failed:   0,
			skipped:  0,
			running:  2,
			expected: "PASSED: 1 passed, 0 failed, 0 skipped, 2 running, 1 total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			simple := &SimpleOutput{
				writer:  &buf,
				output:  []string{},
				passed:  tt.passed,
				failed:  tt.failed,
				skipped: tt.skipped,
				running: tt.running,
			}

			err := simple.writeSummary()
			require.NoError(t, err)

			output := strings.TrimSpace(buf.String())
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestSimpleOutput_PackageLevelEvents(t *testing.T) {
	// Package-level events (without Test field) should not affect test counts
	events := make(chan engine.Event, 10)

	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "run",
			Package: "pkg",
			Test:    "Test1",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "pass",
			Package: "pkg",
			Test:    "Test1",
		},
	}
	// Package-level pass event
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Action:  "pass",
			Package: "pkg",
			Elapsed: 1.0,
		},
	}
	events <- engine.Event{Type: engine.EventComplete}
	close(events)

	var buf bytes.Buffer
	simple := NewSimpleOutput(&buf)
	err := simple.ProcessEvents(events)
	require.NoError(t, err)

	output := buf.String()
	// Should only count the test-level pass, not the package-level pass
	assert.Contains(t, output, "PASSED: 1 passed, 0 failed, 0 skipped, 0 running, 1 total")
}
