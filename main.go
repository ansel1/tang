package main

import (
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
	junitfile := flag.String("junitout", "", "Save cumulative test results to the specified JUnit XML file")
	notty := flag.Bool("notty", false, "Don't use TUI, output to stdout")
	replay := flag.Bool("replay", false, "Replay events with timing from original test run (requires -f)")
	rate := flag.Float64("rate", 1.0, "Replay rate multiplier (0=instant, 1=original speed, 0.5=2x speed)")
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
		// Re-trigger exit to ensure default behavior if not in TUI or if TUI hangs
		// In TUI mode, bubbletea usually catches SIGINT first, but we need to ensure this output happens.
		// However, signal.Notify might intercept it from bubbletea if we aren't careful.
		// Actually, bubbletea handles interrupts internally if not specified otherwise.
		// But in this global handler, we want to ensure we write the file.
		// If the main goroutine exits (defer), we are good.
		// If the user force kills, we might miss it.
		// Let's rely on defer for normal exit, and this goroutine for signals.
		// After writing, we should probably let the program terminate naturally or force it if it's stuck.
		// But TUI also listens for signals. This might compete.
		//
		// Correct approach: output generation is fast. We can do it and let default handler proceed or exit.
		os.Exit(1)
	}()

	var exitCode int

	// Skip TUI if:
	// 1. -notty flag is set, OR
	// 2. -f is used without -replay (reading from file without replay)
	skipTUI := *notty || (*infile != "" && !*replay)

	if skipTUI {
		// Simple output mode (no TUI)
		simple := output.NewSimpleOutput(os.Stdout, collector)
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
		// TUI mode
		var p *tea.Program
		var pDone chan struct{}
		var eventCount int

		printSummary := func() {
			// finish the current run, in case it was interrupted
			collector.Finish()

			lastRun := collector.State().MostRecentRun()
			if lastRun != nil {
				// Reprint non-test output
				for _, line := range lastRun.NonTestOutput {
					fmt.Print(line)
				}
				fmt.Print("\n")
				formatter := format.NewSummaryFormatter(80) // 80 is fallback width, should ideally be dynamic
				summary := format.ComputeSummary(lastRun, 10*time.Second)
				if summary != nil {
					fmt.Println(formatter.Format(summary))
				}
			}

		}

		// Consume events
	EventLoop:
		for evt := range engineEvents {
			collector.Push(evt)

			if p == nil {
				// TUI is NOT running
				// safe to access state without lock

				// Check if run is active
				if collector.State().CurrentRun != nil {
					// Run started! Start TUI.
					m := tui.NewModel(*replay, *rate, collector)
					p = tea.NewProgram(m)
					pDone = make(chan struct{})

					go func() {
						if _, err := p.Run(); err != nil {
							fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
						}
						close(pDone)
					}()
				} else {
					// No run active. Print raw lines directly.
					if evt.Type == engine.EventRawLine {
						fmt.Print(string(evt.RawLine))
					}
				}
			} else {
				// TUI IS running
				// Check if TUI exited unexpectedly (e.g. user pressed q)
				select {
				case <-pDone:
					// TUI finished but run is still active (or we haven't processed the end event yet)
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
					// Run finished! Quit TUI with an empty final render.
					p.Send(tui.QuitMsg{})
					<-pDone
					p = nil
					pDone = nil

					// Print Summary for the finished run
					printSummary()

					// If the run finished because of a raw line, print it now
					if evt.Type == engine.EventRawLine {
						fmt.Print(string(evt.RawLine))
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

		// Ensure TUI is closed if loop finishes
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
