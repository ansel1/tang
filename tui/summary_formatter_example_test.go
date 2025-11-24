package tui

import (
	"fmt"
	"time"
)

// ExampleSummaryFormatter demonstrates the formatter output
func ExampleSummaryFormatter() {
	formatter := NewSummaryFormatter(80)

	// Create a comprehensive summary
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg/utils",
				Status:       "ok",
				Elapsed:      2500 * time.Millisecond,
				PassedTests:  15,
				FailedTests:  0,
				SkippedTests: 1,
			},
			{
				Name:         "github.com/user/project/pkg/handlers",
				Status:       "FAIL",
				Elapsed:      3200 * time.Millisecond,
				PassedTests:  8,
				FailedTests:  2,
				SkippedTests: 0,
			},
			{
				Name:         "github.com/user/project/pkg/models",
				Status:       "ok",
				Elapsed:      65 * time.Second,
				PassedTests:  20,
				FailedTests:  0,
				SkippedTests: 0,
			},
		},
		TotalTests:   46,
		PassedTests:  43,
		FailedTests:  2,
		SkippedTests: 1,
		TotalTime:    70700 * time.Millisecond,
		PackageCount: 3,
		Failures: []*TestResult{
			{
				Package: "github.com/user/project/pkg/handlers",
				Name:    "TestHandleRequest",
				Status:  "fail",
				Elapsed: 1200 * time.Millisecond,
				Output: []string{
					"Error: expected status 200, got 500",
					"at handlers_test.go:42",
				},
			},
			{
				Package: "github.com/user/project/pkg/handlers",
				Name:    "TestValidation",
				Status:  "fail",
				Elapsed: 800 * time.Millisecond,
				Output: []string{
					"Error: validation failed",
					"expected error, got nil",
				},
			},
		},
		Skipped: []*TestResult{
			{
				Package: "github.com/user/project/pkg/utils",
				Name:    "TestExperimentalFeature",
				Status:  "skip",
				Elapsed: 0,
				Output: []string{
					"Skipping: feature not yet implemented",
				},
			},
		},
		SlowTests: []*TestResult{
			{
				Package: "github.com/user/project/pkg/models",
				Name:    "TestDatabaseMigration",
				Status:  "pass",
				Elapsed: 65 * time.Second,
			},
		},
		FastestPackage: &PackageResult{
			Name:    "github.com/user/project/pkg/utils",
			Elapsed: 2500 * time.Millisecond,
		},
		SlowestPackage: &PackageResult{
			Name:    "github.com/user/project/pkg/models",
			Elapsed: 65 * time.Second,
		},
		MostTestsPackage: &PackageResult{
			Name:         "github.com/user/project/pkg/models",
			PassedTests:  20,
			FailedTests:  0,
			SkippedTests: 0,
		},
	}

	output := formatter.Format(summary)
	fmt.Println(output)
}
