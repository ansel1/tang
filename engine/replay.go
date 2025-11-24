package engine

import (
	"bufio"
	"io"
	"time"

	"github.com/ansel1/tang/parser"
)

// lineWithTiming represents a line from the input with its associated timestamp
type lineWithTiming struct {
	line      []byte
	timestamp time.Time
	isJSON    bool // true if this line parsed as a TestEvent
}

// ReplayReader wraps an io.Reader and replays its content with timing delays
// based on timestamps found in go test -json output
type ReplayReader struct {
	lines         []lineWithTiming
	rate          float64
	currentIdx    int
	lineBuffer    []byte
	bufferPos     int
	firstRead     bool
	lastEventTime time.Time
}

// NewReplayReader creates a new replay reader that simulates timing from test events
func NewReplayReader(r io.Reader, rate float64) (*ReplayReader, error) {
	// Read and parse all lines upfront
	var lines []lineWithTiming
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Bytes()
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)

		// Try to parse as test event to get timestamp
		testEvent, err := parser.ParseEvent(lineCopy)
		if err == nil && !testEvent.Time.IsZero() {
			// Successfully parsed with timestamp
			lines = append(lines, lineWithTiming{
				line:      lineCopy,
				timestamp: testEvent.Time,
				isJSON:    true,
			})
		} else {
			// Raw line or JSON without timestamp - use previous timestamp
			var ts time.Time
			if len(lines) > 0 {
				ts = lines[len(lines)-1].timestamp
			}
			lines = append(lines, lineWithTiming{
				line:      lineCopy,
				timestamp: ts,
				isJSON:    false,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &ReplayReader{
		lines:      lines,
		rate:       rate,
		currentIdx: 0,
		firstRead:  true,
	}, nil
}

// Read implements io.Reader, returning data line-by-line with timing delays
func (r *ReplayReader) Read(p []byte) (n int, err error) {
	// If we're in the middle of returning a line, continue from buffer
	if r.bufferPos < len(r.lineBuffer) {
		n = copy(p, r.lineBuffer[r.bufferPos:])
		r.bufferPos += n
		return n, nil
	}

	// Check if we've exhausted all lines
	if r.currentIdx >= len(r.lines) {
		return 0, io.EOF
	}

	// Get current line
	current := r.lines[r.currentIdx]

	// Calculate and apply delay (if not first read and rate > 0)
	if !r.firstRead && r.rate > 0 && !r.lastEventTime.IsZero() && !current.timestamp.IsZero() {
		actualDelay := current.timestamp.Sub(r.lastEventTime)
		if actualDelay > 0 {
			adjustedDelay := time.Duration(float64(actualDelay) * r.rate)
			time.Sleep(adjustedDelay)
		}
	}

	// Update state for next iteration
	r.firstRead = false
	if !current.timestamp.IsZero() {
		r.lastEventTime = current.timestamp
	}

	// Prepare line buffer (line + newline)
	r.lineBuffer = make([]byte, len(current.line)+1)
	copy(r.lineBuffer, current.line)
	r.lineBuffer[len(current.line)] = '\n'
	r.bufferPos = 0
	r.currentIdx++

	// Copy what we can to p
	n = copy(p, r.lineBuffer[r.bufferPos:])
	r.bufferPos += n

	return n, nil
}
