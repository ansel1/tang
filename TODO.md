TODO
----

- [x] -1 exit code if tests fail
- [x] -f <filename>: read input from file rather than stdin, and just display a summary of the results
- [x] -replay: when used with -f, replay events with pauses to simulate original test run
- [x] -rate: when used with -replay, set rate to replay.  Defaults to 1 (original speed), 0.5 = double speed, 0 = no pause
- [x] pass any lines which are not `go test -json` output directly to the terminal
- [x] if the output stream contains multiple, seperate test runs (like the output of a make target that calls `go test` twice)
      then detect the start and end of each run.  At the end of a run, print the summary to the terminal.  Stream any additional 
      non-test ouput to the terminal.  When the next test run start is detected, restart the TUI for the duration of the next run.  Etc.
- [x] Keep printing output after tests complete until pipe is empty
- [ ] detect and report panics
- [x] -slow-threshold: set slow test threshold
- [x] -notty: don't use the tui, just stream results to terminal, then print the summary
- [x] -help
- [x] LICENSE
- [x] if output lines have escape sequences, they bleed into the tui.  Need to reset terminal state after test output
- [x] elide output lines from all finished tests (even failed tests)
- [x] try a spinner column on the left for the status of each package and test.  Use the colors to indicate the status. When a package is finished, replace the spinner with a checkmark or xmark.
- [x] when in replay mode using a sped up rate, show the elapsed time of the original test run next to the elapsed time of the replayed run.
- [x] add an elapsed time to the tui summary line

**Bugs**
- [ ] with `-count=2`, the package summary will show 1, but the total summary will show 2.  I think
  I'd prefer if package count showed 2 as well, and generally, just treat duplicate runs of the same
  test as seperate runs.  Just print out the result for each run, and keep incrementing the counts
  and elapsed times.  So with `-count 100`, it would show 100 runs.

**Summary**


**Browse**
- [ ] add browse command: must pass a file, and opens a tui which lets user navigate to any test run, and view the output
- [ ] add option to locate specific test in file (maybe with regex?) and dump it

**Run**
- [ ] add a subcommand which runs `go test`, as an alternative to piping output, e.g.:

        tang test ./... -v

**tui**
- [ ] 

**optimizations**
- [x] update to bubbletea v2
- [ ] The replay reader parses each line.  The same line gets parsed again later.  So we're double parsing.  Would be more efficient to implement the replay logic later in the data pipeline.  Also, the replay reader reads all the lines into memory.
- [x] in replay mode, it looks like printing a lot of output is *much* slower than the pace at which the logs are read from the input stream.  A test might appear to take 5 minutes to complete, but actually took .04 seconds.  Let's experiment with draining the event channel in batches between display frames.  Not sure if that's going to make the UI too jumpy...
- [ ] If there's a line in the output which is json, but not a test/build event, I'm not sure what we'll do
- [x] Don't bother with the gutter icon for paused tests
- [ ] need to rethink the notty mode
- [ ] ctrl-c doesn't work if the tui hasn't started yet
- [x] `cat simple.out | tang` doesn't show anything.  
- [ ] handle "action":"bench" events 
- [ ] in Collector.handleBuildEvent(), why not just start a new run if needed?
  // The Action field is one of a fixed set of action descriptions:
//
//	start  - the test binary is about to be executed
//	run    - the test has started running
//	pause  - the test has been paused
//	cont   - the test has continued running
//	pass   - the test passed
//	bench  - the benchmark printed log output but did not fail
//	fail   - the test or benchmark failed
//	output - the test printed output
//	skip   - the test was skipped or the package contained no tests
