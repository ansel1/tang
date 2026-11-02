package format

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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
	layout := f.computeTableLayout(summary)
	f.formatTestDetails(&sb, summary, layout)
	f.formatPackageSummary(&sb, summary, layout)
	return sb.String()
}

type packageIssue struct {
	kind     string // "fail", "skip", "slow", "build", "output"
	entry    *TestExecutionEntry
	buildPkg *results.PackageResult
	pkg      *results.PackageResult
}

func (f *SummaryFormatter) formatTestDetails(sb *strings.Builder, summary *Summary, layout summaryTableLayout) {
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

	messagesPrefixPlain := LineHorizontal + LineHorizontal + LineHorizontal + " Messages "
	messagesPrefixLen := utf8.RuneCountInString(messagesPrefixPlain)
	messagesDashes := max(0, layout.separatorLen-messagesPrefixLen)
	sb.WriteString(f.dimRule(3))
	sb.WriteString(" Messages ")
	sb.WriteString(f.dimRule(messagesDashes))
	sb.WriteString("\n")

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

type pkgLine struct {
	statusWord   string
	name         string
	extra        string
	showDuration bool
	pkg          *results.PackageResult
}

type summaryTableLayout struct {
	lines           []pkgLine
	pkgLabel        string
	maxStatusLen    int
	maxNameExtraLen int
	maxPassedLen    int
	maxFailedLen    int
	maxSkippedLen   int
	maxTotalLen     int
	maxElapsedLen   int
	labelWidth      int
	countsWidth     int
	lineWidth       int
	separatorLen    int
}

const maxSummaryWidth = 150

func (f *SummaryFormatter) computeTableLayout(summary *Summary) summaryTableLayout {
	maxWidth := maxSummaryWidth
	if f.width > 0 && f.width < maxWidth {
		maxWidth = f.width
	}

	if len(summary.Packages) == 0 {
		return summaryTableLayout{separatorLen: maxWidth}
	}

	lines := make([]pkgLine, 0, len(summary.Packages))

	maxStatusLen := 0
	maxNameExtraLen := 0
	maxPassedLen := len(fmt.Sprintf("%d", summary.PassedTests))
	maxFailedLen := len(fmt.Sprintf("%d", summary.FailedTests))
	maxSkippedLen := len(fmt.Sprintf("%d", summary.SkippedTests))
	maxTotalLen := len(fmt.Sprintf("%d", summary.TotalTests))

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
				rest = stripLeadingDuration(rest)
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
		if nameExtraLen := utf8.RuneCountInString(nameExtra); nameExtraLen > maxNameExtraLen {
			maxNameExtraLen = nameExtraLen
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

	pkgLabel := fmt.Sprintf("(%d packages)", summary.PackageCount)
	summaryPrefixLen := utf8.RuneCountInString(LineHorizontal + LineHorizontal + LineHorizontal + " Summary ")
	messagesPrefixLen := utf8.RuneCountInString(LineHorizontal + LineHorizontal + LineHorizontal + " Messages ")
	minLabelWidth := max(len(pkgLabel), summaryPrefixLen, messagesPrefixLen)
	if maxStatusLen+4+maxNameExtraLen < minLabelWidth {
		maxNameExtraLen = minLabelWidth - maxStatusLen - 4
	}
	labelWidth := maxStatusLen + 4 + maxNameExtraLen

	// NN   NN   NN   NN (4 count columns separated by 3 spaces so " ─ <icon>" fits above each gap)
	countsWidth := maxPassedLen + 3 + maxFailedLen + 3 + maxSkippedLen + 3 + maxTotalLen
	rightWidth := 2 + countsWidth + 2 + maxElapsedLen
	if labelWidth+rightWidth > maxWidth {
		maxNameExtraLen = max(minLabelWidth-maxStatusLen-4, maxWidth-rightWidth-maxStatusLen-4)
		labelWidth = maxStatusLen + 4 + maxNameExtraLen
	}
	lineWidth := labelWidth + rightWidth
	separatorLen := min(lineWidth, maxWidth)

	return summaryTableLayout{
		lines:           lines,
		pkgLabel:        pkgLabel,
		maxStatusLen:    maxStatusLen,
		maxNameExtraLen: maxNameExtraLen,
		maxPassedLen:    maxPassedLen,
		maxFailedLen:    maxFailedLen,
		maxSkippedLen:   maxSkippedLen,
		maxTotalLen:     maxTotalLen,
		maxElapsedLen:   maxElapsedLen,
		labelWidth:      labelWidth,
		countsWidth:     countsWidth,
		lineWidth:       lineWidth,
		separatorLen:    separatorLen,
	}
}

func (f *SummaryFormatter) dimRule(n int) string {
	if n <= 0 {
		return ""
	}
	return f.dimStyle.Render(strings.Repeat(LineHorizontal, n))
}

func (f *SummaryFormatter) formatPackageSummary(sb *strings.Builder, summary *Summary, layout summaryTableLayout) {
	if len(layout.lines) == 0 {
		return
	}

	summaryPrefixPlain := LineHorizontal + LineHorizontal + LineHorizontal + " Summary "
	summaryPrefixLen := utf8.RuneCountInString(summaryPrefixPlain)
	leadDashes := max(0, layout.labelWidth+layout.maxPassedLen-summaryPrefixLen)
	headerColumns := fmt.Sprintf("%s %s %s %s %s %s %s %s %s",
		f.passStyle.Render(SymbolPass),
		f.dimRule(layout.maxFailedLen),
		f.failStyle.Render(SymbolFail),
		f.dimRule(layout.maxSkippedLen),
		f.skipStyle.Render(SymbolSkip),
		f.dimRule(layout.maxTotalLen),
		f.neutralStyle.Render(SymbolTotal),
		f.dimRule(max(1, layout.maxElapsedLen-1)),
		f.neutralStyle.Render(SymbolTime),
	)
	fmt.Fprintf(sb, "%s Summary %s %s\n",
		f.dimRule(3),
		f.dimRule(leadDashes),
		headerColumns,
	)

	for _, pl := range layout.lines {
		var statusStr string
		switch pl.statusWord {
		case "FAIL":
			statusStr = f.boldFail.Render(fmt.Sprintf("%-*s", layout.maxStatusLen, pl.statusWord))
		case "ok":
			statusStr = f.boldPass.Render(fmt.Sprintf("%-*s", layout.maxStatusLen, pl.statusWord))
		case "?":
			statusStr = f.boldSkip.Render(fmt.Sprintf("%-*s", layout.maxStatusLen, pl.statusWord))
		}

		nameExtra := pl.name
		if pl.extra != "" {
			nameExtra += " " + pl.extra
		}
		nameExtra = truncateStr(nameExtra, layout.maxNameExtraLen)

		// Package name+info renders in the terminal's default foreground; the
		// color-coded status word (FAIL/ok/?) alone signals package status.
		paddedNameExtra := fmt.Sprintf("%-*s", layout.maxNameExtraLen, nameExtra)

		hasCounts := pl.pkg.Counts.Passed > 0 || pl.pkg.Counts.Failed > 0 || pl.pkg.Counts.Skipped > 0
		countsStr := ""
		if hasCounts {
			passedStr := formatCount(pl.pkg.Counts.Passed, layout.maxPassedLen, f.passStyle, true)
			failedStr := formatCount(pl.pkg.Counts.Failed, layout.maxFailedLen, f.failStyle, true)
			skippedStr := formatCount(pl.pkg.Counts.Skipped, layout.maxSkippedLen, f.skipStyle, true)

			total := pl.pkg.Counts.Passed + pl.pkg.Counts.Failed + pl.pkg.Counts.Skipped
			totalStr := f.neutralStyle.Render(fmt.Sprintf("%*d", layout.maxTotalLen, total))

			countsStr = fmt.Sprintf("%s   %s   %s   %s", passedStr, failedStr, skippedStr, totalStr)
		}

		elapsed := ""
		if pl.showDuration {
			elapsed = fmt.Sprintf("  %*s", layout.maxElapsedLen, formatDuration(pl.pkg.Elapsed))
		}

		if countsStr != "" {
			fmt.Fprintf(sb, "%s    %s  %s%s\n",
				statusStr, paddedNameExtra, countsStr, elapsed)
		} else {
			emptyCounts := strings.Repeat(" ", layout.countsWidth)
			fmt.Fprintf(sb, "%s    %s  %s%s\n",
				statusStr, paddedNameExtra, emptyCounts, elapsed)
		}
	}

	bottomLeadDashes := layout.labelWidth + layout.maxPassedLen
	fmt.Fprintf(sb, "%s %s\n",
		f.dimRule(bottomLeadDashes),
		headerColumns,
	)

	var passColorForSummary lipgloss.Style
	if summary.PassedTests > 0 {
		passColorForSummary = f.passStyle
	} else {
		passColorForSummary = f.neutralStyle
	}
	passedStr := formatCount(summary.PassedTests, layout.maxPassedLen, passColorForSummary, false)

	var failColorForSummary lipgloss.Style
	if summary.FailedTests > 0 {
		failColorForSummary = f.failStyle
	} else {
		failColorForSummary = f.neutralStyle
	}
	failedStr := formatCount(summary.FailedTests, layout.maxFailedLen, failColorForSummary, false)

	var skipColorForSummary lipgloss.Style
	if summary.SkippedTests > 0 {
		skipColorForSummary = f.skipStyle
	} else {
		skipColorForSummary = f.neutralStyle
	}
	skippedStr := formatCount(summary.SkippedTests, layout.maxSkippedLen, skipColorForSummary, false)

	totalStr := f.neutralStyle.Render(fmt.Sprintf("%*d", layout.maxTotalLen, summary.TotalTests))
	countsStr := fmt.Sprintf("%s   %s   %s   %s", passedStr, failedStr, skippedStr, totalStr)
	elapsed := fmt.Sprintf("%*s", layout.maxElapsedLen, formatDuration(summary.TotalTime))

	fmt.Fprintf(sb, "%-*s  %s  %s\n", layout.labelWidth, layout.pkgLabel, countsStr, elapsed)
}

func formatCount(count int, width int, style lipgloss.Style, hideZero bool) string {
	if count == 0 && hideZero {
		return strings.Repeat(" ", width)
	}
	raw := fmt.Sprintf("%*d", width, count)
	return style.Render(raw)
}

func stripLeadingDuration(s string) string {
	fields := strings.Fields(s)
	if len(fields) > 0 {
		if _, err := time.ParseDuration(fields[0]); err == nil {
			idx := strings.Index(s, fields[0])
			return strings.TrimSpace(s[idx+len(fields[0]):])
		}
	}
	return s
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}


