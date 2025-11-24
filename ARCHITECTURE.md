# Architecture

## Overview

`tang` is a command-line tool for summarizing Go test results in real-time. It processes the JSON output from `go test -json` and presents it either through an interactive Terminal User Interface (TUI) or as simple text output. The tool is designed with an event-driven architecture that separates parsing, processing, and presentation concerns.

**Key Design Principles:**
- **Event-Driven**: Components communicate through typed events via Go channels
- **Separation of Concerns**: Engine, TUI, and output formats are independent
- **Stateless Processing**: The engine parses without maintaining test state
- **Consumer Independence**: Multiple output consumers can process the same event stream
- **Extensibility**: New output formats can be added without modifying existing code

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
        │  (buffered)  │
        └──────┬───────┘
               │
       ┌───────┴────────┐
       │                │
       ▼                ▼
┌─────────────┐  ┌─────────────┐
│     TUI     │  │   Simple    │
│   (Model)   │  │   Output    │
└─────────────┘  └─────────────┘
       │                │
       ▼                ▼
  Interactive      Text Output
   Terminal          (stdout)
```

### Package Structure

```
tang/
├── main.go              # Entry point, CLI flag parsing, wiring
├── engine/              # Event processing and streaming
│   ├── engine.go        # Core engine implementation
│   └── engine_test.go   # Engine tests
├── parser/              # JSON parsing
│   └── parser.go        # go test -json event parser
├── output/              # Output format consumers
│   ├── simple.go        # Simple text output
│   └── simple_test.go   # Simple output tests
└── tui/                 # Terminal UI
    ├── model.go         # Bubbletea model
    └── tui_test.go      # TUI tests
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
- **Buffered Channel**: 100-event buffer prevents blocking between producer/consumer
- **Buffer Copying**: Raw lines are copied since `bufio.Scanner` reuses buffers
- **Optional File Writers**: Support simultaneous raw and JSON output files

### 2. Parser (`parser/parser.go`)

**Responsibility:** Parse `go test -json` events

Minimal JSON unmarshaling layer that converts JSON lines into structured `TestEvent` objects.

**Key Type:**

```go
type TestEvent struct {
    Time       time.Time
    Action     string    // "run", "pass", "fail", "skip", "output", etc.
    Package    string
    Test       string    // Empty for package-level events
    Output     string
    Elapsed    float64
    Source     string
    ImportPath string
}
```

**Design Decisions:**
- **Thin Layer**: Just JSON unmarshaling, no business logic
- **Unchanged from Original**: Pre-dates the refactoring, works perfectly as-is

### 3. TUI (`tui/model.go`)

**Responsibility:** Interactive terminal interface using Bubbletea

The TUI is a consumer that:
- Receives events via Bubbletea messages
- Maintains its own model state (test counts, output lines)
- Renders live updates during test execution
- Displays final summary with styled output

**Key Types:**

```go
type Model struct {
    events         []parser.TestEvent
    output         []string
    packageResults map[string]*PackageResult
    testResults    map[string]string
    passed, failed, skipped, running int
    // Styling...
}

type EngineEventMsg engine.Event  // Wrapper for Bubbletea
```

**Design Decisions:**
- **Independent State**: TUI maintains its own copy of test state
- **Event Wrapper**: `EngineEventMsg` adapts engine events to Bubbletea's message system
- **Backward Compatible**: Still accepts direct `parser.TestEvent` messages for testing
- **Lipgloss Styling**: Green/red/yellow colored output

### 4. Simple Output (`output/simple.go`)

**Responsibility:** Plain text output for `-notty` mode

The simple output consumer:
- Accumulates test events and output lines
- Tracks summary statistics (pass/fail/skip counts)
- Writes final summary and all output when complete

**Design Decisions:**
- **Accumulate Then Write**: Collects all data before writing (vs. streaming)
- **Same Format**: Matches TUI output format for consistency
- **No Colors**: Plain text suitable for CI/CD logs

### 5. Main (`main.go`)

**Responsibility:** CLI entry point, component wiring

The main function:
- Parses command-line flags
- Sets up input source (stdin or file)
- Creates engine with options (raw/JSON output files)
- Routes events to appropriate consumer (TUI or simple)
- Handles exit codes (planned)

**Design Decisions:**
- **Minimal Logic**: Just wiring, no business logic
- **Flag-Based Routing**: `-notty` flag selects output mode
- **Goroutine for TUI**: Events forwarded to Bubbletea via `Send()`

## Data Flow

### Event Flow Sequence

1. **Input** → Lines read from stdin or file
2. **Engine** → Parses each line, emits events to channel
3. **Consumer** → Receives events, updates model state
4. **Output** → Renders/writes final results

### Event Types and Handling

| Event Type | Trigger | Engine Action | TUI Action | Simple Action |
|------------|---------|---------------|------------|---------------|
| `EventRawLine` | Non-JSON line | Copy buffer, emit | Append to output | Append to output |
| `EventTest` | Valid JSON | Parse, emit | Update model state | Update stats, append output |
| `EventError` | Scanner error | Emit error | Display error | Append error |
| `EventComplete` | EOF/error | Close channel | Quit program | Write all output |

### File Output Flow

When `-outfile` or `-jsonfile` flags are used:
- Engine writes to files **in addition to** emitting events
- Raw output: Every line written (before parsing)
- JSON output: Only successfully parsed test events
- File writing happens synchronously in the engine goroutine

## Key Design Decisions

### 1. Event-Driven Architecture

**Decision:** Use Go channels to stream events from engine to consumers

**Rationale:**
- Eliminates shared mutable state
- Natural flow control via channel buffering
- Easy to add new consumers (e.g., JUnit XML writer)
- Testable components in isolation

**Trade-offs:**
- Slightly more complex than direct function calls
- Requires goroutines for concurrent processing

### 2. Stateless Engine

**Decision:** Engine parses but doesn't track test state

**Rationale:**
- Single Responsibility Principle - engine only parses
- Consumers maintain their own models
- Same event stream can feed multiple consumers with different needs
- Engine stays simple and highly testable

### 3. Buffer Copying for Raw Lines

**Decision:** Copy scanner buffer before emitting raw line events

**Rationale:**
- `bufio.Scanner` reuses its internal buffer
- Without copying, all events would reference the same (changing) buffer
- Small performance cost for safety

### 4. Backward Compatible TUI

**Decision:** TUI accepts both `EngineEventMsg` and `parser.TestEvent`

**Rationale:**
- Existing tests send `parser.TestEvent` directly
- Don't break tests during refactoring
- Tests can remain simple without mocking engine

### 5. Buffered Event Channel

**Decision:** 100-event buffer on the channel

**Rationale:**
- Prevents engine from blocking while consumers process
- Better throughput for high-volume test output
- 100 events is ~10KB memory - acceptable overhead

## Technology Stack

### Dependencies

- **[Bubbletea v2](https://github.com/charmbracelet/bubbletea)** - TUI framework (Elm architecture)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Terminal styling
- **[teatest](https://github.com/charmbracelet/x/exp/teatest)** - Bubbletea testing utilities
- **[testify](https://github.com/stretchr/testify)** - Test assertions

### Standard Library

- `encoding/json` - JSON parsing
- `bufio` - Line scanning
- `io` - Reader/writer interfaces
- `flag` - CLI argument parsing
- `os` - File operations, stdin/stdout

## Testing Strategy

### Unit Tests

Each package has comprehensive unit tests:

**Engine Tests** (`engine/engine_test.go`):
- Valid JSON parsing → `EventTest`
- Non-JSON lines → `EventRawLine`
- Scanner errors → `EventError`
- Raw output file writing
- JSON output file writing
- Buffer copying safety

**Simple Output Tests** (`output/simple_test.go`):
- Event processing and accumulation
- Summary statistics calculation
- Output formatting
- Multiple test handling

**TUI Tests** (`tui/tui_test.go`):
- Event processing via `EngineEventMsg`
- Summary rendering
- Backward compatibility with `parser.TestEvent`
- teatest framework for TUI testing

### Integration Tests

Planned in `main_test.go`:
- Full pipeline from input to output
- Multiple output formats simultaneously
- Both TUI and `-notty` modes

### Test Coverage

Current coverage: ~30+ tests across packages
Focus areas:
- ✅ Parsing edge cases
- ✅ Event type handling
- ✅ State accumulation
- ✅ Output formatting
- ⏳ End-to-end scenarios (planned)

## Future Extensibility

The architecture supports several planned features:

### 1. JUnit XML Output

**Implementation:** New consumer in `output/junit.go`

```go
type JUnitWriter struct {
    writer io.Writer
    suites map[string]*JUnitTestSuite
}

func (j *JUnitWriter) ProcessEvents(events <-chan engine.Event) error {
    // Accumulate test data, write XML on EventComplete
}
```

**Integration:** Add `-junitfile` flag, wire up in `main.go`

### 2. Multiple Output Formats

**Pattern:** Fan-out events to multiple consumers

```go
func broadcastEvents(source <-chan Event, consumers ...chan<- Event) {
    for evt := range source {
        for _, c := range consumers {
            c <- evt
        }
    }
}
```

### 3. Multiple Test Run Detection

**Approach:** Add state machine to engine or dedicated coordinator
- Detect `{"Action": "start"}` events
- Signal run boundaries to consumers
- Reset consumer state between runs

### 4. Exit Code Support

**Implementation:** Consumers return failure status

```go
type Consumer interface {
    ProcessEvents(<-chan Event) error
    Failed() bool
}

// In main.go:
if consumer.Failed() {
    os.Exit(1)
}
```

### 5. Output Filtering

**Approach:** Add configuration to consumers

```go
type OutputConfig struct {
    IncludePassed  bool
    IncludeFailed  bool
    IncludeSkipped bool
    SlowThreshold  time.Duration
}

simple := output.NewSimpleOutput(os.Stdout, cfg)
```

### 6. Replay Mode

**Implementation:** Add delay injection to engine

```go
func (e *Engine) StreamWithTiming(input io.Reader) <-chan TimedEvent {
    // Emit events with original timestamps
    // Consumer can delay based on timestamp deltas
}
```

## Performance Considerations

### Memory

- **Bounded Event Channel**: 100-event buffer limits memory growth
- **Consumers Accumulate**: TUI and simple output keep all events in memory
  - Acceptable for typical test runs (thousands of tests)
  - May need streaming for very large test suites
- **Buffer Copying**: Small overhead for safety

### Throughput

- **Concurrent Processing**: Engine and consumers run in parallel
- **Buffered I/O**: `bufio.Scanner` for efficient line reading
- **Minimal Allocations**: Reuse of scanner buffer (with copying for events)

### Bottlenecks

- **TUI Rendering**: Bubbletea updates can be slow with very long output
  - Mitigation: Limit visible output lines
- **File I/O**: Writing raw/JSON files is synchronous in engine
  - Acceptable: File I/O is fast, parsing is the bottleneck

## Error Handling

### Error Flow

1. **Scanner Errors**: Emitted as `EventError`, consumers decide how to handle
2. **File Errors**: Fail fast in `main.go` before starting engine
3. **Parsing Errors**: Silently treat as `EventRawLine` (not all input is JSON)

### Graceful Degradation

- Non-JSON lines displayed as-is (don't break on stderr in input)
- Partial JSON is treated as raw line
- Missing test fields handled gracefully (zero values)

## Security Considerations

- **No External Network**: Tool is fully local
- **File Permissions**: Respects OS file permissions for output files
- **Input Sanitization**: Terminal escapes in output could be a concern
  - Currently: No sanitization (trusts test output)
  - Future: Could add escape sequence filtering

## Migration Path

The architecture was developed through a refactoring:

**Before:** Processing logic tightly coupled with TUI code

**After:** Clean separation with event-driven architecture

**Backward Compatibility:**
- ✅ All existing TUI tests pass unchanged
- ✅ Command-line flags remain the same (added new ones)
- ✅ Output format identical

See `REFACTORING_SUMMARY.md` for detailed migration notes.

## Contributing

When adding new features:

1. **New Output Format**: Create consumer in `output/` package
2. **New Event Type**: Add to `engine.EventType` enum, handle in all consumers
3. **New Engine Feature**: Update `engine.Option` pattern
4. **Tests Required**: All new code must have unit tests

## References

- **SPEC.md** - Feature roadmap and implementation phases
- **REFACTORING_SUMMARY.md** - Details of the architecture refactoring
- **README.md** - User-facing documentation and usage examples
