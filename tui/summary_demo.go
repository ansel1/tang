//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"time"

	"github.com/ansel1/tang/tui"
)

func main() {
	formatter := tui.NewSummaryFormatter(80)

	// Create a comprehensive summary
	summary := &tui.Summary{
		Packages: []*tui.PackageResult{
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
		Failures: []*tui.TestResult{
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
		Skipped: []*tui.TestResult{
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
		SlowTests: []*tui.TestResult{
			{
				Package: "github.com/user/project/pkg/models",
				Name:    "TestDatabaseMigration",
				Status:  "pass",
				Elapsed: 65 * time.Second,
			},
		},
		FastestPackage: &tui.PackageResult{
			Name:    "github.com/user/project/pkg/utils",
			Elapsed: 2500 * time.Millisecond,
		},
		SlowestPackage: &tui.PackageResult{
			Name:    "github.com/user/project/pkg/models",
			Elapsed: 65 * time.Second,
		},
		MostTestsPackage: &tui.PackageResult{
			Name:         "github.com/user/project/pkg/models",
			PassedTests:  20,
			FailedTests:  0,
			SkippedTests: 0,
		},
	}

	output := formatter.Format(summary)
	fmt.Println(output)
}
