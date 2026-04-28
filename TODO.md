TODO
----
- [ ] need to review all the unit tests for redundancy
- [ ] what value is Summary bringing?  it seems to be a middleman between the run and the formatting code
- [ ] FastestPackage   *results.PackageResult
	SlowestPackage   *results.PackageResult
	MostTestsPackage *results.PackageResult don't seem to be used anymore
- [ ] an immediate fail leads to a stuck terminal.  I think it's the bubbletea issue
- [ ] try not using color for passing tests/packages
- [ ] When running with -count > 1, the running count goes negative.

        ⠦ (1 packages: 1 running, 0 done)                                                                                                                                   ▶-1053 ⏸903 (✓1378 ✗4 ∅22) 1404 15.5m
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
⠦ gitlab.protectv.local/ncryptify/henry                                                                                                                             ▶-1053 ⏸903 (✓1378 ✗4 ∅22) 1404 15.5m

- [ ] also, when -count > 0, failures that happened during the first run of the test are erased when the second
      run starts, so they don't show up in the summary, or the tui.  The only evidence that they happened is in the counts
- [x] detect and report panics
- [x] Play with the UX: make the live feed more like the summary format
- [x] add a `-v` flag.  In both live and notty, outputs something closer to the original 
      test verbose output, including all tests
- [x] rename `--junitout` to `--junitfile`
- [x] add COLUMNS support if we don't already have it
- [x] still something wrong with indentation of log lines in the summary.  when running sample, they are indented too far, but not when running henry's tests
- [x] add a `--no-color` flag
- [x] if panic happens in TestMain, tang prints nothing
- [x] when eliding, have RUN and CONT taking higher precendence than PAUSE
- [x] don't count paused tests in the running count
- [x] clear up the help string
- [x] in live mode, try using the icons again in the left gutter for finished packages, so isntead of:

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

**Browse**
- [ ] add browse command: must pass a file, and opens a tui which lets user navigate to any test run, and view the output
- [ ] add option to locate specific test in file (maybe with regex?) and dump it

**optimizations**
- [ ] The replay reader parses each line.  The same line gets parsed again later.  So we're double parsing.  Would be more efficient to implement the replay logic later in the data pipeline.  Also, the replay reader reads all the lines into memory.
- [ ] If there's a line in the output which is json, but not a test/build event, I'm not sure what we'll do
- [x] need to rethink the notty mode
- [ ] ctrl-c doesn't work if the live UI hasn't started yet
- [ ] handle "action":"bench" events 
