package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output"
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
	events := eng.Stream(inputSource)

	var exitCode int

	if *notty {
		// Simple output mode (no TUI)
		simple := output.NewSimpleOutput(os.Stdout)
		if err := simple.ProcessEvents(events); err != nil {
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
		m := tui.NewModel()
		p := tea.NewProgram(m)

		// Forward engine events to bubbletea
		go func() {
			for evt := range events {
				// Handle raw output lines directly using Println()
				// This prints them above the TUI without mixing with test output
				if evt.Type == engine.EventRawLine {
					p.Println(string(evt.RawLine))
				} else {
					// Send all other events to the model
					p.Send(tui.EngineEventMsg(evt))
				}
			}
		}()

		// Run the program
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}

		// Set exit code based on test failures
		if model, ok := finalModel.(*tui.Model); ok && model.HasFailures() {
			exitCode = 1
		} else {
			exitCode = 0
		}
	}

	os.Exit(exitCode)
}
