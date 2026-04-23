# Code Context

## Project Overview

**Tang** is a command-line tool for summarizing Go test results in real-time. It processes the JSON output from `go test -json` and presents it either through an interactive terminal UI (TUI) or as simple text output.

## Files Retrieved

1. `/Users/regan/dev/tang/README.md` (full file) - User-facing documentation
2. `/Users/regan/dev/tang/ARCHITECTURE.md` (full file) - System architecture details
3. `/Users/regan/dev/tang/main.go` (lines 1-250+) - Entry point, CLI parsing, pipeline wiring

## Key Code

### Main Entry Point (`main.go`)
The `main.go` serves as the pipeline orchestrator:
- Parses CLI flags (tang's flags vs go test flags)
- Supports running `go test` directly: `tang test ./...`
- Supports reading from stdin or file
- Creates the Engine to process input
- Wires to either TUI (`tui.Model`) or Simple Output (`output.SimpleOutput`)
- Handles signals for graceful shutdown

### Engine (`engine/engine.go`)
```go
type EventType string

const (
    EventRawLine  EventType = "raw"      // Non-JSON line
    EventTest     EventType = "test"     // Parsed test event
    EventBuild    EventType = "build"    // Parsed build event
    EventError    EventType = "error"    // Processing error
    EventComplete EventType = "complete" // Stream finished
)

type Event struct {
    Type       EventType
    RawLine    []byte
    TestEvent  parser.TestEvent
    BuildEvent parser.BuildEvent
    Error      error
}
```
The engine reads input line-by-line, parses JSON events from `go test -json`, and emits typed events through a buffered channel. It's stateless.

### Results Model (`results/model.go`)
```go
type Run struct {
    ID             int
    Packages       map[string]*PackageResult
    PackageOrder   []string
    TestResults    map[string]*TestResult
    FirstEventTime time.Time
    Counts         struct { Passed, Failed, Skipped, Running, Paused int }
    Status         Status
}

type TestResult struct {
    Name     string
    Package  string
    Status   Status
    Duration time.Duration
    Output   string
    TestID   int
}
```
The results package reconstitutes test state from the event stream. It's a **passive** component - doesn't run its own goroutine, methods are called by the consumer.

## Architecture

```
┌─────────────┐
│   Input     │
│  (stdin/    │
│   file)     │
└──────┬──────┘
       ▼
┌─────────────────────────────────┐
│         Engine                  │
│  Line Scanner → JSON Parser    │
│  → Event Emitter               │
└──────────────┼──────────────────┘
               │
               ▼
        ┌──────────────┐
        │Event Channel │
        └──────┬───────┘
               │
               ▼
      ┌────────────────────┐
      │  Active Consumer   │
      │                    │
┌─────▼────┐          ┌────▼─────┐
│  Live    │   OR     │  Simple  │
│ (TUI)    │          │  Output  │
└─────┬────┘          └────┬─────┘
      │                    │
      │   Push Events      │
      └─────────┬──────────┘
                ▼
      ┌────────────────────┐
      │  Results Collector │
      │  (Passive State)   │
      └────────────────────┘
```

### Key Design Decisions:
1. **Pipeline Processing**: Linear flow from Engine → Event Stream → Consumer → Collector
2. **Separation of Concerns**: Parsing, state management, and presentation are decoupled
3. **Passive State Reconstitution**: The `results` package rebuilds test state from events
4. **Consumer Ownership**: Active consumer (TUI or Simple Output) owns the update loop
5. **Confinement over Locking**: Results collector is NOT thread-safe, designed to be owned by a single consumer

### Package Structure:
- `engine/` - Event processing and streaming
- `parser/` - JSON parsing of go test -json events
- `results/` - State management and domain model
- `output/` - Simple text output consumer
- `tui/` - Interactive terminal UI (using Bubbletea)
- `output/format/` - Summary text generation
- `output/junit/` - JUnit XML output

## Start Here

Start with **`/Users/regan/dev/tang/main.go`** if you need to:
- Understand how flags are parsed
- See how the TUI vs simple output decision is made
- Understand the overall pipeline wiring

Start with **`/Users/regan/dev/tang/engine/engine.go`** if you need to:
- Understand how input is parsed
- See how events are emitted to consumers

Start with **`/Users/regan/dev/tang/results/collector.go`** if you need to:
- Understand how test state is built from events
- See how multiple test runs are detected

Start with **`/Users/regan/dev/tang/tui/model.go`** if you need to:
- Understand the interactive TUI implementation