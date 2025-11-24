Features
========

Multiple test run detection
---------------------------

The input stream might contain multiple, independent test runs.  For example, if the utility was invoked like this:

    ./build.sh | tang

...and build.sh looked like this:

    echo Testing component 1
    cd component1; && go test -json ./...
    echo Testing component 2
    cd component2; && go test -json ./...

...then the input stream to tang would have json test output from two separate runs, with some additional, non-test output in between.

Go test json output is organized by go package.  There is a "Package" attribute on each test output line, and test output can be grouped
by these package values.  During a single test run which is testing multiple packages concurrently, you will see lines denoting that
particular packages have started, you'll see lines about tests in that package starting and ending, and then you'll see lines denoting
that particular packages have finished.  For example:

      {"Time":"2025-11-12T20:45:29.148037-06:00","Action":"start","Package":"github.com/ansel1/tang/testdata"}
      {"Time":"2025-11-12T20:45:29.148113-06:00","Action":"start","Package":"github.com/ansel1/tang/testdata/groups"}
      {"Time":"2025-11-12T20:45:29.148894-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata/groups","Output":"?   \tgithub.com/ansel1/tang/testdata/groups\t[no test files]\n"}
      {"Time":"2025-11-12T20:45:29.148975-06:00","Action":"skip","Package":"github.com/ansel1/tang/testdata/groups","Elapsed":0.001}
      {"Time":"2025-11-12T20:45:29.14917-06:00","Action":"start","Package":"github.com/ansel1/tang/testdata/widgets"}
      {"Time":"2025-11-12T20:45:29.149195-06:00","Action":"run","Package":"github.com/ansel1/tang/testdata/widgets","Test":"TestPass"}
      {"Time":"2025-11-12T20:45:29.149202-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata/widgets","Test":"TestPass","Output":"=== RUN   TestPass\n"}
      {"Time":"2025-11-12T20:45:29.14921-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata/widgets","Test":"TestPass","Output":"    widgets_test.go:6: passes\n"}
      {"Time":"2025-11-12T20:45:29.149222-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata/widgets","Test":"TestPass","Output":"--- PASS: TestPass (0.00s)\n"}
      {"Time":"2025-11-12T20:45:29.149225-06:00","Action":"pass","Package":"github.com/ansel1/tang/testdata/widgets","Test":"TestPass","Elapsed":0}
      {"Time":"2025-11-12T20:45:29.14923-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata/widgets","Output":"PASS\n"}
      {"Time":"2025-11-12T20:45:29.149232-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata/widgets","Output":"ok  \tgithub.com/ansel1/tang/testdata/widgets\t(cached)\n"}
      {"Time":"2025-11-12T20:45:29.149241-06:00","Action":"pass","Package":"github.com/ansel1/tang/testdata/widgets","Elapsed":0}
      ...
      {"Time":"2025-11-12T20:45:31.464036-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata","Output":"FAIL\n"}
      {"Time":"2025-11-12T20:45:31.465406-06:00","Action":"output","Package":"github.com/ansel1/tang/testdata","Output":"FAIL\tgithub.com/ansel1/tang/testdata\t2.317s\n"}
      {"Time":"2025-11-12T20:45:31.465437-06:00","Action":"fail","Package":"github.com/ansel1/tang/testdata","Elapsed":2.317}

Here, there are 3 packages running concurrently:

- github.com/ansel1/tang/testdata
- github.com/ansel1/tang/testdata/groups
- github.com/ansel1/tang/testdata/widgets

Each has a start line, and a line that indicates the end of the package.  `testdata` ends with a `fail` line, `groups` ends with a `skip` line (because
that package had no tests), and `widgets` ends with a `pass` line.

Packages are tested concurrently, so the lines associated with each package are interleaved.

In the build.sh example above, instead of thinking of the output as two discrete test runs, we could think if it as many 
 *package* test runs.  The fact that some packages were tests by the first command and some by the second command doesn't really
 matter.  All those packages could just as easily have been run all together in a single test run.



The expected behavior of tang should be:

- if the tui is enabled, activate the tui output when the first test run starts, and finish the tui output
  when the first run ends.  Then tang should continue forwarding any non-test input lines directly to the output (as tang
  was doing before the first run started).  When the second test run starts, start a new tui output for it.  Generally, whenever a 
  test run ends, stop the tui output and resume simple output forwarding, then start a new tui session when the next run starts.
- If using the -outfile option, forward all output to that file as usual.
- If using the -jsonfile option, forward all json lines from all test runs to that file as usual.

junit xml output
----------------

This feature adds a new flag, `-junitfile <filename>`.  If specified, the test results should be converted to junit xml
format and saved to the specified file.

I'm not sure what would be ideal behavior if there are multiple, discrete go test runs in the input stream.  Maybe all the results from
all runs should be summarized in a single junit xml?  Please suggest something.





TODO
----

- [x] -1 exit code if tests fail
- [x] -f <filename>: read input from file rather than stdin, and just display a summary of the results
- [x] -replay: when used with -f, replay events with pauses to simulate original test run
- [x] -rate: when used with -replay, set rate to replay.  Defaults to 1 (original speed), 0.5 = double speed, 0 = no pause
- [x] pass any lines which are not `go test -json` output directly to the terminal
- [ ] if the output stream contains multiple, seperate test runs (like the output of a make target that calls `go test` twice)
      then detect the start and end of each run.  At the end of a run, print the summary to the terminal.  Stream any additional 
      non-test ouput to the terminal.  When the next test run start is detected, restart the TUI for the duration of the next run.  Etc.
- [ ] Keep printing output after tests complete until pipe is empty
- [ ] detect and report panics
- [ ] -slow-threshold: set slow test threshold
- [x] -notty: don't use the tui, just stream results to terminal, then print the summary
- [x] -help
- [ ] LICENSE
- [x] if output lines have escape sequences, they bleed into the tui.  Need to reset terminal state after test output
- [x] elide output lines from all finished tests (even failed tests)
- [ ] add a slow icon on slow test lines, next to the test's elapsed time
- [ ] try a spinner column on the left for the status of each package and test.  Use the colors to indicate the status. When a package is finished, replace the spinner with a checkmark or xmark.
- [ ] when in replay mode using a sped up rate, show the elapsed time of the original test run next to the elapsed time of the replayed run.

**Summary**
- [ ] final summary, including failed tests with last 20 lines of output, slow tests, and skipped tests
- [ ] if quitting the tui with ctrl-c, print the summary to the terminal
- [ ] if just passing file, and not replaying, don't use tui, just print summary

**Browse**
- [ ] add browse command: must pass a file, and opens a tui which lets user navigate to any test run, and view the output
- [ ] add option to locate specific test in file (maybe with regex?) and dump it

**junit**
- [ ] -junitfile <filename>: write junit xml to the specified file