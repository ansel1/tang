package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output"
	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse command-line flags
	infile := flag.String("f", "", "Read from file instead of stdin")
	outfile := flag.String("outfile", "", "Save all input to the specified file")
	jsonfile := flag.String("jsonfile", "", "Save JSON events to the specified file")
	notty := flag.Bool("notty", false, "Don't use TUI, output to stdout")
	replay := flag.Bool("replay", false, "Replay events with timing from original test run (requires -f)")
	rate := flag.Float64("rate", 1.0, "Replay rate multiplier (0=instant, 1=original speed, 0.5=2x speed)")
	flag.Parse()

	// Validate flag combinations
	if *replay && *infile == "" {
		fmt.Fprintf(os.Stderr, "Error: -replay requires -f <filename>\n")
		os.Exit(1)
	}
	if *rate < 0 {
		fmt.Fprintf(os.Stderr, "Error: -rate must be >= 0\n")
		os.Exit(1)
	}

	// Setup input source (file or stdin)
	var inputSource io.Reader = os.Stdin
	if *infile != "" {
		f, err := os.Open(*infile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		// Wrap with ReplayReader if replay mode is enabled
		if *replay {
			replayReader, err := engine.NewReplayReader(f, *rate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating replay reader: %v\n", err)
				os.Exit(1)
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
			os.Exit(1)
		}
		defer f.Close()
		opts = append(opts, engine.WithRawOutput(f))
	}

	// JSON output file
	if *jsonfile != "" {
		f, err := os.Create(*jsonfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating JSON file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
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
			os.Exit(1)
		}
		// Set exit code based on test failures
		if simple.HasFailures() {
			exitCode = 1
		} else {
			exitCode = 0
		}
	} else {
		// TUI mode
		m := tui.NewModel(*replay, *rate, collector)
		p := tea.NewProgram(m)

		// Forward engine events to bubbletea
		go func() {
			var batch []engine.Event
			const maxBatchSize = 50

			// Consume events until channel closes
			for {
				// Block waiting for the first event
				evt, ok := <-engineEvents
				if !ok {
					break
				}

				// Start a new batch
				// Note: We intentionally allocate a new slice backing array here
				// so that we can safely pass it to the TUI (which runs in another goroutine)
				// without race conditions.
				batch = make([]engine.Event, 0, maxBatchSize)
				batch = append(batch, evt)

				// Drain additional available events up to maxBatchSize
			DrainLoop:
				for len(batch) < maxBatchSize {
					select {
					case nextEvt, ok := <-engineEvents:
						if !ok {
							// Channel closed
							if len(batch) > 0 {
								p.Send(tui.EngineEventBatchMsg(batch))
							}
							p.Send(tui.EOFMsg{})
							return
						}

						batch = append(batch, nextEvt)
					default:
						// No more events immediately available
						break DrainLoop
					}
				}

				if len(batch) > 0 {
					p.Send(tui.EngineEventBatchMsg(batch))
					batch = nil
				}
			}

			// Signal completion
			p.Send(tui.EOFMsg{})
		}()

		// Run the program
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}

		// Set exit code based on test failures
		if _, ok := finalModel.(*tui.Model); ok {
			for _, run := range collector.State().Runs {
				if run.Counts.Failed > 0 {
					exitCode = 1
					break
				}
			}
		}
	}

	os.Exit(exitCode)
}
