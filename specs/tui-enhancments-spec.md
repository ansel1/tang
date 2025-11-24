Let's make the tui more sophisticated.  The purpose of the tui output is to summarize the test run progress, and show constant feedback
to the user about which packages and tests are currently running, and how long they have been running, so the user doesn't think the tests have stalled.

It would also be nice to summarize the test log output for running tests, but elide some of it so we can show feedback about multiple, 
simultaneous tests on a single screen.

It would also be nice to group the output by package...so I guess when a package starts, it's starting output line should be pinned, and then all
test output for that package should appear underneath it.

Here's what I'm thinking:

- Group the output on the screen by package and then by test.
- Each package and each test should be summarized in a single line.
- The summary line for a package while it is running should be the name of the package, left aligned, and a summary of passed and failed tests and elapsed time, right aligned.  For example:

        github.com/ansel1/tang/testdata                                                                        4 passed, 3 failed 0.1s

- The summary line for a package that has finished should be the last Output value for that package.  For example, for these test lines:

        {"Time":"2024-10-27T23:48:48.064013-05:00","Action":"start","Package":"gitlab.protectv.local/ncryptify/minerva.git/cmd/minerva"}
        {"Time":"2024-10-27T23:48:48.072235-05:00","Action":"output","Package":"gitlab.protectv.local/ncryptify/minerva.git/cmd/minerva","Output":"\tgitlab.protectv.local/ncryptify/minerva.git/cmd/minerva\t\tcoverage: 0.0% of statements\n"}
        {"Time":"2024-10-27T23:48:48.084766-05:00","Action":"pass","Package":"gitlab.protectv.local/ncryptify/minerva.git/cmd/minerva","Elapsed":0.021}

  ...the summary line should be:

        gitlab.protectv.local/ncryptify/minerva.git/cmd/minerva         coverage: 0.0% of statements                     4 passed 0.2s

- The summary line for a test which is running, paused or resumed should be the last Output value for that test which started with "===", left aligned, and the elapsed time right aligned.
  For example:

        === RUN   Test_NewAesKey                                                                                                  0.2s

- The summary line for a test which has finished should be the last Output value for that test, for example:

        --- PASS: Test_NewAesKey (0.06s)                                                                                          0.0s

- Tests which have passed or have been skipped may have their summary lines removed from the output, so the tui bringing more attention
  to tests which are running, paused, or failed.
- For the other output lines for a test, like log output lines, show up to the most recent 6 lines of that output under the test summary line.
- At the bottom of the output, pin the overall summary line

Layout and visual hierarchy
---------------------------

The display should organize test output into a clear visual hierarchy:

**Package Headers and Grouping:**
- Each package appears as a pinned header line that stays visible as content scrolls
- All test output for a package appears directly below that package's header
- Packages are displayed in the order they start (chronological appearance order)

**Test Organization Within Packages:**
- Tests within a package are indented one level (2 spaces recommended) under the package header
- Tests are listed in chronological order (in the order they start within that package)
- When a test is running, its most recent "===" output line appears as the test summary
- When a test finishes, its final output line (typically "--- PASS" or "--- FAIL") appears as the test summary

**Log Output Under Tests:**
- Test output lines (log output, assertion messages, etc.) appear indented further (4 spaces recommended) under the test summary
- Log lines are displayed in the order they were produced (oldest to newest)
- A maximum of 6 log output lines appear under each test (see "Output Filtering and Elision Rules" for details)

**Terminal Width and Truncation:**
- Package names and test output should not wrap or break across multiple lines
- If a line would exceed the terminal width, it should be truncated at the right edge
- Right-aligned elapsed time information should be preserved even if package/test names are truncated
- The TUI should not assume a minimum terminal width, but should display reasonably at widths of 80 characters or greater
- When the terminal width is less than can accommodate the full layout, prioritize showing the summary line intact and truncate log output lines first
- When the terminal is resized, the display should adapt to the new width

**Alignment Strategy:**
- Package headers use right-aligned elapsed time/status info separated from the package name by padding
- Test summary lines use right-aligned elapsed time separated from the test output by padding
- Right alignment should attempt to position information at the same column across packages when possible, but will vary based on content length
- Use at least 2 spaces of padding between left-aligned content and right-aligned content

Real-time update behavior
-------------------------

The display should provide constant, responsive feedback about test progress:

**Update Frequency:**
- The display should refresh each time a new event is received (test start, test completion, output line, etc.)
- Between events, the elapsed time display for running tests and packages should update at least once per second
- Timer-based updates (for elapsed time) should not cause unnecessary redraws if no time change is visible at the current precision

**Scrolling and Viewport:**
- When new content is added that would exceed the terminal height, scroll the display upward to show the newest content
- The overall summary line (pinned at the bottom) should always be visible
- Scrolling should be smooth and follow the "tail" of test output (always showing the most recent activity)
- Package headers should remain visible even as their test output scrolls

**Package Lifecycle on Screen:**
- When a package starts, its header line immediately appears on screen
- While a package is running, its header is updated in place with current pass/fail counts and elapsed time
- When a package finishes, the header is updated with final results but remains on screen (not removed)
- Finished packages remain visible and can be scrolled to, allowing users to review results

**Test Lifecycle on Screen:**
- When a test starts within a package, an indented test summary line appears under the package header
- As the test runs, the test summary line is updated with current elapsed time and latest "===" output
- When the test produces log output, those lines appear indented under the test summary (up to 6 lines, see filtering rules)
- When a test finishes, its summary line is updated with final output and elapsed time
- Test summary lines remain on screen after completion (not removed), unless explicitly filtered

**Elapsed Time Display:**
- Elapsed times should be displayed in the format "X.Xs" where X is a number (e.g., "0.1s", "1.2s", "10.5s")
- For durations less than 0.1 seconds, display as "0.0s"
- For durations greater than 60 seconds, display the format in minutes (e.g., "1.1m")
- The elapsed time for a running test or package is computed from the event timestamp to the current time
- The elapsed time for a finished test or package is the elapsed time from the event (provided in the event data)
- Elapsed time should update frequently for running tests (at least once per second) to show progress

Output filtering and elision rules
----------------------------------

To fit multiple concurrent tests on a single screen while preserving important information:

**Log Output Line Limits:**
- For each test, display up to 6 most recent log output lines (not including the test summary line itself)
- "Log output lines" are output lines that do not contain the test status summary (lines not starting with "---" or "===")
- If a test produces more than 6 output lines before completion, only the most recent 6 are displayed
- When a new output line arrives for a test that already has 6 displayed, the oldest displayed line is removed and the new line is added at the bottom

**Passed and Skipped Test Handling:**
- Passed tests retain their summary line on screen showing "--- PASS: TestName (duration)"
- Skipped tests retain their summary line on screen showing their skip summary
- Log output lines for passed and skipped tests are still displayed (up to 6 lines per the above rule)
- Passed and skipped tests are NOT automatically removed from the display, as the TUI shows all activity (with "Show everything, scroll as needed" philosophy)
- If configuration options are added later to hide passed/skipped tests, they should remove all associated output lines

**Long Line Handling:**
- Output lines longer than the terminal width should be truncated at the right edge
- Text should not wrap to multiple lines
- The truncation should preserve the beginning of the line for readability

**Empty Test Output:**
- If a test completes without any log output lines, only the test summary line is displayed (no blank space is reserved)
- If a test has some output lines, but fewer than 6, display only the lines that exist

**Coverage and Special Output:**
- Coverage information that appears as package-level output (e.g., "coverage: 95.2% of statements") should appear in the package header if it's the last output value received for that package
- Build errors, panic stacks, or compilation output should be displayed as raw output lines at their original indentation level
- Non-test output (lines that are not valid JSON test events) should be displayed at the top level without indentation, as separate sections

Overall display structure
------------------------

**Screen Layout (Top to Bottom):**
1. Non-test output lines (if any) in the order received
2. Grouped test packages with their headers, tests, and log output (in chronological order of package start)
3. Running/incomplete packages show current statistics in their header
4. Finished packages show final status in their header
5. Overall summary line (pinned at bottom) - always visible, not scrolled off

**Overall Summary Line Format:**
- Positioned at the absolute bottom of the display
- Format: `<count> passed, <count> failed, <count> skipped, <count> running, total <count>`
- Example: `42 passed, 3 failed, 5 skipped, 2 running, total 52`
- This line is continuously updated as test results arrive
- It provides a quick overview of the entire test run status
- The line should be visually distinct (e.g., using a separator line above it, different color, or fixed position)

**Status Indicators and Formatting:**
- Running tests/packages: Display their elapsed time and current state (test name for tests, pass/fail counts for packages)
- Passed tests/packages: Display final elapsed time and "PASS" indicator
- Failed tests/packages: Display final elapsed time and "FAIL" indicator
- Skipped tests/packages: Display final elapsed time and "SKIP" indicator
- Color coding (if supported): green for pass, red for fail, yellow for skip, white/neutral for running

**Visual Separators:**
- No blank lines.  Maximize screen real estate.
- Log output lines have consistent indentation to show their nesting level

Edge cases and special scenarios
--------------------------------

**Long Package Names:**
- Package names may be very long (80+ characters)
- When displayed with right-aligned elapsed time, the package name should be truncated at the right edge rather than wrapping
- The full package name and time should both be preserved in the data; only the display truncates

**Packages with No Tests:**
- Packages that complete with no test executions (e.g., packages with no test files) still have a summary line
- The summary line should show the package's output (e.g., "[no test files]") and status (skipped)

**Tests with No Output:**
- Tests that complete without producing any log output should display only the test summary line
- No blank space should be allocated for missing log output lines

**Excessive Output:**
- If a single test produces hundreds or thousands of output lines, only the most recent 6 are shown (per filtering rules)
- This prevents the display from becoming unresponsive or memory-intensive

**Build Errors and Non-Test Output:**
- Build errors, compiler errors, or panic stacks that prevent test execution should be displayed at the top level
- These should appear in the order received, before the first package header
- Non-JSON test output lines should be displayed with minimal indentation
- If a build fails for a package, that package may never appear in the JSON output; the build error output should still be visible

**Terminal Resize Events:**
- When the terminal is resized, the display should adapt to the new width
- Content should be re-rendered to fit the new dimensions
- Scrolling position should be maintained when possible (users should still see the same test results, just reformatted)

**Parallel Test Execution Display:**
- Multiple tests running in the same package should each have their own summary line
- Tests are listed in the order they started (chronological within a package)
- Elapsed times for each running test should update independently
- The package-level pass/fail counts should sum across all tests in that package

Spinner indicators
------------------

**Purpose:**
Spinners provide visual feedback that tests and packages are actively running, helping users understand that the system hasn't stalled.

**Spinner Placement:**
- Package lines: Display a spinner on the left side of the package name while the package is running
- Test lines: Display a spinner on the left side of the test summary while the test is running
- Summary line: Display a spinner on the left side of the summary text while tests are running (already implemented)

**Spinner Behavior:**
- **Running state**: Spinner is visible and animating
- **Finished state**: Spinner is hidden (no character displayed in its place)
- **Package with failures**: When a package finishes with failures, the spinner color changes to red before being hidden

**Spacing:**
- One space of padding between the spinner and the text it precedes
- The spinner character itself takes one character width
- Total overhead: 2 characters (spinner + space) when visible

**Visual Format:**
```
<spinner> <text>                                                    <right-aligned-info>
```

When not running:
```
<text>                                                              <right-aligned-info>
```

**Implementation Notes:**
- Use the same spinner component (bubbles/spinner) that's already used in the summary line
- The spinner should animate at the same rate across all lines
- Package spinners should turn red when `pkg.Failed > 0` and `pkg.Status != "running"`
- Test spinners should be neutral colored (match the current summary spinner color)
- When rendering, check the status to determine whether to show the spinner

Display examples and mockups
----------------------------

This section shows ASCII mockups of what the TUI display should look like in various states. These examples illustrate the layout, hierarchy, and formatting described in the specification above.

**Example 1: Multiple Packages with Concurrent Tests Running**

This shows an early stage of a test run where multiple packages are running concurrently, with different numbers of tests in each:

```
github.com/ansel1/tang/internal/parser                           1 passed, 2 running 0.8s
  === RUN   TestParseBasic                                                           0.2s
  === RUN   TestParseComplex                                                         0.6s
    parser_test.go:15: Starting complex parse
    parser_test.go:20: Nested structure parsed successfully
github.com/ansel1/tang/internal/engine                            3 passed, 1 failed 1.2s
  --- PASS: TestNewEngine (0.05s)                                                    0.0s
  --- PASS: TestEventProcessing (0.12s)                                              0.0s
  --- FAIL: TestErrorHandling (0.34s)                                                0.0s
    engine_test.go:89: assertion failed: got nil, want error
    engine_test.go:90: context: processing malformed JSON
github.com/ansel1/tang/tui                                       0 passed, 1 running 0.3s
  === RUN   TestRender                                                               0.3s
    tui_test.go:42: Rendering 1000 lines
    tui_test.go:45: Layout calculated
github.com/ansel1/tang/output                                     0 passed, 0 failed 0.0s
═════════════════════════════════════════════════════════════════════════════════════════
4 passed, 1 failed, 0 skipped, 3 running, total 8
```

**Example 2: Tests with Nested Log Output (Multiple Lines)**

This shows a test that produces multiple log output lines, with the most recent 6 displayed:

```
github.com/myapp/database                                        1 passed, 1 running 2.1s
  === RUN   TestDatabaseMigration                                                    1.8s
    database_test.go:23: Starting migration from v1 to v2
    database_test.go:45: Creating backup
    database_test.go:48: Backup completed in 523ms
    database_test.go:51: Applying migration script
    database_test.go:78: Migration validation passed
    database_test.go:82: Verifying schema changes
═════════════════════════════════════════════════════════════════════════════════════════
1 passed, 0 failed, 0 skipped, 1 running, total 2
```

(Note: Only the 6 most recent log lines are shown. If the test had produced more lines before these, the older lines would be removed from the display.)

**Example 3: Failed Test with Error Output**

This shows a failed test with assertion or error information displayed in the log lines:

```
github.com/myapp/api                                              2 passed, 1 failed 0.6s
  --- PASS: TestListUsers (0.08s)                                                    0.0s
  --- FAIL: TestCreateUser (0.18s)                                                   0.0s
    api_test.go:64: unexpected status code
    api_test.go:65: got: 400, want: 201
    api_test.go:72: request body: {"name": "Alice"}
    api_test.go:73: response body: {"error": "invalid request"}
═════════════════════════════════════════════════════════════════════════════════════════
2 passed, 1 failed, 0 skipped, 0 running, total 3
```

**Example 4: Transition as Package Completes**

This shows a package as it transitions from running to completed.

Before (package running):
```
github.com/myapp/handlers                                        3 passed, 1 running 1.2s
  --- PASS: TestGetHandler (0.08s)                                                   0.0s
  --- PASS: TestPostHandler (0.15s)                                                  0.0s
  --- PASS: TestDeleteHandler (0.09s)                                                0.0s
  === RUN   TestPutHandler                                                           0.8s
═════════════════════════════════════════════════════════════════════════════════════════
4 passed, 0 failed, 0 skipped, 1 running, total 5
```

After (package finished, test completed with pass):
```
github.com/myapp/handlers                                                ok 4 passed 1.3s
  --- PASS: TestGetHandler (0.08s)                                                   0.0s
  --- PASS: TestPostHandler (0.15s)                                                  0.0s
  --- PASS: TestDeleteHandler (0.09s)                                                0.0s
  --- PASS: TestPutHandler (0.87s)                                                   0.0s
═════════════════════════════════════════════════════════════════════════════════════════
5 passed, 0 failed, 0 skipped, 0 running, total 5
```

**Example 5: Long Package Name with Truncation**

This shows how long package names are handled when they exceed terminal width:

Terminal width: 100 characters

```
gitlab.protectv.local/ncryptify/minerva.git/cmd/minerva...        3 passed, 2 failed 2.5s
  --- PASS: TestInitialize (0.12s)                                                   0.0s
  --- FAIL: TestEncryption (0.89s)                                                   0.0s
    minerva_test.go:156: encryption failed: invalid key format
  --- FAIL: TestDecryption (0.67s)                                                   0.0s
    minerva_test.go:201: decryption failed: authentication tag mismatch
═════════════════════════════════════════════════════════════════════════════════════════
3 passed, 2 failed, 0 skipped, 0 running, total 5
```

(The long package name is truncated with "..." to fit within the terminal width while preserving the elapsed time on the right.)

**Example 6: Complex Multi-Package Scenario with Mixed Statuses**

This shows a more realistic scenario with multiple packages in different states:

```
github.com/myapp/utils                                            5 passed, 0 failed 0.4s
  --- PASS: TestStringUtils (0.05s)                                                  0.0s
  --- PASS: TestMathUtils (0.07s)                                                    0.0s
  --- PASS: TestFileUtils (0.08s)                                                    0.0s
  --- PASS: TestJSONUtils (0.06s)                                                    0.0s
  --- PASS: TestTimeUtils (0.11s)                                                    0.0s
github.com/myapp/cache                                           1 passed, 2 running 1.8s
  --- PASS: TestCacheInit (0.12s)                                                    0.0s
  === RUN   TestCacheGet                                                             1.1s
    cache_test.go:34: Cache populated with 1000 items
  === RUN   TestCacheEvict                                                           0.6s
    cache_test.go:78: Testing LRU eviction policy
github.com/myapp/storage                                         0 passed, 0 skipped 0.0s
github.com/myapp/api                                              2 passed, 1 failed 1.5s
  --- PASS: TestAuthMiddleware (0.18s)                                               0.0s
  --- FAIL: TestRateLimiter (0.72s)                                                  0.0s
    api_test.go:142: rate limit threshold exceeded
    api_test.go:143: expected: 100 req/sec, got: 95 req/sec
  === RUN   TestValidation                                                           0.5s
═════════════════════════════════════════════════════════════════════════════════════════
8 passed, 1 failed, 1 skipped, 3 running, total 13
```

**Example 7: Scrolled View (Some Packages Off-Screen)**

When there are more packages or tests than fit on screen, older packages scroll off the top while newer ones appear at the bottom. The summary line remains pinned at the bottom:

```
(...)
github.com/myapp/validators                                       6 passed, 0 failed 0.9s
  --- PASS: TestEmailValidator (0.08s)                                               0.0s
  --- PASS: TestPhoneValidator (0.07s)                                               0.0s
  --- PASS: TestZipValidator (0.06s)                                                 0.0s
  --- PASS: TestURLValidator (0.09s)                                                 0.0s
  --- PASS: TestIPValidator (0.08s)                                                  0.0s
  --- PASS: TestCIDRValidator (0.10s)                                                0.0s

github.com/myapp/server                                          2 passed, 1 running 3.2s
  --- PASS: TestServerStart (0.45s)                                                  0.0s
  === RUN   TestServerShutdown                                                       2.8s
    server_test.go:212: Graceful shutdown initiated
    server_test.go:219: Waiting for active connections to close
═════════════════════════════════════════════════════════════════════════════════════════
47 passed, 3 failed, 2 skipped, 8 running, total 60
```

(The "(...)" indicates that some content has scrolled off-screen above. The summary at the bottom shows the aggregate of all packages, including those not currently visible.)

**Example 8: Test Output Truncation (Long Lines)**

When a test produces output lines longer than the terminal width, they are truncated:

```
github.com/myapp/logging                                         0 passed, 1 running 0.5s
  === RUN   TestLongStackTrace                                                       0.5s
    stack_trace.go:1: goroutine 42 [running]: github.com/myapp/logging.TestLongStackTr...
    stack_trace.go:5: github.com/myapp/internal/processor.(*Processor).Process(0xc00...
    stack_trace.go:9: main.main()

═════════════════════════════════════════════════════════════════════════════════════════
0 passed, 0 failed, 0 skipped, 1 running, total 1
```

(Long lines are truncated at the right edge, preserving the beginning of the line for readability.)