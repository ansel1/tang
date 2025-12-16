package junit

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ansel1/tang/results"
)

// JUnitTestSuites represents the root <testsuites> element
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       string           `xml:"time,attr"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a <testsuite> element (one per package per run)
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Name       string          `xml:"name,attr"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Skipped    int             `xml:"skipped,attr"`
	Time       string          `xml:"time,attr"`
	Timestamp  string          `xml:"timestamp,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase `xml:"testcase"`
}

// JUnitProperty represents a property in <properties>
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitTestCase represents a <testcase> element (one per test)
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
}

// JUnitFailure represents a <failure> element
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Content string `xml:",chardata"`
}

// JUnitError represents an <error> element
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Content string `xml:",chardata"`
}

// JUnitSkipped represents a <skipped> element
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// WriteXML writes the current results state to the writer in JUnit XML format
func WriteXML(w io.Writer, state *results.State) error {
	suites := JUnitTestSuites{
		TestSuites: make([]JUnitTestSuite, 0),
	}

	var totalTime float64
	var errorCount int // Count of packages with build errors

	for _, run := range state.Runs {
		// We want to output suites in a deterministic order.
		// The Runs are already ordered by ID.
		// Within a Run, we should follow PackageOrder.

		for _, pkgName := range run.PackageOrder {
			pkgResult := run.Packages[pkgName]
			if pkgResult == nil {
				continue
			}

			suite := JUnitTestSuite{
				Name:      pkgResult.Name,
				Tests:     pkgResult.Counts.Passed + pkgResult.Counts.Failed + pkgResult.Counts.Skipped,
				Failures:  pkgResult.Counts.Failed,
				Skipped:   pkgResult.Counts.Skipped,
				Time:      fmt.Sprintf("%.3f", pkgResult.Elapsed.Seconds()),
				Timestamp: pkgResult.StartTime.Format(time.RFC3339),
				Properties: []JUnitProperty{
					{Name: "run_id", Value: fmt.Sprintf("%d", run.ID)},
				},
				TestCases: make([]JUnitTestCase, 0),
			}

			suites.Tests += suite.Tests
			suites.Failures += suite.Failures
			totalTime += pkgResult.Elapsed.Seconds()

			// Check if this package had a build failure - create TestMain pseudo-test
			if pkgResult.FailedBuild != "" {
				// Increment error count for this package
				errorCount++

				// Get build errors for this package
				buildErrors := run.GetBuildErrors(pkgResult.FailedBuild)

				// Combine build error output
				var buildOutput strings.Builder
				for _, be := range buildErrors {
					if be.Output != "" {
						buildOutput.WriteString(be.Output)
					}
				}

				// Create TestMain pseudo-test with error element
				buildFailureCase := JUnitTestCase{
					Name:      "TestMain",
					ClassName: "", // Empty classname like gotestsum
					Time:      "0.000",
					Error: &JUnitError{
						Message: "Build failed",
						Type:    "BuildError",
						Content: buildOutput.String(),
					},
				}
				suite.TestCases = append(suite.TestCases, buildFailureCase)
				// Note: Don't increment suite.Tests (gotestsum keeps it at actual test count)
				// Failures are already counted in lines 83, 94
			}

			// Add tests in order
			for _, testName := range pkgResult.TestOrder {
				// The test name in pkgResult.TestOrder is usually the full name "TestName",
				// but TestResults map keys are "pkgname/TestName" or just "TestName"?
				// Let's check how TestResults are keyed in results/model.go or collector.go.
				// Looking at results/model.go: TestResults map[string]*TestResult // "package/testname" -> TestResult

				lookupKey := pkgName + "/" + testName
				testResult, ok := run.TestResults[lookupKey]

				// Fallback: sometimes the key might just be the testName if there's ambiguity in my understanding,
				// but based on `collector.go` usually it keys by full path.
				// However, `PackageResult.TestOrder` likely stores just the test name relative to package.
				// Let's verify `results` logic later if needed, but `pkgName/testName` is the standard pattern in this codebase.

				if !ok {
					// Should not happen if data integrity is maintained
					continue
				}

				testCase := JUnitTestCase{
					Name:      testResult.Name,
					ClassName: pkgResult.Name,
					Time:      fmt.Sprintf("%.3f", testResult.Elapsed.Seconds()),
				}

				switch testResult.Status {
				case results.StatusFailed:
					// Join output lines for the failure message
					content := ""
					if len(testResult.Output) > 0 {
						// Simple join, or maybe pick the last line as message?
						// Usually the output contains the failure details.
						content = fmt.Sprintf("%v", testResult.Output)
					}
					testCase.Failure = &JUnitFailure{
						Message: "Failed",
						Content: content,
					}
				case results.StatusSkipped:
					testCase.Skipped = &JUnitSkipped{
						Message: "Skipped",
					}
				}

				suite.TestCases = append(suite.TestCases, testCase)
			}

			suites.TestSuites = append(suites.TestSuites, suite)
		}
	}

	suites.Time = fmt.Sprintf("%.3f", totalTime)
	suites.Errors = errorCount

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return err
	}
	// Flush and add newline
	if err := enc.Flush(); err != nil {
		return err
	}
	_, err := w.Write([]byte("\n"))
	return err
}
