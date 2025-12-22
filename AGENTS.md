# Agent Guide for tang

## Build & Test Commands
- Build: `go build -o tang .`
- Run all tests: `go test ./...`
- Run single test: `go test -run TestName ./path/to/package`
- Run with coverage: `go test -cover ./...`
- Format code: `go fmt ./...`
- Lint: `go vet ./...`

## Code Style
- **Imports**: Standard library first, blank line, then third-party packages (grouped by domain)
- **Formatting**: Use `go fmt` - standard Go formatting applies
- **Types**: Explicit types preferred for struct fields; use type inference for local variables when clear
- **Naming**: Follow Go conventions - exportedNames, privateNames, short variable names (i, m, pkg) in limited scope
- **Error handling**: Always check errors immediately; return errors up the call stack; use fmt.Errorf for context
- **Comments**: Exported functions/types must have doc comments starting with the name (e.g., "ParseEvent parses...")

## Project-Specific Notes
- This is a Bubbletea TUI app - **do not run it directly in tests/CI** (requires TTY)
- Test events use `go test -json` format (see parser.TestEvent)
- Test data format: blocks separated by `###`, input/expected separated by `===`
- Focus on compilation correctness over runtime verification
