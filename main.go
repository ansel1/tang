package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/internal/termwidth"
	"github.com/ansel1/tang/output"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/output/junit"
	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
	"github.com/charmbracelet/colorprofile"
)

func main() {
	os.Exit(run())
}

func run() int {
	testIdx := scanForTestSubcommand()

	infile := flag.String("f", "", "Read from file instead of stdin")
	outfile := flag.String("outfile", "", "Save all input to the specified file")
	jsonfile := flag.String("jsonfile", "", "Save JSON events to the specified file")
	junitfile := flag.String("junitfile", "", "Save cumulative test results to the specified JUnit XML file")
	notty := flag.Bool("notty", false, "Don't use live UI, output to stdout")
	verbose := flag.Bool("v", false, "Verbose output (show all test output in -notty mode)")
	replay := flag.Bool("replay", false, "Replay events with timing from original test run (requires -f)")
	rate := flag.Float64("rate", 1.0, "Replay rate multiplier (0=instant, 1=original speed, 0.5=2x speed)")
	slowThreshold := flag.Duration("slow-threshold", 10*time.Second, "Duration threshold for slow test detection")
	includeSkipped := flag.Bool("include-skipped", false, "Include skipped tests in summary")
	includeSlow := flag.Bool("include-slow", false, "Include slow tests in summary")
	noColorFlag := flag.Bool("no-color", false, "Disable all ANSI color and style escape codes")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tang [flags] [test [go test flags]]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  test    Run go test and summarize results (auto-adds -json)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	var goTestArgs []string
	var hasVerboseAfterTest bool
	isTestMode := testIdx != -1

	if isTestMode {
		preTestArgs := os.Args[1:testIdx]

		for _, arg := range preTestArgs {
			if arg == "-h" || arg == "-help" || arg == "--help" {
				flag.Usage()
				return 0
			}
		}

		if err := validatePreTestArgs(preTestArgs); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}

		var tangArgs []string
		tangArgs, goTestArgs, hasVerboseAfterTest = splitTestArgs(os.Args[1:])

		os.Args = append([]string{os.Args[0]}, tangArgs...)
	}

	flag.Parse()

	if isTestMode {
		if *infile != "" {
			fmt.Fprintf(os.Stderr, "Error: -f is not compatible with 'test' subcommand\n")
			return 1
		}
		if *replay {
			fmt.Fprintf(os.Stderr, "Error: -replay is not compatible with 'test' subcommand\n")
			return 1
		}
		if *rate != 1.0 {
			fmt.Fprintf(os.Stderr, "Error: -rate is not compatible with 'test' subcommand\n")
			return 1
		}
		if hasVerboseAfterTest {
			*verbose = true
		}
	}

	profile := colorprofile.Detect(os.Stdout, os.Environ())
	if *noColorFlag {
		profile = colorprofile.NoTTY
	}
	noColor := profile == colorprofile.NoTTY

	if !isTestMode {
		if *replay && *infile == "" {
			fmt.Fprintf(os.Stderr, "Error: -replay requires -f <filename>\n")
			return 1
		}
		if *rate < 0 {
			fmt.Fprintf(os.Stderr, "Error: -rate must be >= 0\n")
			return 1
		}
	}

	var inputSource io.Reader
	var goTestCmd *goTestProcess

	if isTestMode {
		proc, err := startGoTest(goTestArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		defer proc.cleanup()
		goTestCmd = proc
		inputSource = proc.stdout
	} else if *infile != "" {
		f, err := os.Open(*infile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()

		if *replay {
			replayReader, err := engine.NewReplayReader(f, *rate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating replay reader: %v\n", err)
				return 1
			}
			inputSource = replayReader
		} else {
			inputSource = f
		}
	} else {
		inputSource = os.Stdin
	}

	var opts []engine.Option

	if *outfile != "" {
		f, err := os.Create(*outfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()
		opts = append(opts, engine.WithRawOutput(f))
	}

	if *jsonfile != "" {
		f, err := os.Create(*jsonfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating JSON file: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()
		opts = append(opts, engine.WithJSONOutput(f))
	}

	eng := engine.NewEngine(opts...)
	engineEvents := eng.Stream(inputSource)

	collector := results.NewCollector()
	if *replay {
		collector.SetReplay(true, *rate)
	}

	var writeJUnitOnce sync.Once
	writeJUnit := func() {
		writeJUnitOnce.Do(func() {
			if *junitfile != "" {
				f, err := os.Create(*junitfile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating JUnit file: %v\n", err)
					return
				}
				defer func() { _ = f.Close() }()

				if err := junit.WriteXML(f, collector.State()); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing JUnit XML: %v\n", err)
				}
			}
		})
	}
	defer writeJUnit()

	var (
		interrupted    atomic.Bool
		shutdownOnce   sync.Once
		shutdownMu     sync.Mutex
		forceKillTimer *time.Timer
		forceKillAfter = 2 * time.Second
	)
	triggerShutdown := func() {
		shutdownOnce.Do(func() {
			interrupted.Store(true)
			shutdownMu.Lock()
			if goTestCmd != nil {
				_ = goTestCmd.signal(os.Interrupt)
				forceKillTimer = time.AfterFunc(forceKillAfter, func() {
					goTestCmd.cleanup()
				})
			}
			shutdownMu.Unlock()
		})
	}
	defer func() {
		shutdownMu.Lock()
		if forceKillTimer != nil {
			forceKillTimer.Stop()
		}
		shutdownMu.Unlock()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		count := 0
		for range sigChan {
			count++
			if count == 1 {
				triggerShutdown()
			} else {
				if goTestCmd != nil {
					goTestCmd.cleanup()
				}
			}
		}
	}()

	var exitCode int

	skipLive := *notty || (*infile != "" && !*replay)

	termWidth := termwidth.Get(os.Stdout.Fd())
	columnsOverride := termwidth.FromEnv()

	summaryOpts := format.SummaryOptions{
		IncludeSkipped: *includeSkipped,
		IncludeSlow:    *includeSlow,
	}

	// A single SimpleOutput renders output identically in both modes. It
	// writes into a buffer which the event loop dumps to stdout whenever the
	// TUI is not on screen (in notty mode, that's after every event).
	var outputBuf bytes.Buffer
	simple := output.NewSimpleOutput(&outputBuf, collector, *slowThreshold, summaryOpts, *verbose, termWidth, noColor)

	// In live mode, the event loop starts (and restarts) the TUI via this
	// callback. In notty mode it stays nil and output streams directly.
	var startTUI func() tuiController
	if !skipLive {
		startTUI = func() tuiController {
			m := tui.NewModel(*replay, *rate, collector)
			m.SlowThreshold = *slowThreshold
			m.OnInterrupt = triggerShutdown
			var progOpts []tea.ProgramOption
			progOpts = append(progOpts, tea.WithColorProfile(profile))
			if columnsOverride > 0 {
				progOpts = append(progOpts, tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
					if ws, ok := msg.(tea.WindowSizeMsg); ok {
						ws.Width = columnsOverride
						return ws
					}
					return msg
				}))
			}
			p := tea.NewProgram(m, progOpts...)
			done := make(chan struct{})

			go func() {
				if _, err := p.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "Error running live UI: %v\n", err)
				}
				close(done)
			}()

			return &teaController{program: p, doneCh: done}
		}
	}

	runEventLoop(engineEvents, collector, simple, &outputBuf, os.Stdout, startTUI)

	if simple.HasFailures() || interrupted.Load() {
		exitCode = 1
	}

	if goTestCmd != nil {
		childExit := goTestCmd.wait()
		if childExit > exitCode {
			exitCode = childExit
		}
	}

	return exitCode
}
