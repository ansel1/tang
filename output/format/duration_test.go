package format

import (
	"fmt"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string // duration string to parse
		expected string
	}{
		// Test cases < 60s (should use HH:MM:SS.mmm format now)
		{
			name:     "zero duration",
			duration: "0s",
			expected: "00:00:00.000",
		},
		{
			name:     "small duration",
			duration: "5.2s",
			expected: "00:00:05.200",
		},
		{
			name:     "boundary case 59.9s",
			duration: "59.9s",
			expected: "00:00:59.900",
		},
		{
			name:     "boundary case exactly 60s",
			duration: "60s",
			expected: "00:01:00.000",
		},
		// Test cases >= 60s (should use HH:MM:SS.mmm format)
		{
			name:     "just over 60s",
			duration: "60.1s",
			expected: "00:01:00.100",
		},
		{
			name:     "1 minute 30 seconds",
			duration: "1m30s",
			expected: "00:01:30.000",
		},
		{
			name:     "boundary case 3599.9s (just under 1 hour)",
			duration: "3599.9s",
			expected: "00:59:59.900",
		},
		{
			name:     "boundary case exactly 3600s (1 hour)",
			duration: "3600s",
			expected: "01:00:00.000",
		},
		{
			name:     "1 hour 23 minutes 45 seconds 678 milliseconds",
			duration: "1h23m45s678ms",
			expected: "01:23:45.678",
		},
		{
			name:     "multiple hours",
			duration: "2h30m15s",
			expected: "02:30:15.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the duration string
			d, err := parseDuration(tt.duration)
			if err != nil {
				t.Fatalf("Failed to parse duration %s: %v", tt.duration, err)
			}

			result := formatDuration(d)
			if result != tt.expected {
				t.Errorf("formatDuration(%s) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

// parseDuration is a helper function to parse duration strings for testing.
// It handles the standard Go duration format (e.g., "1h23m45s678ms").
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// TestPropertyTimeFormatConsistency is a property-based test that verifies time format consistency.
// **Feature: test-summary, Property 8: Time format consistency**
// **Validates: Requirements 5.3, 6.4, 9.5**
//
// Property: For any duration the format SHALL be "HH:MM:SS.mmm".
func TestPropertyTimeFormatConsistency(t *testing.T) {
	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random durations across the full range
		durations := generateRandomDurations(i)

		for _, d := range durations {
			formatted := formatDuration(d)

			// Should be in "HH:MM:SS.mmm" format
			// Verify format matches pattern: HH:MM:SS.mmm
			if !isHMSFormat(formatted) {
				t.Errorf("Iteration %d: Duration %v formatted as '%s', expected HH:MM:SS.mmm format",
					i, d, formatted)
			}

			// Verify the time components are correct
			if !verifyHMSFormat(formatted, d) {
				t.Errorf("Iteration %d: Duration %v formatted as '%s', components don't match",
					i, d, formatted)
			}
		}
	}
}

// generateRandomDurations generates a diverse set of durations for property testing.
// It includes edge cases around the 60-second boundary and various ranges.
func generateRandomDurations(seed int) []time.Duration {
	// Use seed for deterministic randomness
	rng := func(n int) int {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		return seed % n
	}

	durations := make([]time.Duration, 0)

	// Generate 10-20 random durations per iteration
	numDurations := rng(11) + 10

	for i := 0; i < numDurations; i++ {
		roll := rng(100)

		var d time.Duration
		if roll < 30 {
			// 30% - Very short durations (0-10s)
			milliseconds := rng(10000)
			d = time.Duration(milliseconds) * time.Millisecond
		} else if roll < 50 {
			// 20% - Near 60s boundary (55s-65s)
			milliseconds := 55000 + rng(10000)
			d = time.Duration(milliseconds) * time.Millisecond
		} else if roll < 70 {
			// 20% - Medium durations (10s-55s)
			milliseconds := 10000 + rng(45000)
			d = time.Duration(milliseconds) * time.Millisecond
		} else if roll < 85 {
			// 15% - Long durations (1min-10min)
			seconds := 60 + rng(540)
			d = time.Duration(seconds) * time.Second
		} else {
			// 15% - Very long durations (10min-2hours)
			seconds := 600 + rng(6600)
			d = time.Duration(seconds) * time.Second
		}

		durations = append(durations, d)
	}

	// Always include critical boundary cases
	durations = append(durations,
		0*time.Second,                                     // Zero
		100*time.Millisecond,                              // Very short
		59900*time.Millisecond,                            // Just below 60s
		60000*time.Millisecond,                            // Exactly 60s
		60100*time.Millisecond,                            // Just above 60s
		3599900*time.Millisecond,                          // Just below 1 hour
		3600000*time.Millisecond,                          // Exactly 1 hour
		3600100*time.Millisecond,                          // Just above 1 hour
		time.Duration(rng(59000))*time.Millisecond,        // Random < 60s
		time.Duration(60000+rng(300000))*time.Millisecond, // Random >= 60s
	)

	return durations
}

// isHMSFormat checks if a string matches the "HH:MM:SS.mmm" format.
func isHMSFormat(s string) bool {
	// Expected format: HH:MM:SS.mmm (12 characters minimum)
	if len(s) < 12 {
		return false
	}

	// Check structure: XX:XX:XX.XXX
	if s[2] != ':' || s[5] != ':' || s[8] != '.' {
		return false
	}

	// Check all other positions are digits
	positions := []int{0, 1, 3, 4, 6, 7, 9, 10, 11}
	for _, pos := range positions {
		if pos >= len(s) || s[pos] < '0' || s[pos] > '9' {
			return false
		}
	}

	return true
}

// verifyHMSFormat checks if the formatted string matches the expected duration components.
func verifyHMSFormat(formatted string, d time.Duration) bool {
	// Parse the components
	var hours, minutes, seconds, milliseconds int
	_, err := fmt.Sscanf(formatted, "%02d:%02d:%02d.%03d", &hours, &minutes, &seconds, &milliseconds)
	if err != nil {
		return false
	}

	// Calculate expected components
	expectedHours := int(d.Hours())
	expectedMinutes := int(d.Minutes()) % 60
	expectedSeconds := int(d.Seconds()) % 60
	expectedMilliseconds := int(d.Milliseconds()) % 1000

	// Verify each component
	return hours == expectedHours &&
		minutes == expectedMinutes &&
		seconds == expectedSeconds &&
		milliseconds == expectedMilliseconds
}
