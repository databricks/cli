// Package termtest drives cmdio's interactive prompts as a black box: it
// wires a [cmdio.NewTestIOWithOptions] up to an OS pipe for input and
// in-memory buffers for output, calls the public entry points
// ([cmdio.RunPrompt], [cmdio.Secret], [cmdio.SelectOrdered], …) in a
// goroutine, and exposes the rendered screen via an in-process VT emulator.
//
// Tests neither construct the underlying tea.Model nor reach into cmdio's
// internals — they hand opts in and read (value, err) out, the same way a
// real command does.
//
// Typical usage:
//
//	tm := termtest.NewPrompt(t, cmdio.PromptOptions{Label: "Workspace name"})
//	tm.WaitFor("Workspace name")
//	tm.Golden("01-empty")
//	tm.Type("hello")
//	tm.Golden("02-after-typing")
//	tm.Type(termtest.KeyEnter)
//	value, err := tm.Result()
//
// # Golden files
//
// Golden(step) compares the rendered screen against
// testdata/<TestName>/<step>.golden. The screen comes from the emulator —
// what the user would see — not the raw escape-sequence stream, so the
// goldens are stable across changes to escape sequences as long as the
// visible UI matches. The styled, modes and queries goldens record those
// sequences on purpose. To create or refresh goldens, pass -update to `go test`
// (the flag is registered by [testdiff.OverwriteMode]).
package termtest

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/require"
)

// Key constants — raw byte sequences as they'd arrive on a terminal's stdin.
// Bubbletea's input parser converts them into the same KeyMsg values a real
// keypress would produce. Pass them straight to (*Term).Type — the bytes go
// down the same pipe as ordinary typed text.
const (
	KeyEnter     = "\r"
	KeyTab       = "\t"
	KeyBackspace = "\x7f"
	KeyEsc       = "\x1b"
	KeyCtrlC     = "\x03"
	KeyCtrlD     = "\x04"
	KeyUp        = "\x1b[A"
	KeyDown      = "\x1b[B"
	KeyRight     = "\x1b[C"
	KeyLeft      = "\x1b[D"
	KeyHome      = "\x1b[H"
	KeyEnd       = "\x1b[F"
	KeyDelete    = "\x1b[3~"
	KeyCtrlA     = "\x01"
	KeyCtrlB     = "\x02"
	KeyCtrlE     = "\x05"
	KeyCtrlF     = "\x06"
	KeyCtrlH     = "\x08"
	KeyCtrlJ     = "\x0a"
	KeyCtrlN     = "\x0e"
	KeyCtrlP     = "\x10"
	KeyCtrlU     = "\x15"
	KeyCtrlW     = "\x17"

	// Modified keys in the encodings terminals send for them. Legacy
	// terminals prefix Alt combinations with Esc; terminals with the kitty
	// keyboard protocol or xterm's modifyOtherKeys enabled send CSI forms.
	KeyAltEnter                 = "\x1b\r"
	KeyAltLeft                  = "\x1b[1;3D"
	KeyAltF                     = "\x1bf"
	KeyShiftEnterKitty          = "\x1b[13;2u"
	KeyShiftEnterModifyOtherKey = "\x1b[27;2;13~"
	KeyEscKitty                 = "\x1b[27u"
)

// Paste wraps text in bracketed-paste markers, the way a terminal delivers a
// paste when the program has enabled bracketed paste mode.
func Paste(text string) string {
	return "\x1b[200~" + text + "\x1b[201~"
}

const (
	// defaultTimeout bounds how long any single wait (Golden,
	// WaitFor, Close, Result) blocks before declaring failure. Generous
	// so heavy CI load doesn't fail tests that would otherwise pass — if
	// the test is going to fail it fails anyway, just a bit slower.
	defaultTimeout = 5 * time.Second
	pollInterval   = 5 * time.Millisecond

	// updateSettleWait is the fixed delay Golden() waits in update mode
	// before snapshotting. Update mode is interactive (the author has
	// passed -update to regenerate goldens), so a small fixed sleep is
	// enough — if the result looks mid-frame the author re-runs.
	updateSettleWait = 200 * time.Millisecond

	// escDisambiguationDelay is how long [Term.Type] waits after writing a
	// lone Esc. The kernel pipe doesn't preserve write boundaries, so a
	// follow-up Type can land in the same bubbletea Read and the buffer
	// "\x1ba" gets parsed as Alt+a (bubbletea key.go: any "\x1b" with a
	// trailing byte is an Alt-modifier; only a buffer of exactly "\x1b"
	// becomes KeyEscape). KeyAltEnter and KeyAltF are Esc-prefixed too, but
	// they are written in one Type call, so the Esc is never alone in a read.
	// Other Key* constants are single non-Esc bytes or complete "\x1b[..."
	// CSI sequences, whose terminator commits the parse before any follow-up
	// byte matters.
	// bubbletea v2 additionally holds a lone Esc until ultraviolet's
	// EscTimeout expires, and that timer starts when the reader receives the
	// byte, not when Type writes it. bubbletea leaves the timeout at
	// ultraviolet's DefaultEscTimeout of 50 ms and doesn't expose it
	// (charm.land/bubbletea/v2 v2.1.1 tty.go:75; ultraviolet
	// terminal_reader.go:27,49). 100 ms, twice that, leaves room for a
	// reader running late under CI load. v1 has no such timeout, so there it
	// only needs to beat the next write.
	escDisambiguationDelay = 100 * time.Millisecond

	// settleQuiet is how long [Term.Settle] waits for output to stop. A
	// keystroke's redraw lands well within it.
	settleQuiet = 50 * time.Millisecond
)

// resultMsg carries the (value, err) the goroutine running the cmdio entry
// point produced. T is string for RunPrompt / Secret / SelectOrdered and int
// for RunSelect.
type resultMsg[T any] struct {
	value T
	err   error
}

// Term wraps a goroutine running a cmdio prompt/select call. Stdin is an
// OS-level pipe we write to (bubbletea's cancelreader only knows how to
// unblock os.File-backed inputs cleanly on Quit; io.Pipe would leak the
// input goroutine); cmdio's stderr (where the TUI renders) is a
// synchronized buffer we drain into the VT emulator on demand.
type Term[T any] struct {
	t    *testing.T
	opts Options

	inW *os.File
	inR *os.File

	done     chan resultMsg[T]
	finished chan struct{}

	mu      sync.Mutex
	out     bytes.Buffer
	term    *emulator
	raw     bytes.Buffer
	pending []byte

	// Query responder state; see [Responder].
	answered  int
	replies   sync.WaitGroup
	replyMu   sync.Mutex
	lastReply chan struct{}
	inWMu     sync.Mutex

	// leftover holds the input bytes the program left unread on exit, when
	// [Options.CaptureLeftover] is set.
	leftover    []byte
	leftoverErr error
}

// Options adjusts [RunWith]. The zero value gives the same harness as [Run].
type Options struct {
	// Env sets environment variables in the command's context (see
	// libs/env). An empty value reads as unset to cmdio's color checks.
	Env map[string]string

	// ColorFromEnv derives cmdio's color capability from NO_COLOR and TERM
	// in Env, the way a real command does, instead of forcing it on.
	ColorFromEnv bool

	// StdoutNotTTY and StderrNotTTY make cmdio treat the stream as
	// redirected.
	StdoutNotTTY bool
	StderrNotTTY bool

	// CaptureStdout feeds cmdio's stdout into the emulator as well, for
	// programs that render there through cmdio (the pager). Both streams then
	// share one screen, as on a real terminal. Streams gives a program that
	// takes its streams as arguments (tableview) the emulator directly.
	CaptureStdout bool

	// HeaderTemplate and Template are the command's text templates;
	// [cmdio.RenderIterator] only pages when Template is set.
	HeaderTemplate string
	Template       string

	// Responder, when non-nil, answers terminal queries the program writes.
	Responder *Responder

	// CaptureLeftover reads the input bytes still unread after the program
	// returns; see [Term.Leftover]. Unsupported on Windows.
	CaptureLeftover bool

	// MapNewline makes the emulator treat LF as CR LF, the way a terminal in
	// cooked mode shows plain lines written to it (cmdio.LogString output).
	// Off Windows the emulator already does, so it only has an effect on
	// Windows.
	MapNewline bool

	// StdinTTY makes stdin look like a terminal to cmdio, so the paths that
	// need one run, such as NewTeaProgram's wait for the terminal's replies
	// before quitting. Input still comes from the pipe.
	StdinTTY bool

	// ScreenRows, when positive, replaces the emulator's height of
	// [cmdio.TestTerminalHeight] rows. LF on the last row scrolls the top
	// line off the screen, so programs that print more lines than that (the
	// pager draining several pages) need a taller screen to keep every line.
	// The window size the program is told about is unchanged.
	ScreenRows int

	// Normalize, when non-nil, rewrites every screen snapshot (Snapshot and
	// SnapshotStyled, and the goldens made from them) before it is compared or
	// written, e.g. to blank out a spinner frame that changes over time. The
	// modes and queries logs are not rewritten.
	Normalize func(string) string
}

// Responder answers the terminal queries a program writes (see
// [Term.SnapshotQueries] for their names) by writing replies to its input,
// after Delay. A real terminal answers within milliseconds locally and much
// later over a slow link; a long Delay models the latter.
type Responder struct {
	Delay time.Duration

	// Replies overrides the reply for a query name. Queries without an
	// entry get [DefaultReply].
	Replies map[string]string
}

// DefaultReply returns the reply a typical xterm-compatible terminal sends
// for a query name, or "" for an unknown one: DECRQM modes are reported as
// "reset", the kitty keyboard flags as 0, the foreground as white, the
// background as black and the cursor position as the top-left corner.
func DefaultReply(query string) string {
	if mode, ok := strings.CutPrefix(query, "DECRQM ?"); ok {
		return "\x1b[?" + mode + ";2$y"
	}
	switch query {
	case "kitty-keyboard-flags":
		return "\x1b[?0u"
	// OSC 10 and OSC 11 are the foreground and background color queries.
	case "OSC 10":
		return "\x1b]10;rgb:ffff/ffff/ffff\x1b\\"
	case "OSC 11":
		return "\x1b]11;rgb:0000/0000/0000\x1b\\"
	case "CPR":
		return "\x1b[1;1R"
	case "DA1":
		return "\x1b[?62;22c"
	}
	return ""
}

type streamsKey struct{}

type streams struct {
	in  io.Reader
	out io.Writer
}

// Streams returns the input and output the harness wired into ctx, for
// programs that take their streams as arguments instead of reading them from
// cmdio (e.g. libs/tableview). Output goes to the emulator.
func Streams(ctx context.Context) (io.Reader, io.Writer) {
	s := ctx.Value(streamsKey{}).(streams)
	return s.in, s.out
}

// NewPrompt runs [cmdio.RunPrompt] in a goroutine and returns a harness for
// it.
func NewPrompt(t *testing.T, opts cmdio.PromptOptions) *Term[string] {
	t.Helper()
	return Run(t, func(ctx context.Context) (string, error) {
		return cmdio.RunPrompt(ctx, opts)
	})
}

// NewSecret runs [cmdio.Secret] in a goroutine and returns a harness for it.
func NewSecret(t *testing.T, label string) *Term[string] {
	t.Helper()
	return Run(t, func(ctx context.Context) (string, error) {
		return cmdio.Secret(ctx, label)
	})
}

// NewSelect runs [cmdio.RunSelect] in a goroutine and returns a harness for
// it.
func NewSelect(t *testing.T, opts cmdio.SelectOptions) *Term[int] {
	t.Helper()
	return Run(t, func(ctx context.Context) (int, error) {
		return cmdio.RunSelect(ctx, opts)
	})
}

// NewSelectOrdered runs [cmdio.SelectOrdered] in a goroutine and returns a
// harness for it.
func NewSelectOrdered(t *testing.T, items []cmdio.Tuple, label string) *Term[string] {
	t.Helper()
	return Run(t, func(ctx context.Context) (string, error) {
		return cmdio.SelectOrdered(ctx, items, label)
	})
}

// Run drives an interactive command through the same terminal harness as the prompts.
func Run[T any](t *testing.T, fn func(ctx context.Context) (T, error)) *Term[T] {
	t.Helper()
	return RunWith(t, Options{}, fn)
}

// RunWith is [Run] with the adjustments in opts.
func RunWith[T any](t *testing.T, opts Options, fn func(ctx context.Context) (T, error)) *Term[T] {
	t.Helper()
	inR, inW, err := os.Pipe()
	require.NoError(t, err, "open input pipe")
	tt := &Term[T]{
		t:        t,
		opts:     opts,
		inW:      inW,
		inR:      inR,
		done:     make(chan resultMsg[T], 1),
		finished: make(chan struct{}),
		term:     newEmulator(cmdio.TestTerminalWidth, screenRows(opts.ScreenRows)),
	}
	// MapNewline only turns LF mapping on: newEmulator already maps LF to CR LF
	// off Windows, which bubbletea v2 relies on when its input is a pipe.
	if opts.MapNewline {
		tt.term.mapNewline = true
	}
	ctx := t.Context()
	for k, v := range opts.Env {
		ctx = env.Set(ctx, k, v) //nolint:fatcontext // one value per variable, set once before the run
	}
	screen := &syncWriter{mu: &tt.mu, buf: &tt.out}
	var stdout io.Writer
	if opts.CaptureStdout {
		stdout = screen
	} else {
		stdout = io.Discard
	}
	var in io.Reader = inR
	if opts.StdinTTY {
		in = cmdio.FakeTTYReader(inR)
	}
	cio := cmdio.NewTestIOWithOptions(ctx, in, stdout, screen, cmdio.TestIOOptions{
		HeaderTemplate: opts.HeaderTemplate,
		Template:       opts.Template,
		ColorFromEnv:   opts.ColorFromEnv,
		StdoutNotTTY:   opts.StdoutNotTTY,
		StderrNotTTY:   opts.StderrNotTTY,
	})
	ctx = cmdio.InContext(ctx, cio)
	ctx = context.WithValue(ctx, streamsKey{}, streams{in: inR, out: screen})
	if opts.Responder != nil {
		go tt.respondLoop()
	}
	go func() {
		v, err := fn(ctx)
		if opts.CaptureLeftover {
			tt.captureLeftover()
		}
		tt.done <- resultMsg[T]{value: v, err: err}
		// Deliver the result before closing inR: on Windows bubbletea cannot
		// cancel a read on a pipe, so its read loop can stay blocked in
		// ReadFile on inR and inR.Close waits for that read until Close
		// closes inW.
		_ = inR.Close()
		close(tt.finished)
	}()
	t.Cleanup(tt.Close)
	return tt
}

// screenRows returns the emulator height for ScreenRows: the default unless
// the value is positive, so a negative value cannot size a grid.
func screenRows(n int) int {
	if n > 0 {
		return n
	}
	return cmdio.TestTerminalHeight
}

// syncWriter is the io.Writer cmdio gives bubbletea as stderr. Writes from
// the program goroutine race with our drain calls, so the buffer is mutex-
// guarded.
type syncWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

// TypeBurst writes keys into the program's stdin in a single write, the way
// input arrives when the user types faster than the program reads (over a
// slow link, from an IME commit, or from an unbracketed paste).
func (tt *Term[T]) TypeBurst(keys ...string) {
	tt.t.Helper()
	tt.write(strings.Join(keys, ""))
}

func (tt *Term[T]) write(s string) {
	tt.t.Helper()
	tt.inWMu.Lock()
	defer tt.inWMu.Unlock()
	_, err := tt.inW.Write([]byte(s))
	if err == nil {
		return
	}
	// A write fails once the program has returned and its end of the pipe is
	// closed. The keys are lost, as on a real terminal, and the goldens that
	// follow show the early exit. The pipe is closed just before finished,
	// so wait for it.
	select {
	case <-tt.finished:
		return
	case <-time.After(defaultTimeout):
	}
	require.NoError(tt.t, err, "write to stdin pipe")
}

// Type writes a string into the program's stdin. The string can be ordinary
// text or one of the Key* constants — they're all just bytes the user would
// have typed on a real terminal. After a lone Esc the call sleeps for
// [escDisambiguationDelay] so a follow-up Type lands in a separate bubbletea
// Read; see the constant's comment for why that matters.
func (tt *Term[T]) Type(s string) {
	tt.t.Helper()
	tt.write(s)
	if s == KeyEsc {
		time.Sleep(escDisambiguationDelay)
	}
}

// Settle waits until the program has written nothing for settleQuiet, or has
// returned. Call it between writes whose meaning changes when the program
// reads them together, e.g. "/" and the search text after it, which a single
// read would deliver as one "/it" key.
func (tt *Term[T]) Settle() {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	last, quietSince := len(tt.Raw()), time.Now()
	for time.Since(quietSince) < settleQuiet {
		select {
		case <-tt.finished:
			return
		case <-time.After(pollInterval):
		}
		if n := len(tt.Raw()); n != last {
			last, quietSince = n, time.Now()
		}
		if time.Now().After(deadline) {
			tt.t.Fatalf("Settle: program still writing or reading input after %s", defaultTimeout)
		}
	}
}

// WaitForExit waits up to grace for the program to return and reports whether
// it did. It never fails, so it can wait on a key that a program may ignore.
func (tt *Term[T]) WaitForExit(grace time.Duration) bool {
	select {
	case <-tt.finished:
		return true
	case <-time.After(grace):
		return false
	}
}

// WaitForReplies blocks until every query the program has written so far has
// been answered: the replies are written to the program's input, which may not
// have read them yet. Call it before typing when the test must not interleave
// keys with replies. It needs Options.Responder.
func (tt *Term[T]) WaitForReplies() {
	tt.t.Helper()
	tt.drainPending()
	tt.scheduleReplies()
	tt.replyMu.Lock()
	last := tt.lastReply
	tt.replyMu.Unlock()
	if last != nil {
		select {
		case <-last:
		case <-time.After(defaultTimeout):
			tt.t.Fatalf("WaitForReplies: replies not written within %s", defaultTimeout)
		}
	}
}

// WaitForScreen blocks until substr is visible on the rendered screen, or
// the default timeout elapses. Unlike [Term.WaitFor], which matches anything
// ever written, it only matches what is on screen now, e.g. a spinner that
// replaced a prompt.
func (tt *Term[T]) WaitForScreen(substr string) {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		got := tt.Snapshot()
		if strings.Contains(got, substr) {
			return
		}
		if time.Now().After(deadline) {
			tt.t.Fatalf("WaitForScreen(%q) timed out after %s\n--- current screen ---\n%s", substr, defaultTimeout, got)
		}
		time.Sleep(pollInterval)
	}
}

// WaitForOrExit is [Term.WaitFor] for programs that may return before
// rendering anything, e.g. a prompt that is not supported in the current
// environment. It reports whether substr appeared; false means the program
// returned first.
func (tt *Term[T]) WaitForOrExit(substr string) bool {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		if strings.Contains(tt.Raw(), substr) {
			return true
		}
		select {
		case <-tt.finished:
			return strings.Contains(tt.Raw(), substr)
		default:
		}
		if time.Now().After(deadline) {
			tt.t.Fatalf("WaitForOrExit(%q) timed out after %s\n--- captured raw output ---\n%s", substr, defaultTimeout, tt.Raw())
		}
		time.Sleep(pollInterval)
	}
}

// WaitFor blocks until substr appears in the raw output stream, or the
// default timeout elapses. Returns the captured output for use in failure
// messages.
func (tt *Term[T]) WaitFor(substr string) string {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		got := tt.Raw()
		if strings.Contains(got, substr) {
			return got
		}
		if time.Now().After(deadline) {
			tt.t.Fatalf("WaitFor(%q) timed out after %s\n--- captured raw output ---\n%s", substr, defaultTimeout, got)
		}
		time.Sleep(pollInterval)
	}
}

// Snapshot returns the current emulator screen contents with trailing
// whitespace trimmed from each line and trailing empty lines dropped.
func (tt *Term[T]) Snapshot() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	lines := strings.Split(tt.term.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return tt.normalize(strings.Join(lines, "\n"))
}

// SnapshotStyled returns the screen like [Term.Snapshot], with the cell
// styles as inline markers: `{fg=4 bold}text{/}`. `{attrs}` opens a styled
// span and `{/}` returns to the default style; a change between two styles
// emits just the new opening marker. Spans never cross a newline, and a
// literal `{` is written as `{{`. Attributes come in a fixed order (fg, bg,
// bold, faint, italic, underline, blink, reverse, strikethrough, overline),
// and colors keep the form the program sent: `4` (16-color index, bright
// colors as 8-15), `@208` (256-color) or `#8fb3dc` (truecolor). A screen
// without styles renders the same as Snapshot.
func (tt *Term[T]) SnapshotStyled() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.normalize(tt.term.styledString())
}

// SnapshotModes returns the terminal mode changes the program has written so
// far (DEC private modes, kitty keyboard protocol pushes and pops,
// modifyOtherKeys, tab-stop resets), followed by the modes still active
// relative to a terminal's default state.
func (tt *Term[T]) SnapshotModes() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return strings.TrimSuffix(tt.term.modesString(), "\n")
}

// SnapshotQueries returns the terminal queries the program has written, one
// per line: "DECRQM ?<mode>", "kitty-keyboard-flags", "OSC <n>" (e.g. OSC 11,
// the background color), "CPR" (cursor position), "DA1", "DA2", "XTVERSION".
func (tt *Term[T]) SnapshotQueries() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return strings.TrimSuffix(tt.term.queriesString(), "\n")
}

func (tt *Term[T]) normalize(s string) string {
	if tt.opts.Normalize == nil {
		return s
	}
	return tt.opts.Normalize(s)
}

// Raw returns the raw byte stream captured from the program, including all
// escape sequences. Useful for debugging when Snapshot output is unexpected.
func (tt *Term[T]) Raw() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.raw.String()
}

// LogRawOnFailure logs the raw output stream when the test fails, so a flake
// shows the exact escape sequences that produced the screen. The log is only
// printed on failure.
func (tt *Term[T]) LogRawOnFailure() {
	tt.t.Cleanup(func() {
		if tt.t.Failed() {
			tt.t.Logf("raw output:\n%q", tt.Raw())
		}
	})
}

// Golden waits for the rendered screen to match
// testdata/<TestName>/<step>.golden, then asserts they're equal. With
// the testdiff -update flag set ([testdiff.OverwriteMode]) the golden
// file is (re)written from the settled snapshot instead. Step names are
// used verbatim as filenames; prefer "01-", "02-", … to keep listings
// in checkpoint order.
func (tt *Term[T]) Golden(step string) {
	tt.t.Helper()
	tt.golden(step+".golden", tt.Snapshot)
}

// GoldenStyled is [Term.Golden] for [Term.SnapshotStyled], stored as
// <step>.styled.golden.
func (tt *Term[T]) GoldenStyled(step string) {
	tt.t.Helper()
	tt.golden(step+".styled.golden", tt.SnapshotStyled)
}

// GoldenModes is [Term.Golden] for [Term.SnapshotModes], stored as
// <step>.modes.golden.
func (tt *Term[T]) GoldenModes(step string) {
	tt.t.Helper()
	tt.golden(step+".modes.golden", tt.SnapshotModes)
}

// GoldenQueries is [Term.Golden] for [Term.SnapshotQueries], stored as
// <step>.queries.golden.
func (tt *Term[T]) GoldenQueries(step string) {
	tt.t.Helper()
	tt.golden(step+".queries.golden", tt.SnapshotQueries)
}

func (tt *Term[T]) golden(file string, snapshot func() string) {
	tt.t.Helper()
	safeName := strings.ReplaceAll(tt.t.Name(), "/", "_")
	path := filepath.Join("testdata", safeName, file)

	if testdiff.OverwriteMode {
		time.Sleep(updateSettleWait)
		got := snapshot() + "\n"
		require.NoError(tt.t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(tt.t, os.WriteFile(path, []byte(got), 0o644))
		return
	}

	wantBytes, err := os.ReadFile(path)
	if err != nil {
		tt.t.Fatalf("read golden %s: %v\nrun with -update to create it\n--- current screen ---\n%s", path, err, snapshot())
	}
	want := string(wantBytes)
	got := tt.waitForMatch(want, snapshot) + "\n"
	testdiff.AssertEqualTexts(tt.t, path, "actual", want, got)
}

// GoldenText compares text against testdata/<TestName>/<step>.golden, or
// writes it there with -update. Use it to pin values that are not a screen,
// such as a prompt's result or the input left unread.
func GoldenText(t *testing.T, step, text string) {
	t.Helper()
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	path := filepath.Join("testdata", safeName, step+".golden")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if testdiff.OverwriteMode {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\nrun with -update to create it\n--- actual ---\n%s", path, err, text)
	}
	testdiff.AssertEqualTexts(t, path, "actual", string(want), text)
}

// waitForMatch polls snapshot until it (with a trailing newline appended,
// matching how goldens are stored on disk) equals want, then returns. If
// defaultTimeout elapses without a match, returns the last snapshot seen so
// the caller can produce a diff against it.
func (tt *Term[T]) waitForMatch(want string, snapshot func() string) string {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		got := snapshot()
		if got+"\n" == want {
			return got
		}
		if time.Now().After(deadline) {
			return got
		}
		time.Sleep(pollInterval)
	}
}

// Leftover returns the input bytes the program left unread when it
// returned, i.e. what a shell would read next. It requires
// [Options.CaptureLeftover] and waits for the program to finish. Replies
// the [Responder] still had scheduled are written before the read.
func (tt *Term[T]) Leftover() ([]byte, error) {
	tt.t.Helper()
	require.True(tt.t, tt.opts.CaptureLeftover, "Leftover requires Options.CaptureLeftover")
	select {
	case <-tt.finished:
	case <-time.After(defaultTimeout):
		tt.t.Fatalf("cmdio call did not finish in %s", defaultTimeout)
	}
	return tt.leftover, tt.leftoverErr
}

// captureLeftover runs on the program goroutine after the program returns
// and before the input pipe is closed.
func (tt *Term[T]) captureLeftover() {
	tt.drainPending()
	tt.scheduleReplies()
	tt.replies.Wait()
	// Closing the write end makes ReadAll stop at the end of the buffered
	// bytes instead of blocking on the still-open pipe.
	tt.inWMu.Lock()
	_ = tt.inW.Close()
	tt.inWMu.Unlock()
	tt.leftover, tt.leftoverErr = io.ReadAll(tt.inR)
}

// respondLoop answers queries while the program runs.
func (tt *Term[T]) respondLoop() {
	for {
		select {
		case <-tt.finished:
			return
		case <-time.After(pollInterval):
			tt.drainPending()
			tt.scheduleReplies()
		}
	}
}

// scheduleReplies queues the replies to every query not yet answered. They
// are written after the responder's delay, in the order the queries were
// written, as a terminal answers them.
func (tt *Term[T]) scheduleReplies() {
	r := tt.opts.Responder
	if r == nil {
		return
	}
	tt.replyMu.Lock()
	defer tt.replyMu.Unlock()
	tt.mu.Lock()
	queries := tt.term.queries[tt.answered:]
	tt.answered = len(tt.term.queries)
	tt.mu.Unlock()
	var batch strings.Builder
	for _, q := range queries {
		reply, ok := r.Replies[q]
		if !ok {
			reply = DefaultReply(q)
		}
		batch.WriteString(reply)
	}
	if batch.Len() == 0 {
		return
	}
	// Each batch waits for the one before it, so replies never overtake
	// each other even when their timers fire together.
	prev, done := tt.lastReply, make(chan struct{})
	tt.lastReply = done
	tt.replies.Add(1)
	time.AfterFunc(r.Delay, func() {
		defer tt.replies.Done()
		defer close(done)
		if prev != nil {
			<-prev
		}
		tt.inWMu.Lock()
		defer tt.inWMu.Unlock()
		_, _ = tt.inW.Write([]byte(batch.String()))
	})
}

// Close releases the input pipe and waits for the goroutine running the
// cmdio entry point to finish. Safe to call after Result has already drained
// the return value, and safe to call repeatedly.
func (tt *Term[T]) Close() {
	_ = tt.inW.Close()
	select {
	case <-tt.finished:
	case <-time.After(defaultTimeout):
		tt.t.Logf("Close: cmdio call did not finish in %s; raw output:\n%s", defaultTimeout, tt.Raw())
	}
}

// Result waits for the goroutine running the cmdio entry point to finish and
// returns the (value, error) it produced. For NewPrompt / NewSecret /
// NewSelectOrdered T is string; for NewSelect T is int.
func (tt *Term[T]) Result() (T, error) {
	tt.t.Helper()
	select {
	case r := <-tt.done:
		return r.value, r.err
	case <-time.After(defaultTimeout):
		tt.t.Fatalf("cmdio call did not finish in %s; raw output:\n%s", defaultTimeout, tt.Raw())
		var zero T
		return zero, nil
	}
}

// drainPending moves whatever bytes the program has written since the last
// drain into the raw buffer and the VT emulator. The emulator's parser is
// fed in UTF-8 boundary-aligned chunks because a multi-byte rune split
// across two writes would otherwise be reported invalid and skip cells.
func (tt *Term[T]) drainPending() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	if tt.out.Len() == 0 {
		return
	}
	b := make([]byte, tt.out.Len())
	_, _ = tt.out.Read(b)
	tt.raw.Write(b)
	input := b
	if len(tt.pending) > 0 {
		input = append(tt.pending, input...)
		tt.pending = tt.pending[:0]
	}
	consumed := completeUTF8End(input)
	_, _ = tt.term.Write(input[:consumed])
	if consumed < len(input) {
		tt.pending = append(tt.pending, input[consumed:]...)
	}
}

// completeUTF8End returns the offset such that p[:offset] contains only
// complete UTF-8 sequences and p[offset:] is the start of an incomplete rune
// that needs more bytes. If p ends on a complete rune, the offset is len(p).
func completeUTF8End(p []byte) int {
	n := len(p)
	// A UTF-8 rune is at most 4 bytes, so the partial start can only be in
	// the last 3 positions. Walk backwards looking for a rune-start byte.
	for i := 1; i <= 3 && i <= n; i++ {
		if utf8.RuneStart(p[n-i]) {
			if !utf8.FullRune(p[n-i:]) {
				return n - i
			}
			return n
		}
	}
	return n
}
