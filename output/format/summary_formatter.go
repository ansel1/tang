package format

import (
	"fmt"
	"strings"

	"os"

	"charm.land/lipgloss/v2"
	"github.com/ansel1/tang/results"
	"github.com/mattn/go-isatty"
)

// SummaryFormatter formats a Summary for display, with test details grouped by
// package, go-test-style package summary, then totals.
type SummaryFormatter struct {
	width     int
	useColors bool

	failStyle    lipgloss.Style
	passStyle    lipgloss.Style
	skipStyle    lipgloss.Style
	slowStyle    lipgloss.Style
	boldFail     lipgloss.Style
	boldSkip     lipgloss.Style
	boldSlow     lipgloss.Style
	boldPass     lipgloss.Style
	dimStyle     lipgloss.Style
	boldWhite    lipgloss.Style
	neutralStyle lipgloss.Style
}

func NewSummaryFormatter(width int) *SummaryFormatter {
	if width <= 0 {
		width = 80
	}
	useColors := isatty.IsTerminal(os.Stdout.Fd())

	return &SummaryFormatter{
		width:        width,
		useColors:    useColors,
		failStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		passStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		skipStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		slowStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
		boldFail:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		boldSkip:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
		boldSlow:     lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true),
		boldPass:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		dimStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		boldWhite:    lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true),
		neutralStyle: lipgloss.NewStyle(),
	}
}

func (f *SummaryFormatter) Format(summary *Summary) string {
	var sb strings.Builder
	f.formatTestDetails(&sb, summary)
	f.formatPackageSummary(&sb, summary)
	f.formatTotalSummary(&sb, summary)
	return sb.String()
}

type packageIssue struct {
	kind     string // "fail", "skip", "slow", "build"
	test     *results.TestResult
	buildPkg *results.PackageResult
}

func (f *SummaryFormatter) formatTestDetails(sb *strings.Builder, summary *Summary) {
	type pkgData struct {
		issues []packageIssue
	}
	pkgMap := make(map[string]*pkgData)
	pkgOrder := make([]string, 0)

	ensurePkg := func(name string) *pkgData {
		if _, ok := pkgMap[name]; !ok {
			pkgMap[name] = &pkgData{}
			pkgOrder = append(pkgOrder, name)
		}
		return pkgMap[name]
	}

	for _, pkg := range summary.BuildFailures {
		pd := ensurePkg(pkg.Name)
		pd.issues = append(pd.issues, packageIssue{kind: "build", buildPkg: pkg})
	}

	slowSet := make(map[string]bool, len(summary.SlowTests))
	for _, slow := range summary.SlowTests {
		slowSet[slow.Package+"/"+slow.Name] = true
	}

	classifyTest := func(pkg *results.PackageResult, testName string) (string, *results.TestResult) {
		testKey := pkg.Name + "/" + testName
		tr, ok := summary.Run.TestResults[testKey]
		if !ok {
			return "", nil
		}
		switch tr.Status {
		case results.StatusFailed:
			return "fail", tr
		case results.StatusSkipped:
			return "skip", tr
		default:
			if slowSet[testKey] {
				return "slow", tr
			}
		}
		return "", nil
	}

	// Group subtests under their parent so they render nested in the output.
	if summary.Run != nil {
		for _, pkg := range summary.Packages {
			subtestsByParent := make(map[string][]string)
			topLevel := make([]string, 0)
			seen := make(map[string]bool)

			for _, testName := range pkg.TestOrder {
				if isSubtest(testName) {
					parent := testName[:strings.Index(testName, "/")]
					subtestsByParent[parent] = append(subtestsByParent[parent], testName)
				} else {
					if !seen[testName] {
						topLevel = append(topLevel, testName)
						seen[testName] = true
					}
				}
			}

			for _, parentName := range topLevel {
				parentKind, parentTR := classifyTest(pkg, parentName)

				var subtestIssues []packageIssue
				for _, subName := range subtestsByParent[parentName] {
					kind, tr := classifyTest(pkg, subName)
					if kind != "" {
						subtestIssues = append(subtestIssues, packageIssue{kind: kind, test: tr})
					}
				}

				if parentKind == "" && len(subtestIssues) == 0 {
					continue
				}

				pd := ensurePkg(pkg.Name)

				if parentKind != "" {
					pd.issues = append(pd.issues, packageIssue{kind: parentKind, test: parentTR})
				}

				pd.issues = append(pd.issues, subtestIssues...)
			}
		}
	}

	if len(pkgOrder) == 0 {
		return
	}

	for _, pkgName := range pkgOrder {
		pd := pkgMap[pkgName]

		sb.WriteString("=== ")
		sb.WriteString(pkgName)
		sb.WriteString("\n")

		for _, issue := range pd.issues {
			switch issue.kind {
			case "build":
				f.formatBuildIssue(sb, issue.buildPkg, summary)
			case "fail":
				f.formatTestIssue(sb, issue.test, "FAIL", f.boldFail, f.failStyle)
			case "skip":
				f.formatTestIssue(sb, issue.test, "SKIP", f.boldSkip, f.skipStyle)
			case "slow":
				f.formatSlowTestIssue(sb, issue.test)
			}
		}

		sb.WriteString("\n")
	}
}

func isSubtest(name string) bool {
	return strings.Contains(name, "/")
}

func (f *SummaryFormatter) formatTestIssue(sb *strings.Builder, tr *results.TestResult, label string, boldStyle, colorStyle lipgloss.Style) {
	indent := IndentLevel2             // 4 spaces for top-level tests
	logIndent := indent + IndentLevel2 // 8 spaces for log output
	if isSubtest(tr.Name) {
		indent = IndentLevel2 + IndentLevel2 // 8 spaces for subtests
		logIndent = indent + IndentLevel2    // 12 spaces for subtest log output
	}

	elapsed := fmt.Sprintf("(%.2fs)", tr.Elapsed.Seconds())

	sb.WriteString(indent)
	sb.WriteString("--- ")
	sb.WriteString(boldStyle.Render(label))
	sb.WriteString(": ")
	sb.WriteString(colorStyle.Render(tr.Name))
	sb.WriteString(" ")
	sb.WriteString(f.dimStyle.Render(elapsed))
	sb.WriteString("\n")

	for _, line := range tr.Output {
		sb.WriteString(logIndent)
		sb.WriteString(ensureReset(strings.TrimLeft(line, " \t")))
		sb.WriteString("\n")
	}
}

func (f *SummaryFormatter) formatSlowTestIssue(sb *strings.Builder, tr *results.TestResult) {
	indent := IndentLevel2
	if isSubtest(tr.Name) {
		indent = IndentLevel2 + IndentLevel2
	}

	elapsed := fmt.Sprintf("(%.2fs)", tr.Elapsed.Seconds())

	sb.WriteString(indent)
	sb.WriteString("--- ")
	sb.WriteString(f.boldSlow.Render("SLOW"))
	sb.WriteString(": ")
	sb.WriteString(f.slowStyle.Render(tr.Name))
	sb.WriteString(" ")
	sb.WriteString(f.boldWhite.Render(elapsed))
	sb.WriteString("\n")
}

func (f *SummaryFormatter) formatBuildIssue(sb *strings.Builder, pkg *results.PackageResult, summary *Summary) {
	if summary.Run == nil || pkg.FailedBuild == "" {
		return
	}

	buildErrors := summary.Run.GetBuildErrors(pkg.FailedBuild)
	for _, be := range buildErrors {
		if be.Action == "build-output" && be.Output != "" {
			lines := strings.Split(strings.TrimRight(be.Output, "\n"), "\n")
			for _, line := range lines {
				if line != "" {
					sb.WriteString(IndentLevel2)
					sb.WriteString(f.failStyle.Render(ensureReset(line)))
					sb.WriteString("\n")
				}
			}
		}
	}
}

func (f *SummaryFormatter) formatPackageSummary(sb *strings.Builder, summary *Summary) {
	if len(summary.Packages) == 0 {
		return
	}

	type pkgLine struct {
		statusWord string
		name       string
		extra      string
		pkg        *results.PackageResult
	}

	lines := make([]pkgLine, 0, len(summary.Packages))

	maxStatusLen := 0
	maxNameExtraLen := 0
	maxPassedLen := 0
	maxFailedLen := 0
	maxSkippedLen := 0
	maxTotalLen := 0

	for _, pkg := range summary.Packages {
		pl := pkgLine{pkg: pkg}

		switch {
		case pkg.FailedBuild != "":
			pl.statusWord = "FAIL"
		case pkg.Status == results.StatusFailed:
			pl.statusWord = "FAIL"
		case pkg.Status == results.StatusSkipped:
			pl.statusWord = "?"
		default:
			pl.statusWord = "ok"
		}

		pl.name = pkg.Name
		if pkg.FailedBuild != "" {
			pl.extra = "[build failed]"
		} else if pkg.Output != "" {
			output := expandTabs(pkg.Output, 8)
			nameIdx := strings.Index(output, pkg.Name)
			if nameIdx >= 0 {
				rest := strings.TrimSpace(output[nameIdx+len(pkg.Name):])
				if rest != "" {
					pl.extra = rest
				}
			}
		}

		passedStr := fmt.Sprintf("%d", pkg.Counts.Passed)
		failedStr := fmt.Sprintf("%d", pkg.Counts.Failed)
		skippedStr := fmt.Sprintf("%d", pkg.Counts.Skipped)
		if len(passedStr) > maxPassedLen {
			maxPassedLen = len(passedStr)
		}
		if len(failedStr) > maxFailedLen {
			maxFailedLen = len(failedStr)
		}
		if len(skippedStr) > maxSkippedLen {
			maxSkippedLen = len(skippedStr)
		}
		totalStr := fmt.Sprintf("%d", pkg.Counts.Passed+pkg.Counts.Failed+pkg.Counts.Skipped)
		if len(totalStr) > maxTotalLen {
			maxTotalLen = len(totalStr)
		}

		if len(pl.statusWord) > maxStatusLen {
			maxStatusLen = len(pl.statusWord)
		}

		nameExtra := pl.name
		if pl.extra != "" {
			nameExtra += " " + pl.extra
		}
		if len(nameExtra) > maxNameExtraLen {
			maxNameExtraLen = len(nameExtra)
		}

		lines = append(lines, pl)
	}

	separatorLen := 94
	if f.width > separatorLen {
		separatorLen = f.width
	}
	sb.WriteString(strings.Repeat("-", separatorLen))
	sb.WriteString("\n")

	for _, pl := range lines {
		var statusStr string
		switch pl.statusWord {
		case "FAIL":
			statusStr = f.boldFail.Render(fmt.Sprintf("%-*s", maxStatusLen, pl.statusWord))
		case "ok":
			statusStr = f.boldPass.Render(fmt.Sprintf("%-*s", maxStatusLen, pl.statusWord))
		case "?":
			statusStr = f.boldSkip.Render(fmt.Sprintf("%-*s", maxStatusLen, pl.statusWord))
		}

		nameExtra := pl.name
		if pl.extra != "" {
			nameExtra += " " + pl.extra
		}

		var coloredNameExtra string
		switch pl.statusWord {
		case "FAIL":
			coloredNameExtra = f.failStyle.Render(fmt.Sprintf("%-*s", maxNameExtraLen, nameExtra))
		case "ok":
			coloredNameExtra = f.passStyle.Render(fmt.Sprintf("%-*s", maxNameExtraLen, nameExtra))
		case "?":
			coloredNameExtra = f.skipStyle.Render(fmt.Sprintf("%-*s", maxNameExtraLen, nameExtra))
		}

		hasCounts := pl.pkg.Counts.Passed > 0 || pl.pkg.Counts.Failed > 0 || pl.pkg.Counts.Skipped > 0
		countsStr := ""
		if hasCounts {
			passedStr := fmt.Sprintf("%s%*d", SymbolPass, maxPassedLen, pl.pkg.Counts.Passed)
			if pl.pkg.Counts.Passed > 0 {
				passedStr = f.passStyle.Render(passedStr)
			} else {
				passedStr = f.neutralStyle.Render(passedStr)
			}

			failedStr := fmt.Sprintf("%s%*d", SymbolFail, maxFailedLen, pl.pkg.Counts.Failed)
			if pl.pkg.Counts.Failed > 0 {
				failedStr = f.failStyle.Render(failedStr)
			} else {
				failedStr = f.neutralStyle.Render(failedStr)
			}

			skippedStr := fmt.Sprintf("%s%*d", SymbolSkip, maxSkippedLen, pl.pkg.Counts.Skipped)
			if pl.pkg.Counts.Skipped > 0 {
				skippedStr = f.skipStyle.Render(skippedStr)
			} else {
				skippedStr = f.neutralStyle.Render(skippedStr)
			}

			total := pl.pkg.Counts.Passed + pl.pkg.Counts.Failed + pl.pkg.Counts.Skipped
			totalStr := f.neutralStyle.Render(fmt.Sprintf("=%*d", maxTotalLen, total))

			countsStr = fmt.Sprintf("%s %s %s %s", passedStr, failedStr, skippedStr, totalStr)
		}

		elapsed := formatDuration(pl.pkg.Elapsed)

		if countsStr != "" {
			fmt.Fprintf(sb, "%s    %s  %s  %s\n",
				statusStr, coloredNameExtra, countsStr, elapsed)
		} else {
			countsWidth := 4 + 3 + maxPassedLen + maxFailedLen + maxSkippedLen + maxTotalLen
			emptyCounts := strings.Repeat(" ", countsWidth)
			fmt.Fprintf(sb, "%s    %s  %s  %s\n",
				statusStr, coloredNameExtra, emptyCounts, elapsed)
		}
	}

	sb.WriteString("\n")
}

func (f *SummaryFormatter) formatTotalSummary(sb *strings.Builder, summary *Summary) {
	passPercent := 0.0
	failPercent := 0.0
	skipPercent := 0.0
	if summary.TotalTests > 0 {
		passPercent = float64(summary.PassedTests) / float64(summary.TotalTests) * 100
		failPercent = float64(summary.FailedTests) / float64(summary.TotalTests) * 100
		skipPercent = float64(summary.SkippedTests) / float64(summary.TotalTests) * 100
	}

	passIcon := SymbolPass
	failIcon := SymbolFail
	skipIcon := SymbolSkip
	if f.useColors {
		passIcon = f.passStyle.Render(SymbolPass)
		failIcon = f.failStyle.Render(SymbolFail)
		skipIcon = f.skipStyle.Render(SymbolSkip)
	}

	fmt.Fprintf(sb, "Total tests:    %d\n", summary.TotalTests)
	fmt.Fprintf(sb, "Passed:         %d %s (%.1f%%)\n", summary.PassedTests, passIcon, passPercent)
	fmt.Fprintf(sb, "Failed:         %d %s (%.1f%%)\n", summary.FailedTests, failIcon, failPercent)
	fmt.Fprintf(sb, "Skipped:        %d %s (%.1f%%)\n", summary.SkippedTests, skipIcon, skipPercent)
	fmt.Fprintf(sb, "Total time:     %s\n", formatDuration(summary.TotalTime))
	fmt.Fprintf(sb, "Packages:       %d\n", summary.PackageCount)
}
