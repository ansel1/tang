package format

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
)

// TestFormatBuildErrors tests the ERRORS section formatting
func TestFormatBuildErrors(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	run := results.NewRun(1)
	run.BuildEvents = []parser.BuildEvent{
		{
			ImportPath: "github.com/user/project/broken",
			Action:     "build-output",
			Output:     "# github.com/user/project/broken\n",
		},
		{
			ImportPath: "github.com/user/project/broken",
			Action:     "build-output",
			Output:     "broken/file.go:10:5: syntax error: unexpected name\n",
		},
		{
			ImportPath: "github.com/user/project/broken",
			Action:     "build-output",
			Output:     "broken/file.go:15:10: undefined: someFunc\n",
		},
		{
			ImportPath: "github.com/user/project/broken",
			Action:     "build-fail",
		},
	}

	pkg1 := &results.PackageResult{
		Name:        "github.com/user/project/broken",
		Status:      results.StatusFailed,
		FailedBuild: "github.com/user/project/broken",
		Elapsed:     0,
	}

	pkg2 := &results.PackageResult{
		Name:    "github.com/user/project/working",
		Status:  results.StatusFailed,
		Elapsed: 2 * time.Second,
	}
	pkg2.Counts.Failed = 1

	run.Packages["github.com/user/project/broken"] = pkg1
	run.Packages["github.com/user/project/working"] = pkg2
	run.PackageOrder = []string{"github.com/user/project/broken", "github.com/user/project/working"}

	summary := &Summary{
		Packages:      []*results.PackageResult{pkg1, pkg2},
		BuildFailures: []*results.PackageResult{pkg1},
		Run:           run,
		TotalTests:    1,
		FailedTests:   1,
		TotalTime:     2 * time.Second,
		PackageCount:  2,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "syntax error: unexpected name") {
		t.Error("Expected build error message")
	}
	if !strings.Contains(output, "undefined: someFunc") {
		t.Error("Expected second build error message")
	}
	if !strings.Contains(output, "github.com/user/project/broken") {
		t.Error("Expected broken package name")
	}
	if !strings.Contains(output, "FAIL") {
		t.Error("Expected FAIL status for broken package")
	}
}

// TestComputeSummaryWithBuildFailures tests that ComputeSummary correctly identifies build failures
func TestComputeSummaryWithBuildFailures(t *testing.T) {
	run := results.NewRun(1)
	run.FirstEventTime = time.Now()
	run.LastEventTime = run.FirstEventTime.Add(2 * time.Second)

	// Add build events for the broken package
	run.BuildEvents = []parser.BuildEvent{
		{
			ImportPath: "github.com/test/broken",
			Action:     "build-output",
			Output:     "# github.com/test/broken\n",
		},
		{
			ImportPath: "github.com/test/broken",
			Action:     "build-output",
			Output:     "error: compilation failed\n",
		},
		{
			ImportPath: "github.com/test/broken",
			Action:     "build-fail",
		},
	}

	// Package with build failure
	pkg1 := &results.PackageResult{
		Name:        "github.com/test/broken",
		Status:      results.StatusFailed,
		FailedBuild: "github.com/test/broken",
		Elapsed:     0,
	}

	// Package with test failure (not build failure)
	pkg2 := &results.PackageResult{
		Name:    "github.com/test/working",
		Status:  results.StatusFailed,
		Elapsed: 1 * time.Second,
	}
	pkg2.Counts.Failed = 1

	// Package that passed
	pkg3 := &results.PackageResult{
		Name:    "github.com/test/passing",
		Status:  results.StatusPassed,
		Elapsed: 500 * time.Millisecond,
	}
	pkg3.Counts.Passed = 3

	run.Packages["github.com/test/broken"] = pkg1
	run.Packages["github.com/test/working"] = pkg2
	run.Packages["github.com/test/passing"] = pkg3
	run.PackageOrder = []string{"github.com/test/broken", "github.com/test/working", "github.com/test/passing"}

	summary := ComputeSummary(run, 10*time.Second)

	// Verify Run field is set
	if summary.Run != run {
		t.Error("Expected summary.Run to be set to the run")
	}

	// Verify only pkg1 is in BuildFailures
	if len(summary.BuildFailures) != 1 {
		t.Errorf("Expected 1 build failure, got %d", len(summary.BuildFailures))
	}
	if len(summary.BuildFailures) > 0 && summary.BuildFailures[0].Name != "github.com/test/broken" {
		t.Errorf("Expected build failure for github.com/test/broken, got %s", summary.BuildFailures[0].Name)
	}

	// Verify build errors can be retrieved
	if summary.Run != nil {
		buildErrors := summary.Run.GetBuildErrors("github.com/test/broken")
		if len(buildErrors) != 3 {
			t.Errorf("Expected 3 build events, got %d", len(buildErrors))
		}
	}
}
