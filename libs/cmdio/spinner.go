package cmdio

import (
	"context"
	"sync"
	"time"

	bubblespinner "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerModel is the Bubble Tea model for the spinner.
type spinnerModel struct {
	spinner  bubblespinner.Model
	suffix   string
	quitting bool
}

// Message types for spinner updates.
type (
	suffixMsg string
	quitMsg   struct{}
)

// newSpinnerModel creates a new spinner model.
func newSpinnerModel() spinnerModel {
	s := bubblespinner.New()
	// Braille spinner frames with 200ms timing
	s.Spinner = bubblespinner.Spinner{
		Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		FPS:    time.Second / 5, // 200ms = 5 FPS
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	return spinnerModel{
		spinner:  s,
		suffix:   "",
		quitting: false,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case suffixMsg:
		m.suffix = string(msg)
		return m, nil

	case quitMsg:
		m.quitting = true
		return m, tea.Quit

	case bubblespinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}

	if m.suffix != "" {
		return m.spinner.View() + " " + m.suffix
	}

	return m.spinner.View()
}

// spinner provides a structured interface for displaying progress indicators.
// Use NewSpinner to create an instance, Update to send status messages,
// and Close to stop the spinner and clean up resources.
//
// The spinner automatically degrades in non-interactive terminals.
// Context cancellation will automatically close the spinner.
type spinner struct {
	p    *tea.Program // nil in non-interactive mode
	c    *cmdIO
	ctx  context.Context
	once sync.Once
	done chan struct{} // Closed when tea.Program finishes
}

// Update sends a status message to the spinner.
// This operation sends directly to the tea.Program.
func (sp *spinner) Update(msg string) {
	if sp.p != nil {
		sp.p.Send(suffixMsg(msg))
	}
}

// Close stops the spinner and releases resources.
// It waits for the spinner to fully terminate before returning.
// It is safe to call Close multiple times and from multiple goroutines.
func (sp *spinner) Close() {
	sp.once.Do(func() {
		if sp.p != nil {
			sp.p.Send(quitMsg{})
			// Wait for tea.Program to finish
			<-sp.done
		}
	})
}

// NewSpinner creates a new spinner for displaying progress.
// The spinner should be closed when done to clean up resources.
//
// Example:
//
//	sp := cmdio.NewSpinner(ctx)
//	defer sp.Close()
//	sp.Update("processing files")
func (c *cmdIO) NewSpinner(ctx context.Context) *spinner {
	// Don't show spinner if not interactive
	if !c.capabilities.SupportsInteractive() {
		return &spinner{p: nil, c: c, ctx: ctx}
	}

	// Create model and program
	m := newSpinnerModel()
	p := tea.NewProgram(
		m,
		tea.WithInput(nil),
		tea.WithOutput(c.err),

		// Note: We don't let tea capture signals to match current behavior.
		// This allows Ctrl-C to immediately terminate instead of being captured by Bubble Tea.
		tea.WithoutSignalHandler(),
	)

	// Acquire program slot (queues if another program is running)
	c.acquireTeaProgram(p)

	done := make(chan struct{})
	sp := &spinner{
		p:    p,
		c:    c,
		ctx:  ctx,
		done: done,
	}

	// Start program in background
	go func() {
		_, _ = p.Run()
		c.releaseTeaProgram()
		close(done)
	}()

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		sp.Close()
	}()

	return sp
}

// Spinner returns a channel for updating spinner status messages.
// Send messages to update the suffix, close the channel to stop.
// The spinner runs until the channel is closed or context is cancelled.
func (c *cmdIO) Spinner(ctx context.Context) chan string {
	updates := make(chan string)
	sp := c.NewSpinner(ctx)

	// Bridge goroutine: channel -> spinner.Update()
	go func() {
		defer sp.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-updates:
				if !ok {
					// Channel closed
					return
				}
				sp.Update(msg)
			}
		}
	}()

	return updates
}
