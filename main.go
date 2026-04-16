package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		if goTestCmd != nil {
			_ = goTestCmd.signal(sig)
		}
		writeJUnit()
		os.Exit(1)
	}()

	var exitCode int

	skipLive := *notty || (*infile != "" && !*replay)

	termWidth := termwidth.Get(os.Stdout.Fd())
	columnsOverride := termwidth.FromEnv()

	summaryOpts := format.SummaryOptions{
		IncludeSkipped: *includeSkipped,
		IncludeSlow:    *includeSlow,
	}

	if skipLive {
		simple := output.NewSimpleOutput(os.Stdout, collector, *slowThreshold, summaryOpts, *verbose, termWidth, noColor)
		if err := simple.ProcessEvents(engineEvents); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing events: %v\n", err)
			return 1
		}
		if simple.HasFailures() {
			exitCode = 1
		}
	} else {
		var p *tea.Program
		var pDone chan struct{}
		var eventCount int

		var outputBuf bytes.Buffer
		simpleOut := output.NewSimpleOutput(&outputBuf, collector, *slowThreshold, summaryOpts, *verbose, termWidth, noColor)
		simpleOut.Init()

		printSummary := func() {
			collector.Finish()

			if *verbose {
				simpleOut.Flush()
				if outputBuf.Len() > 0 {
					fmt.Print(outputBuf.String())
				}
			}

			lastRun := collector.State().MostRecentRun()
			if lastRun != nil {
				for _, line := range lastRun.NonTestOutput {
					fmt.Print(line)
				}
				summary := format.ComputeSummary(lastRun, *slowThreshold)
				if summary != nil {
					summaryText := format.NewSummaryFormatter(termWidth, noColor, summaryOpts).Format(summary)
					if len(lastRun.NonTestOutput) > 0 || summary.HasTestDetailsWithOptions(summaryOpts) {
						fmt.Print("\n")
					}
					fmt.Println(summaryText)
				}
			}
		}

	EventLoop:
		for evt := range engineEvents {
			collector.Push(evt)
			if evt.Type != engine.EventRawLine {
				simpleOut.ProcessEvent(evt)
			}

			if p == nil {
				if collector.State().CurrentRun != nil {
					m := tui.NewModel(*replay, *rate, collector)
					m.SlowThreshold = *slowThreshold
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
					p = tea.NewProgram(m, progOpts...)
					pDone = make(chan struct{})

					go func() {
						if _, err := p.Run(); err != nil {
							fmt.Fprintf(os.Stderr, "Error running live UI: %v\n", err)
						}
						close(pDone)
					}()
				} else {
					if evt.Type == engine.EventRawLine {
						fmt.Println(string(evt.RawLine))
					}
				}
			} else {
				select {
				case <-pDone:
					printSummary()
					p = nil
					pDone = nil
					break EventLoop
				default:
				}

				collector.Lock()
				currentRun := collector.State().CurrentRun
				collector.Unlock()

				if currentRun == nil {
					p.Send(tui.QuitMsg{})
					<-pDone
					p = nil
					pDone = nil

					printSummary()

					outputBuf.Reset()
					simpleOut.Init()

					if evt.Type == engine.EventRawLine {
						fmt.Println(string(evt.RawLine))
					}
				} else {
					eventCount++
					if eventCount%50 == 0 {
						p.Send(tui.RepaintMsg{})
					}
				}
			}
		}

		if p != nil {
			p.Send(tui.QuitMsg{})
			<-pDone
		}

		for _, run := range collector.State().Runs {
			if run.Counts.Failed > 0 {
				exitCode = 1
				break
			}
		}
	}

	if goTestCmd != nil {
		childExit := goTestCmd.wait()
		if childExit > exitCode {
			exitCode = childExit
		}
	}

	return exitCode
}
