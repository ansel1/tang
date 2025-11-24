package engine

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ansel1/tang/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Stream_ParsesValidJSON(t *testing.T) {
	input := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":1.5}`

	eng := NewEngine()
	events := eng.Stream(strings.NewReader(input))

	var collected []Event
	for evt := range events {
		collected = append(collected, evt)
	}

	// Should have 2 test events + 1 complete event
	require.Len(t, collected, 3)

	// First event: run
	assert.Equal(t, EventTest, collected[0].Type)
	assert.Equal(t, "run", collected[0].TestEvent.Action)
	assert.Equal(t, "TestFoo", collected[0].TestEvent.Test)

	// Second event: pass
	assert.Equal(t, EventTest, collected[1].Type)
	assert.Equal(t, "pass", collected[1].TestEvent.Action)
	assert.Equal(t, 1.5, collected[1].TestEvent.Elapsed)

	// Third event: complete
	assert.Equal(t, EventComplete, collected[2].Type)
}

func TestEngine_Stream_HandlesNonJSONLines(t *testing.T) {
	input := `This is not JSON
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
Another non-JSON line
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":1.5}`

	eng := NewEngine()
	events := eng.Stream(strings.NewReader(input))

	var collected []Event
	for evt := range events {
		collected = append(collected, evt)
	}

	// Should have: 2 raw lines + 2 test events + 1 complete = 5 events
	require.Len(t, collected, 5)

	// First: raw line
	assert.Equal(t, EventRawLine, collected[0].Type)
	assert.Equal(t, "This is not JSON", string(collected[0].RawLine))

	// Second: test event
	assert.Equal(t, EventTest, collected[1].Type)
	assert.Equal(t, "run", collected[1].TestEvent.Action)

	// Third: raw line
	assert.Equal(t, EventRawLine, collected[2].Type)
	assert.Equal(t, "Another non-JSON line", string(collected[2].RawLine))

	// Fourth: test event
	assert.Equal(t, EventTest, collected[3].Type)
	assert.Equal(t, "pass", collected[3].TestEvent.Action)

	// Fifth: complete
	assert.Equal(t, EventComplete, collected[4].Type)
}

func TestEngine_Stream_WritesRawOutput(t *testing.T) {
	input := `This is not JSON
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}`

	var rawBuf bytes.Buffer
	eng := NewEngine(WithRawOutput(&rawBuf))
	events := eng.Stream(strings.NewReader(input))

	// Consume all events
	for range events {
	}

	// Check raw output contains both lines
	output := rawBuf.String()
	assert.Contains(t, output, "This is not JSON\n")
	assert.Contains(t, output, `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}`)
}

func TestEngine_Stream_WritesJSONOutput(t *testing.T) {
	input := `This is not JSON
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}
Another non-JSON line
{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":1.5}`

	var jsonBuf bytes.Buffer
	eng := NewEngine(WithJSONOutput(&jsonBuf))
	events := eng.Stream(strings.NewReader(input))

	// Consume all events
	for range events {
	}

	// Check JSON output contains only valid JSON lines
	output := jsonBuf.String()
	assert.Contains(t, output, `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}`)
	assert.Contains(t, output, `{"Time":"2024-01-01T00:00:01Z","Action":"pass","Package":"example.com/pkg","Test":"TestFoo","Elapsed":1.5}`)
	assert.NotContains(t, output, "This is not JSON")
	assert.NotContains(t, output, "Another non-JSON line")
}

func TestEngine_Stream_BothRawAndJSONOutput(t *testing.T) {
	input := `Non-JSON line
{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"example.com/pkg","Test":"TestFoo"}`

	var rawBuf, jsonBuf bytes.Buffer
	eng := NewEngine(
		WithRawOutput(&rawBuf),
		WithJSONOutput(&jsonBuf),
	)
	events := eng.Stream(strings.NewReader(input))

	// Consume all events
	for range events {
	}

	// Raw should have both lines
	rawOutput := rawBuf.String()
	assert.Contains(t, rawOutput, "Non-JSON line\n")
	assert.Contains(t, rawOutput, `{"Time":"2024-01-01T00:00:00Z"`)

	// JSON should have only the JSON line
	jsonOutput := jsonBuf.String()
	assert.Contains(t, jsonOutput, `{"Time":"2024-01-01T00:00:00Z"`)
	assert.NotContains(t, jsonOutput, "Non-JSON line")
}

func TestEngine_Stream_EmptyInput(t *testing.T) {
	eng := NewEngine()
	events := eng.Stream(strings.NewReader(""))

	var collected []Event
	for evt := range events {
		collected = append(collected, evt)
	}

	// Should only have complete event
	require.Len(t, collected, 1)
	assert.Equal(t, EventComplete, collected[0].Type)
}

func TestEngine_Stream_PreservesEventOrder(t *testing.T) {
	input := `{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"pkg1","Test":"Test1"}
{"Time":"2024-01-01T00:00:01Z","Action":"run","Package":"pkg2","Test":"Test2"}
{"Time":"2024-01-01T00:00:02Z","Action":"pass","Package":"pkg1","Test":"Test1"}
{"Time":"2024-01-01T00:00:03Z","Action":"pass","Package":"pkg2","Test":"Test2"}`

	eng := NewEngine()
	events := eng.Stream(strings.NewReader(input))

	var testNames []string
	for evt := range events {
		if evt.Type == EventTest {
			testNames = append(testNames, evt.TestEvent.Test)
		}
	}

	// Events should be in order
	require.Equal(t, []string{"Test1", "Test2", "Test1", "Test2"}, testNames)
}

// errReader simulates a reader that returns an error
type errReader struct{}

func (e errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func TestEngine_Stream_HandlesReadError(t *testing.T) {
	eng := NewEngine()
	events := eng.Stream(errReader{})

	var collected []Event
	for evt := range events {
		collected = append(collected, evt)
	}

	// Should have error event and complete event
	require.Len(t, collected, 2)
	assert.Equal(t, EventError, collected[0].Type)
	assert.Error(t, collected[0].Error)
	assert.Equal(t, EventComplete, collected[1].Type)
}

func TestEngine_Stream_CopiesLineBuffer(t *testing.T) {
	// This test ensures that raw line bytes are properly copied
	// Scanner reuses its internal buffer, so we need to copy
	input := `line1
line2
line3`

	eng := NewEngine()
	events := eng.Stream(strings.NewReader(input))

	var rawLines [][]byte
	for evt := range events {
		if evt.Type == EventRawLine {
			rawLines = append(rawLines, evt.RawLine)
		}
	}

	require.Len(t, rawLines, 3)
	assert.Equal(t, "line1", string(rawLines[0]))
	assert.Equal(t, "line2", string(rawLines[1]))
	assert.Equal(t, "line3", string(rawLines[2]))
}

func TestEngine_Stream_RealGoTestOutput(t *testing.T) {
	// Test with actual go test -json output format
	input := `{"Time":"2024-01-01T10:00:00.123456Z","Action":"start","Package":"example.com/mypackage"}
{"Time":"2024-01-01T10:00:00.234567Z","Action":"run","Package":"example.com/mypackage","Test":"TestExample"}
{"Time":"2024-01-01T10:00:00.345678Z","Action":"output","Package":"example.com/mypackage","Test":"TestExample","Output":"=== RUN   TestExample\n"}
{"Time":"2024-01-01T10:00:00.456789Z","Action":"output","Package":"example.com/mypackage","Test":"TestExample","Output":"--- PASS: TestExample (0.10s)\n"}
{"Time":"2024-01-01T10:00:00.567890Z","Action":"pass","Package":"example.com/mypackage","Test":"TestExample","Elapsed":0.1}
{"Time":"2024-01-01T10:00:00.678901Z","Action":"pass","Package":"example.com/mypackage","Elapsed":0.2}`

	eng := NewEngine()
	events := eng.Stream(strings.NewReader(input))

	var testEvents []parser.TestEvent
	for evt := range events {
		if evt.Type == EventTest {
			testEvents = append(testEvents, evt.TestEvent)
		}
	}

	require.Len(t, testEvents, 6)
	assert.Equal(t, "start", testEvents[0].Action)
	assert.Equal(t, "run", testEvents[1].Action)
	assert.Equal(t, "output", testEvents[2].Action)
	assert.Equal(t, "output", testEvents[3].Action)
	assert.Equal(t, "pass", testEvents[4].Action)
	assert.Equal(t, "pass", testEvents[5].Action)
}
