# Architecture

## Overview

`tang` is a command-line tool for summarizing Go test results in real-time. It processes the JSON output from `go test -json` and presents it either through an interactive Terminal User Interface (TUI) or as simple text output. The tool uses a pipeline architecture where a single active consumer drives a passive state collector.

**Key Design Principles:**
- **Pipeline Processing**: Linear flow from Engine -> Event Stream -> Consumer -> Collector
- **Separation of Concerns**: Parsing, state management, and presentation are decoupled
- **Passive State Reconstitution**: The `results` package rebuilds test state from events
- **Consumer Ownership**: The active consumer (TUI or Simple Output) owns the update loop
- **Extensibility**: New output formats can be added as new consumers

## System Architecture

### High-Level Architecture

```
┌─────────────┐
│   Input     │
│  (stdin/    │
│   file)     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│         Engine                  │
│  ┌──────────────────────────┐   │
│  │ Line Scanner             │   │
│  └───────────┬──────────────┘   │
│              │                  │
│  ┌───────────▼──────────────┐   │
│  │ JSON Parser (optional)   │   │
│  └───────────┬──────────────┘   │
│              │                  │
│  ┌───────────▼──────────────┐   │
│  │ Event Emitter            │   │
│  └───────────┬──────────────┘   │
└──────────────┼──────────────────┘
               │
               ▼
        ┌──────────────┐
        │Event Channel │
        │              │
        └──────┬───────┘
               │
               ▼
      (One Active Consumer)
      ┌────────────────────┐
      │                    │
┌─────▼────┐          ┌────▼─────┐
│   TUI    │   OR     │  Simple  │
│ (Model)  │          │  Output  │
└─────┬────┘          └────┬─────┘
      │                    │
      │   Push Events      │
      └─────────┬──────────┘
                │
                ▼
      ┌────────────────────┐
      │  Results Collector │
      │  (results package) │
      │                    │
      │   Full State       │
      │   Model            │
      └─────────┬──────────┘
                │
                ▼
      ┌────────────────────┐
      │ Summary Formatter  │
      └─────────┬──────────┘
                │
                ▼
         Final Output
```

### Package Structure

```
tang/
├── main.go              # Entry point, CLI flag parsing, pipeline wiring
├── engine/              # Event processing and streaming
│   ├── engine.go        # Core engine implementation
│   └── engine_test.go   # Engine tests
├── parser/              # JSON parsing
│   └── parser.go        # go test -json event parser
├── output/              # Output format consumers
│   ├── simple.go        # Simple text output
│   └── format/          # Shared formatting logic
│       └── summary.go   # Summary text generation
├── results/             # State management
│   ├── collector.go     # Event processor and state builder
│   ├── model.go         # Data structures (Run, Package, Test)
│   └── collector_test.go# Tests
└── tui/                 # Terminal UI
    ├── model.go         # Bubbletea model (active consumer)
    └── *_test.go        # Tests
```

## Core Components

### 1. Engine (`engine/engine.go`)

**Responsibility:** Read input, parse lines, emit events

The engine is a lightweight, stateless component that:
- Reads input line-by-line from an `io.Reader`
- Attempts to parse each line as JSON (from `go test -json`)
- Emits typed events through a buffered channel
- Optionally writes raw and JSON output to files

**Key Types:**

```go
type EventType string

const (
    EventRawLine  EventType = "raw"      // Non-JSON line
    EventTest     EventType = "test"     // Parsed test event
    EventError    EventType = "error"    // Processing error
    EventComplete EventType = "complete" // Stream finished
)

type Event struct {
    Type      EventType
    RawLine   []byte
    TestEvent parser.TestEvent
    Error     error
}
```

**Design Decisions:**
- **Stateless**: Engine doesn't track test results or maintain models
- **Buffered Channel**: Prevents blocking between producer and consumer
- **Buffer Copying**: Raw lines are copied since `bufio.Scanner` reuses buffers

### 2. Results (`results/`)

**Responsibility:** Reconstitute test state from the event stream

This package contains the domain model and the logic to build it.

**Collector (`results/collector.go`):**
- **Passive Component**: Does not run its own goroutine. Methods are called by the consumer.
- **State Building**: Processes `engine.Event`s to update the `State`.
- **Run Detection**: Identifies boundaries between multiple test runs (e.g., `go test -count=2`).
- **Timing handling**: Manages wall-clock time vs event timestamps, supporting replay scaling.

**Model (`results/model.go`):**
- `State`: Top-level container, holds list of `Run`s.
- `Run`: Represents a single execution of the test suite.
- `PackageResult`: Results for a Go package.
- `TestResult`: Results for a single test.

**Design Decisions:**
- **Logic Centralization**: All logic for interpreting test events happens here.
- **Passive Design**: Puts control in the hands of the consumer (TUI/Simple), simplifying threading.
- **Detailed State**: Tracks everything needed for the final summary and TUI display.

### 3. TUI (`tui/model.go`)

**Responsibility:** Interactive terminal interface using Bubbletea

The TUI is an **active consumer** that:
- Runs the Bubbletea event loop.
- Receives events from the engine channel (batched for performance).
- Pushes events into the `results.Collector`.
- Reads state from the `results.Collector` to render the UI.

**Key Design Decisions:**
- **Batch Processing**: Groups engine events to reduce render cycles (`EngineEventBatchMsg`).
- **Single Threaded Update**: The `Update()` method is the single writer to the collector.
- **View Logic**: Renders the current run state from the collector.

### 4. Simple Output (`output/simple.go`)

**Responsibility:** Plain text output for `-notty` mode

The simple output is an alternative **active consumer** that:
- Reads from the engine channel in a loop.
- Pushes events to the `results.Collector`.
- buffer buffering output lines or streaming them to stdout.
- Prints the summary at the end.

**Design Decisions:**
- **Shared Logic**: Uses the same `results.Collector` as the TUI.
- **Polymorphism**: `main.go` chooses between TUI and Simple Output based on flags.

### 5. Summary Formatter (`output/format/summary.go`)

**Responsibility:** Generate the text summary

- Shared by both TUI and Simple Output.
- Computes statistics (slow tests, pass/fail counts).
- Formats the "OVERALL RESULTS" block.
- Handles colorization (red/green/yellow) via Lipgloss.

## Data Flow

### Pipeline Sequence

1. **Input**: Lines read from stdin or file.
2. **Engine**: Parses lines -> `engine.Event` channel.
3. **Main**: Wires the channel to the chosen consumer.
    - **TUI Mode**: Channel -> `tui.Model` (via Bubbletea)
    - **Simple Mode**: Channel -> `output.SimpleOutput`
4. **Consumer**:
    - Receives Event.
    - Calls `collector.Push(event)`.
    - `Collector` updates `State`.
    - Consumer renders output based on new `State`.

### Thread-Safety

The architecture relies on **confinement** rather than locking for the primary data path:

- The `results.Collector` is **NOT thread-safe**.
- It is designed to be owned by a single active consumer.
- **TUI Mode**: The Bubbletea `Update` loop allows exclusive access to the collector.
- **Simple Mode**: The `ProcessEvents` loop has exclusive access.

*Note: The TUI does read from the collector in its `View()` method. In Bubbletea, `View` is technically concurrent with `Update`, but in practice, they are often coordinated. If strict concurrency safety is needed between TUI Update and View, a lock would be added to the Model, not the Collector.*

## Key Design Decisions

### 1. Replay Mode Support
To support replaying past logs with realistic timing:
- **ReplayReader**: A wrapper around `io.Reader` that simulates delays based on timestamp deltas.
- **Timing Logic**: The `Collector` distinguishes between `EventTime` (log timestamp) and `WallStartTime` (replay wall clock) to show accurate "Elapsed" timers during replay.

### 2. Event Batching in TUI
The TUI batches events from the engine channel before sending them to the Bubbletea update loop. This prevents the UI from freezing when processing high-volume output (e.g., thousands of lines per second).

### 3. Multiple Test Runs
The `results.Collector` can detect when a new test run starts (e.g., `go test -count=N`). It creates a new `Run` object in the state. The consumers are designed to display the "Current Run".

- **REFACTORING_SUMMARY.md** - Details of the architecture refactoring
- **README.md** - User-facing documentation and usage examples
