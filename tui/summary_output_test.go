package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
)

func TestPackageOutputCapture(t *testing.T) {
	collector := NewSummaryCollector()
	baseTime := time.Now()

	// Simulate a package run with output
	pkgName := "github.com/example/pkg"

	// 1. Start package (implicit in events usually, but we can just send output)

	// 2. Send some package output
	outputLines := []string{
		"some output line 1",
		"some output line 2",
		"ok  \tgithub.com/example/pkg\t0.123s",
	}

	for i, line := range outputLines {
		collector.AddTestEvent(parser.TestEvent{
			Time:    baseTime.Add(time.Duration(i) * time.Millisecond),
			Action:  "output",
			Package: pkgName,
			Test:    "", // Package-level output
			Output:  line + "\n",
		})
	}

	// 3. Finish package
	collector.AddPackageResult(pkgName, "ok", 123*time.Millisecond)

	// 4. Get summary
	summary := ComputeSummary(collector, 10*time.Second)

	// 5. Check if output is captured
	if len(summary.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(summary.Packages))
	}

	pkgResult := summary.Packages[0]
	expectedOutput := "ok  \tgithub.com/example/pkg\t0.123s"

	// We expect the LAST line of output to be captured
	if pkgResult.Output != expectedOutput {
		t.Errorf("Expected package output %q, got %q", expectedOutput, pkgResult.Output)
	}
}

func TestSummaryFormatting(t *testing.T) {
	// Create a summary with some packages
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "pkg1",
				Status:       "ok",
				Elapsed:      1234 * time.Millisecond,
				PassedTests:  10,
				FailedTests:  0,
				SkippedTests: 0,
				Output:       "ok\tpkg1\t1.234s",
			},
			{
				Name:         "pkg2",
				Status:       "?",
				Elapsed:      0,
				PassedTests:  0,
				FailedTests:  0,
				SkippedTests: 0,
				Output:       "?\tpkg2\t[no test files]",
			},
		},
		TotalTests:   10,
		PassedTests:  10,
		TotalTime:    2 * time.Second,
		PackageCount: 2,
	}

	formatter := NewSummaryFormatter(80)
	output := formatter.Format(summary)

	// Check for pkg1 line
	// Expected: ✓ ok      pkg1    1.234s  ✓ 10  ✗ 0  ∅ 0  00:00:01.234
	// Note: tabs in output are expanded with width 8.
	// "ok\tpkg1\t1.234s" -> "ok      pkg1    1.234s"
	// We verify the presence of key parts and order.

	if !containsSequence(output, "✓", "ok      pkg1    1.234s", "✓ 10", "00:00:01.234") {
		t.Errorf("Output missing expected pkg1 format. Got:\n%s", output)
	}

	// Check for pkg2 line (no counts)
	// Expected: ∅ ?    pkg2    [no test files]       00:00:00.000
	// The counts column should be empty (spaces).
	// "✓ 0" should NOT be present for this line.

	// We can check that the line for pkg2 does not contain "✓ 0".
	lines := splitLines(output)
	var pkg2Line string
	for _, line := range lines {
		if strings.Contains(line, "pkg2") {
			pkg2Line = line
			break
		}
	}

	if pkg2Line == "" {
		t.Fatal("Could not find line for pkg2")
	}

	if strings.Contains(pkg2Line, "✓ 0") {
		t.Errorf("pkg2 line should not contain counts, got: %q", pkg2Line)
	}

	if !strings.Contains(pkg2Line, "00:00:00.000") {
		t.Errorf("pkg2 line missing full elapsed time, got: %q", pkg2Line)
	}
}

func containsSequence(s string, parts ...string) bool {
	lastIndex := -1
	for _, part := range parts {
		idx := strings.Index(s, part)
		if idx == -1 || idx < lastIndex {
			return false
		}
		lastIndex = idx
	}
	return true
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
