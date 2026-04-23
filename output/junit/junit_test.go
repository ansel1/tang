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
	tr1 := results.NewTestResult(pkgName, "TestPass")
	tr1.Latest().Status = results.StatusPassed
	tr1.Latest().Elapsed = 100 * time.Millisecond
	tr1.Latest().StartTime = pkg.StartTime
	run.TestResults[pkgName+"/TestPass"] = tr1

	tr2 := results.NewTestResult(pkgName, "TestFail")
	tr2.Latest().Status = results.StatusFailed
	tr2.Latest().Elapsed = 200 * time.Millisecond
	tr2.Latest().StartTime = pkg.StartTime
	tr2.Latest().Output = []string{"assertion failed", "expected true got false"}
	run.TestResults[pkgName+"/TestFail"] = tr2

	tr3 := results.NewTestResult(pkgName, "TestSkip")
	tr3.Latest().Status = results.StatusSkipped
	tr3.Latest().Elapsed = 0
	tr3.Latest().StartTime = pkg.StartTime
	run.TestResults[pkgName+"/TestSkip"] = tr3

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

func TestWriteXML_MultiExecution(t *testing.T) {
	// Test -count=2 scenario where first execution fails, second passes
	state := results.NewState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	pkgName := "github.com/example/pkg"
	pkg := &results.PackageResult{
		Name:      pkgName,
		Status:    results.StatusPassed, // Package passes because second execution passed
		StartTime: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Elapsed:   500 * time.Millisecond,
		TestOrder: []string{"TestFoo"},
	}
	pkg.Counts.Passed = 1
	pkg.Counts.Failed = 1 // First execution failed
	run.Packages[pkgName] = pkg
	run.PackageOrder = append(run.PackageOrder, pkgName)

	// Create test result with 2 executions
	tr := results.NewTestResult(pkgName, "TestFoo")
	// First execution - failed
	tr.Executions[0].Status = results.StatusFailed
	tr.Executions[0].Elapsed = 100 * time.Millisecond
	tr.Executions[0].Output = []string{"FAIL: assertion failed", "expected true got false"}
	tr.Executions[0].SummaryLine = "--- FAIL: TestFoo (0.10s)"
	// Second execution - passed
	tr.AppendExecution()
	tr.Latest().Status = results.StatusPassed
	tr.Latest().Elapsed = 150 * time.Millisecond
	tr.Latest().SummaryLine = "--- PASS: TestFoo (0.15s)"

	run.TestResults[pkgName+"/TestFoo"] = tr

	var buf bytes.Buffer
	err := WriteXML(&buf, state)
	if err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	output := buf.String()

	// Should have 2 testcases (one per execution)
	if !strings.Contains(output, `tests="2"`) {
		t.Error("Expected 2 tests in output")
	}
	// First execution should have #01 suffix (per design decision)
	if !strings.Contains(output, `name="TestFoo#01"`) {
		t.Error("Expected first execution to have #01 suffix")
	}
	// Second execution should have #02 suffix
	if !strings.Contains(output, `name="TestFoo#02"`) {
		t.Error("Expected second execution to have #02 suffix")
	}
	// First execution should have failure
	if !strings.Contains(output, `name="TestFoo#01"`) || !strings.Contains(strings.Split(output, "TestFoo#01")[1], "<failure") {
		t.Error("Expected first execution to have failure element")
	}
	// Second execution should not have failure
	if !strings.Contains(output, `name="TestFoo#02"`) || strings.Contains(strings.Split(output, "TestFoo#02")[1], "<failure>") {
		t.Error("Expected second execution to not have failure element")
	}

	// XML Validation
	var val JUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Verify we have 2 testcases
	if len(val.TestSuites) != 1 {
		t.Errorf("Expected 1 testsuite, got %d", len(val.TestSuites))
	}
	if len(val.TestSuites[0].TestCases) != 2 {
		t.Errorf("Expected 2 testcases, got %d", len(val.TestSuites[0].TestCases))
	}

	// Verify first testcase has failure
	if val.TestSuites[0].TestCases[0].Name != "TestFoo#01" {
		t.Errorf("Expected first testcase name 'TestFoo#01', got '%s'", val.TestSuites[0].TestCases[0].Name)
	}
	if val.TestSuites[0].TestCases[0].Failure == nil {
		t.Error("Expected first testcase to have failure")
	}

	// Verify second testcase passes
	if val.TestSuites[0].TestCases[1].Name != "TestFoo#02" {
		t.Errorf("Expected second testcase name 'TestFoo#02', got '%s'", val.TestSuites[0].TestCases[1].Name)
	}
	if val.TestSuites[0].TestCases[1].Failure != nil {
		t.Error("Expected second testcase to not have failure")
	}
}

func TestWriteXML_MultiExecutionBothFail(t *testing.T) {
	// Test -count=2 scenario where both executions fail
	state := results.NewState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	pkgName := "github.com/example/pkg"
	pkg := &results.PackageResult{
		Name:      pkgName,
		Status:    results.StatusFailed,
		StartTime: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Elapsed:   500 * time.Millisecond,
		TestOrder: []string{"TestFoo"},
	}
	pkg.Counts.Failed = 2 // Both executions failed
	run.Packages[pkgName] = pkg
	run.PackageOrder = append(run.PackageOrder, pkgName)

	// Create test result with 2 failed executions
	tr := results.NewTestResult(pkgName, "TestFoo")
	// First execution - failed
	tr.Executions[0].Status = results.StatusFailed
	tr.Executions[0].Elapsed = 100 * time.Millisecond
	tr.Executions[0].Output = []string{"FAIL: first failure"}
	tr.Executions[0].SummaryLine = "--- FAIL: TestFoo (0.10s)"
	// Second execution - also failed
	tr.AppendExecution()
	tr.Latest().Status = results.StatusFailed
	tr.Latest().Elapsed = 200 * time.Millisecond
	tr.Latest().Output = []string{"FAIL: second failure"}
	tr.Latest().SummaryLine = "--- FAIL: TestFoo (0.20s)"

	run.TestResults[pkgName+"/TestFoo"] = tr

	var buf bytes.Buffer
	err := WriteXML(&buf, state)
	if err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	output := buf.String()

	// Should have 2 testcases
	if !strings.Contains(output, `tests="2"`) {
		t.Error("Expected 2 tests in output")
	}
	if !strings.Contains(output, `failures="2"`) {
		t.Error("Expected 2 failures in output")
	}

	// XML Validation
	var val JUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Verify we have 2 testcases with failures
	if len(val.TestSuites[0].TestCases) != 2 {
		t.Errorf("Expected 2 testcases, got %d", len(val.TestSuites[0].TestCases))
	}
	if val.TestSuites[0].TestCases[0].Failure == nil {
		t.Error("Expected first testcase to have failure")
	}
	if val.TestSuites[0].TestCases[1].Failure == nil {
		t.Error("Expected second testcase to have failure")
	}
	// Verify both failures have different content
	if !strings.Contains(val.TestSuites[0].TestCases[0].Failure.Content, "first failure") {
		t.Error("Expected first failure to contain 'first failure'")
	}
	if !strings.Contains(val.TestSuites[0].TestCases[1].Failure.Content, "second failure") {
		t.Error("Expected second failure to contain 'second failure'")
	}
}

func TestWriteXML_MultiExecutionSubtest(t *testing.T) {
	// Test multi-execution subtest naming
	state := results.NewState()
	run := results.NewRun(1)
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	pkgName := "github.com/example/pkg"
	pkg := &results.PackageResult{
		Name:      pkgName,
		Status:    results.StatusFailed,
		StartTime: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Elapsed:   500 * time.Millisecond,
		TestOrder: []string{"TestFoo/sub"}, // Use full subtest name in TestOrder
	}
	pkg.Counts.Failed = 2
	run.Packages[pkgName] = pkg
	run.PackageOrder = append(run.PackageOrder, pkgName)

	// Create subtest with 2 executions: TestFoo/sub
	tr := results.NewTestResult(pkgName, "TestFoo/sub")
	// First execution - failed
	tr.Executions[0].Status = results.StatusFailed
	tr.Executions[0].Elapsed = 100 * time.Millisecond
	tr.Executions[0].Output = []string{"FAIL: subtest failed"}
	// Second execution - passed
	tr.AppendExecution()
	tr.Latest().Status = results.StatusPassed
	tr.Latest().Elapsed = 150 * time.Millisecond

	run.TestResults[pkgName+"/TestFoo/sub"] = tr

	var buf bytes.Buffer
	err := WriteXML(&buf, state)
	if err != nil {
		t.Fatalf("WriteXML failed: %v", err)
	}

	output := buf.String()

	// Subtest should use TestFoo#02/sub format (suffix on parent, not on subtest name)
	if !strings.Contains(output, `name="TestFoo#01/sub"`) {
		t.Error("Expected first subtest to have #01 suffix on parent")
	}
	if !strings.Contains(output, `name="TestFoo#02/sub"`) {
		t.Error("Expected second subtest to have #02 suffix on parent")
	}

	// XML Validation
	var val JUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &val); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Verify testcase names
	if val.TestSuites[0].TestCases[0].Name != "TestFoo#01/sub" {
		t.Errorf("Expected first testcase name 'TestFoo#01/sub', got '%s'", val.TestSuites[0].TestCases[0].Name)
	}
	if val.TestSuites[0].TestCases[1].Name != "TestFoo#02/sub" {
		t.Errorf("Expected second testcase name 'TestFoo#02/sub', got '%s'", val.TestSuites[0].TestCases[1].Name)
	}
}
