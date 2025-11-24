

```
OVERALL RESULTS
  Total Tests:    945
  ✓ Passed:       940 (99.5%)
  ✗ Failed:         3 (0.3%)
  ⊘ Skipped:        2 (0.2%)

  Total Time:     00:04:15.221
  Packages:       234

FAILURES (3)
  ✗ github.com/user/project/pkg/replay
    TestConcurrentRuns
      assertion failed: expected 5, got 3
      at integration_test.go:45

  ✗ github.com/user/project/pkg/validator
    TestValidateJSON
      unexpected error: invalid character '}' at position 42
      at validator_test.go:89

    TestValidateSchema
      schema validation failed: missing required field 'type'
      at validator_test.go:124

SLOWEST TESTS (>10s)
  1. TestEndToEndLargeProject      (tests/e2e)            00:02:35.221
  2. TestEndToEndWithErrors         (tests/e2e)            00:01:12.445
  3. TestHighVolumeOutput           (tests/stress)         00:01:08.923
  4. TestDatabaseMigration          (tests/integration)    00:00:35.234
  5. TestConcurrentStress           (tests/benchmark)      00:00:28.921

PACKAGE STATISTICS
  Fastest:  pkg/utils                                      00:00:00.234
  Slowest:  tests/e2e                                      00:03:47.666
  Most Tests: tests/integration (45 tests)
  ```