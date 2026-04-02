TODO
----
- [x] detect and report panics
- [x] Play with the UX: make the tui feed more like the summary format
- [x] add a `-v` flag.  In both tui and notty, outputs something closer to the original 
      test verbose output, including all tests
- [x] rename `--junitout` to `--junitfile`
- [ ] add a `--no-color` flag
- [ ] clear up the help string
- [ ] in the tui, try using the icons again in the left gutter for finished packages, so isntead of:

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

- [ ] the summary isn't preserving the indentation of the logs

        --- FAIL: TestHenry_NatsDomainDeletionHandler (1.92s)
            pgtestdb.go:177: testdbconf: postgres://postgres:postgres@localhost:5453/testdb_tpl_fce91eba4683f9a625fb30b55a97e628_inst_d65482ca?sslmode=disable
            henry_nats_subscriber_test.go:3572:
                        Error Trace:    /Users/regan/dev/ncryptify/henry/henry_nats_subscriber_test.go:3572
                        Error:          Received unexpected error:
                                        Cannot find the specified domain: SubDomainNu4yKMDw
                                        HTTP Code: 400
                                        User Message: Cannot find the specified domain: SubDomainNu4yKMDw

                                        gitlab.protectv.local/ncryptify/henry.(*Henry).updateCTEClient.func1
                                                /Users/regan/dev/ncryptify/henry/henry_client_2.go:3199
                                        gitlab.protectv.local/ncryptify/henry.(*Henry).updateResourceInAccount
                                                /Users/regan/dev/ncryptify/henry/henry_utilities.go:777
                                        gitlab.protectv.local/ncryptify/henry.(*Henry).updateCTEClient
                                                /Users/regan/dev/ncryptify/henry/henry_client_2.go:2792
                                        gitlab.protectv.local/ncryptify/henry.TestHenry_NatsDomainDeletionHandler
                                                /Users/regan/dev/ncryptify/henry/henry_nats_subscriber_test.go:3571
                                        testing.tRunner
                                                /opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036
                                        runtime.goexit
                                                /opt/homebrew/Cellar/go/1.26.1/libexec/src/runtime/asm_arm64.s:1447
                        Test:           TestHenry_NatsDomainDeletionHandler
            === ARTIFACTS TestHenry_NatsDomainDeletionHandler /Users/regan/dev/ncryptify/henry/build/_artifacts/TestHenry_NatsDomainDeletionHandler/4156307301
        FAIL    gitlab.protectv.local/ncryptify/henry   363.414s
        ok      gitlab.protectv.local/ncryptify/henry/benchmarktests    1.229s
        ?       gitlab.protectv.local/ncryptify/henry/cmd/henry [no test files]
        ok      gitlab.protectv.local/ncryptify/henry/common    (cached)
        ?       gitlab.protectv.local/ncryptify/henry/henryconfig       [no test files]
        ok      gitlab.protectv.local/ncryptify/henry/httptransport     197.717s
        ok      gitlab.protectv.local/ncryptify/henry/labels    (cached)
        ok      gitlab.protectv.local/ncryptify/henry/logging   0.312s
        ok      gitlab.protectv.local/ncryptify/henry/logging/auditlogs 0.321s
        ?       gitlab.protectv.local/ncryptify/henry/messages  [no test files]
        ok      gitlab.protectv.local/ncryptify/henry/models    0.738s
        ?       gitlab.protectv.local/ncryptify/henry/qos       [no test files]
        ?       gitlab.protectv.local/ncryptify/henry/records   [no test files]

        === gitlab.protectv.local/ncryptify/henry
          --- FAIL: TestHenry_UpdateCteCsiStorageGroup_Quorum (0.27s)
              pgtestdb.go:177: testdbconf: postgres://postgres:postgres@localhost:5453/testdb_tpl_fce91eba4683f9a625fb30b55a97e628_inst_7a3d2e14?sslmode=disable
              henry_csi_storagegroup_test.go:300: Test: Create CSI StorageGroup
              henry_csi_storagegroup_test.go:318: 2) Activate Quorum Policy
              test_utils.go:417: Activate Quorum Policy
              henry_csi_storagegroup_test.go:324:
              Error Trace:      /Users/regan/dev/ncryptify/henry/henry_csi_storagegroup_test.go:324
              Error:            Received unexpected error:
              Failed to activate quorum policy with err: [NCERRBadRequest: Bad HTTP request]: actions not supported for quorum
              HTTP Code: 400
              User Message: Failed to activate quorum policy with err: [NCERRBadRequest: Bad HTTP request]: actions not supported for quorum

              gitlab.protectv.local/ncryptify/henry.(*Henry).ActivateQuorumPolicy
              /Users/regan/dev/ncryptify/henry/test_utils.go:421
              gitlab.protectv.local/ncryptify/henry.TestHenry_UpdateCteCsiStorageGroup_Quorum
              /Users/regan/dev/ncryptify/henry/henry_csi_storagegroup_test.go:323
              testing.tRunner
              /opt/homebrew/Cellar/go/1.26.1/libexec/src/testing/testing.go:2036
              runtime.goexit
              /opt/homebrew/Cellar/go/1.26.1/libexec/src/runtime/asm_arm64.s:1447
              Test:             TestHenry_UpdateCteCsiStorageGroup_Quorum
              === ARTIFACTS TestHenry_UpdateCteCsiStorageGroup_Quorum /Users/regan/dev/ncryptify/henry/build/_artifacts/TestHenry_UpdateCteCsiStorageGroup_Quorum/4113618586
              
- [ ] final package count summary line was missing from running henry

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
- [x] need to rethink the notty mode
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
