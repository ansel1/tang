## Context

Tang parses `go test -json` output and presents test results in a live TUI, non-TTY streaming mode, summary table, and JUnit XML. The current data model uses a single `TestResult` per unique test name (`run.TestResults["pkg/TestFoo"]`), storing one status, one output buffer, and one elapsed time. When `go test -count=N` reruns a test, the second iteration's events overwrite the first's fields without correctly adjusting the running counter (causing negative counts) and without preserving earlier failure details.

The collector (`results/collector.go`) creates a `TestResult` on the first `"run"` event and increments `Running++`. On subsequent iterations, the existing `TestResult` is found in the map, so `Running++` is skipped — but the terminal event (`pass`/`fail`/`skip`) always decrements `Running--`, driving it negative. Additionally, the `Status`, `Output`, `SummaryLine`, and `Elapsed` fields are overwritten, erasing any prior failure evidence.

## Goals / Non-Goals

**Goals:**
- Each test execution under `-count=N` is tracked as a distinct record with its own status, output, elapsed time, and summary line
- Running/paused counters remain non-negative and accurate at all times
- All failed executions appear in the summary and JUnit output, even if later executions of the same test pass
- Live TUI and simple-mode output continue to work correctly, showing the latest execution's state
- Minimal disruption to existing consumers — callers that only need the latest execution use accessor methods

**Non-Goals:**
- Full "test history" UX (e.g., a dedicated view to browse all N executions of a flaky test) — that's a future feature
- Changes to watch-mode rerun semantics — those correctly reset per-package and should remain as-is
- Handling of `"bench"` action events (separate TODO item)
- Browse command (separate TODO item)

## Decisions

### Decision 1: Execution slice on TestResult (vs. keying by iteration)

**Choice:** Add `Executions []*TestExecution` to `TestResult`, with accessor methods for the latest execution. `TestResult` instances SHALL maintain the invariant that `Executions` is non-empty after construction; use helpers such as `NewTestResult(...)`, `NewTestExecution(...)`, and/or `AppendExecution(...)` so collector and fixture code do not hand-roll execution initialization. `Latest()` may defensively handle an empty slice, but empty execution slices are a programming error, not a normal state.

**Alternative considered:** Key `TestResults` by `"pkg/TestFoo#02"` (include iteration in the map key). This would avoid changing `TestResult`'s structure but would break every consumer that looks up test results by the natural `"pkg/TestName"` key (TUI, simple output, summary, JUnit, `failInterruptedTests`). It would also require tracking iteration counts externally.

**Rationale:** The execution-slice approach keeps the `TestResults` map keyed by natural test name, preserving all existing lookups. The accessor methods (`Latest()`, `Status()`, `Elapsed()`, etc.) keep TUI/simple-mode call sites concise. Only consumers that need multi-execution awareness (summary, JUnit) iterate the `Executions` slice directly.

### Decision 2: Rerun detection in the collector

**Choice:** When `handleTestLevelEvent` receives any event for an existing `TestResult` whose `Latest().Status` is terminal (`Passed`, `Failed`, `Skipped`), and the event action is `"run"`, append a new `TestExecution` and increment `Running++`.

**Alternative considered:** Detecting reruns at the package level (similar to the existing watch-mode `start` handler at collector.go:135-169). However, `-count=N` does not emit an intervening package `start` event between iterations — all iterations run within a single package run. So detection must happen at the test level.

**Rationale:** The `"run"` action is always emitted by `go test` at the start of each iteration. Checking `Latest().Status` is terminal is sufficient to distinguish a rerun from an initial run. The collector already has the `TestResult` in hand at this point, so the check is cheap.

### Decision 3: Iteration labeling convention

**Choice:** When a test has only one execution, display the plain test name (`TestFoo`, `TestFoo/sub`). When a test has two or more executions, label every execution — including the first — with a zero-padded `#NN` suffix: `TestFoo#01`, `TestFoo#02`, `TestFoo#03`, etc. For subtests, the suffix goes on the top-level test name before the subtest path: `TestFoo#01/sub`, `TestFoo#02/sub`. For deeper subtests, the suffix stays on the top-level segment (e.g. `TestFoo#02/sub/nested`). Centralize this formatting in a helper shared by summary and JUnit so the convention cannot drift.

**Alternative considered:** Leave the first execution unlabeled (`TestFoo`, then `TestFoo#02`, `TestFoo#03`). This minimizes diff vs. current single-execution output but is visually inconsistent in multi-execution lists — the first row reads as the canonical name and the rest read as variants, which makes lists like a 5x rerun harder to scan and harder to count.

**Alternative considered:** `TestFoo (run 2 of 3)` — more verbose, takes more horizontal space in the already-tight summary lines. `TestFoo (iteration 2)` — similarly verbose.

**Rationale:** The `#NN` format is compact and echoes Go's subtest naming convention (e.g., `TestFoo/sub`). Labeling every execution when there is more than one keeps multi-execution lists visually uniform and makes iteration counts immediately obvious. The plain-name case for single executions keeps output identical to today's for the common case.

### Decision 4: TestExecution fields (moved from TestResult)

**Choice:** Move `Status`, `StartTime`, `WallStartTime`, `Elapsed`, `Output`, `SummaryLine`, `Interrupted`, `ActiveDuration`, `LastResumeTime` into `TestExecution`. Keep `Package` and `Name` on `TestResult` (they're per-test, not per-execution).

**Rationale:** These fields all represent per-execution state. The current bug exists precisely because they're shared across iterations. Moving them into `TestExecution` makes the data model correctly represent reality.

### Decision 5: Summary failure collection uses structured entries

**Choice:** Change `Summary.Failures`, `Summary.Skipped`, and `Summary.SlowTests` from `[]*TestResult` to structured per-execution entries (e.g. `TestExecutionEntry`) holding `{ TestResult *TestResult; Execution *TestExecution; Iteration int; TotalExecutions int }`.

**Alternative considered:** Keep `[]*TestResult` and add the same `*TestResult` once per failed execution — but this loses the execution identity (which execution failed?) and doesn't carry the iteration index for labeling.

**Rationale:** The structured entry gives the formatter everything it needs: the test identity (`TestResult.Name`, `TestResult.Package`), the execution-specific data (`Execution.Status`, `Execution.Output`, `Execution.Elapsed`), and the iteration index/total count for `#NN` labeling.

The formatter should be entry-driven, not test-driven. The current formatter's `classifyTest` shape returns at most one issue for a unique test name; after this change, classification/grouping must be able to return multiple package issues for the same test (e.g. two failed executions of `TestFoo`). Build indexes from `Summary.Failures`, `Summary.Skipped`, and `Summary.SlowTests` keyed by package/test for grouping, but render the per-execution entries themselves.

### Decision 6: Watch-mode rerun behavior unchanged

**Choice:** The package-level `start` handler (collector.go:135-169) that deletes `TestResults` for a restarted package continues to work as today. It creates fresh `TestResult` instances with fresh non-empty `Executions` slices when new test-level `run` events arrive. The reset path must also clear all package-local failure metadata, including `PanicTestKey`, so a panic source from a previous watch cycle cannot affect interrupted-test output handling in the new cycle.

**Rationale:** Watch-mode reruns are conceptually a fresh run of the package. The user expects to see current results, not accumulated history from prior watch cycles. This is semantically different from `-count=N`, which is a deliberate multi-iteration run.

## Risks / Trade-offs

- **[Breaking change to Summary types]** Changing `Summary.Failures`, `Summary.Skipped`, and `Summary.SlowTests` from `[]*TestResult` to structured per-execution entries (e.g. `[]*TestExecutionEntry`) will break any external consumers of the `format` package. → Mitigation: This is an internal package with no known external consumers. All call sites are within the tang codebase and will be updated together.

- **[TestResult accessor method naming]** `TestResult.Status()` conflicts with the current `Status` field. Go doesn't allow a method and field with the same name. → Mitigation: Remove the field (it moves to `TestExecution`), so the method name is available. Callers that previously wrote `tr.Status` will write `tr.Status()` (adding parens). The compiler catches every missed site.

- **[Output accumulation in simple mode]** In non-TTY mode, `output/simple.go` reads `tr.Output` and `tr.SummaryLine` on `"fail"` events. With the refactor, these are on `tr.Latest()`. If the `"fail"` event handler runs before the collector processes the event (race), `Latest()` might not yet be updated. → Mitigation: The main event loop and `SimpleOutput.ProcessEvents` both call `collector.Push(event)` before rendering. The collector is synchronous, so `Latest()` is always up-to-date by the time the renderer reads it. Verify in implementation.

- **[Memory for large -count values]** With `-count=10000`, each test accumulates 10000 `TestExecution` records in memory. Each execution holds an output slice. → Mitigation: This is a niche use case. The output per-execution is typically small (a few lines for failures, empty for passes). For `-count=10000` with 1000 tests, memory overhead is bounded (10M execution structs at ~100 bytes each = ~1GB without output). Acceptable for now; could add a cap later if needed.

- **[Test fixture churn]** Many test files construct `TestResult` directly. The struct change will cause compile errors across ~6 test files. → Mitigation: Update `internal/testutil/builders.go` first (the shared fixture builder), which automatically fixes most call sites. Remaining direct constructions are updated file-by-file.
