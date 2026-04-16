package junit

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
)

func TestWriteXML(t *testing.T) {
	// Setup a sample state
	state := results.NewState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	// Create a package result
	pkgName := "github.com/ansel1/tang/example"
	pkg := &results.PackageResult{
		Name:      pkgName,
		Status:    results.StatusFailed,
		StartTime: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Elapsed:   1500 * time.Millisecond,
		TestOrder: []string{"TestPass", "TestFail", "TestSkip"},
	}
	pkg.Counts.Passed = 1
	pkg.Counts.Failed = 1
	pkg.Counts.Skipped = 1
	run.Packages[pkgName] = pkg
	run.PackageOrder = append(run.PackageOrder, pkgName)

	// Add test results
	run.TestResults[pkgName+"/TestPass"] = &results.TestResult{
		Name:      "TestPass",
		Package:   pkgName,
		Status:    results.StatusPassed,
		Elapsed:   100 * time.Millisecond,
		StartTime: pkg.StartTime,
	}

	run.TestResults[pkgName+"/TestFail"] = &results.TestResult{
		Name:      "TestFail",
		Package:   pkgName,
		Status:    results.StatusFailed,
		Elapsed:   200 * time.Millisecond,
		StartTime: pkg.StartTime,
		Output:    []string{"assertion failed", "expected true got false"},
	}

	run.TestResults[pkgName+"/TestSkip"] = &results.TestResult{
		Name:      "TestSkip",
		Package:   pkgName,
		Status:    results.StatusSkipped,
		Elapsed:   0,
		StartTime: pkg.StartTime,
	}

	// Capture output
	var buf bytes.Buffer
	err := WriteXML(&buf, state)
	if err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	output := buf.String()

	// Validate Output (Basic checks)
	if !strings.Contains(output, `<testsuites`) {
		t.Error("Output missing root <testsuites>")
	}
	if !strings.Contains(output, `tests="3"`) {
		t.Error("Incorrect total test count")
	}
	if !strings.Contains(output, `failures="1"`) {
		t.Error("Incorrect failure count")
	}
	if !strings.Contains(output, `errors="0"`) {
		t.Error("Error count should be 0 when there are no build failures")
	}
	// Total time should be approx 1.5s
	if !strings.Contains(output, `time="1.500"`) {
		t.Error("Incorrect total time")
	}

	if !strings.Contains(output, `name="github.com/ansel1/tang/example"`) {
		t.Error("Output missing package name")
	}
	if !strings.Contains(output, `tests="3"`) {
		t.Error("Incorrect total test count")
	}
	if !strings.Contains(output, `failures="1"`) {
		t.Error("Incorrect failure count")
	}
	if !strings.Contains(output, `skipped="1"`) {
		t.Error("Incorrect skipped count")
	}
	if !strings.Contains(output, `name="TestPass"`) {
		t.Error("Missing TestPass")
	}
	if !strings.Contains(output, `name="TestFail"`) {
		t.Error("Missing TestFail")
	}
	if !strings.Contains(output, `name="TestSkip"`) {
		t.Error("Missing TestSkip")
	}
	if !strings.Contains(output, `<failure message="Failed">`) {
		t.Error("Missing failure message")
	}
	if !strings.Contains(output, `assertion failed`) {
		t.Error("Missing failure content")
	}
	if !strings.Contains(output, `<skipped message="Skipped"></skipped>`) && !strings.Contains(output, `<skipped message="Skipped"/>`) {
		t.Error("Missing skipped message")
	}
	if !strings.Contains(output, `value="1"`) {
		t.Error("Missing run_id property")
	}

	// XML Validation
	var val JUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}
}

func TestWriteXML_BuildFailure(t *testing.T) {
	state := results.NewState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)

	// Add build events
	run.BuildEvents = []parser.BuildEvent{
		{
			ImportPath: "github.com/example/pkg",
			Action:     "build-output",
			Output:     "# github.com/example/pkg\n",
		},
		{
			ImportPath: "github.com/example/pkg",
			Action:     "build-output",
			Output:     "pkg/file.go:10:5: syntax error\n",
		},
		{
			ImportPath: "github.com/example/pkg",
			Action:     "build-fail",
		},
	}

	// Create a package with build failure reference
	pkgName := "github.com/example/pkg"
	pkg := &results.PackageResult{
		Name:        pkgName,
		Status:      results.StatusFailed,
		StartTime:   time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Elapsed:     0,
		TestOrder:   []string{}, // No tests ran
		FailedBuild: "github.com/example/pkg",
		SummaryLine: "FAIL\tgithub.com/example/pkg [build failed]",
	}
	run.Packages[pkgName] = pkg
	run.PackageOrder = append(run.PackageOrder, pkgName)

	var buf bytes.Buffer
	err := WriteXML(&buf, state)
	if err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	output := buf.String()

	// Verify TestMain pseudo-test is present
	if !strings.Contains(output, `name="TestMain"`) {
		t.Error("Missing TestMain pseudo-test for build failure")
	}
	// Verify build error output is in error content
	if !strings.Contains(output, "syntax error") {
		t.Error("Missing build error in error content")
	}
	// Verify error element is used instead of failure
	if !strings.Contains(output, `<error`) {
		t.Error("Build failure should use <error> element")
	}
	if strings.Contains(output, `<failure`) {
		t.Error("Build failure should not use <failure> element")
	}
	// Verify error count is 1 (one package with build error)
	if !strings.Contains(output, `errors="1"`) {
		t.Error("Error count should be 1 for build failure")
	}
	// Verify test count is 0
	if !strings.Contains(output, `tests="0"`) {
		t.Error("Test count should be 0 for build failure")
	}
	// Verify failure count is 0 (build failures don't count as test failures)
	if !strings.Contains(output, `failures="0"`) {
		t.Error("Failure count should be 0 for build failure (no tests ran)")
	}

	// XML Validation
	var val JUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Verify structure
	if len(val.TestSuites) != 1 {
		t.Errorf("Expected 1 testsuite, got %d", len(val.TestSuites))
	}
	if len(val.TestSuites[0].TestCases) != 1 {
		t.Errorf("Expected 1 testcase, got %d", len(val.TestSuites[0].TestCases))
	}
	tc := val.TestSuites[0].TestCases[0]
	if tc.Name != "TestMain" {
		t.Errorf("Expected testcase name 'TestMain', got '%s'", tc.Name)
	}
	if tc.Error == nil {
		t.Error("Expected error element in TestMain testcase")
	} else {
		if tc.Error.Message != "Build failed" {
			t.Errorf("Expected error message 'Build failed', got '%s'", tc.Error.Message)
		}
		if tc.Error.Type != "BuildError" {
			t.Errorf("Expected error type 'BuildError', got '%s'", tc.Error.Type)
		}
	}
	if tc.Failure != nil {
		t.Error("Expected no failure element in TestMain testcase (should use error instead)")
	}
	if val.Errors != 1 {
		t.Errorf("Expected errors count of 1, got %d", val.Errors)
	}
}
