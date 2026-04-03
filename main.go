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
	"github.com/ansel1/tang/output"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/output/junit"
	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse command-line flags
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
	flag.Parse()

	// Validate flag combinations
	if *replay && *infile == "" {
		fmt.Fprintf(os.Stderr, "Error: -replay requires -f <filename>\n")
		return 1
	}
	if *rate < 0 {
		fmt.Fprintf(os.Stderr, "Error: -rate must be >= 0\n")
		return 1
	}

	// Setup input source (file or stdin)
	var inputSource io.Reader = os.Stdin
	if *infile != "" {
		f, err := os.Open(*infile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()

		// Wrap with ReplayReader if replay mode is enabled
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
	}

	// Setup engine options
	var opts []engine.Option

	// Raw output file
	if *outfile != "" {
		f, err := os.Create(*outfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()
		opts = append(opts, engine.WithRawOutput(f))
	}

	// JSON output file
	if *jsonfile != "" {
		f, err := os.Create(*jsonfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating JSON file: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()
		opts = append(opts, engine.WithJSONOutput(f))
	}

	// Create engine and start streaming
	eng := engine.NewEngine(opts...)
	engineEvents := eng.Stream(inputSource)

	// Create results collector
	collector := results.NewCollector()
	if *replay {
		collector.SetReplay(true, *rate)
	}

	// Setup JUnit output handling
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

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		writeJUnit()
		// Re-trigger exit to ensure default behavior if not in live mode or if it hangs
		// In live mode, bubbletea usually catches SIGINT first, but we need to ensure this output happens.
		// However, signal.Notify might intercept it from bubbletea if we aren't careful.
		// Actually, bubbletea handles interrupts internally if not specified otherwise.
		// But in this global handler, we want to ensure we write the file.
		// If the main goroutine exits (defer), we are good.
		// If the user force kills, we might miss it.
		// Let's rely on defer for normal exit, and this goroutine for signals.
		// After writing, we should probably let the program terminate naturally or force it if it's stuck.
		// But the live UI also listens for signals. This might compete.
		//
		// Correct approach: output generation is fast. We can do it and let default handler proceed or exit.
		os.Exit(1)
	}()

	var exitCode int

	// Skip live mode if:
	// 1. -notty flag is set, OR
	// 2. -f is used without -replay (reading from file without replay)
	skipLive := *notty || (*infile != "" && !*replay)

	summaryOpts := format.SummaryOptions{
		IncludeSkipped: *includeSkipped,
		IncludeSlow:    *includeSlow,
	}

	if skipLive {
		// Simple output mode (no live UI)
		simple := output.NewSimpleOutput(os.Stdout, collector, *slowThreshold, summaryOpts, *verbose)
		if err := simple.ProcessEvents(engineEvents); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing events: %v\n", err)
			return 1
		}
		// Set exit code based on test failures
		if simple.HasFailures() {
			exitCode = 1
		} else {
			exitCode = 0
		}
	} else {
		// Live mode
		var p *tea.Program
		var pDone chan struct{}
		var eventCount int

		// Buffer go test output to print after live UI exits
		var outputBuf bytes.Buffer
		simpleOut := output.NewSimpleOutput(&outputBuf, collector, *slowThreshold, summaryOpts, *verbose)
		simpleOut.Init()

		printSummary := func() {
			collector.Finish()

			// Print buffered go test output
			simpleOut.Flush()
			if outputBuf.Len() > 0 {
				fmt.Print(outputBuf.String())
			}

			lastRun := collector.State().MostRecentRun()
			if lastRun != nil {
				for _, line := range lastRun.NonTestOutput {
					fmt.Print(line)
				}
				summary := format.ComputeSummary(lastRun, *slowThreshold)
				if summary != nil {
					summaryText := format.NewSummaryFormatter(80, summaryOpts).Format(summary)
					if len(lastRun.NonTestOutput) > 0 || summary.HasTestDetailsWithOptions(summaryOpts) {
						fmt.Print("\n")
					}
					fmt.Println(summaryText)
				}
			}
		}

		// Consume events
	EventLoop:
		for evt := range engineEvents {
			collector.Push(evt)
			if evt.Type != engine.EventRawLine {
				simpleOut.ProcessEvent(evt)
			}

			if p == nil {
				// Live UI is NOT running
				// safe to access state without lock

				// Check if run is active
				if collector.State().CurrentRun != nil {
					// Run started! Start live UI.
					m := tui.NewModel(*replay, *rate, collector)
					m.SlowThreshold = *slowThreshold
					p = tea.NewProgram(m)
					pDone = make(chan struct{})

					go func() {
						if _, err := p.Run(); err != nil {
							fmt.Fprintf(os.Stderr, "Error running live UI: %v\n", err)
						}
						close(pDone)
					}()
				} else {
					// No run active. Print raw lines directly.
					if evt.Type == engine.EventRawLine {
						fmt.Println(string(evt.RawLine))
					}
				}
			} else {
				// Live UI IS running
				// Check if live UI exited unexpectedly (e.g. user pressed q)
				select {
				case <-pDone:
					// Live UI finished but run is still active (or we haven't processed the end event yet)
					// This implies user requested quit.
					// We should exit the whole program.
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
					// Run finished! Quit live UI with an empty final render.
					p.Send(tui.QuitMsg{})
					<-pDone
					p = nil
					pDone = nil

					// Print buffered output and summary for the finished run
					printSummary()

					// Reset for next run
					outputBuf.Reset()
					simpleOut.Init()

					// If the run finished because of a raw line, print it now
					if evt.Type == engine.EventRawLine {
						fmt.Println(string(evt.RawLine))
					}
				} else {
					// Run still running.
					// Send repaint occasionally to keep UI updated
					eventCount++
					if eventCount%50 == 0 {
						p.Send(tui.RepaintMsg{})
					}
				}
			}
		}

		// Ensure live UI is closed if loop finishes
		if p != nil {
			p.Send(tui.QuitMsg{})
			<-pDone
		}

		// Set exit code based on test failures
		for _, run := range collector.State().Runs {
			if run.Counts.Failed > 0 {
				exitCode = 1
				break
			}
		}
	}

	return exitCode
}
