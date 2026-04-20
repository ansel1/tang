package tui_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/internal/testutil"
	"github.com/ansel1/tang/results"
)

// ansiRe strips ANSI SGR escape sequences so assertions can match raw text.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

// TestFinishedPackageGutterIcons verifies that finished package header lines
// use a colored gutter icon (✓/✗/∅) in place of the literal "ok"/"FAIL"/"?"
// status token, so the package name aligns at column 3 across all states.
func TestFinishedPackageGutterIcons(t *testing.T) {
	cases := []struct {
		name        string
		status      results.Status
		summaryLine string
		wantIcon    string
		wantTail    string
		forbidWord  string
	}{
		{
			name:        "failed package",
			status:      results.StatusFailed,
			summaryLine: "FAIL\tgithub.com/ansel1/tang/sample/badtestmain\t0.202s",
			wantIcon:    "✗",
			wantTail:    "github.com/ansel1/tang/sample/badtestmain",
			forbidWord:  "FAIL\t",
		},
		{
			name:        "failed build",
			status:      results.StatusFailed,
			summaryLine: "FAIL\tgithub.com/ansel1/tang/sample/broken [build failed]",
			wantIcon:    "✗",
			wantTail:    "github.com/ansel1/tang/sample/broken [build failed]",
			forbidWord:  "FAIL\t",
		},
		{
			name:        "no test files",
			status:      results.StatusSkipped,
			summaryLine: "?   \tgithub.com/ansel1/tang/sample/models\t[no test files]",
			wantIcon:    "∅",
			wantTail:    "github.com/ansel1/tang/sample/models",
			forbidWord:  "?   \t",
		},
		{
			name:        "passed package",
			status:      results.StatusPassed,
			summaryLine: "ok  \tgithub.com/test/pkg1\t0.10s",
			wantIcon:    "✓",
			wantTail:    "github.com/test/pkg1    0.10s",
			forbidWord:  "ok      ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testutil.BuildModel(
				testutil.WithTermSize(200, 24),
				testutil.WithRunStatus(results.StatusPassed),
				testutil.WithPackage("pkg",
					testutil.PkgStatus(tc.status),
					testutil.PkgElapsed(100*time.Millisecond),
					testutil.PkgOutput(tc.summaryLine),
				),
			)

			raw := m.String()
			plain := stripAnsi(raw)

			var pkgLine string
			for _, line := range strings.Split(plain, "\n") {
				if strings.Contains(line, tc.wantTail) {
					pkgLine = line
					break
				}
			}
			if pkgLine == "" {
				t.Fatalf("no line contained %q; full output:\n%s", tc.wantTail, plain)
			}

			// Gutter icon must be at column 1 (index 0), followed by a space,
			// and the package/summary content must start at column 3.
			if !strings.HasPrefix(pkgLine, tc.wantIcon+" ") {
				t.Errorf("expected line to start with %q followed by space; got %q", tc.wantIcon, pkgLine)
			}
			if strings.Contains(pkgLine, tc.forbidWord) {
				t.Errorf("status word %q should have been replaced by gutter icon; got %q", tc.forbidWord, pkgLine)
			}
		})
	}
}
