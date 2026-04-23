## Why

When `go test -count=N` reruns tests multiple times, tang's running count goes negative (e.g. `▶-1053`) and failures from earlier iterations are silently erased — they don't appear in the summary or TUI. The root cause is that `TestResult` stores only a single status/output per unique test name, so the second iteration's events overwrite the first without properly adjusting counters.

## What Changes

- Refactor `TestResult` to hold a chronological `Executions []*TestExecution` slice, where each iteration of a test under `-count=N` gets its own execution record.
- Add accessor methods on `TestResult` (`Latest()`, `Status()`, `Elapsed()`, etc.) plus construction/append helpers so every `TestResult` maintains a non-empty executions invariant.
- Fix the collector's event handling: when a `"run"` event arrives for an existing test whose latest execution is terminal (passed/failed/skipped), append a new execution and correctly increment the running counter.
- Update the summary formatter to list every failed execution (not just the latest), and to treat skipped/slow detail entries as execution-specific where those sections are enabled, using centralized `TestFoo#NN` labeling whenever a test has more than one execution (every execution is labeled, including the first).
- Update JUnit XML output to emit one `<testcase>` per execution, with the `#NN` suffix applied to every `<testcase>` when a test has more than one execution, and line-preserving failure content.
- Update the TUI and simple-mode renderers to read from the latest execution (no multi-execution awareness needed in live views).

## Capabilities

### New Capabilities
- `test-execution-tracking`: Per-execution result tracking within a single test run, enabling correct counters and preserved failure history when `go test -count=N` causes tests to execute multiple times.

### Modified Capabilities
<!-- No existing specs to modify -->

## Impact

- **Core model**: `results/model.go` — new `TestExecution` type, slimmed `TestResult` struct with accessor methods. All fields that were directly on `TestResult` (Status, Output, Elapsed, SummaryLine, etc.) move into `TestExecution`.
- **Collector**: `results/collector.go` — rerun detection logic in `handleTestLevelEvent`; all field writes route through `Latest()`.
- **Summary**: `output/format/summary.go` and `summary_formatter.go` — iterate executions when building failure/skip/slow lists; render per-execution failure blocks with iteration labels; avoid collapsing multiple entries for the same unique test name.
- **JUnit**: `output/junit/junit.go` — emit one `<testcase>` per execution.
- **TUI**: `tui/model.go` — minor accessor updates to read from latest execution.
- **Simple output**: `output/simple.go` — minor accessor updates.
- **Test fixtures**: `internal/testutil/builders.go` and all test files that construct `TestResult` instances.
- **Watch-mode reruns** (collector.go:135-169): semantics unchanged — package-level `start` events still wipe and recreate `TestResults` for that package, and reset package-local failure metadata such as `PanicTestKey`, which is correct for watch-mode semantics (fresh run, not an intentional multi-iteration).
