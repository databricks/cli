package cmdio

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerModel is the Bubble Tea model for the spinner.
type spinnerModel struct {
	spinner  spinner.Model
	suffix   string
	quitting bool
}

// Message types for spinner updates.
type suffixMsg string
type quitMsg struct{}

// newSpinnerModel creates a new spinner model with exact Brian Downs charset 11.
func newSpinnerModel() spinnerModel {
	s := spinner.New()
	// CharSet 11 from briandowns/spinner: {"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	s.Spinner = spinner.Spinner{
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

	case spinner.TickMsg:
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

// Spinner returns a channel for updating spinner status messages.
// Send messages to update the suffix, close the channel to stop.
// The spinner runs until the channel is closed or context is cancelled.
func (c *cmdIO) Spinner(ctx context.Context) chan string {
	updates := make(chan string)

	// Don't show spinner if not interactive
	if !c.capabilities.SupportsInteractive() {
		// Return channel but don't start program - just drain messages
		go func() {
			for range updates {
				// Discard messages
			}
		}()
		return updates
	}

	// Create model and program
	m := newSpinnerModel()
	p := tea.NewProgram(
		m,
		tea.WithInput(nil),
		tea.WithOutput(c.err),
	)

	// Acquire program slot (queues if another program is running)
	c.acquireTeaProgram(p)

	// Start program in background
	go func() {
		defer c.releaseTeaProgram()
		_, _ = p.Run()
	}()

	// Bridge goroutine: channel -> tea messages
	go func() {
		defer func() {
			p.Send(quitMsg{})
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-updates:
				if !ok {
					// Channel closed
					return
				}
				p.Send(suffixMsg(msg))
			}
		}
	}()

	return updates
}
