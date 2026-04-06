TODO
----
- [x] detect and report panics
- [x] Play with the UX: make the live feed more like the summary format
- [x] add a `-v` flag.  In both live and notty, outputs something closer to the original 
      test verbose output, including all tests
- [x] rename `--junitout` to `--junitfile`
- [x] add COLUMNS support if we don't already have it
- [x] still something wrong with indentation of log lines in the summary.  when running sample, they are indented too far, but not when running henry's tests
- [ ] add a `--no-color` flag
- [ ] if panic happens in TestMain, tang prints nothing
- [x] when eliding, have RUN and CONT taking higher precendence than PAUSE
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

⠼ (13 packages: 1 running, 12 done)                                                                                            ▶7|⏸132|644|✓412|✗223|∅123  6.1s
-----------------------------------------------------------------------------------------------------------------------------------------------------------------
⠼ github.com/ansel1/tang/sample/auth                                                                                           ▶7|⏸ 32| 28|✓  2|✗  3|∅ 23| 6.1s
ok      gitlab.protectv.local/ncryptify/henry/benchmarktests    (cached)                                                                 1|✓  0|✗  0|∅  1| 0.0s
?       gitlab.protectv.local/ncryptify/henry/cmd/henry [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/common    (cached)                                                                        14|✓ 14|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/henryconfig       [no test files]                                                          0|✓  0|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/labels    (cached)                                                                        36|✓ 36|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/logging   (cached)                                                                         1|✓  1|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/logging/auditlogs (cached)                                                                32|✓ 32|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/messages  [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/models    (cached)                                                                        21|✓ 21|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/qos       [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/records   [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s

⠼ (13 packages: 1 running, 12 done)                                                                                            ▶7 ⏸132 644:✓412 ✗223 ∅123  6.1s
-----------------------------------------------------------------------------------------------------------------------------------------------------------------
⠼ github.com/ansel1/tang/sample/auth                                                                                           ▶7  ⏸32   28  ✓2   ✗3   ∅23  6.1s
ok      gitlab.protectv.local/ncryptify/henry/benchmarktests    (cached)                                                                 1|✓  0|✗  0|∅  1| 0.0s
?       gitlab.protectv.local/ncryptify/henry/cmd/henry [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/common    (cached)                                                                        14|✓ 14|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/henryconfig       [no test files]                                                          0|✓  0|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/labels    (cached)                                                                        36|✓ 36|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/logging   (cached)                                                                         1|✓  1|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/logging/auditlogs (cached)                                                                32|✓ 32|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/messages  [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s
ok      gitlab.protectv.local/ncryptify/henry/models    (cached)                                                                        21|✓ 21|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/qos       [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s
?       gitlab.protectv.local/ncryptify/henry/records   [no test files]                                                                  0|✓  0|✗  0|∅  0| 0.0s

⠼ ▶run ⏸pause ✓pass|✗fail|∅skip|done                                                                                           ▶7|⏸132 ✓412|✗223|∅123=644  6.1s
---------------------------------------------------------------------------------------------------------------------------------------------------------------
⠼ github.com/ansel1/tang/sample/auth                                                                                           ▶7| ⏸32   ✓2|  ✗3|∅23=  28  6.1s
✓ gitlab.protectv.local/ncryptify/henry/benchmarktests    (cached)                                                                       ✓0|  ✗0| ∅1=   1  0.0s
? gitlab.protectv.local/ncryptify/henry/cmd/henry [no test files]                                                                        ✓0|  ✗0| ∅0=   0  0.0s
✓ gitlab.protectv.local/ncryptify/henry/common    (cached)                                                                              ✓14|  ✗0| ∅0=  14  0.0s
? gitlab.protectv.local/ncryptify/henry/henryconfig       [no test files]                                                                ✓0|  ✗0| ∅0=   0  0.0s
✓ gitlab.protectv.local/ncryptify/henry/httptransport     (cached)                                                                     ✓198|  ✗0|∅33= 231  0.0s
✓ gitlab.protectv.local/ncryptify/henry/labels    (cached)                                                                              ✓36|  ✗0| ∅0=  36  0.0s
✓ gitlab.protectv.local/ncryptify/henry/logging   (cached)                                                                               ✓1|  ✗0| ∅0=   1  0.0s
✓ gitlab.protectv.local/ncryptify/henry/logging/auditlogs (cached)                                                                      ✓32|  ✗0| ∅0=  32  0.0s
? gitlab.protectv.local/ncryptify/henry/messages  [no test files]                                                                        ✓0|  ✗0| ∅0=   0  0.0s
✗ gitlab.protectv.local/ncryptify/henry/models    (cached)                                                                              ✓21|  ✗3| ∅0=  21  0.0s
? gitlab.protectv.local/ncryptify/henry/qos       [no test files]                                                                        ✓0|  ✗0| ∅0=   0  0.0s
? gitlab.protectv.local/ncryptify/henry/records   [no test files]                                                                        ✓0|  ✗0| ∅0=   0  0.0s

  ok      gitlab.protectv.local/ncryptify/henry/benchmarktests    (cached)                                                               ✓  0 ✗0 ∅  1 =   1  0.0s
  ?       gitlab.protectv.local/ncryptify/henry/cmd/henry [no test files]                                                                ✓  0 ✗0 ∅  0 =   0  0.0s
  ok      gitlab.protectv.local/ncryptify/henry/common    (cached)                                                                       ✓ 14 ✗0 ∅  0 =  14  0.0s
  ?       gitlab.protectv.local/ncryptify/henry/henryconfig       [no test files]                                                        ✓  0 ✗0 ∅  0 =   0  0.0s
  ok      gitlab.protectv.local/ncryptify/henry/httptransport     (cached)                                                               ✓198 ✗0 ∅ 33 = 231  0.0s
  ok      gitlab.protectv.local/ncryptify/henry/labels    (cached)                                                                       ✓ 36 ✗0 ∅  0 =  36  0.0s
  ok      gitlab.protectv.local/ncryptify/henry/logging   (cached)                                                                       ✓  1 ✗0 ∅  0 =   1  0.0s
  ok      gitlab.protectv.local/ncryptify/henry/logging/auditlogs (cached)                                                               ✓ 32 ✗0 ∅  0 =  32  0.0s
  ?       gitlab.protectv.local/ncryptify/henry/messages  [no test files]                                                                ✓  0 ✗0 ∅  0 =   0  0.0s
  ok      gitlab.protectv.local/ncryptify/henry/models    (cached)                                                                       ✓ 21 ✗0 ∅  0 =  21  0.0s
  ?       gitlab.protectv.local/ncryptify/henry/qos       [no test files]                                                                ✓  0 ✗0 ∅  0 =   0  0.0s
  ?       gitlab.protectv.local/ncryptify/henry/records   [no test files]                                                                ✓  0 ✗0 ∅  0 =   0  0.0s


FAIL    github.com/ansel1/tang/sample/auth 7.938s             ( ✓17 ✗2 ∅1)  20  7.939s
FAIL    github.com/ansel1/tang/sample/broken [build failed]
ok      github.com/ansel1/tang/sample/cache (cached)          ( ✓12 ✗0 ∅0)  12
ok      github.com/ansel1/tang/sample/mathutil (cached)       ( ✓26 ✗0 ∅0)  26
?       github.com/ansel1/tang/sample/models [no test files]
ok      github.com/ansel1/tang/sample/stringutil (cached)     ( ✓26 ✗0 ∅2)  28
ok      github.com/ansel1/tang/sample/validator (cached)      ( ✓33 ✗0 ∅1)  34
-------------------------------------------------------------------------------------
(7 packages)                                                  (✓114 ✗2 ∅4) 120  7.939s

=== gitlab.protectv.local/ncryptify/solo.git
    --- FAIL: TestSolo_AccountReset_AmbiguousDomain (6.90s)
        === ARTIFACTS TestSolo_AccountReset_AmbiguousDomain /Users/regan/dev/ncryptify/solo/build/_artifacts/TestSolo_AccountReset_AmbiguousDomain/3086546935
        --- FAIL: TestSolo_AccountReset_AmbiguousDomain/domain_user_only_in_one_ambiguous_domain_should_resolve (0.01s)
            accountreset_test.go:631:
                        Error Trace:    /Users/regan/dev/ncryptify/solo/accountreset_test.go:631
                        Error:          Received unexpected error:
                                        Failed to reset the user account: User not found
                                        HTTP Code: 404
                                        User Message: User not found
                                        errorResponse: <nil>

                                        gitlab.protectv.local/ncryptify/solo%2egit.(*Kylo).getHeidiUserByUsername
                                                /Users/regan/dev/ncryptify/solo/heidizones.go:740
                                        gitlab.protectv.local/ncryptify/solo%2egit.(*Kylo).locateDomain
                                                /Users/regan/dev/ncryptify/solo/domains.go:1155
                                        gitlab.protectv.local/ncryptify/solo%2egit.(*Kylo).AccountReset
                                                /Users/regan/dev/ncryptify/solo/accountreset.go:16
                                        gitlab.protectv.local/ncryptify/solo%2egit.TestSolo_AccountReset_AmbiguousDomain.func2
                                                /Users/regan/dev/ncryptify/solo/accountreset_test.go:630
                                        testing.tRunner
                                                /opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036
                                        runtime.goexit
                                                /opt/homebrew/Cellar/go/1.26.1/libexec/src/runtime/asm_arm64.s:1447
                        Test:           TestSolo_AccountReset_AmbiguousDomain/domain_user_only_in_one_ambiguous_domain_should_resolve
                        Messages:       AccountReset should find domain user bob in deepblue despite ambiguous domain name
            === ARTIFACTS TestSolo_AccountReset_AmbiguousDomain/domain_user_only_in_one_ambiguous_domain_should_resolve /Users/regan/dev/ncryptify/solo/build/_artifacts/TestSolo_AccountReset_AmbiguousDomain__domain_usc38ea456620119ce/256525650
        --- FAIL: TestSolo_AccountReset_AmbiguousDomain/root_user_and_domain_user_in_different_ambiguous_domains_should_return_ambiguity_error (0.29s)
            accountreset_test.go:663:
                        Error Trace:    /Users/regan/dev/ncryptify/solo/accountreset_test.go:663
                        Error:          Error "Failed to reset the user account: failed to find user" does not contain "ambiguous"
                        Test:           TestSolo_AccountReset_AmbiguousDomain/root_user_and_domain_user_in_different_ambiguous_domains_should_return_ambiguity_error
                        Messages:       AccountReset should reject ambiguous domain+user combination
            === ARTIFACTS TestSolo_AccountReset_AmbiguousDomain/root_user_and_domain_user_in_different_ambiguous_domains_should_return_ambiguity_error /Users/regan/dev/ncryptify/solo/build/_artifacts/TestSolo_AccountReset_AmbiguousDomain__root_userb1233b24a8c92ba3/1599749544
FAIL gitlab.protectv.local/ncryptify/solo.git

FAIL    gitlab.protectv.local/ncryptify/solo.git 21.509s  (✓0 ✗3 ∅0) 3  21.51s
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
(1 packages)                                              (✓0 ✗3 ∅0) 3  21.51s