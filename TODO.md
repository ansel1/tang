TODO
----
- [x] detect and report panics
- [ ] Play with the UX: make the tui feed more like the summary format
- [ ] add a `-v` flag.  In both tui and notty, outputs something closer to the original 
      test verbose output, including all tests
- [ ] Consider leaving failed (and skipped and slow?) tests in the tui

**Bugs**
- [ ] with `-count=2`, the package summary will show 1, but the total summary will show 2.  I think
  I'd prefer if package count showed 2 as well, and generally, just treat duplicate runs of the same
  test as seperate runs.  Just print out the result for each run, and keep incrementing the counts
  and elapsed times.  So with `-count 100`, it would show 100 runs.

**Browse**
- [ ] add browse command: must pass a file, and opens a tui which lets user navigate to any test run, and view the output
- [ ] add option to locate specific test in file (maybe with regex?) and dump it

**Run**
- [ ] add a subcommand which runs `go test`, as an alternative to piping output, e.g.:

        tang test ./... -v

**tui**
- [ ] 

**optimizations**
- [ ] The replay reader parses each line.  The same line gets parsed again later.  So we're double parsing.  Would be more efficient to implement the replay logic later in the data pipeline.  Also, the replay reader reads all the lines into memory.
- [ ] If there's a line in the output which is json, but not a test/build event, I'm not sure what we'll do
- [ ] need to rethink the notty mode
- [ ] ctrl-c doesn't work if the tui hasn't started yet
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
