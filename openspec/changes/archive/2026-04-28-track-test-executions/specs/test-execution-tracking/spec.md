## ADDED Requirements

### Requirement: Per-execution result tracking
When `go test -count=N` causes a test to execute multiple times within a single run, the system SHALL record each execution as a separate record with its own status, output, elapsed time, and summary line. The system SHALL NOT overwrite earlier execution records when later executions of the same test begin.

#### Scenario: Test executes twice and both pass
- **WHEN** a test runs twice due to `-count=2` and passes both times
- **THEN** the test result contains 2 execution records, both with status Passed
- **THEN** the package and run passed counts equal 2

#### Scenario: Test fails first then passes second
- **WHEN** a test fails on the first execution and passes on the second due to `-count=2`
- **THEN** the test result contains 2 execution records: first with status Failed, second with status Passed
- **THEN** the package failed count equals 1 and passed count equals 1
- **THEN** the first execution's failure output is preserved

#### Scenario: Test fails on both executions
- **WHEN** a test fails on both executions due to `-count=2`
- **THEN** the test result contains 2 execution records, both with status Failed
- **THEN** the package failed count equals 2
- **THEN** both executions' failure output is preserved independently

### Requirement: Running count remains non-negative
The system SHALL correctly track the number of currently running tests when tests execute multiple times. Each new execution of a previously-completed test SHALL increment the running counter, and each terminal event (pass/fail/skip) SHALL decrement it, maintaining a balanced count.

#### Scenario: Running count with -count=2
- **WHEN** a test executes twice due to `-count=2`
- **THEN** the running count increments to 1 when the first execution starts
- **THEN** the running count returns to 0 when the first execution completes
- **THEN** the running count increments to 1 when the second execution starts
- **THEN** the running count returns to 0 when the second execution completes
- **THEN** the running count is never negative

#### Scenario: Running count with -count=N and parallel tests
- **WHEN** multiple tests execute with `-count=N` and `t.Parallel()` is used
- **THEN** the running count reflects the actual number of concurrently executing tests at all times
- **THEN** the running count is never negative

### Requirement: Summary includes all failed executions
The summary output SHALL list every failed execution across all iterations, not just the latest execution of each unique test name. Each failed execution SHALL include its full failure output.

#### Scenario: First execution fails, second passes
- **WHEN** a test fails on the first execution and passes on the second
- **THEN** the summary failure section includes the first execution's failure with its output
- **THEN** the summary's total failed count reflects the failure

#### Scenario: Multiple executions of the same test fail
- **WHEN** a test fails on both execution 1 and execution 2
- **THEN** the summary failure section includes both failures with their respective outputs
- **THEN** the two failure entries are rendered as separate failure blocks

### Requirement: Summary optional detail sections are execution-specific
When skipped-test or slow-test detail sections are enabled, the summary SHALL treat those details as execution-specific entries rather than collapsing to the latest execution of each unique test name.

#### Scenario: One execution skipped and one execution passes
- **WHEN** a test is skipped on one execution and passes on another execution
- **THEN** the skipped-test detail section includes the skipped execution when skipped details are enabled

#### Scenario: Multiple slow executions of the same test
- **WHEN** two executions of the same test exceed the slow-test threshold
- **THEN** the slow-test detail section includes both slow executions when slow details are enabled

### Requirement: Multi-execution iteration labeling
When a test has only one execution, the system SHALL display the test name without any iteration suffix. When a test has two or more executions, the system SHALL label every execution (including the first) with a `#NN` suffix (e.g. `TestFoo#01`, `TestFoo#02`) so each execution is unambiguously identifiable. The suffix is always written with at least two digits, zero-padded.

#### Scenario: Single execution test naming
- **WHEN** a test executes only once (no `-count` or `-count=1`)
- **THEN** the test name is displayed without any iteration suffix (e.g. `TestFoo`)

#### Scenario: Multi-execution test naming
- **WHEN** a test executes 3 times due to `-count=3`
- **THEN** the first execution is labeled `TestFoo#01`
- **THEN** the second execution is labeled `TestFoo#02`
- **THEN** the third execution is labeled `TestFoo#03`

#### Scenario: Multi-execution subtest naming
- **WHEN** a subtest `TestFoo/sub` executes twice due to `-count=2`
- **THEN** the first execution is labeled `TestFoo#01/sub`
- **THEN** the second execution is labeled `TestFoo#02/sub`

#### Scenario: Multi-execution nested subtest naming
- **WHEN** a nested subtest `TestFoo/sub/nested` executes twice due to `-count=2`
- **THEN** the first execution is labeled `TestFoo#01/sub/nested`
- **THEN** the second execution is labeled `TestFoo#02/sub/nested`

### Requirement: JUnit XML emits one testcase per execution
The JUnit XML output SHALL emit one `<testcase>` element per execution of each test. When a test has two or more executions, the `name` attribute of every emitted `<testcase>` for that test SHALL use the `#NN` suffix convention. When a test has only one execution, the `name` attribute SHALL be the plain test name with no suffix.

#### Scenario: JUnit with -count=2 where first fails
- **WHEN** a test executes twice, failing first and passing second, and JUnit output is enabled
- **THEN** the XML contains two `<testcase>` elements for that test
- **THEN** the first `<testcase>` has `name="TestFoo#01"` with a `<failure>` child element
- **THEN** the first `<failure>` content contains that execution's failure output as line-preserving text
- **THEN** the second `<testcase>` has `name="TestFoo#02"` without a `<failure>` child element

#### Scenario: JUnit with single execution
- **WHEN** a test executes once and JUnit output is enabled
- **THEN** the XML contains one `<testcase>` with the plain test name (no `#NN` suffix)

### Requirement: TUI displays latest execution state
The live TUI SHALL display the most recent execution's state for each unique test name. Per-test rows SHALL reflect the latest execution's status, output, and elapsed time. Package and run-level counters SHALL reflect cumulative counts across all executions.

#### Scenario: TUI during second execution
- **WHEN** a test's first execution has completed and the second execution is running
- **THEN** the TUI test row shows the second execution as running
- **THEN** the package running count reflects the second execution

#### Scenario: TUI package status with mixed results
- **WHEN** a test fails on execution 1 and passes on execution 2 within a package
- **THEN** the package may remain Running while later executions are still in progress
- **THEN** after the package-level terminal event arrives, the final package status is Failed (any failure = failed package)
- **THEN** the package counts show both the failure and the pass

### Requirement: Watch-mode reruns remain independent
Watch-mode package reruns (triggered by a package-level `start` event for an already-completed package) SHALL continue to reset test results for that package. This behavior is distinct from `-count=N` iteration tracking and SHALL NOT be changed.

#### Scenario: Watch-mode rerun clears previous results
- **WHEN** a package completes and then receives a new `start` event (watch-mode rerun)
- **THEN** all test results for that package are cleared
- **THEN** package-local failure metadata from the previous run, including panic-source tracking, is cleared
- **THEN** the package begins collecting fresh results as if it were the first run

### Requirement: Simple (non-TTY) mode failure output per execution
In non-TTY mode, when a test's `"fail"` event arrives, the system SHALL emit the failure output for that specific execution. Earlier executions' outputs SHALL have already been emitted when their respective `"fail"` events arrived, and SHALL NOT be lost or overwritten.

#### Scenario: Non-TTY output with -count=2 where both fail
- **WHEN** a test fails on both executions in non-TTY mode
- **THEN** the first execution's failure output is emitted when its `"fail"` event arrives
- **THEN** the second execution's failure output is emitted when its `"fail"` event arrives
- **THEN** both failure outputs appear in the stream
