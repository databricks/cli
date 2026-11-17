package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// keysExitCtrlC is the number of Ctrl+C presses that end the keys program.
// A single press is recorded like any other key so it can be inspected.
const keysExitCtrlC = 2

// keysModel prints one line per key message. It draws nothing itself so the
// output stays a plain log of decoded input. Lines are written directly from
// Update: tea.Println runs its commands concurrently, so the order is not kept.
type keysModel struct {
	out   io.Writer
	eol   string
	ctrlC int
}

func (m *keysModel) Init() tea.Cmd { return nil }

func (m *keysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	line := fmt.Sprintf("key=%q type=%d runes=%q alt=%t paste=%t",
		key.String(), key.Type, string(key.Runes), key.Alt, key.Paste)
	fmt.Fprint(m.out, line+m.eol)
	if key.Type == tea.KeyCtrlC {
		m.ctrlC++
		if m.ctrlC >= keysExitCtrlC {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *keysModel) View() string { return "" }

func newKeysCmd() *cobra.Command {
	var (
		raw     bool
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Print each key message bubbletea decodes from stdin",
		Long: `Print one line per key message that bubbletea decodes from stdin.

Uses the same tea.Program options as cmdio's prompts, so the output shows how
this terminal encodes Esc, Alt+x, Shift+Enter, Ctrl+H vs Backspace, AltGr and
bracketed paste. Exits on two Ctrl+C presses or after --timeout.

With --raw, bubbletea is bypassed: when stdin is a terminal it is put in raw
mode, and each chunk of bytes read is printed as hex and quoted text.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			in := cmd.InOrStdin()
			out := cmd.ErrOrStderr()
			if raw {
				return runRawKeys(ctx, in, out, timeout)
			}
			// Mirrors cmdio's runTUI options. cmdio does not expose a way to run a
			// custom model, so keep these in sync with libs/cmdio/io.go.
			// Raw mode clears OPOST, so "\n" would not return the carriage.
			eol := "\n"
			if _, ok := rawInputFile(ctx, in); ok {
				eol = "\r\n"
			}
			p := tea.NewProgram(&keysModel{out: out, eol: eol},
				tea.WithInput(in),
				tea.WithOutput(out),
				tea.WithoutSignalHandler(),
			)
			if timeout > 0 {
				timer := time.AfterFunc(timeout, p.Quit)
				defer timer.Stop()
			}
			_, err := p.Run()
			return err
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "bypass bubbletea and print the raw bytes read from stdin")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "exit after this duration (0 waits for Ctrl+C)")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if timeout < 0 {
			return fmt.Errorf("--timeout must not be negative, got %s", timeout)
		}
		return nil
	}
	return cmd
}

// rawInputFile returns stdin as a file when it is a terminal. bubbletea and the
// --raw path put that file in raw mode, which is why output needs "\r\n".
func rawInputFile(ctx context.Context, in io.Reader) (*os.File, bool) {
	f, ok := in.(*os.File)
	return f, ok && cmdio.IsInputTTY(ctx)
}

// runRawKeys reads stdin in raw mode without bubbletea. After an early return
// the reader goroutine can be blocked on Read or on sending a chunk; the
// process exits once the command finishes, so it does not need to be joined.
func runRawKeys(ctx context.Context, in io.Reader, out io.Writer, timeout time.Duration) error {
	// Raw mode clears OPOST, so "\n" would not return the carriage. Print
	// "\r\n" in that case so each line starts at column 0.
	eol := "\n"
	if f, ok := rawInputFile(ctx, in); ok {
		state, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return err
		}
		defer func() { _ = term.Restore(int(f.Fd()), state) }()
		eol = "\r\n"
	}

	chunks := make(chan []byte)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := in.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				chunks <- chunk
			}
			if err != nil {
				readErr <- err
				return
			}
		}
	}()

	var timeoutC <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timeoutC = timer.C
	}

	ctrlC := 0
	for {
		select {
		case chunk := <-chunks:
			fmt.Fprintf(out, "raw bytes=% x text=%q%s", chunk, string(chunk), eol)
			for _, b := range chunk {
				if b == 0x03 {
					ctrlC++
				}
			}
			if ctrlC >= keysExitCtrlC {
				return nil
			}
		case err := <-readErr:
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		case <-timeoutC:
			return nil
		}
	}
}
