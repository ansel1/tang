# Add Install Target to Makefile

This plan outlines the addition of an `install` target to the `Makefile`.

## Proposed Changes

### Root Directory

#### [MODIFY] [Makefile](file:///Users/regan/dev/tang-opencode/Makefile)

-   **Add** `install` target.
    -   It will run `go install .`.
    -   This installs the binary to `$GOPATH/bin` (or `$GOBIN` if set in the environment), making it available system-wide (assuming `$GOPATH/bin` is in `$PATH`).

## Verification Plan

### Automated Tests
-   Run `make install`.
-   Check if the binary is installed in `$(go env GOPATH)/bin`.

### Manual Verification
-   Run `tang` (assuming it's in PATH) to see if it executes.
