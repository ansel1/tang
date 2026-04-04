TODO
----
- [x] detect and report panics
- [x] Play with the UX: make the live feed more like the summary format
- [x] add a `-v` flag.  In both live and notty, outputs something closer to the original 
      test verbose output, including all tests
- [x] rename `--junitout` to `--junitfile`
- [x] add COLUMNS support if we don't already have it
- [ ] still something wrong with indentation of log lines in the summary.  when running sample, they are indented too far, but not when running henry's tests
- [ ] add a `--no-color` flag
- [ ] when eliding, have RUN and CONT taking higher precendence than PAUSE
- [ ] don't count paused tests in the running count
- [ ] clear up the help string
- [ ] in live mode, try using the icons again in the left gutter for finished packages, so isntead of:

        ⠦ gitlab.protectv.local/ncryptify/solo.git                                                                                                                                                                                 ✓279 ✗0 ∅1 =280 3.4m
            --- PASS: TestGetLoginURI/good_URL_w/javascript-quote,_should_be_escaped (0.00s)                                                                                                                                                       0.0s
            --- PASS: TestGetLoginURI/bad_scheme_(data) (0.00s)                                                                                                                                                                                    0.0s
            --- PASS: TestGetLoginURI/bad_scheme_(file) (0.00s)                                                                                                                                                                                    0.0s
            --- PASS: Test_listAllDomains (2.25s)                                                                                                                                                                                                  2.2s
            === RUN   Test_EnableLogRedirection                                                                                                                                                                                                    0.9s
        ?       gitlab.protectv.local/ncryptify/solo.git/clientmgmt     [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ?       gitlab.protectv.local/ncryptify/solo.git/cmd/mockoidc   [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ?       gitlab.protectv.local/ncryptify/solo.git/cmd/solo       [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ok      gitlab.protectv.local/ncryptify/solo.git/httptransport  95.097s                                                                                                                                                  ✓264 ✗0 ∅1 =265 1.6m
        ?       gitlab.protectv.local/ncryptify/solo.git/models [no test files]                                                                                                                                                  ✓  0 ✗0 ∅0 =  0 0.0s
        ok      gitlab.protectv.local/ncryptify/solo.git/ssl    0.467s                                                                                                                                                           ✓ 10 ✗0 ∅0 = 10 0.5s
        ok      gitlab.protectv.local/ncryptify/solo.git/usermgmt       3.162s

    try:

        ⠦ gitlab.protectv.local/ncryptify/solo.git                                                                                                                                                                                 ✓279 ✗0 ∅1 =280 3.4m
            --- PASS: TestGetLoginURI/good_URL_w/javascript-quote,_should_be_escaped (0.00s)                                                                                                                                                       0.0s
            --- PASS: TestGetLoginURI/bad_scheme_(data) (0.00s)                                                                                                                                                                                    0.0s
            --- PASS: TestGetLoginURI/bad_scheme_(file) (0.00s)                                                                                                                                                                                    0.0s
            --- PASS: Test_listAllDomains (2.25s)                                                                                                                                                                                                  2.2s
            === RUN   Test_EnableLogRedirection                                                                                                                                                                                                    0.9s
        ? gitlab.protectv.local/ncryptify/solo.git/clientmgmt     [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ? gitlab.protectv.local/ncryptify/solo.git/cmd/mockoidc   [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ? gitlab.protectv.local/ncryptify/solo.git/cmd/solo       [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ✓ gitlab.protectv.local/ncryptify/solo.git/httptransport  95.097s                                                                                                                                                  ✓264 ✗0 ∅1 =265 1.6m
        ? gitlab.protectv.local/ncryptify/solo.git/models [no test files]                                                                                                                                                  ✓  0 ✗0 ∅0 =  0 0.0s
        ✓ gitlab.protectv.local/ncryptify/solo.git/ssl    0.467s                                                                                                                                                           ✓ 10 ✗0 ∅0 = 10 0.5s
        ✓ gitlab.protectv.local/ncryptify/solo.git/usermgmt       3.162s

    or: 

        ⠦       gitlab.protectv.local/ncryptify/solo.git                                                                                                                                                                                 ✓279 ✗0 ∅1 =280 3.4m
            --- PASS: TestGetLoginURI/good_URL_w/javascript-quote,_should_be_escaped (0.00s)                                                                                                                                                       0.0s
            --- PASS: TestGetLoginURI/bad_scheme_(data) (0.00s)                                                                                                                                                                                    0.0s
            --- PASS: TestGetLoginURI/bad_scheme_(file) (0.00s)                                                                                                                                                                                    0.0s
            --- PASS: Test_listAllDomains (2.25s)                                                                                                                                                                                                  2.2s
            === RUN   Test_EnableLogRedirection                                                                                                                                                                                                    0.9s
        ?       gitlab.protectv.local/ncryptify/solo.git/clientmgmt     [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ?       gitlab.protectv.local/ncryptify/solo.git/cmd/mockoidc   [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ?       gitlab.protectv.local/ncryptify/solo.git/cmd/solo       [no test files]                                                                                                                                          ✓  0 ✗0 ∅0 =  0 0.0s
        ok      gitlab.protectv.local/ncryptify/solo.git/httptransport  95.097s                                                                                                                                                  ✓264 ✗0 ∅1 =265 1.6m
        ?       gitlab.protectv.local/ncryptify/solo.git/models [no test files]                                                                                                                                                  ✓  0 ✗0 ∅0 =  0 0.0s
        ok      gitlab.protectv.local/ncryptify/solo.git/ssl    0.467s                                                                                                                                                           ✓ 10 ✗0 ∅0 = 10 0.5s
        ok      gitlab.protectv.local/ncryptify/solo.git/usermgmt       3.162s

- [ ] final package count summary line was missing from running henry

**Browse**
- [ ] add browse command: must pass a file, and opens a tui which lets user navigate to any test run, and view the output
- [ ] add option to locate specific test in file (maybe with regex?) and dump it

**Run**
- [ ] add a subcommand which runs `go test`, as an alternative to piping output, e.g.:

        tang test ./... -v

**live**
- [ ] 

**optimizations**
- [ ] The replay reader parses each line.  The same line gets parsed again later.  So we're double parsing.  Would be more efficient to implement the replay logic later in the data pipeline.  Also, the replay reader reads all the lines into memory.
- [ ] If there's a line in the output which is json, but not a test/build event, I'm not sure what we'll do
- [x] need to rethink the notty mode
- [ ] ctrl-c doesn't work if the live UI hasn't started yet
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
