package cmdio

import (
	"context"
	"fmt"
	"sync"
	"time"

	bubblespinner "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerModel is the Bubble Tea model for the spinner.
type spinnerModel struct {
	spinner   bubblespinner.Model
	suffix    string
	quitting  bool
	startTime time.Time // non-zero when elapsed time display is enabled
}

// Message types for spinner updates.
type (
	suffixMsg string
	quitMsg   struct{}
)

// SpinnerOption configures spinner behavior.
type SpinnerOption func(*spinnerModel)

// WithElapsedTime enables an elapsed time prefix (MM:SS) on the spinner.
func WithElapsedTime() SpinnerOption {
	return func(m *spinnerModel) {
		m.startTime = time.Now()
	}
}

// newSpinnerModel creates a new spinner model.
func newSpinnerModel(opts ...SpinnerOption) spinnerModel {
	s := bubblespinner.New()
	// Braille spinner frames with 200ms timing
	s.Spinner = bubblespinner.Spinner{
		Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		FPS:    time.Second / 5, // 200ms = 5 FPS
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	m := spinnerModel{
		spinner: s,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
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

	var result string
	if !m.startTime.IsZero() {
		elapsed := time.Since(m.startTime)
		result += fmt.Sprintf("%02d:%02d ", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
	}
	result += m.spinner.View()
	if m.suffix != "" {
		result += " " + m.suffix
	}
	return result
}

// spinner provides a structured interface for displaying progress indicators.
// Use NewSpinner to create an instance, Update to send status messages,
// and Close to stop the spinner and clean up resources.
//
// The spinner automatically degrades in non-interactive terminals.
// Context cancellation will automatically close the spinner.
type spinner struct {
	p        *tea.Program // nil in non-interactive mode
	c        *cmdIO
	ctx      context.Context
	sendQuit func()
	done     chan struct{} // Closed when tea.Program finishes
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
	sp.sendQuit()
	// Always wait for termination, even if we weren't the first caller
	if sp.p != nil {
		<-sp.done
	}
}

// NewSpinner creates a new spinner for displaying progress.
// The spinner should be closed when done to clean up resources.
//
// Example:
//
//	sp := cmdio.NewSpinner(ctx)
//	defer sp.Close()
//	sp.Update("processing files")
//
// Use WithElapsedTime() to show a running MM:SS prefix:
//
//	sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
func (c *cmdIO) NewSpinner(ctx context.Context, opts ...SpinnerOption) *spinner {
	// Don't show spinner if not interactive
	if !c.capabilities.SupportsInteractive() {
		return &spinner{p: nil, c: c, ctx: ctx, sendQuit: func() {}}
	}

	// Create model and program
	m := newSpinnerModel(opts...)
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
		p:        p,
		c:        c,
		ctx:      ctx,
		sendQuit: sync.OnceFunc(func() { p.Send(quitMsg{}) }),
		done:     done,
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
