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
               ▼
        ┌──────────────┐
        │  Event       │
        │  Fan-out     │
        │  (main.go)   │
        └──────┬───────┘
               │
       ┌───────┼────────┬──────────────┐
       │       │        │              │
       ▼       ▼        ▼              ▼
┌──────────┐ ┌────┐ ┌────────┐  ┌──────────────┐
│   TUI    │ │Sim-│ │Summary │  │   Future     │
│ (Model)  │ │ple │ │Collec- │  │  Consumers   │
│          │ │Out │ │tor     │  │ (JUnit, etc) │
│Minimal   │ │put │ │        │  └──────────────┘
│State     │ │    │ │Detailed│
│          │ │    │ │State   │
└────┬─────┘ └─┬──┘ └───┬────┘
     │         │        │
     │         │        │ (thread-safe)
     │         │        │
     ▼         ▼        ▼
┌──────────────────────────────┐
│   Summary Formatter          │
│   (shared by all consumers)  │
└──────────────┬───────────────┘
               │
       ┌───────┴────────┐
       ▼                ▼
  Interactive      Text Output
   Terminal          (stdout)
```

### Package Structure

```
tang/
├── main.go              # Entry point, CLI flag parsing, event fan-out wiring
├── engine/              # Event processing and streaming
│   ├── engine.go        # Core engine implementation
│   └── engine_test.go   # Engine tests
├── parser/              # JSON parsing
│   └── parser.go        # go test -json event parser
├── output/              # Output format consumers
│   ├── simple.go        # Simple text output
│   └── simple_test.go   # Simple output tests
└── tui/                 # Terminal UI and summary
    ├── model.go         # Bubbletea model (minimal state)
    ├── summary.go       # Summary collector (detailed state)
    ├── summary_*.go     # Summary computation and formatting
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
- Maintains **minimal state** for real-time display (running, passed, failed, skipped counts)
- Renders live updates during test execution
- Retrieves detailed summary from Summary Collector when complete

**Key Types:**

```go
type Model struct {
    output         []string
    passed, failed, skipped, running int
    summaryCollector *SummaryCollector  // Reference to shared collector
    // Styling...
}

type EngineEventMsg engine.Event  // Wrapper for Bubbletea
```

**Design Decisions:**
- **Minimal State**: TUI only tracks lightweight counters for real-time display
- **Shared Summary**: Delegates detailed tracking to Summary Collector
- **Event Wrapper**: `EngineEventMsg` adapts engine events to Bubbletea's message system
- **Backward Compatible**: Still accepts direct `parser.TestEvent` messages for testing
- **Lipgloss Styling**: Green/red/yellow colored output

### 3a. Summary Collector (`tui/summary.go`)

**Responsibility:** Independent consumer that accumulates detailed test results

The Summary Collector is a **parallel consumer** that:
- Runs in its own goroutine, consuming events from a dedicated channel
- Maintains comprehensive test state (all test results, package data, output)
- Provides thread-safe access to summary data via mutex
- Computes final statistics when requested

**Key Types:**

```go
type SummaryCollector struct {
    packages      map[string]*PackageResult
    testResults   map[string]*TestResult
    startTime     time.Time
    endTime       time.Time
    packageOrder  []string
    mu            sync.RWMutex  // Thread-safe access
}

type PackageResult struct {
    Name          string
    Status        string
    Elapsed       time.Duration
    PassedTests   int
    FailedTests   int
    SkippedTests  int
    Output        string
}

type TestResult struct {
    Package       string
    Name          string
    Status        string
    Elapsed       time.Duration
    Output        []string
}
```

**Design Decisions:**
- **Independent Consumer**: Runs in parallel with TUI/Simple Output
- **Thread-Safe**: Uses `sync.RWMutex` for concurrent access
- **Detailed State**: Keeps all test results and output for final summary
- **Shared by All Consumers**: Both TUI and Simple Output use the same collector
- **No Duplication**: Summary logic exists in one place only

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

**Responsibility:** CLI entry point, component wiring, event fan-out

The main function:
- Parses command-line flags
- Sets up input source (stdin or file)
- Creates engine with options (raw/JSON output files)
- **Implements event fan-out** to multiple consumers
- Routes events to appropriate consumer (TUI or simple)
- Manages Summary Collector lifecycle
- Handles exit codes (planned)

**Event Fan-out Pattern:**

```go
// Create engine and get event stream
eng := engine.NewEngine(opts...)
events := eng.Stream(inputSource)

// Create channels for each consumer
tuiEvents := make(chan engine.Event, 100)
summaryEvents := make(chan engine.Event, 100)

// Create and start summary collector
summaryCollector := tui.NewSummaryCollector()
go summaryCollector.ProcessEvents(summaryEvents)

// Fan out events to all consumers
go func() {
    for evt := range events {
        tuiEvents <- evt
        summaryEvents <- evt
    }
    close(tuiEvents)
    close(summaryEvents)
}()

// TUI or Simple Output consumes tuiEvents
// When complete, retrieve summary from collector
summary := tui.ComputeSummary(summaryCollector, 10*time.Second)
```

**Design Decisions:**
- **Event Fan-out**: Broadcasts each event to multiple consumers via separate channels
- **Parallel Consumers**: Summary collector runs independently of display consumers
- **Shared Summary**: Both TUI and Simple Output use the same collector instance
- **Channel Management**: Main is responsible for creating and closing all channels
- **Minimal Logic**: Just wiring, no business logic

## Hybrid Architecture: Minimal vs. Detailed State

The system uses a **hybrid approach** where different consumers maintain different levels of state:

### Minimal State (TUI/Simple Output)

**Purpose:** Real-time display and immediate feedback

**What's Tracked:**
- Running test count (incremented on "run", decremented on "pass"/"fail"/"skip")
- Passed test count
- Failed test count
- Skipped test count
- Recent output lines (for display)

**Benefits:**
- Lightweight memory footprint
- Fast updates for real-time display
- Simple state management
- Can drop old output to save memory

### Detailed State (Summary Collector)

**Purpose:** Comprehensive final summary

**What's Tracked:**
- All test results with full metadata
- All package results with timing
- Complete failure output (up to 10 lines per test)
- Complete skip reasons (up to 3 lines per test)
- Package chronological order
- Slow test detection data

**Benefits:**
- Complete data for final summary
- No information loss
- Shared across all consumers
- Single source of truth for statistics

### Why This Approach?

1. **No Duplication of Complex Logic**: Summary computation exists in one place
2. **Minimal Duplication of Simple State**: Lightweight counters are cheap to duplicate
3. **Independent Evolution**: Can enhance summary without touching TUI rendering
4. **Memory Efficiency**: TUI can drop old output while summary keeps statistics
5. **Testability**: Summary collector can be tested in isolation
6. **Reusability**: Same summary works for TUI and `-notty` modes

## Data Flow

### Event Flow Sequence

1. **Input** → Lines read from stdin or file
2. **Engine** → Parses each line, emits events to channel
3. **Fan-out** → Broadcasts events to multiple consumer channels
4. **Consumers** → Receive events in parallel, update their respective states
5. **Summary** → Retrieved from collector when complete
6. **Output** → Renders/writes final results

### Event Types and Handling

| Event Type | Trigger | Engine Action | TUI Action | Simple Action | Summary Collector Action |
|------------|---------|---------------|------------|---------------|--------------------------|
| `EventRawLine` | Non-JSON line | Copy buffer, emit | Append to output | Append to output | Ignore |
| `EventTest` | Valid JSON | Parse, emit | Update counters | Update stats | Track detailed results |
| `EventError` | Scanner error | Emit error | Display error | Append error | Ignore |
| `EventComplete` | EOF/error | Close channel | Display summary, quit | Display summary | Finalize timing |

### Parallel Event Processing

```
Engine Event → Fan-out ─┬→ TUI Channel → TUI Model (minimal state)
                        │                     ↓
                        │              Update counters only
                        │
                        ├→ Summary Channel → Summary Collector (detailed state)
                        │                          ↓
                        │                   Track all test results
                        │
                        └→ Future Consumer Channels
                                   ↓
                            (JUnit, etc.)

On EventComplete:
    TUI/Simple → summaryCollector.GetSummary() → Format → Display
```

### File Output Flow

When `-outfile` or `-jsonfile` flags are used:
- Engine writes to files **in addition to** emitting events
- Raw output: Every line written (before parsing)
- JSON output: Only successfully parsed test events
- File writing happens synchronously in the engine goroutine

## Thread-Safety Considerations

### Summary Collector Concurrency

The Summary Collector is designed for concurrent access:

**Write Path (Event Processing):**
- `ProcessEvents()` runs in its own goroutine
- Consumes events from dedicated channel
- Updates internal state without locks (single writer)
- Terminates when channel closes

**Read Path (Summary Retrieval):**
- `GetSummary()` can be called from any goroutine
- Uses `sync.RWMutex` for thread-safe reads
- Can be called while `ProcessEvents()` is still running
- Returns snapshot of current state

**Synchronization Pattern:**

```go
type SummaryCollector struct {
    mu sync.RWMutex
    // ... state fields
}

// Single writer (no lock needed in ProcessEvents)
func (sc *SummaryCollector) ProcessEvents(events <-chan Event) {
    for evt := range events {
        // Update state directly (no concurrent writers)
        sc.packages[pkg] = result
    }
}

// Multiple readers (lock required)
func (sc *SummaryCollector) GetSummary() *Summary {
    sc.mu.RLock()
    defer sc.mu.RUnlock()
    // Read state safely
    return summary
}
```

**Why This Works:**
- **Single Writer Guarantee**: Only `ProcessEvents()` modifies state
- **Channel Semantics**: Go channels provide synchronization
- **Read Lock Only**: `GetSummary()` uses read lock for concurrent reads
- **No Write Lock Needed**: Single writer doesn't need write lock for its own updates

**Important Note:** If future features require multiple writers (e.g., manual state updates), write locks would be needed in modification methods.

### Channel Management

**Buffered Channels:**
- All consumer channels use 100-event buffers
- Prevents fan-out goroutine from blocking
- Allows consumers to process at different speeds

**Channel Lifecycle:**
- Main creates all channels
- Fan-out goroutine closes channels when engine completes
- Consumers terminate when their channel closes

**No Deadlocks:**
- Engine never blocks on channel send (buffered)
- Fan-out never blocks (buffered consumer channels)
- Consumers never block (reading from channel)

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

### 1a. Event Fan-out Pattern

**Decision:** Broadcast events to multiple consumers via separate channels

**Rationale:**
- Consumers can process at different speeds
- Easy to add new consumers without modifying existing ones
- Each consumer gets its own buffered channel
- No consumer can block others

**Trade-offs:**
- Small memory overhead (multiple channel buffers)
- Requires goroutine for fan-out logic
- Main is responsible for channel lifecycle management

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

### 6. Hybrid State Management

**Decision:** TUI maintains minimal state, Summary Collector maintains detailed state

**Rationale:**
- TUI needs fast updates for real-time display (lightweight counters)
- Summary needs complete data for final report (all test results)
- Avoids duplicating complex summary logic
- Allows TUI to drop old output while preserving statistics
- Single source of truth for summary data

**Trade-offs:**
- Small duplication of simple counters (acceptable)
- Requires coordination between TUI and Summary Collector
- Summary Collector must be thread-safe

### 7. Independent Summary Collector

**Decision:** Summary Collector runs as independent consumer in parallel

**Rationale:**
- Decouples summary logic from display logic
- Can be shared by TUI and Simple Output
- Testable in isolation
- Can be enhanced without touching TUI code
- Supports future consumers (JUnit, etc.)

**Trade-offs:**
- Requires event fan-out in main
- Requires thread-safe access patterns
- Slightly more complex wiring

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

**Integration:** Add `-junitfile` flag, add to fan-out in `main.go`

```go
// Add JUnit channel to fan-out
junitEvents := make(chan engine.Event, 100)
junitWriter := output.NewJUnitWriter(junitFile)
go junitWriter.ProcessEvents(junitEvents)

// Update fan-out to include JUnit
go func() {
    for evt := range events {
        tuiEvents <- evt
        summaryEvents <- evt
        junitEvents <- evt  // Add new consumer
    }
    close(tuiEvents)
    close(summaryEvents)
    close(junitEvents)  // Close new channel
}()
```

### 2. Multiple Output Formats

**Pattern:** Already implemented via event fan-out

The fan-out pattern in `main.go` makes it trivial to add new consumers:
1. Create channel for new consumer
2. Start consumer goroutine
3. Add channel to fan-out loop
4. Close channel when engine completes

**No changes needed to existing consumers!**

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
