# Architecture

## Overview

`tang` is a command-line tool for summarizing Go test results in real-time. It processes the JSON output from `go test -json` and presents it either through an interactive live display or as simple text output. The tool uses a pipeline architecture where a single event loop drives a passive state collector and a single text renderer.

**Key Design Principles:**
- **Pipeline Processing**: Linear flow from Engine -> Event Stream -> Event Loop -> Collector + Renderer
- **Separation of Concerns**: Parsing, state management, and presentation are decoupled
- **Passive State Reconstitution**: The `results` package rebuilds test state from events
- **One Canonical Renderer**: `output.SimpleOutput` renders the text output in *both* modes; live mode only adds a TUI on top of it
- **Mode Equivalence**: After tang exits, the terminal contents are identical whether the live TUI was used or not

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
      ┌────────────────────┐
      │    Event Loop      │
      │  (eventloop.go)    │
      └──┬───────┬──────┬──┘
         │       │      │
         │ Push  │Render│ (live mode only)
         ▼       ▼      ▼
┌────────────┐ ┌────────┐ ┌──────────┐
│  Results   │ │ Simple │ │ Live TUI │
│ Collector  │ │ Output │ │ (Model)  │
│ (results)  │ │        │ │          │
└────────────┘ └───┬────┘ └────┬─────┘
       ▲           │           │
       │ Read      │ buffer    │ reads collector,
       └───────────┼───────────┘ owns screen while
                   │             a run is active
                   ▼
       stdout (dumped whenever the
        TUI is not on screen)
```

In both modes, all rendering flows through one `output.SimpleOutput`, which
writes into a buffer. The event loop dumps the buffer to stdout after every
event while no TUI is on screen. In notty mode there is never a TUI, so
output effectively streams directly. In live mode the buffer accumulates
while the TUI owns the screen and is dumped when the TUI closes (at the end
of each run). A summary block is written through the same renderer after
every run.

### Package Structure

```
tang/
├── main.go              # Entry point, CLI flag parsing, pipeline wiring
├── eventloop.go         # Unified event loop driving collector, renderer, TUI
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

### 3. Event Loop (`eventloop.go`)

**Responsibility:** The single active consumer, shared by both modes

`runEventLoop` owns the engine channel and, for each event:
- Pushes it to the `results.Collector`.
- Renders it through the `output.SimpleOutput` (into a buffer).
- Dumps the buffer to stdout if no TUI is on screen.
- Detects run boundaries (the collector finishes a run on raw lines and
  stream completion) and writes a summary after every run.
- In live mode, starts the TUI when a run starts and quits it when the run
  finishes; if the user quits the TUI, the partial run is finished and
  summarized and remaining events are discarded (break-and-summarize).

The TUI is abstracted behind the `tuiController` interface so the loop can
be tested without a terminal (see `eventloop_test.go`, which asserts that
notty and live modes produce byte-identical output).

### 4. Live Display (`tui/model.go`)

**Responsibility:** Interactive terminal interface using Bubbletea

The live display:
- Runs the Bubbletea event loop while a run is active.
- Reads state from the `results.Collector` to render the UI (it does not
  push events; the event loop does).
- Receives periodic `RepaintMsg`s from the event loop.
- Renders inline (no alt screen) and clears its render on exit, so the
  scrollback left behind is only what the event loop printed.

### 5. Simple Output (`output/simple.go`)

**Responsibility:** Canonical plain text rendering in both modes

`SimpleOutput` reconstructs `go test`-style text output from events:
- One package at a time is "focused" and streams; others buffer.
- Verbose mode reconstructs `go test -v` output (including `=== NAME`
  injection and subtest grouping); non-verbose mode emits failures only.
- Raw (non-JSON) lines are rendered inline, in stream order.
- `WriteRunSummaries` writes a summary block for each finished run exactly
  once.

**Design Decisions:**
- **Always On**: Runs in both notty and live modes; the only difference is
  where its writer's bytes end up (streamed vs. buffered until TUI close).
- **Caller-Driven**: `Init`/`ProcessEvent`/`Flush`/`WriteRunSummaries` are
  called by the event loop, which also owns `collector.Push`.

### 6. Summary Formatter (`output/format/summary.go`)

**Responsibility:** Generate the text summary

- Used by `SimpleOutput.WriteRunSummaries` (both modes) and the TUI.
- Computes statistics (slow tests, pass/fail counts).
- Formats the "OVERALL RESULTS" block.
- Handles colorization (red/green/yellow) via Lipgloss.

## Data Flow

### Pipeline Sequence

1. **Input**: Lines read from stdin or file.
2. **Engine**: Parses lines -> `engine.Event` channel.
3. **Event Loop** (`runEventLoop`), per event:
    - Calls `collector.Push(event)`; `Collector` updates `State`.
    - Calls `simple.ProcessEvent(event)` to render into the buffer.
    - Dumps the buffer to stdout if no TUI is on screen.
    - On run boundaries: quits the TUI (live mode), flushes, and writes the
      run summary.

### Thread-Safety

- Events are pushed to the collector only by the event loop goroutine.
- The live TUI reads collector state from its own goroutine; reads and
  writes are coordinated via the collector's mutex (`Lock`/`Unlock`,
  internal locking in `Push`).
- The output buffer is confined to the event loop goroutine; the TUI never
  touches it.

## Key Design Decisions

### 1. Replay Mode Support
To support replaying past logs with realistic timing:
- **ReplayReader**: A wrapper around `io.Reader` that simulates delays based on timestamp deltas.
- **Timing Logic**: The `Collector` distinguishes between `EventTime` (log timestamp) and `WallStartTime` (replay wall clock) to show accurate "Elapsed" timers during replay.

### 2. Event Batching in Live Mode
The live display batches events from the engine channel before sending them to the Bubbletea update loop. This prevents the UI from freezing when processing high-volume output (e.g., thousands of lines per second).

### 3. Multiple Test Runs
The `results.Collector` can detect when a new test run starts (e.g., `go test -count=N`). It creates a new `Run` object in the state. The consumers are designed to display the "Current Run".

- **REFACTORING_SUMMARY.md** - Details of the architecture refactoring
- **README.md** - User-facing documentation and usage examples
