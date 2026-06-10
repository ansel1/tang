package main

import (
	"bytes"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output"
	"github.com/ansel1/tang/results"
	"github.com/ansel1/tang/tui"
)

// tuiController abstracts the live TUI program so the event loop can be
// driven in tests without a terminal.
type tuiController interface {
	// Send delivers a message to the running program.
	Send(msg tea.Msg)
	// Done returns a channel that is closed when the program has exited.
	Done() <-chan struct{}
}

// teaController adapts a bubbletea program to the tuiController interface.
type teaController struct {
	program *tea.Program
	doneCh  chan struct{}
}

func (t *teaController) Send(msg tea.Msg)      { t.program.Send(msg) }
func (t *teaController) Done() <-chan struct{} { return t.doneCh }

// runEventLoop consumes engine events and renders output through a single
// SimpleOutput, in both notty and live modes.
//
// All rendering goes through simple, which writes into buf. After each
// event, if no TUI is on screen, buf is dumped to stdout. So notty mode
// effectively streams directly, and live mode diverts output to the buffer
// only while the TUI owns the screen, dumping it when the TUI closes. This
// makes the final terminal contents identical in both modes.
//
// startTUI is nil in notty mode. In live mode it starts the TUI program,
// which is quit whenever a run finishes and restarted when a new run
// starts. A summary block is written after every run in both modes.
//
// If the user quits the TUI, the in-progress run is finished and
// summarized, and the loop stops without processing further events.
func runEventLoop(
	events <-chan engine.Event,
	collector *results.Collector,
	simple *output.SimpleOutput,
	buf *bytes.Buffer,
	stdout io.Writer,
	startTUI func() tuiController,
) {
	simple.Init()

	flushBuf := func() {
		if buf.Len() > 0 {
			_, _ = stdout.Write(buf.Bytes())
			buf.Reset()
		}
	}

	// finishRun flushes any remaining buffered package output and writes
	// summaries for any newly-finished runs. Idempotent.
	finishRun := func() {
		simple.Flush()
		simple.WriteRunSummaries()
	}

	currentRun := func() *results.Run {
		collector.Lock()
		defer collector.Unlock()
		return collector.State().CurrentRun
	}

	var p tuiController
	var eventCount int

EventLoop:
	for evt := range events {
		prevRun := currentRun()

		collector.Push(evt)

		if p != nil {
			select {
			case <-p.Done():
				// User quit the TUI: finish and summarize the partial run,
				// then stop processing.
				p = nil
				collector.Finish()
				finishRun()
				flushBuf()
				break EventLoop
			default:
			}
		}

		run := currentRun()

		// Did this event finish a run? (Raw lines and stream completion
		// force the collector to finish the current run.)
		if prevRun != nil && run == nil {
			if p != nil {
				p.Send(tui.QuitMsg{})
				<-p.Done()
				p = nil
			}
			finishRun()
		}

		simple.ProcessEvent(evt)

		if p == nil {
			flushBuf()
			if run != nil && startTUI != nil {
				p = startTUI()
			}
		} else {
			eventCount++
			if eventCount%50 == 0 {
				p.Send(tui.RepaintMsg{})
			}
		}
	}

	// Stream ended (or was interrupted): close the TUI and emit any
	// remaining output and summaries.
	if p != nil {
		p.Send(tui.QuitMsg{})
		<-p.Done()
		p = nil
	}
	collector.Finish()
	finishRun()
	flushBuf()
}
