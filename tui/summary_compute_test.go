package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
)

// TestComputeSummaryBasic tests basic summary computation with mixed results.
func TestComputeSummaryBasic(t *testing.T) {
	collector := NewSummaryCollector()

	// Add some test events
	baseTime := time.Now()

	// Package 1: 2 pass, 1 fail
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime,
		Action:  "run",
		Package: "pkg1",
		Test:    "TestA",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(1 * time.Second),
		Action:  "pass",
		Package: "pkg1",
		Test:    "TestA",
		Elapsed: 1.0,
	})

	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(2 * time.Second),
		Action:  "run",
		Package: "pkg1",
		Test:    "TestB",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(3 * time.Second),
		Action:  "fail",
		Package: "pkg1",
		Test:    "TestB",
		Elapsed: 1.0,
	})

	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(4 * time.Second),
		Action:  "run",
		Package: "pkg1",
		Test:    "TestC",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(5 * time.Second),
		Action:  "pass",
		Package: "pkg1",
		Test:    "TestC",
		Elapsed: 1.0,
	})

	collector.AddPackageResult("pkg1", "FAIL", 5*time.Second)

	// Package 2: 1 pass, 1 skip
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(6 * time.Second),
		Action:  "run",
		Package: "pkg2",
		Test:    "TestD",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(7 * time.Second),
		Action:  "pass",
		Package: "pkg2",
		Test:    "TestD",
		Elapsed: 1.0,
	})

	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(8 * time.Second),
		Action:  "run",
		Package: "pkg2",
		Test:    "TestE",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(9 * time.Second),
		Action:  "skip",
		Package: "pkg2",
		Test:    "TestE",
		Elapsed: 1.0,
	})

	collector.AddPackageResult("pkg2", "ok", 3*time.Second)

	// Compute summary
	summary := ComputeSummary(collector, 10*time.Second)

	// Verify overall statistics
	if summary.TotalTests != 5 {
		t.Errorf("Expected 5 total tests, got %d", summary.TotalTests)
	}
	if summary.PassedTests != 3 {
		t.Errorf("Expected 3 passed tests, got %d", summary.PassedTests)
	}
	if summary.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", summary.FailedTests)
	}
	if summary.SkippedTests != 1 {
		t.Errorf("Expected 1 skipped test, got %d", summary.SkippedTests)
	}
	if summary.PackageCount != 2 {
		t.Errorf("Expected 2 packages, got %d", summary.PackageCount)
	}

	// Verify failures collection
	if len(summary.Failures) != 1 {
		t.Errorf("Expected 1 failure, got %d", len(summary.Failures))
	}
	if len(summary.Failures) > 0 && summary.Failures[0].Name != "TestB" {
		t.Errorf("Expected failure TestB, got %s", summary.Failures[0].Name)
	}

	// Verify skipped collection
	if len(summary.Skipped) != 1 {
		t.Errorf("Expected 1 skipped test, got %d", len(summary.Skipped))
	}
	if len(summary.Skipped) > 0 && summary.Skipped[0].Name != "TestE" {
		t.Errorf("Expected skipped TestE, got %s", summary.Skipped[0].Name)
	}

	// Verify package statistics
	if summary.FastestPackage == nil {
		t.Error("Expected fastest package to be set")
	} else if summary.FastestPackage.Name != "pkg2" {
		t.Errorf("Expected fastest package to be pkg2, got %s", summary.FastestPackage.Name)
	}

	if summary.SlowestPackage == nil {
		t.Error("Expected slowest package to be set")
	} else if summary.SlowestPackage.Name != "pkg1" {
		t.Errorf("Expected slowest package to be pkg1, got %s", summary.SlowestPackage.Name)
	}

	if summary.MostTestsPackage == nil {
		t.Error("Expected most tests package to be set")
	} else if summary.MostTestsPackage.Name != "pkg1" {
		t.Errorf("Expected most tests package to be pkg1, got %s", summary.MostTestsPackage.Name)
	}
}

// TestComputeSummarySlowTests tests slow test detection and sorting.
func TestComputeSummarySlowTests(t *testing.T) {
	collector := NewSummaryCollector()
	baseTime := time.Now()

	// Add tests with varying durations
	tests := []struct {
		name    string
		elapsed float64
	}{
		{"TestFast", 5.0},
		{"TestSlow1", 15.0},
		{"TestSlow2", 25.0},
		{"TestSlow3", 12.0},
	}

	for i, test := range tests {
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(i) * time.Second),
			Action:  "run",
			Package: "pkg1",
			Test:    test.name,
		})
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(i+1) * time.Second),
			Action:  "pass",
			Package: "pkg1",
			Test:    test.name,
			Elapsed: test.elapsed,
		})
	}

	collector.AddPackageResult("pkg1", "ok", 60*time.Second)

	// Compute summary with 10s threshold
	summary := ComputeSummary(collector, 10*time.Second)

	// Verify slow tests detected
	if len(summary.SlowTests) != 3 {
		t.Errorf("Expected 3 slow tests, got %d", len(summary.SlowTests))
	}

	// Verify slow tests are sorted by duration (descending)
	if len(summary.SlowTests) >= 3 {
		if summary.SlowTests[0].Name != "TestSlow2" {
			t.Errorf("Expected first slow test to be TestSlow2 (25s), got %s", summary.SlowTests[0].Name)
		}
		if summary.SlowTests[1].Name != "TestSlow1" {
			t.Errorf("Expected second slow test to be TestSlow1 (15s), got %s", summary.SlowTests[1].Name)
		}
		if summary.SlowTests[2].Name != "TestSlow3" {
			t.Errorf("Expected third slow test to be TestSlow3 (12s), got %s", summary.SlowTests[2].Name)
		}
	}
}

// TestComputeSummaryEmptyResults tests summary with no tests.
func TestComputeSummaryEmptyResults(t *testing.T) {
	collector := NewSummaryCollector()

	summary := ComputeSummary(collector, 10*time.Second)

	if summary.TotalTests != 0 {
		t.Errorf("Expected 0 total tests, got %d", summary.TotalTests)
	}
	if summary.PackageCount != 0 {
		t.Errorf("Expected 0 packages, got %d", summary.PackageCount)
	}
	if len(summary.Failures) != 0 {
		t.Errorf("Expected 0 failures, got %d", len(summary.Failures))
	}
	if len(summary.Skipped) != 0 {
		t.Errorf("Expected 0 skipped, got %d", len(summary.Skipped))
	}
	if len(summary.SlowTests) != 0 {
		t.Errorf("Expected 0 slow tests, got %d", len(summary.SlowTests))
	}
}

// TestComputeSummaryAllPass tests summary with all passing tests.
func TestComputeSummaryAllPass(t *testing.T) {
	collector := NewSummaryCollector()
	baseTime := time.Now()

	// Add 3 passing tests
	for i := 0; i < 3; i++ {
		testName := string(rune('A' + i))
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(i*2) * time.Second),
			Action:  "run",
			Package: "pkg1",
			Test:    "Test" + testName,
		})
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(i*2+1) * time.Second),
			Action:  "pass",
			Package: "pkg1",
			Test:    "Test" + testName,
			Elapsed: 1.0,
		})
	}

	collector.AddPackageResult("pkg1", "ok", 6*time.Second)

	summary := ComputeSummary(collector, 10*time.Second)

	if summary.TotalTests != 3 {
		t.Errorf("Expected 3 total tests, got %d", summary.TotalTests)
	}
	if summary.PassedTests != 3 {
		t.Errorf("Expected 3 passed tests, got %d", summary.PassedTests)
	}
	if summary.FailedTests != 0 {
		t.Errorf("Expected 0 failed tests, got %d", summary.FailedTests)
	}
	if summary.SkippedTests != 0 {
		t.Errorf("Expected 0 skipped tests, got %d", summary.SkippedTests)
	}
	if len(summary.Failures) != 0 {
		t.Errorf("Expected empty failures list, got %d", len(summary.Failures))
	}
	if len(summary.Skipped) != 0 {
		t.Errorf("Expected empty skipped list, got %d", len(summary.Skipped))
	}
}

// TestPropertySlowTestThreshold is a property-based test that verifies slow test threshold detection.
// **Feature: test-summary, Property 5: Slow test threshold**
// **Validates: Requirements 5.1, 5.5**
//
// Property: For any test with elapsed time greater than the threshold, that test SHALL appear in the SLOW TESTS section.
func TestPropertySlowTestThreshold(t *testing.T) {
	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random test events with varying durations
		events, threshold, expectedSlowTests := generateRandomTestsWithDurations(i)

		// Create collector and process events
		collector := NewSummaryCollector()
		for _, event := range events {
			collector.AddTestEvent(event)
		}

		// Add package results for all packages
		packages := make(map[string]bool)
		for _, event := range events {
			if event.Package != "" {
				packages[event.Package] = true
			}
		}
		for pkg := range packages {
			collector.AddPackageResult(pkg, "ok", 100*time.Millisecond)
		}

		// Compute summary with the generated threshold
		summary := ComputeSummary(collector, threshold)

		// Verify that all tests exceeding threshold appear in slow tests
		for testKey, duration := range expectedSlowTests {
			found := false
			for _, slowTest := range summary.SlowTests {
				if slowTest.Package+"/"+slowTest.Name == testKey {
					found = true
					// Verify the duration is correct
					if slowTest.Elapsed != duration {
						t.Errorf("Iteration %d: Slow test %s has incorrect duration: expected %v, got %v",
							i, testKey, duration, slowTest.Elapsed)
					}
					break
				}
			}
			if !found {
				t.Errorf("Iteration %d: Test %s with duration %v (>= threshold %v) not found in slow tests",
					i, testKey, duration, threshold)
			}
		}

		// Verify that no tests below threshold appear in slow tests
		for _, slowTest := range summary.SlowTests {
			if slowTest.Elapsed < threshold {
				t.Errorf("Iteration %d: Test %s/%s with duration %v (< threshold %v) incorrectly appears in slow tests",
					i, slowTest.Package, slowTest.Name, slowTest.Elapsed, threshold)
			}
		}

		// Verify count matches
		if len(summary.SlowTests) != len(expectedSlowTests) {
			t.Errorf("Iteration %d: Expected %d slow tests, got %d (threshold: %v)",
				i, len(expectedSlowTests), len(summary.SlowTests), threshold)
		}
	}
}

// generateRandomTestsWithDurations generates random test events with varying durations.
// It returns the events, the threshold to use, and a map of tests that should be considered slow.
func generateRandomTestsWithDurations(seed int) ([]parser.TestEvent, time.Duration, map[string]time.Duration) {
	// Use seed for deterministic randomness
	rng := func(n int) int {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		return seed % n
	}

	now := time.Now()
	events := make([]parser.TestEvent, 0)
	expectedSlowTests := make(map[string]time.Duration)

	// Generate a random threshold between 5s and 20s
	thresholdSeconds := rng(16) + 5 // 5-20 seconds
	threshold := time.Duration(thresholdSeconds) * time.Second

	// Generate 1-30 tests
	numTests := rng(30) + 1
	numPackages := rng(5) + 1
	packages := make([]string, numPackages)
	for i := 0; i < numPackages; i++ {
		packages[i] = "example.com/pkg" + string(rune('A'+i))
	}

	for i := 0; i < numTests; i++ {
		pkg := packages[rng(numPackages)]
		testName := "Test" + string(rune('A'+i%26))
		if i >= 26 {
			testName += string(rune('0' + i/26))
		}

		// Generate random duration between 0.1s and 30s
		// Use a distribution that creates both fast and slow tests
		durationRoll := rng(100)
		var durationSeconds float64
		if durationRoll < 60 {
			// 60% fast tests (0.1s - 4.9s)
			durationSeconds = 0.1 + float64(rng(49))/10.0
		} else if durationRoll < 80 {
			// 20% medium tests (5s - threshold-1s)
			if thresholdSeconds > 5 {
				durationSeconds = 5.0 + float64(rng((thresholdSeconds-5)*10))/10.0
			} else {
				durationSeconds = 0.1 + float64(rng(49))/10.0
			}
		} else {
			// 20% slow tests (threshold to 30s)
			durationSeconds = float64(thresholdSeconds) + float64(rng((30-thresholdSeconds)*10))/10.0
		}

		duration := time.Duration(durationSeconds * float64(time.Second))

		// Add run event
		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100) * time.Millisecond),
			Action:  "run",
			Package: pkg,
			Test:    testName,
		})

		// Add pass event with duration
		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100+50) * time.Millisecond),
			Action:  "pass",
			Package: pkg,
			Test:    testName,
			Elapsed: durationSeconds,
		})

		// Track if this test should be slow
		if duration >= threshold {
			testKey := pkg + "/" + testName
			expectedSlowTests[testKey] = duration
		}
	}

	return events, threshold, expectedSlowTests
}

// TestPropertySlowTestOrdering is a property-based test that verifies slow tests are sorted by duration.
// **Feature: test-summary, Property 6: Slow test ordering**
// **Validates: Requirements 5.2**
//
// Property: For any two slow tests A and B where A.elapsed > B.elapsed, test A SHALL appear before test B in the SLOW TESTS section.
func TestPropertySlowTestOrdering(t *testing.T) {
	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random slow tests with varying durations
		events, threshold := generateRandomSlowTests(i)

		// Create collector and process events
		collector := NewSummaryCollector()
		for _, event := range events {
			collector.AddTestEvent(event)
		}

		// Add package results for all packages
		packages := make(map[string]bool)
		for _, event := range events {
			if event.Package != "" {
				packages[event.Package] = true
			}
		}
		for pkg := range packages {
			collector.AddPackageResult(pkg, "ok", 100*time.Millisecond)
		}

		// Compute summary
		summary := ComputeSummary(collector, threshold)

		// Verify that slow tests are sorted in descending order by duration
		for j := 0; j < len(summary.SlowTests)-1; j++ {
			currentTest := summary.SlowTests[j]
			nextTest := summary.SlowTests[j+1]

			if currentTest.Elapsed < nextTest.Elapsed {
				t.Errorf("Iteration %d: Slow tests not sorted correctly at index %d: "+
					"test %s/%s (%v) should be >= test %s/%s (%v)",
					i, j,
					currentTest.Package, currentTest.Name, currentTest.Elapsed,
					nextTest.Package, nextTest.Name, nextTest.Elapsed)
			}
		}

		// Additional verification: all slow tests should exceed threshold
		for _, slowTest := range summary.SlowTests {
			if slowTest.Elapsed < threshold {
				t.Errorf("Iteration %d: Test %s/%s with duration %v (< threshold %v) incorrectly appears in slow tests",
					i, slowTest.Package, slowTest.Name, slowTest.Elapsed, threshold)
			}
		}
	}
}

// generateRandomSlowTests generates random test events where all tests exceed the threshold.
// This ensures we're testing the sorting property specifically.
func generateRandomSlowTests(seed int) ([]parser.TestEvent, time.Duration) {
	// Use seed for deterministic randomness
	rng := func(n int) int {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		return seed % n
	}

	now := time.Now()
	events := make([]parser.TestEvent, 0)

	// Generate a random threshold between 5s and 15s
	thresholdSeconds := rng(11) + 5 // 5-15 seconds
	threshold := time.Duration(thresholdSeconds) * time.Second

	// Generate 2-20 slow tests (all exceeding threshold)
	numTests := rng(19) + 2
	numPackages := rng(5) + 1
	packages := make([]string, numPackages)
	for i := 0; i < numPackages; i++ {
		packages[i] = "example.com/pkg" + string(rune('A'+i))
	}

	for i := 0; i < numTests; i++ {
		pkg := packages[rng(numPackages)]
		testName := "Test" + string(rune('A'+i%26))
		if i >= 26 {
			testName += string(rune('0' + i/26))
		}

		// Generate random duration that exceeds threshold
		// Range: threshold to threshold + 30s
		durationSeconds := float64(thresholdSeconds) + float64(rng(300))/10.0

		// Add run event
		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100) * time.Millisecond),
			Action:  "run",
			Package: pkg,
			Test:    testName,
		})

		// Add pass event with duration
		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100+50) * time.Millisecond),
			Action:  "pass",
			Package: pkg,
			Test:    testName,
			Elapsed: durationSeconds,
		})
	}

	return events, threshold
}

// TestPropertyTimeFormatConsistency is a property-based test that verifies time format consistency.
// **Feature: test-summary, Property 8: Time format consistency**
// **Validates: Requirements 5.3, 6.4, 9.5**
//
// Property: For any duration less than 60 seconds, the format SHALL be "X.Xs",
// and for any duration >= 60 seconds, the format SHALL be "HH:MM:SS.mmm".
func TestPropertyTimeFormatConsistency(t *testing.T) {
	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random durations across the full range
		durations := generateRandomDurations(i)

		for _, d := range durations {
			formatted := formatDuration(d)

			if d < 60*time.Second {
				// Should be in "X.Xs" format
				// Verify format matches pattern: digits, optional dot, optional digit, 's'
				// Examples: "0.0s", "5.2s", "59.9s"
				if !isSecondsFormat(formatted) {
					t.Errorf("Iteration %d: Duration %v (< 60s) formatted as '%s', expected seconds format (X.Xs)",
						i, d, formatted)
				}

				// Verify the numeric value is correct
				expectedSeconds := d.Seconds()
				if !verifySecondsFormat(formatted, expectedSeconds) {
					t.Errorf("Iteration %d: Duration %v formatted as '%s', expected %.1fs",
						i, d, formatted, expectedSeconds)
				}
			} else {
				// Should be in "HH:MM:SS.mmm" format
				// Verify format matches pattern: HH:MM:SS.mmm
				if !isHMSFormat(formatted) {
					t.Errorf("Iteration %d: Duration %v (>= 60s) formatted as '%s', expected HH:MM:SS.mmm format",
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

// isSecondsFormat checks if a string matches the "X.Xs" format.
func isSecondsFormat(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[len(s)-1] != 's' {
		return false
	}
	// Check that everything before 's' is a valid float
	numPart := s[:len(s)-1]
	if len(numPart) == 0 {
		return false
	}
	// Simple validation: should contain digits and at most one dot
	dotCount := 0
	for _, c := range numPart {
		if c == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
		} else if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// verifySecondsFormat checks if the formatted string matches the expected seconds value.
func verifySecondsFormat(formatted string, expectedSeconds float64) bool {
	// Parse the numeric part
	if len(formatted) < 2 || formatted[len(formatted)-1] != 's' {
		return false
	}
	numPart := formatted[:len(formatted)-1]

	// Parse as float
	var parsed float64
	_, err := fmt.Sscanf(numPart, "%f", &parsed)
	if err != nil {
		return false
	}

	// The format uses %.1f which rounds to 1 decimal place
	// We need to verify that the parsed value is within 0.06s of the expected value
	// (since rounding can change the value by up to 0.05s, plus floating point errors)
	diff := parsed - expectedSeconds
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.06
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

// TestComputeSummaryRequirements tests that ComputeSummary meets all specified requirements.
func TestComputeSummaryRequirements(t *testing.T) {
	collector := NewSummaryCollector()
	baseTime := time.Now()

	// Create a comprehensive test scenario
	// Package 1: 3 pass, 2 fail, 1 skip, 1 slow test
	pkg1Tests := []struct {
		name    string
		action  string
		elapsed float64
		output  string
	}{
		{"TestPass1", "pass", 2.0, ""},
		{"TestPass2", "pass", 3.0, ""},
		{"TestPass3", "pass", 15.0, ""}, // slow test
		{"TestFail1", "fail", 1.0, "assertion failed"},
		{"TestFail2", "fail", 2.0, "panic: runtime error"},
		{"TestSkip1", "skip", 0.5, "skipping test"},
	}

	offset := 0
	for _, test := range pkg1Tests {
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(offset) * time.Second),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    test.name,
		})
		if test.output != "" {
			collector.AddTestEvent(parser.TestEvent{
				Time:    baseTime.Add(time.Duration(offset) * time.Second),
				Action:  "output",
				Package: "github.com/test/pkg1",
				Test:    test.name,
				Output:  test.output + "\n",
			})
		}
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(offset+1) * time.Second),
			Action:  test.action,
			Package: "github.com/test/pkg1",
			Test:    test.name,
			Elapsed: test.elapsed,
		})
		offset += 2
	}
	collector.AddPackageResult("github.com/test/pkg1", "FAIL", 25*time.Second)

	// Package 2: 2 pass, faster package
	pkg2Tests := []struct {
		name    string
		elapsed float64
	}{
		{"TestQuick1", 0.5},
		{"TestQuick2", 0.5},
	}

	for _, test := range pkg2Tests {
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(offset) * time.Second),
			Action:  "run",
			Package: "github.com/test/pkg2",
			Test:    test.name,
		})
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(offset+1) * time.Second),
			Action:  "pass",
			Package: "github.com/test/pkg2",
			Test:    test.name,
			Elapsed: test.elapsed,
		})
		offset += 2
	}
	collector.AddPackageResult("github.com/test/pkg2", "ok", 1*time.Second)

	// Package 3: 1 pass, 1 slow test
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(time.Duration(offset) * time.Second),
		Action:  "run",
		Package: "github.com/test/pkg3",
		Test:    "TestSlowest",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    baseTime.Add(time.Duration(offset+1) * time.Second),
		Action:  "pass",
		Package: "github.com/test/pkg3",
		Test:    "TestSlowest",
		Elapsed: 30.0,
	})
	collector.AddPackageResult("github.com/test/pkg3", "ok", 30*time.Second)

	// Compute summary with 10s threshold
	summary := ComputeSummary(collector, 10*time.Second)

	// Requirement 3.1, 3.2: Failures collection and grouping
	t.Run("FailuresCollection", func(t *testing.T) {
		if len(summary.Failures) != 2 {
			t.Errorf("Expected 2 failures, got %d", len(summary.Failures))
		}
		// Verify failures are from the correct package
		for _, failure := range summary.Failures {
			if failure.Package != "github.com/test/pkg1" {
				t.Errorf("Expected failure from pkg1, got %s", failure.Package)
			}
			if failure.Status != "fail" {
				t.Errorf("Expected failure status 'fail', got %s", failure.Status)
			}
		}
	})

	// Requirement 4.1, 4.2: Skipped test collection and grouping
	t.Run("SkippedCollection", func(t *testing.T) {
		if len(summary.Skipped) != 1 {
			t.Errorf("Expected 1 skipped test, got %d", len(summary.Skipped))
		}
		if len(summary.Skipped) > 0 {
			if summary.Skipped[0].Package != "github.com/test/pkg1" {
				t.Errorf("Expected skipped test from pkg1, got %s", summary.Skipped[0].Package)
			}
			if summary.Skipped[0].Status != "skip" {
				t.Errorf("Expected skipped status 'skip', got %s", summary.Skipped[0].Status)
			}
		}
	})

	// Requirement 5.1, 5.2: Slow test detection and sorting
	t.Run("SlowTestDetectionAndSorting", func(t *testing.T) {
		if len(summary.SlowTests) != 2 {
			t.Errorf("Expected 2 slow tests (>10s), got %d", len(summary.SlowTests))
		}
		// Verify all slow tests exceed threshold
		for _, slowTest := range summary.SlowTests {
			if slowTest.Elapsed < 10*time.Second {
				t.Errorf("Slow test %s has duration %v, expected >= 10s", slowTest.Name, slowTest.Elapsed)
			}
		}
		// Verify sorting (descending order)
		if len(summary.SlowTests) >= 2 {
			if summary.SlowTests[0].Elapsed < summary.SlowTests[1].Elapsed {
				t.Errorf("Slow tests not sorted correctly: %v should be >= %v",
					summary.SlowTests[0].Elapsed, summary.SlowTests[1].Elapsed)
			}
			// First should be TestSlowest (30s), second should be TestPass3 (15s)
			if summary.SlowTests[0].Name != "TestSlowest" {
				t.Errorf("Expected slowest test to be TestSlowest, got %s", summary.SlowTests[0].Name)
			}
			if summary.SlowTests[1].Name != "TestPass3" {
				t.Errorf("Expected second slowest test to be TestPass3, got %s", summary.SlowTests[1].Name)
			}
		}
	})

	// Requirement 6.1: Fastest package
	t.Run("FastestPackage", func(t *testing.T) {
		if summary.FastestPackage == nil {
			t.Fatal("FastestPackage should not be nil")
		}
		if summary.FastestPackage.Name != "github.com/test/pkg2" {
			t.Errorf("Expected fastest package to be pkg2 (1s), got %s (%v)",
				summary.FastestPackage.Name, summary.FastestPackage.Elapsed)
		}
	})

	// Requirement 6.2: Slowest package
	t.Run("SlowestPackage", func(t *testing.T) {
		if summary.SlowestPackage == nil {
			t.Fatal("SlowestPackage should not be nil")
		}
		if summary.SlowestPackage.Name != "github.com/test/pkg3" {
			t.Errorf("Expected slowest package to be pkg3 (30s), got %s (%v)",
				summary.SlowestPackage.Name, summary.SlowestPackage.Elapsed)
		}
	})

	// Requirement 6.3: Package with most tests
	t.Run("MostTestsPackage", func(t *testing.T) {
		if summary.MostTestsPackage == nil {
			t.Fatal("MostTestsPackage should not be nil")
		}
		if summary.MostTestsPackage.Name != "github.com/test/pkg1" {
			t.Errorf("Expected package with most tests to be pkg1 (6 tests), got %s",
				summary.MostTestsPackage.Name)
		}
		totalTests := summary.MostTestsPackage.PassedTests +
			summary.MostTestsPackage.FailedTests +
			summary.MostTestsPackage.SkippedTests
		if totalTests != 6 {
			t.Errorf("Expected pkg1 to have 6 tests, got %d", totalTests)
		}
	})

	// Overall statistics verification
	t.Run("OverallStatistics", func(t *testing.T) {
		expectedTotal := 9 // 6 from pkg1 + 2 from pkg2 + 1 from pkg3
		if summary.TotalTests != expectedTotal {
			t.Errorf("Expected %d total tests, got %d", expectedTotal, summary.TotalTests)
		}
		expectedPassed := 6 // 3 from pkg1 + 2 from pkg2 + 1 from pkg3
		if summary.PassedTests != expectedPassed {
			t.Errorf("Expected %d passed tests, got %d", expectedPassed, summary.PassedTests)
		}
		if summary.FailedTests != 2 {
			t.Errorf("Expected 2 failed tests, got %d", summary.FailedTests)
		}
		if summary.SkippedTests != 1 {
			t.Errorf("Expected 1 skipped test, got %d", summary.SkippedTests)
		}
		if summary.PackageCount != 3 {
			t.Errorf("Expected 3 packages, got %d", summary.PackageCount)
		}
	})
}
