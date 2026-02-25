package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UpdateGolden controls whether golden files are rewritten.
// Run with: go test ./tui/ -update
var UpdateGolden = flag.Bool("update", false, "update golden files")

// AssertGolden compares actual against a golden file stored at
// testdata/golden/<name>.golden relative to the caller's package directory.
// If -update is set, the golden file is written instead.
func AssertGolden(t *testing.T, name, actual string) {
	t.Helper()

	scrubbed := ScrubNonDeterministic(actual)

	// Resolve testdata/golden relative to the caller's file
	_, callerFile, _, ok := runtime.Caller(1)
	require.True(t, ok, "runtime.Caller failed")

	goldenDir := filepath.Join(filepath.Dir(callerFile), "testdata", "golden")
	goldenPath := filepath.Join(goldenDir, name+".golden")

	if *UpdateGolden {
		require.NoError(t, os.MkdirAll(goldenDir, 0o755))
		require.NoError(t, os.WriteFile(goldenPath, []byte(scrubbed), 0o644))
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "golden file missing; run with -update to create: %s", goldenPath)

	assert.Equal(t, string(expected), scrubbed,
		"golden mismatch for %s (run with -update to accept)", name)
}

// spinnerRE matches the MiniDot spinner characters.
var spinnerRE = regexp.MustCompile("[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]")

// elapsedRE matches an elapsed-time value (e.g. "1.2s", "606090.4m") that may
// be wrapped in ANSI escape sequences (bold, color).
//
// Group layout:
//
//	(1) leading whitespace + optional ANSI codes
//	(2) the numeric value including unit  e.g. "0.1s"
//	(3) optional trailing ANSI codes
var elapsedRE = regexp.MustCompile(`(\s(?:\x1b\[[0-9;]*m)*)(\d+\.\d+[sm])((?:\x1b\[[0-9;]*m)*)`)

// ScrubNonDeterministic replaces non-deterministic content so that golden
// files can be compared stably across runs.
//
//   - Spinner frames (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏) → ~
//   - Trailing elapsed time values (\d+\.\d+[sm]) → X.Xs
func ScrubNonDeterministic(s string) string {
	s = spinnerRE.ReplaceAllString(s, "~")

	// Scrub elapsed times line-by-line: only replace the last occurrence of an
	// elapsed-time pattern on each line (the right-aligned column), leaving
	// times embedded in package output (left side) untouched.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		matches := elapsedRE.FindAllStringSubmatchIndex(line, -1)
		if len(matches) == 0 {
			continue
		}
		// Use the last match — this is the right-aligned elapsed column.
		last := matches[len(matches)-1]
		// Replace group 2 (the numeric value) with X.Xs, preserving
		// surrounding whitespace and ANSI codes.
		numStart := last[4]
		numEnd := last[5]
		lines[i] = line[:numStart] + "X.Xs" + line[numEnd:]
	}
	return strings.Join(lines, "\n")
}
