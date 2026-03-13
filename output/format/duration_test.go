package format

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"negative", -1 * time.Second, "0s"},
		{"sub-millisecond", 500 * time.Microsecond, "500µs"},
		{"1.23ms", 1230 * time.Microsecond, "1.23ms"},
		{"12.3ms", 12300 * time.Microsecond, "12.3ms"},
		{"123ms", 123 * time.Millisecond, "123ms"},
		{"5.2s", 5200 * time.Millisecond, "5.2s"},
		{"59.9s", 59900 * time.Millisecond, "59.9s"},
		{"exactly 60s", 60 * time.Second, "1m0s"},
		{"60.1s", 60100 * time.Millisecond, "1m0.1s"},
		{"1m30s", 90 * time.Second, "1m30s"},
		{"5m45.5s", 5*time.Minute + 45500*time.Millisecond, "5m45.5s"},
		{"1h0m0s", 1 * time.Hour, "1h0m0s"},
		{"2h30m15s", 2*time.Hour + 30*time.Minute + 15*time.Second, "2h30m15s"},
		{"1h23m45.678s", 1*time.Hour + 23*time.Minute + 45*time.Second + 678*time.Millisecond, "1h23m45.678s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatDurationTruncates(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"truncates seconds to ms", 7925430 * time.Microsecond, "7.925s"},
		{"truncates ms to µs", 3455555 * time.Nanosecond, "3.455ms"},
		{"truncates µs to ns", 123456 * time.Nanosecond, "123.456µs"},
		{"no extra precision seconds", 1 * time.Second, "1s"},
		{"no extra precision ms", 1 * time.Millisecond, "1ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}
