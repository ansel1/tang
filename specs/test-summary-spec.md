When tests finish, print a final summary of the test run.

The final summary should look like the following:

```
  ok      gitlab.protectv.local/ncryptify/clerk.git       26.303s                                              ✓ 206  ✗ 0  ∅ 0  26.3s
  ok      gitlab.protectv.local/ncryptify/clerk.git/alarm 2.411s                                               ✓  19  ✗ 0  ∅ 0   2.4s
  ?       gitlab.protectv.local/ncryptify/clerk.git/alarm/alarmtest       [no test files]                      ✓   0  ✗ 0  ∅ 0   0.0s
  ?       gitlab.protectv.local/ncryptify/clerk.git/cmd/clerk     [no test files]                              ✓   0  ✗ 0  ∅ 0   0.0s
  ok      gitlab.protectv.local/ncryptify/clerk.git/fips  0.693s                                               ✓   1  ✗ 0  ∅ 0   0.7s
  ok      gitlab.protectv.local/ncryptify/clerk.git/httptransport 1.944s                                       ✓  20  ✗ 0  ∅ 0   1.9s
-------------------------------------------------------------------------------------------------------------------------------------

OVERALL RESULTS
  Total Tests:    945
  ✓ Passed:       940 (99.5%)
  ✗ Failed:         3 (0.3%)
  ⊘ Skipped:        2 (0.2%)

  Total Time:     00:04:15.221
  Packages:       234

FAILURES (3)
  github.com/user/project/pkg/replay
    TestConcurrentRuns
      assertion failed: expected 5, got 3
      at integration_test.go:45

  github.com/user/project/pkg/validator
    TestValidateJSON
      unexpected error: invalid character '}' at position 42
      at validator_test.go:89

    TestValidateSchema
      schema validation failed: missing required field 'type'
      at validator_test.go:124

SKIPPED (2)
  github.com/user/project/pkg/alarm
    TestAlarm
      skipped because alarm is disabled

SLOW TESTS (>10s)
  TestEndToEndLargeProject       (tests/e2e)            00:02:35.221
  TestEndToEndWithErrors         (tests/e2e)            00:01:12.445
  TestHighVolumeOutput           (tests/stress)         00:01:08.923
  TestDatabaseMigration          (tests/integration)    00:00:35.234
  TestConcurrentStress           (tests/benchmark)      00:00:28.921

PACKAGE STATISTICS
  Fastest:  pkg/utils                                      00:00:00.234
  Slowest:  tests/e2e                                      00:03:47.666
  Most Tests: tests/integration (45 tests)
  ```

The first section shows the summary of each package, with the following format:

```
  <final package output line>   <passed> <failed> <skipped> <elapsed>
```

The next section shows the overall results, with the following format:

```
  Total Tests:      <total tests>
  ✓ Passed:         <passed tests> (<passed percentage>)
  ✗ Failed:         <failed tests> (<failed percentage>)
  ⊘ Skipped:        <skipped tests> (<skipped percentage>)

  Total Time:       <total time>
  Packages:         <packages>
```

The next section shows the failures, with the following format:

```
  <package name>
    <test name>
      <up to 10 lines of failure output>
```

The next section shows the skipped tests, with the following format:

```
  <package name>
    <test name>
      <up to 3 lines of skipped output>
```
  
The next section shows the slow tests, sorted by elapsed time descending, with the following format:

```
SLOW TESTS (<threshold>)
  <test name> (<package name>) <elapsed time>
```

The next section shows the package statistics, with the following format:

```
PACKAGE STATISTICS
  Fastest:  <package name>                                      <elapsed time>
  Slowest:  <package name>                                      <elapsed time>
  Most Tests: <package name> (<tests>)
```

The summary should be printed to the terminal when the number of concurrently running packages drops to 0.  It should also be printed if the users terminates the program (e.g. with ctrl-c).

If the -f flag is specified without the -replay flag, the tui and feedback should be disabled, and the summary should be printed immediately.