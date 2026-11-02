package format

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ansel1/tang/results"
)

func TestSummaryFormatterBasic(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 10

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "ok") {
		t.Error("Expected 'ok' status for passed package")
	}
	if !strings.Contains(output, "github.com/user/project/pkg1") {
		t.Error("Expected package name")
	}
	if !strings.Contains(output, "(1 packages)") {
		t.Error("Expected totals line with package count")
	}
	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol")
	}
}

func TestSummaryFormatterWithFailures(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusFailed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Failed = 2

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}

	failTest := results.NewTestResult("github.com/user/project/pkg1", "TestFailing")
	failTest.Latest().Status = results.StatusFailed
	failTest.Latest().Elapsed = 1 * time.Second
	failTest.Latest().Output = []string{"Error: expected true, got false", "at line 42"}
	run.TestResults["github.com/user/project/pkg1/TestFailing"] = failTest
	pkg1.TestOrder = []string{"TestFailing"}

	// Create TestExecutionEntry for the failure
	failEntry := &TestExecutionEntry{
		TestResult:      failTest,
		TestExecution:   failTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  2,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Failures:     []*TestExecutionEntry{failEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "FAIL") {
		t.Error("Expected FAIL label in output")
	}
	if !strings.Contains(output, "TestFailing") {
		t.Error("Expected test name in failures")
	}
	if !strings.Contains(output, "Error: expected true, got false") {
		t.Error("Expected failure output")
	}
}

func TestSummaryFormatterWithSkipped(t *testing.T) {
	formatter := NewSummaryFormatter(80, false, SummaryOptions{IncludeSkipped: true})

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Skipped = 2

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}

	skipTest := results.NewTestResult("github.com/user/project/pkg1", "TestSkipped")
	skipTest.Latest().Status = results.StatusSkipped
	skipTest.Latest().Elapsed = 0
	skipTest.Latest().Output = []string{"Skipping: not implemented yet"}
	run.TestResults["github.com/user/project/pkg1/TestSkipped"] = skipTest
	pkg1.TestOrder = []string{"TestSkipped"}

	// Create TestExecutionEntry for the skipped test
	skipEntry := &TestExecutionEntry{
		TestResult:      skipTest,
		TestExecution:   skipTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  0,
		SkippedTests: 2,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Skipped:      []*TestExecutionEntry{skipEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "SKIP") {
		t.Error("Expected SKIP label")
	}
	if !strings.Contains(output, "TestSkipped") {
		t.Error("Expected test name in skipped")
	}
	if !strings.Contains(output, "Skipping: not implemented yet") {
		t.Error("Expected skip reason")
	}
}

func TestSummaryFormatterWithSlowTests(t *testing.T) {
	formatter := NewSummaryFormatter(80, false, SummaryOptions{IncludeSlow: true})

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 65 * time.Second,
	}
	pkg1.Counts.Passed = 2

	slowTest := results.NewTestResult("github.com/user/project/pkg1", "TestSlow")
	slowTest.Latest().Status = results.StatusPassed
	slowTest.Latest().Elapsed = 65 * time.Second

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}
	run.TestResults["github.com/user/project/pkg1/TestSlow"] = slowTest
	pkg1.TestOrder = []string{"TestSlow"}

	// Create TestExecutionEntry for the slow test
	slowEntry := &TestExecutionEntry{
		TestResult:      slowTest,
		TestExecution:   slowTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   2,
		PassedTests:  2,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    65 * time.Second,
		PackageCount: 1,
		SlowTests:    []*TestExecutionEntry{slowEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "SLOW") {
		t.Error("Expected SLOW label")
	}
	if !strings.Contains(output, "TestSlow") {
		t.Error("Expected test name in slow tests")
	}
}

func TestSummaryFormatterSkippedHiddenByDefault(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Skipped = 2

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}

	skipTest := results.NewTestResult("github.com/user/project/pkg1", "TestSkipped")
	skipTest.Latest().Status = results.StatusSkipped
	skipTest.Latest().Elapsed = 0
	skipTest.Latest().Output = []string{"Skipping: not implemented yet"}
	run.TestResults["github.com/user/project/pkg1/TestSkipped"] = skipTest
	pkg1.TestOrder = []string{"TestSkipped"}

	// Create TestExecutionEntry for the skipped test
	skipEntry := &TestExecutionEntry{
		TestResult:      skipTest,
		TestExecution:   skipTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  0,
		SkippedTests: 2,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Skipped:      []*TestExecutionEntry{skipEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if strings.Contains(output, "--- SKIP") {
		t.Error("Skipped test details should be hidden by default")
	}
	if strings.Contains(output, "TestSkipped") {
		t.Error("Skipped test name should be hidden by default")
	}
}

func TestSummaryFormatterSlowHiddenByDefault(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 65 * time.Second,
	}
	pkg1.Counts.Passed = 2

	slowTest := results.NewTestResult("github.com/user/project/pkg1", "TestSlow")
	slowTest.Latest().Status = results.StatusPassed
	slowTest.Latest().Elapsed = 65 * time.Second

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}
	run.TestResults["github.com/user/project/pkg1/TestSlow"] = slowTest
	pkg1.TestOrder = []string{"TestSlow"}

	// Create TestExecutionEntry for the slow test
	slowEntry := &TestExecutionEntry{
		TestResult:      slowTest,
		TestExecution:   slowTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   2,
		PassedTests:  2,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    65 * time.Second,
		PackageCount: 1,
		SlowTests:    []*TestExecutionEntry{slowEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if strings.Contains(output, "SLOW") {
		t.Error("Slow test details should be hidden by default")
	}
}

func TestSummaryFormatterNoFailuresOrSkips(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 10

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	if strings.Contains(output, "--- FAIL") {
		t.Error("FAIL details should be omitted when no failures")
	}
	if strings.Contains(output, "--- SKIP") {
		t.Error("SKIP details should be omitted when no skips")
	}
}

func TestSummaryFormatterTotalsLine(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 3 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Failed = 1
	pkg1.Counts.Skipped = 1

	pkg2 := &results.PackageResult{
		Name:    "github.com/user/project/pkg2",
		Status:  results.StatusPassed,
		Elapsed: 2 * time.Second,
	}
	pkg2.Counts.Passed = 5

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1, pkg2},
		TotalTests:   15,
		PassedTests:  13,
		FailedTests:  1,
		SkippedTests: 1,
		TotalTime:    5 * time.Second,
		PackageCount: 2,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "(2 packages)") {
		t.Error("Expected totals line with package count")
	}
	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol in totals")
	}
	if !strings.Contains(output, SymbolFail) {
		t.Error("Expected fail symbol in totals")
	}
	if !strings.Contains(output, SymbolSkip) {
		t.Error("Expected skip symbol in totals")
	}
}

func TestSummaryFormatterSymbols(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "pkg1",
		Status:  results.StatusPassed,
		Elapsed: 1 * time.Second,
	}
	pkg1.Counts.Passed = 1

	pkg2 := &results.PackageResult{
		Name:    "pkg2",
		Status:  results.StatusFailed,
		Elapsed: 1 * time.Second,
	}
	pkg2.Counts.Failed = 1

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1, pkg2},
		TotalTests:   2,
		PassedTests:  1,
		FailedTests:  1,
		SkippedTests: 0,
		TotalTime:    2 * time.Second,
		PackageCount: 2,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol")
	}
	if !strings.Contains(output, SymbolFail) {
		t.Error("Expected fail symbol")
	}
}

func TestSummaryFormatterNoTestsOmitCounters(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusSkipped,
		Elapsed: 0,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   0,
		PassedTests:  0,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    0,
		PackageCount: 1,
	}

	output := formatter.Format(summary)
	lines := strings.Split(output, "\n")

	var packageLine string
	for _, line := range lines {
		if strings.Contains(line, "github.com/user/project/pkg1") {
			packageLine = line
			break
		}
	}

	if packageLine == "" {
		t.Fatal("Could not find package summary line in output")
	}

	if strings.Contains(packageLine, "(") || strings.Contains(packageLine, ")") {
		t.Errorf("Expected summary package line to omit counters (no parentheses), but got: %q", packageLine)
	}
}

func TestSummaryFormatterHeadersAndAlignment(t *testing.T) {
	formatter := NewSummaryFormatter(120, true)

	pkg1 := &results.PackageResult{
		Name:        "gitlab.protectv.local/ncryptify/stemcell",
		Status:      results.StatusFailed,
		Elapsed:     453 * time.Millisecond,
		SummaryLine: "FAIL\tgitlab.protectv.local/ncryptify/stemcell\t0.453s\n",
	}
	pkg1.Counts.Passed = 124
	pkg1.Counts.Failed = 3
	pkg1.Counts.Skipped = 12

	pkg2 := &results.PackageResult{
		Name:        "gitlab.protectv.local/ncryptify/stemcell/cmd/stemcell",
		Status:      results.StatusSkipped,
		Elapsed:     0,
		SummaryLine: "?\tgitlab.protectv.local/ncryptify/stemcell/cmd/stemcell\t[no test files]\n",
	}

	run := results.NewRun(1)
	run.Packages[pkg1.Name] = pkg1
	run.Packages[pkg2.Name] = pkg2
	run.PackageOrder = []string{pkg1.Name, pkg2.Name}

	failTest := results.NewTestResult(pkg1.Name, "TestSomething")
	failTest.Latest().Status = results.StatusFailed
	failTest.Latest().Elapsed = 120 * time.Millisecond
	failTest.Latest().Output = []string{"something_test.go:42: boom"}
	run.TestResults[pkg1.Name+"/TestSomething"] = failTest
	pkg1.TestOrder = []string{"TestSomething"}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1, pkg2},
		TotalTests:   139,
		PassedTests:  124,
		FailedTests:  3,
		SkippedTests: 12,
		TotalTime:    453358 * time.Microsecond,
		PackageCount: 2,
		Failures: []*TestExecutionEntry{
			{
				TestResult:      failTest,
				TestExecution:   failTest.Latest(),
				Iteration:       1,
				TotalExecutions: 1,
			},
		},
		Run: run,
	}

	output := formatter.Format(summary)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	var messagesHeader, summaryHeader, pkg1Line, bottomSep, totalsLine string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "─── Messages "):
			messagesHeader = line
		case strings.HasPrefix(line, "─── Summary "):
			summaryHeader = line
		case strings.HasPrefix(line, "FAIL"):
			pkg1Line = line
		case strings.HasPrefix(line, "────"):
			bottomSep = line
		case strings.HasPrefix(line, "(2 packages)"):
			totalsLine = line
		}
	}

	if messagesHeader == "" {
		t.Fatalf("expected Messages header in output:\n%s", output)
	}
	if summaryHeader == "" {
		t.Fatalf("expected Summary header in output:\n%s", output)
	}
	if bottomSep == "" {
		t.Fatalf("expected bottom separator line in output:\n%s", output)
	}

	// Verify table-width rule sizing across Messages header, Summary header, bottom separator, and data rows.
	tableWidth := utf8.RuneCountInString(totalsLine)
	for name, line := range map[string]string{
		"messagesHeader": messagesHeader,
		"summaryHeader":  summaryHeader,
		"pkg1Line":       pkg1Line,
		"bottomSep":      bottomSep,
	} {
		if got := utf8.RuneCountInString(line); got != tableWidth {
			t.Errorf("%s rune width = %d, want %d (table width)\nline: %q", name, got, tableWidth, line)
		}
	}

	// Verify column header symbols align with the rightmost character of each column in both summaryHeader and bottomSep.
	runeIndex := func(s, sub string) int {
		byteIdx := strings.Index(s, sub)
		if byteIdx < 0 {
			return -1
		}
		return utf8.RuneCountInString(s[:byteIdx])
	}
	headerRunes := []rune(summaryHeader)
	bottomRunes := []rune(bottomSep)
	totalsRunes := []rune(totalsLine)
	for sym, expectedUnitChar := range map[string]rune{
		SymbolPass:  '4', // 124
		SymbolFail:  '3', // 3
		SymbolSkip:  '2', // 12
		SymbolTotal: '9', // 139
		SymbolTime:  's', // 453.358ms
	} {
		col := runeIndex(summaryHeader, sym)
		if col < 0 {
			t.Fatalf("missing symbol %s in summaryHeader %q", sym, summaryHeader)
		}
		if headerRunes[col] != []rune(sym)[0] {
			t.Fatalf("unexpected rune at col %d", col)
		}
		if bottomRunes[col] != []rune(sym)[0] {
			t.Errorf("bottomSep missing %s at col %d: got %q\nheader: %s\nbottom: %s",
				sym, col, string(bottomRunes[col]), summaryHeader, bottomSep)
		}
		if totalsRunes[col] != expectedUnitChar {
			t.Errorf("under %s (col %d): got %q in totals line, want %q\nheader: %s\ntotals: %s",
				sym, col, string(totalsRunes[col]), string(expectedUnitChar), summaryHeader, totalsLine)
		}
	}
}

func TestSummaryFormatterColors(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkgPass := &results.PackageResult{
		Name:        "github.com/user/project/pkgpass",
		Status:      results.StatusPassed,
		Elapsed:     100 * time.Millisecond,
		SummaryLine: "ok\tgithub.com/user/project/pkgpass\t0.100s\n",
	}
	pkgPass.Counts.Passed = 7

	pkgFail := &results.PackageResult{
		Name:        "github.com/user/project/pkgfail",
		Status:      results.StatusFailed,
		Elapsed:     200 * time.Millisecond,
		SummaryLine: "FAIL\tgithub.com/user/project/pkgfail\t0.200s\n",
	}
	pkgFail.Counts.Passed = 2
	pkgFail.Counts.Failed = 1

	run := results.NewRun(1)
	run.Packages[pkgPass.Name] = pkgPass
	run.Packages[pkgFail.Name] = pkgFail
	run.PackageOrder = []string{pkgPass.Name, pkgFail.Name}

	failTest := results.NewTestResult(pkgFail.Name, "TestBoom")
	failTest.Latest().Status = results.StatusFailed
	failTest.Latest().Elapsed = 50 * time.Millisecond
	failTest.Latest().Output = []string{"boom"}
	run.TestResults[pkgFail.Name+"/TestBoom"] = failTest
	pkgFail.TestOrder = []string{"TestBoom"}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkgPass, pkgFail},
		TotalTests:   10,
		PassedTests:  9,
		FailedTests:  1,
		SkippedTests: 0,
		TotalTime:    200 * time.Millisecond,
		PackageCount: 2,
		Failures: []*TestExecutionEntry{
			{
				TestResult:      failTest,
				TestExecution:   failTest.Latest(),
				Iteration:       1,
				TotalExecutions: 1,
			},
		},
		Run: run,
	}

	output := formatter.Format(summary)

	// Column headers should be colored and horizontal segments dimmed.
	if !strings.Contains(output, formatter.dimStyle.Render("───")+" Messages ") {
		t.Errorf("expected dimmed ─── before Messages in output:\n%s", output)
	}
	if !strings.Contains(output, formatter.dimStyle.Render("───")+" Summary ") {
		t.Errorf("expected dimmed ─── before Summary in output:\n%s", output)
	}
	if !strings.Contains(output, formatter.passStyle.Render(SymbolPass)) {
		t.Errorf("expected green pass symbol in header")
	}
	if !strings.Contains(output, formatter.failStyle.Render(SymbolFail)) {
		t.Errorf("expected red fail symbol in header")
	}
	if !strings.Contains(output, formatter.skipStyle.Render(SymbolSkip)) {
		t.Errorf("expected yellow skip symbol in header")
	}

	// Passing package status "ok" should be bold green, and passed counts green.
	if !strings.Contains(output, formatter.boldPass.Render("ok  ")) {
		t.Errorf("expected bold green 'ok' status prefix")
	}
	if !strings.Contains(output, formatter.passStyle.Render("7")) {
		t.Errorf("expected green passed count '7'")
	}
	if !strings.Contains(output, formatter.passStyle.Render("9")) {
		t.Errorf("expected green total passed count '9'")
	}
}

func TestSummaryFormatterHeadersCappedAtTerminalWidth(t *testing.T) {
	const termWidth = 80
	formatter := NewSummaryFormatter(termWidth, true)

	shortPkg := &results.PackageResult{
		Name:        "github.com/user/project/short",
		Status:      results.StatusFailed,
		Elapsed:     453 * time.Millisecond,
		SummaryLine: "FAIL\tgithub.com/user/project/short\t0.453s\n",
	}
	shortPkg.Counts.Passed = 10
	shortPkg.Counts.Failed = 1

	longPkg := &results.PackageResult{
		Name:        "github.com/user/project/very/deeply/nested/subpackage/with/a/long/import/path",
		Status:      results.StatusSkipped,
		Elapsed:     0,
		SummaryLine: "?\tgithub.com/user/project/very/deeply/nested/subpackage/with/a/long/import/path\t[no test files]\n",
	}

	run := results.NewRun(1)
	run.Packages[shortPkg.Name] = shortPkg
	run.Packages[longPkg.Name] = longPkg
	run.PackageOrder = []string{shortPkg.Name, longPkg.Name}

	failTest := results.NewTestResult(shortPkg.Name, "TestSomething")
	failTest.Latest().Status = results.StatusFailed
	failTest.Latest().Elapsed = 120 * time.Millisecond
	failTest.Latest().Output = []string{"boom"}
	run.TestResults[shortPkg.Name+"/TestSomething"] = failTest
	shortPkg.TestOrder = []string{"TestSomething"}

	summary := &Summary{
		Packages:     []*results.PackageResult{shortPkg, longPkg},
		TotalTests:   11,
		PassedTests:  10,
		FailedTests:  1,
		SkippedTests: 0,
		TotalTime:    453 * time.Millisecond,
		PackageCount: 2,
		Failures: []*TestExecutionEntry{
			{
				TestResult:      failTest,
				TestExecution:   failTest.Latest(),
				Iteration:       1,
				TotalExecutions: 1,
			},
		},
		Run: run,
	}

	output := formatter.Format(summary)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	var messagesHeader, summaryHeader, shortPkgLine, skippedLine, bottomSep, totalsLine string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "─── Messages "):
			messagesHeader = line
		case strings.HasPrefix(line, "─── Summary "):
			summaryHeader = line
		case strings.HasPrefix(line, "FAIL"):
			shortPkgLine = line
		case strings.HasPrefix(line, "?"):
			skippedLine = line
		case strings.HasPrefix(line, "────"):
			bottomSep = line
		case strings.HasPrefix(line, "(2 packages)"):
			totalsLine = line
		}
	}

	for name, line := range map[string]string{
		"messagesHeader": messagesHeader,
		"summaryHeader":  summaryHeader,
		"shortPkgLine":   shortPkgLine,
		"bottomSep":      bottomSep,
		"totalsLine":     totalsLine,
	} {
		if got := utf8.RuneCountInString(line); got != termWidth {
			t.Errorf("%s rune width = %d, want %d (termWidth)\nline: %q", name, got, termWidth, line)
		}
	}

	// Redundant duration token from SummaryLine ("0.453s") should be stripped from the label,
	// leaving only the formatted rightmost duration column ("453ms").
	if strings.Contains(shortPkgLine, "0.453s") {
		t.Errorf("expected redundant SummaryLine duration '0.453s' to be stripped from shortPkgLine: %q", shortPkgLine)
	}

	// Long skipped package should be truncated with an ellipsis on a single line.
	if !strings.Contains(skippedLine, "…") {
		t.Errorf("expected long skipped package line to be truncated with '…', got %q", skippedLine)
	}
	if got := utf8.RuneCountInString(skippedLine); got > termWidth {
		t.Errorf("skippedLine rune width = %d, want <= %d: %q", got, termWidth, skippedLine)
	}
}

func TestSummaryFormatterLongPackageTruncatedAndCappedAt150(t *testing.T) {
	// Even on a wide 200-column terminal, summary lines and Messages header are capped at 150 chars.
	formatter := NewSummaryFormatter(200, true)

	shortPkg := &results.PackageResult{
		Name:        "pkg/short",
		Status:      results.StatusFailed,
		Elapsed:     123 * time.Millisecond,
		SummaryLine: "FAIL\tpkg/short\t0.123s\tcoverage: 87.5% of statements\n",
	}
	shortPkg.Counts.Passed = 4
	shortPkg.Counts.Failed = 1

	longPkg := &results.PackageResult{
		Name:        "github.com/user/project/very/deeply/nested/subpackage/with/an/extremely/long/import/path/that/exceeds/one/hundred/and/fifty/columns/in/the/summary/table",
		Status:      results.StatusPassed,
		Elapsed:     450 * time.Millisecond,
		SummaryLine: "ok\tgithub.com/user/project/very/deeply/nested/subpackage/with/an/extremely/long/import/path/that/exceeds/one/hundred/and/fifty/columns/in/the/summary/table\t0.450s\n",
	}
	longPkg.Counts.Passed = 12
	longPkg.Counts.Skipped = 1

	run := results.NewRun(1)
	run.Packages[shortPkg.Name] = shortPkg
	run.Packages[longPkg.Name] = longPkg
	run.PackageOrder = []string{shortPkg.Name, longPkg.Name}

	failTest := results.NewTestResult(shortPkg.Name, "TestSomething")
	failTest.Latest().Status = results.StatusFailed
	failTest.Latest().Elapsed = 50 * time.Millisecond
	failTest.Latest().Output = []string{"boom"}
	run.TestResults[shortPkg.Name+"/TestSomething"] = failTest
	shortPkg.TestOrder = []string{"TestSomething"}

	summary := &Summary{
		Packages:     []*results.PackageResult{shortPkg, longPkg},
		TotalTests:   18,
		PassedTests:  16,
		FailedTests:  1,
		SkippedTests: 1,
		TotalTime:    573 * time.Millisecond,
		PackageCount: 2,
		Failures: []*TestExecutionEntry{
			{
				TestResult:      failTest,
				TestExecution:   failTest.Latest(),
				Iteration:       1,
				TotalExecutions: 1,
			},
		},
		Run: run,
	}

	output := formatter.Format(summary)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	var messagesHeader, summaryHeader, shortPkgLine, longPkgLine, bottomSep, totalsLine string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "─── Messages "):
			messagesHeader = line
		case strings.HasPrefix(line, "─── Summary "):
			summaryHeader = line
		case strings.HasPrefix(line, "FAIL"):
			shortPkgLine = line
		case strings.HasPrefix(line, "ok"):
			longPkgLine = line
		case strings.HasPrefix(line, "────"):
			bottomSep = line
		case strings.HasPrefix(line, "(2 packages)"):
			totalsLine = line
		}
	}

	// Coverage annotation should be preserved while 0.123s is stripped.
	if !strings.Contains(shortPkgLine, "coverage: 87.5% of statements") {
		t.Errorf("expected coverage annotation to be preserved in shortPkgLine: %q", shortPkgLine)
	}
	if strings.Contains(shortPkgLine, "0.123s") {
		t.Errorf("expected redundant duration '0.123s' to be stripped in shortPkgLine: %q", shortPkgLine)
	}

	// Long package should be truncated on a single line with "…".
	if !strings.Contains(longPkgLine, "…") {
		t.Errorf("expected longPkgLine to be truncated with '…', got %q", longPkgLine)
	}

	// Messages header and all Summary lines should be capped at 150 chars.
	for name, line := range map[string]string{
		"messagesHeader": messagesHeader,
		"summaryHeader":  summaryHeader,
		"shortPkgLine":   shortPkgLine,
		"longPkgLine":    longPkgLine,
		"bottomSep":      bottomSep,
		"totalsLine":     totalsLine,
	} {
		if got := utf8.RuneCountInString(line); got != 150 {
			t.Errorf("%s rune width = %d, want 150\nline: %q", name, got, line)
		}
	}
}
