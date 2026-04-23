package format

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/ansel1/tang/results"
)

// SummaryFormatter formats a Summary for display, with test details grouped by
// package, go-test-style package summary, then totals.
type SummaryFormatter struct {
	width   int
	noColor bool
	options SummaryOptions

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

func NewSummaryFormatter(width int, noColor bool, opts ...SummaryOptions) *SummaryFormatter {
	if width <= 0 {
		width = 80
	}

	var options SummaryOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	neutral := lipgloss.NewStyle()

	f := &SummaryFormatter{
		width:        width,
		noColor:      noColor,
		options:      options,
		neutralStyle: neutral,
	}

	if noColor {
		f.failStyle = neutral
		f.passStyle = neutral
		f.skipStyle = neutral
		f.slowStyle = neutral
		f.boldFail = neutral
		f.boldSkip = neutral
		f.boldSlow = neutral
		f.boldPass = neutral
		f.dimStyle = neutral
		f.boldWhite = neutral
	} else {
		f.failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		f.passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		f.skipStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		f.slowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
		f.boldFail = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
		f.boldSkip = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
		f.boldSlow = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
		f.boldPass = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		f.dimStyle = lipgloss.NewStyle().Faint(true)
		f.boldWhite = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	}

	return f
}

func (f *SummaryFormatter) Format(summary *Summary) string {
	var sb strings.Builder
	f.formatTestDetails(&sb, summary)
	f.formatPackageSummary(&sb, summary)
	return sb.String()
}

type packageIssue struct {
	kind     string // "fail", "skip", "slow", "build", "output"
	entry    *TestExecutionEntry
	buildPkg *results.PackageResult
	pkg      *results.PackageResult
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

	for _, pkg := range summary.Packages {
		if len(pkg.OutputLines) > 0 {
			pd := ensurePkg(pkg.Name)
			pd.issues = append(pd.issues, packageIssue{kind: "output", pkg: pkg})
		}
	}

	for _, pkg := range summary.BuildFailures {
		pd := ensurePkg(pkg.Name)
		pd.issues = append(pd.issues, packageIssue{kind: "build", buildPkg: pkg})
	}

	// Use the execution entries directly from summary for failures, skipped, and slow tests
	// Build a map to look up entries by test key with iteration
	entryByKey := make(map[string][]*TestExecutionEntry)

	for _, entry := range summary.Failures {
		key := entry.TestResult.Package + "/" + entry.TestResult.Name
		entryByKey[key] = append(entryByKey[key], entry)
	}

	if f.options.IncludeSkipped {
		for _, entry := range summary.Skipped {
			key := entry.TestResult.Package + "/" + entry.TestResult.Name
			entryByKey[key] = append(entryByKey[key], entry)
		}
	}

	if f.options.IncludeSlow {
		for _, entry := range summary.SlowTests {
			key := entry.TestResult.Package + "/" + entry.TestResult.Name
			entryByKey[key] = append(entryByKey[key], entry)
		}
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
				parentKey := pkg.Name + "/" + parentName
				parentEntries := entryByKey[parentKey]

				// Collect subtest entries grouped by iteration so each
				// iteration's subtests render under their corresponding
				// parent iteration. Also remember the in-package subtest
				// order so subtests under the same iteration keep stable
				// ordering.
				subEntriesByIter := make(map[int][]*TestExecutionEntry)
				subIters := make(map[int]bool)
				subtestCount := 0
				for _, subName := range subtestsByParent[parentName] {
					subKey := pkg.Name + "/" + subName
					if entries, ok := entryByKey[subKey]; ok {
						for _, entry := range entries {
							subEntriesByIter[entry.Iteration] = append(subEntriesByIter[entry.Iteration], entry)
							subIters[entry.Iteration] = true
							subtestCount++
						}
					}
				}

				if len(parentEntries) == 0 && subtestCount == 0 {
					continue
				}

				pd := ensurePkg(pkg.Name)

				// Determine the full set of iterations to emit (union of
				// parent and subtest iterations) so iteration order is
				// preserved regardless of which side has entries.
				iterSet := make(map[int]bool)
				for _, entry := range parentEntries {
					iterSet[entry.Iteration] = true
				}
				for it := range subIters {
					iterSet[it] = true
				}
				iters := make([]int, 0, len(iterSet))
				for it := range iterSet {
					iters = append(iters, it)
				}
				sort.Ints(iters)

				parentByIter := make(map[int]*TestExecutionEntry, len(parentEntries))
				for _, entry := range parentEntries {
					parentByIter[entry.Iteration] = entry
				}

				for _, it := range iters {
					if entry, ok := parentByIter[it]; ok {
						pd.issues = append(pd.issues, packageIssue{kind: entryKind(entry), entry: entry})
					}
					for _, entry := range subEntriesByIter[it] {
						pd.issues = append(pd.issues, packageIssue{kind: entryKind(entry), entry: entry})
					}
				}
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
			case "output":
				f.formatPackageOutput(sb, issue.pkg)
			case "build":
				f.formatBuildIssue(sb, issue.buildPkg, summary)
			case "fail":
				f.formatTestIssue(sb, issue.entry, "FAIL", f.boldFail, f.failStyle)
			case "skip":
				f.formatTestIssue(sb, issue.entry, "SKIP", f.boldSkip, f.skipStyle)
			case "slow":
				f.formatSlowTestIssue(sb, issue.entry)
			}
		}

		sb.WriteString("\n")
	}
}

func isSubtest(name string) bool {
	return strings.Contains(name, "/")
}

// entryKind classifies a test execution entry for issue rendering.
func entryKind(entry *TestExecutionEntry) string {
	switch entry.TestExecution.Status {
	case results.StatusFailed:
		return "fail"
	case results.StatusSkipped:
		return "skip"
	default:
		return "slow"
	}
}

func (f *SummaryFormatter) formatTestIssue(sb *strings.Builder, entry *TestExecutionEntry, label string, boldStyle, colorStyle lipgloss.Style) {
	tr := entry.TestResult
	exec := entry.TestExecution

	// Use centralized naming helper for multi-execution tests
	name := results.ExecutionDisplayName(tr.Name, entry.Iteration, entry.TotalExecutions)
	indent := testIndent(name)

	annotation := fmt.Sprintf("(%.2fs)", exec.Elapsed.Seconds())
	if exec.Interrupted && len(exec.Output) == 0 {
		annotation = "(interrupted)"
	}

	sb.WriteString(indent)
	sb.WriteString("--- ")
	sb.WriteString(boldStyle.Render(label))
	sb.WriteString(": ")
	sb.WriteString(colorStyle.Render(name))
	sb.WriteString(" ")
	sb.WriteString(f.dimStyle.Render(annotation))
	sb.WriteString("\n")

	for _, line := range exec.Output {
		sb.WriteString(indent)
		if f.noColor {
			sb.WriteString(line)
		} else {
			sb.WriteString(ensureReset(line))
		}
		sb.WriteString("\n")
	}
}

func (f *SummaryFormatter) formatSlowTestIssue(sb *strings.Builder, entry *TestExecutionEntry) {
	tr := entry.TestResult
	exec := entry.TestExecution

	// Use centralized naming helper for multi-execution tests
	name := results.ExecutionDisplayName(tr.Name, entry.Iteration, entry.TotalExecutions)
	indent := testIndent(name)

	elapsed := fmt.Sprintf("(%.2fs)", exec.Elapsed.Seconds())

	sb.WriteString(indent)
	sb.WriteString("--- ")
	sb.WriteString(f.boldSlow.Render("SLOW"))
	sb.WriteString(": ")
	sb.WriteString(f.slowStyle.Render(name))
	sb.WriteString(" ")
	sb.WriteString(f.boldWhite.Render(elapsed))
	sb.WriteString("\n")
}

func (f *SummaryFormatter) formatPackageOutput(sb *strings.Builder, pkg *results.PackageResult) {
	for _, line := range pkg.OutputLines {
		sb.WriteString(IndentLevel)
		if f.noColor {
			sb.WriteString(line)
		} else {
			sb.WriteString(ensureReset(line))
		}
		sb.WriteString("\n")
	}
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
					sb.WriteString(IndentLevel)
					if f.noColor {
						sb.WriteString(line)
					} else {
						sb.WriteString(f.failStyle.Render(ensureReset(line)))
					}
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
		statusWord   string
		name         string
		extra        string
		showDuration bool
		pkg          *results.PackageResult
	}

	lines := make([]pkgLine, 0, len(summary.Packages))

	maxStatusLen := 0
	maxNameExtraLen := 0
	maxPassedLen := 0
	maxFailedLen := 0
	maxSkippedLen := 0
	maxTotalLen := 0

	maxPassedLen = len(fmt.Sprintf("%d", summary.PassedTests))
	maxFailedLen = len(fmt.Sprintf("%d", summary.FailedTests))
	maxSkippedLen = len(fmt.Sprintf("%d", summary.SkippedTests))
	maxTotalLen = len(fmt.Sprintf("%d", summary.TotalTests))

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
		} else if pkg.SummaryLine != "" {
			output := expandTabs(pkg.SummaryLine, 8)
			nameIdx := strings.Index(output, pkg.Name)
			if nameIdx >= 0 {
				rest := strings.TrimSpace(output[nameIdx+len(pkg.Name):])
				if rest != "" {
					pl.extra = rest
				}
			}
		}

		// Omit durations for packages that didn't actually run tests.
		switch pl.extra {
		case "[build failed]", "[no test files]", "(cached)":
			pl.showDuration = false
		default:
			pl.showDuration = true
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

	maxElapsedLen := 0
	for _, pl := range lines {
		if pl.showDuration {
			if el := len(formatDuration(pl.pkg.Elapsed)); el > maxElapsedLen {
				maxElapsedLen = el
			}
		}
	}
	if el := len(formatDuration(summary.TotalTime)); el > maxElapsedLen {
		maxElapsedLen = el
	}

	// (✓NN ✗NN ∅NN) NN
	// parens=2, 3 symbols (multi-byte but 1 display col each), 2 inner spaces, 1 outer space
	countsWidth := 2 + 3 + 2 + maxPassedLen + maxFailedLen + maxSkippedLen + 1 + maxTotalLen
	lineWidth := maxStatusLen + 4 + maxNameExtraLen + 2 + countsWidth + 2 + maxElapsedLen
	separatorLen := lineWidth
	if f.width > separatorLen {
		separatorLen = f.width
	}

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

		// Package name+info renders in the terminal's default foreground; the
		// color-coded status word (FAIL/ok/?) alone signals package status.
		paddedNameExtra := fmt.Sprintf("%-*s", maxNameExtraLen, nameExtra)

		hasCounts := pl.pkg.Counts.Passed > 0 || pl.pkg.Counts.Failed > 0 || pl.pkg.Counts.Skipped > 0
		countsStr := ""
		if hasCounts {
			passedStr := fmt.Sprintf("%*s", maxPassedLen+1, fmt.Sprintf("%s%d", SymbolPass, pl.pkg.Counts.Passed))
			if pl.pkg.Counts.Passed > 0 {
				passedStr = f.passStyle.Render(passedStr)
			} else {
				passedStr = f.neutralStyle.Render(passedStr)
			}

			failedStr := fmt.Sprintf("%*s", maxFailedLen+1, fmt.Sprintf("%s%d", SymbolFail, pl.pkg.Counts.Failed))
			if pl.pkg.Counts.Failed > 0 {
				failedStr = f.failStyle.Render(failedStr)
			} else {
				failedStr = f.neutralStyle.Render(failedStr)
			}

			skippedStr := fmt.Sprintf("%*s", maxSkippedLen+1, fmt.Sprintf("%s%d", SymbolSkip, pl.pkg.Counts.Skipped))
			if pl.pkg.Counts.Skipped > 0 {
				skippedStr = f.skipStyle.Render(skippedStr)
			} else {
				skippedStr = f.neutralStyle.Render(skippedStr)
			}

			total := pl.pkg.Counts.Passed + pl.pkg.Counts.Failed + pl.pkg.Counts.Skipped
			totalStr := f.neutralStyle.Render(fmt.Sprintf("%*d", maxTotalLen, total))

			countsStr = fmt.Sprintf("(%s %s %s) %s", passedStr, failedStr, skippedStr, totalStr)
		}

		elapsed := ""
		if pl.showDuration {
			elapsed = fmt.Sprintf("  %*s", maxElapsedLen, formatDuration(pl.pkg.Elapsed))
		}

		if countsStr != "" {
			fmt.Fprintf(sb, "%s    %s  %s%s\n",
				statusStr, paddedNameExtra, countsStr, elapsed)
		} else {
			emptyCounts := strings.Repeat(" ", countsWidth)
			fmt.Fprintf(sb, "%s    %s  %s%s\n",
				statusStr, paddedNameExtra, emptyCounts, elapsed)
		}
	}

	sb.WriteString(strings.Repeat("-", separatorLen))
	sb.WriteString("\n")

	pkgLabel := fmt.Sprintf("(%d packages)", summary.PackageCount)

	passedStr := fmt.Sprintf("%*s", maxPassedLen+1, fmt.Sprintf("%s%d", SymbolPass, summary.PassedTests))
	if summary.PassedTests > 0 {
		passedStr = f.passStyle.Render(passedStr)
	} else {
		passedStr = f.neutralStyle.Render(passedStr)
	}

	failedStr := fmt.Sprintf("%*s", maxFailedLen+1, fmt.Sprintf("%s%d", SymbolFail, summary.FailedTests))
	if summary.FailedTests > 0 {
		failedStr = f.failStyle.Render(failedStr)
	} else {
		failedStr = f.neutralStyle.Render(failedStr)
	}

	skippedStr := fmt.Sprintf("%*s", maxSkippedLen+1, fmt.Sprintf("%s%d", SymbolSkip, summary.SkippedTests))
	if summary.SkippedTests > 0 {
		skippedStr = f.skipStyle.Render(skippedStr)
	} else {
		skippedStr = f.neutralStyle.Render(skippedStr)
	}

	totalStr := f.neutralStyle.Render(fmt.Sprintf("%*d", maxTotalLen, summary.TotalTests))
	countsStr := fmt.Sprintf("(%s %s %s) %s", passedStr, failedStr, skippedStr, totalStr)
	elapsed := fmt.Sprintf("%*s", maxElapsedLen, formatDuration(summary.TotalTime))

	labelWidth := maxStatusLen + 4 + maxNameExtraLen
	fmt.Fprintf(sb, "%-*s  %s  %s\n", labelWidth, pkgLabel, countsStr, elapsed)
}
