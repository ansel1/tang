# Makefile Creation Walkthrough

I have created a `Makefile` for the `tang` project to automate common development tasks.

## Changes

### [NEW] [Makefile](file:///Users/regan/dev/tang-opencode/Makefile)

The `Makefile` includes the following targets:

-   `all`: Runs `test` and `build`.
-   `build`: Builds the `tang` binary into `bin/tang`.
-   `run`: Runs the application using `go run .`.
-   `test`: Runs all tests in the project.
-   `clean`: Removes the `bin/` directory and runs `go clean`.
-   `fmt`: Formats all Go files.
-   `lint`: Runs `golangci-lint run`.
-   `help`: Displays the help screen with target descriptions.

## Verification Results

### Automated Tests

I ran the following commands to verify the Makefile targets:

-   `make all`: Successfully ran tests and built the binary.
-   `make fmt`: Successfully formatted code.
-   `make lint`: Ran the linter (found some issues, which is expected).
-   `make help`: Successfully displayed the help screen.
-   `make clean`: Successfully cleaned the build artifacts.

```bash
$ make help
all                            Run tests and build the binary
build                          Build the binary
run                            Run the application
test                           Run tests
clean                          Clean build cache and binary
fmt                            Format code
lint                           Lint code using golangci-lint
help                           Display this help screen
```
