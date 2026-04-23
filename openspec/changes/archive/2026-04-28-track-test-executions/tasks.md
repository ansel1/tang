## 1. Core Data Model

- [x] 1.1 Create `TestExecution` struct in `results/model.go` with fields moved from `TestResult`: `Status`, `StartTime`, `WallStartTime`, `Elapsed`, `Output`, `SummaryLine`, `Interrupted`, `ActiveDuration`, `LastResumeTime`
- [x] 1.2 Replace per-execution fields on `TestResult` with `Executions []*TestExecution` slice; keep `Package` and `Name` on `TestResult`
- [x] 1.3 Add construction/append helpers (e.g. `NewTestResult(...)`, `NewTestExecution(...)`, `AppendExecution(...)`) and make the intended invariant explicit: every constructed `TestResult` has at least one execution
- [x] 1.4 Add accessor methods on `*TestResult`: `Latest()`, `Status()`, `Elapsed()`, `Running()`, `SummaryLine()`, `Output()`, `Interrupted()`, `ActiveDuration()`, `LastResumeTime()`, `StartTime()`, `WallStartTime()`
- [x] 1.5 Add centralized iteration-label helper (e.g. `ExecutionDisplayName(name string, iteration, total int)`) used by summary and JUnit; convention is plain name when `total == 1`, otherwise zero-padded `#NN` suffix on every execution including the first (e.g. `TestFoo#01`, `TestFoo#02`), with the suffix anchored on the top-level test name for subtests (e.g. `TestFoo#02/sub`)
- [x] 1.6 Verify the project compiles (fix any immediate breakage from the struct change — compile errors expected in every consumer)

## 2. Collector Rerun Logic

- [x] 2.1 Update `handleTestLevelEvent` in `results/collector.go`: when an event arrives for an existing `TestResult` whose `Latest().Status` is terminal and the action is `"run"`, append a new `TestExecution` and increment `pkg.Counts.Running++` / `run.Counts.Running++`
- [x] 2.2 Route all field writes in `handleTestLevelEvent` switch cases through `Latest()` (status, output, summary line, elapsed, active duration, etc.)
- [x] 2.3 Update `failInterruptedTests` to operate on `Latest()` for each affected test
- [x] 2.4 Update the test-creation block (find-or-create) to initialize `TestResult` with a single `TestExecution` in the `Executions` slice instead of setting fields directly
- [x] 2.5 Verify watch-mode rerun semantics: package-level `start` for a completed package deletes prior `TestResults`, clears `TestOrder`/`DisplayOrder`, resets counts, and clears package-local failure metadata including `PanicTestKey`
- [x] 2.6 Preserve final package semantics: a package whose first `-count` execution fails and later execution passes must finish with package status `Failed` when the package-level `fail` event arrives

## 3. Test Fixture Builder

- [x] 3.1 Update `internal/testutil/builders.go` to construct `TestResult` with an `Executions` slice; update `TStatus`, `TElapsed`, `TSummaryLine`, `TOutput`, and count-adjustment logic to write into the execution
- [x] 3.2 Add a new builder option (e.g. `TIterations(...)` or `TAddExecution(...)`) for constructing multi-execution test fixtures
- [x] 3.3 Update builder count-adjustment logic to count every execution, not just every unique test name

## 4. TUI Rendering

- [x] 4.1 Update `tui/model.go` to use `TestResult` accessor methods (`tr.Status()`, `tr.Elapsed()`, `tr.Output()`, `tr.SummaryLine()`, `tr.Latest()`, etc.) instead of direct field access
- [x] 4.2 Update `tui/smart_render_test.go` test fixtures to use the new `TestResult` structure
- [x] 4.3 Update `tui/bleed_test.go` test fixtures to use the new `TestResult` structure
- [x] 4.4 Add/adjust a TUI test showing that during a second execution the test row reflects the latest execution while package/run counts remain cumulative

## 5. Simple (Non-TTY) Output

- [x] 5.1 Update `output/simple.go` to read from `tr.Latest()` when rendering failure output on `"fail"` events
- [x] 5.2 Add a non-TTY test for `-count=2` where both executions fail; assert both executions' summary/output blocks are emitted in the stream
- [x] 5.3 Verify both simple-output paths call `collector.Push(event)` before rendering (`main.go` live path and `SimpleOutput.ProcessEvents` test/path)

## 6. Summary Computation and Formatting

- [x] 6.1 Introduce a structured per-execution summary entry (e.g. `TestExecutionEntry`) holding `*TestResult`, `*TestExecution`, `Iteration int`, and `TotalExecutions int`
- [x] 6.2 Update `ComputeSummary` to iterate `tr.Executions` and produce one entry per failed execution, one per skipped execution, and one per slow execution
- [x] 6.3 Update `Summary.Failures`, `Summary.Skipped`, and `Summary.SlowTests` types from `[]*TestResult` to `[]*TestExecutionEntry` (or equivalent)
- [x] 6.4 Update `output/format/summary_formatter.go` to render entries, not just unique tests: use centralized `#NN` labeling and render per-execution output blocks
- [x] 6.5 Update formatter grouping/classification so it can return multiple issues for the same unique test name (e.g. two failed executions of `TestFoo`) instead of collapsing to one issue
- [x] 6.6 Preserve subtest grouping while rendering multiple execution entries; subtest labels use `TestParent#02/sub`
- [x] 6.7 Update `output/format/compute_test.go` fixtures and add test cases for multi-execution failures, skips, and slow tests
- [x] 6.8 Update `output/format/formatter_test.go` fixtures for the new entry type and add a test that two failed executions of the same test render as two failure blocks

## 7. JUnit XML Output

- [x] 7.1 Update `output/junit/junit.go` to iterate `tr.Executions` and emit one `<testcase>` per execution; apply centralized `#NN` suffix to `name` attribute for non-first executions
- [x] 7.2 Preserve failure output per execution in JUnit by joining that execution's output lines with newlines (and include the execution summary line when useful) instead of formatting the output slice with `%v`
- [x] 7.3 Update `output/junit/junit_test.go` fixtures and add a test case for multi-execution output, including `TestFoo` + `TestFoo#02` and failure only on the failed execution
- [x] 7.4 Add/adjust a JUnit test for a multi-execution subtest name (`TestFoo#02/sub`) to lock the label convention

## 8. Collector Tests

- [x] 8.1 Update existing assertions in `results/collector_test.go` to use accessor methods (e.g. `tr.Status()` instead of `tr.Status`)
- [x] 8.2 Add test: `-count=2` where both executions pass — assert `len(Executions)==2`, both `StatusPassed`, package/run passed counts equal 2, `Running==0`
- [x] 8.3 Add test: `-count=2` where iter1 fails, iter2 passes — assert both executions preserved, `pkg.Counts.Failed==1`, `pkg.Counts.Passed==1`, `Running==0`, first execution's output retained, and final package status is `Failed` after the package-level `fail` event
- [x] 8.4 Add test: `-count=2` where both executions fail — assert `len(Executions)==2`, both `StatusFailed`, `pkg.Counts.Failed==2`, `Running==0`, and both executions' outputs are preserved independently
- [x] 8.5 Add test: pause/cont within a rerun execution still works correctly
- [x] 8.6 Add test: parallel tests with `-count=2` keep `Running`/`Paused` counts balanced and never negative
- [x] 8.7 Add watch-mode rerun regression test: stale `PanicTestKey` and prior executions are cleared on package-level `start` after completion

## 9. Verification

- [x] 9.1 Run full test suite (`go test ./...`) and fix any remaining failures
- [x] 9.2 Manual verification: run `go test -count=2 ./sample/...` through tang and confirm running count never goes negative and first-iteration failures appear in the summary
- [x] 9.3 Manual verification: run with `--junitfile` and confirm each execution produces a separate `<testcase>` element