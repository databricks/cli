// Package termtest drives cmdio's interactive prompts as a black box: it
// wires a [cmdio.NewTestIO] up to a pair of pipes, calls the public entry
// points ([cmdio.RunPrompt], [cmdio.Secret], [cmdio.SelectOrdered], …) in a
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
// goldens are stable across rendering changes as long as the visible UI
// matches. To create or refresh goldens, set UPDATE_TERMTEST=1.
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
)

const (
	defaultTimeout = 3 * time.Second
	pollInterval   = 5 * time.Millisecond

	// Stability window for Golden(): wait until no new bytes have arrived
	// from the program for this long before snapshotting. Long enough to
	// absorb a keystroke → Update → View → render round trip, short enough
	// that a passing test stays well under a tenth of a second.
	stabilityWindow = 30 * time.Millisecond
	stabilityMax    = 1 * time.Second

	updateEnv = "UPDATE_TERMTEST"
)

// resultMsg carries the (value, err) the goroutine running the cmdio entry
// point produced. value's concrete type depends on which constructor was
// used: string for RunPrompt / Secret / SelectOrdered, int for RunSelect.
type resultMsg struct {
	value any
	err   error
}

// Term wraps a goroutine running a cmdio prompt/select call. Stdin is an
// OS-level pipe we write to (bubbletea's cancelreader only knows how to
// unblock os.File-backed inputs cleanly on Quit; io.Pipe would leak the
// input goroutine); cmdio's stderr (where the TUI renders) is a
// synchronized buffer we drain into the VT emulator on demand.
type Term struct {
	t *testing.T

	inW *os.File
	inR *os.File

	done     chan resultMsg
	finished chan struct{}

	mu      sync.Mutex
	out     bytes.Buffer
	term    *emulator
	raw     bytes.Buffer
	pending []byte
}

// PromptTerm drives a string-returning entry point ([cmdio.RunPrompt],
// [cmdio.Secret], [cmdio.SelectOrdered]).
type PromptTerm struct{ *Term }

// SelectTerm drives an int-returning entry point ([cmdio.RunSelect]).
type SelectTerm struct{ *Term }

// NewPrompt runs [cmdio.RunPrompt] in a goroutine and returns a harness for
// it.
func NewPrompt(t *testing.T, opts cmdio.PromptOptions) *PromptTerm {
	t.Helper()
	return &PromptTerm{Term: run(t, func(ctx context.Context) (any, error) {
		return cmdio.RunPrompt(ctx, opts)
	})}
}

// NewSecret runs [cmdio.Secret] in a goroutine and returns a harness for it.
func NewSecret(t *testing.T, label string) *PromptTerm {
	t.Helper()
	return &PromptTerm{Term: run(t, func(ctx context.Context) (any, error) {
		return cmdio.Secret(ctx, label)
	})}
}

// NewSelect runs [cmdio.RunSelect] in a goroutine and returns a harness for
// it.
func NewSelect(t *testing.T, opts cmdio.SelectOptions) *SelectTerm {
	t.Helper()
	return &SelectTerm{Term: run(t, func(ctx context.Context) (any, error) {
		return cmdio.RunSelect(ctx, opts)
	})}
}

// NewSelectOrdered runs [cmdio.SelectOrdered] in a goroutine and returns a
// harness for it.
func NewSelectOrdered(t *testing.T, items []cmdio.Tuple, label string) *PromptTerm {
	t.Helper()
	return &PromptTerm{Term: run(t, func(ctx context.Context) (any, error) {
		return cmdio.SelectOrdered(ctx, items, label)
	})}
}

func run(t *testing.T, fn func(ctx context.Context) (any, error)) *Term {
	inR, inW, err := os.Pipe()
	require.NoError(t, err, "open input pipe")
	tt := &Term{
		t:        t,
		inW:      inW,
		inR:      inR,
		done:     make(chan resultMsg, 1),
		finished: make(chan struct{}),
		term:     newEmulator(cmdio.TestTerminalWidth, cmdio.TestTerminalHeight),
	}
	io := cmdio.NewTestIO(inR, io.Discard, (*syncWriter)(tt))
	ctx := cmdio.InContext(t.Context(), io)
	go func() {
		v, err := fn(ctx)
		_ = inR.Close()
		tt.done <- resultMsg{value: v, err: err}
		close(tt.finished)
	}()
	t.Cleanup(tt.Close)
	return tt
}

// syncWriter is the io.Writer cmdio gives bubbletea as stderr. Writes from
// the program goroutine race with our drain calls, so the buffer is mutex-
// guarded; the type is a thin alias rather than a wrapper to avoid an extra
// pointer hop.
type syncWriter Term

func (w *syncWriter) Write(p []byte) (int, error) {
	tt := (*Term)(w)
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.out.Write(p)
}

// Type writes a string into the program's stdin. The string can be ordinary
// text or one of the Key* constants — they're all just bytes the user would
// have typed on a real terminal.
func (tt *Term) Type(s string) {
	tt.t.Helper()
	_, err := tt.inW.Write([]byte(s))
	require.NoError(tt.t, err, "write to stdin pipe")
}

// WaitFor blocks until substr appears in the raw output stream, or the
// default timeout elapses. Returns the captured output for use in failure
// messages.
func (tt *Term) WaitFor(substr string) string {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		tt.drainPending()
		tt.mu.Lock()
		got := tt.raw.String()
		tt.mu.Unlock()
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
func (tt *Term) Snapshot() string {
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
	return strings.Join(lines, "\n")
}

// Raw returns the raw byte stream captured from the program, including all
// escape sequences. Useful for debugging when Snapshot output is unexpected.
func (tt *Term) Raw() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.raw.String()
}

// Golden waits for the screen to settle, then compares the rendered output
// against testdata/<TestName>/<step>.golden. With UPDATE_TERMTEST=1 the
// golden file is (re)written instead. Step names are used verbatim as
// filenames; prefer "01-", "02-", … to keep listings in checkpoint order.
func (tt *Term) Golden(step string) {
	tt.t.Helper()
	tt.waitStable()
	got := tt.Snapshot()

	safeName := strings.ReplaceAll(tt.t.Name(), "/", "_")
	path := filepath.Join("testdata", safeName, step+".golden")

	if os.Getenv(updateEnv) != "" { //nolint:forbidigo // test-only UPDATE flag; no ctx available in helper
		require.NoError(tt.t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(tt.t, os.WriteFile(path, []byte(got+"\n"), 0o644))
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		tt.t.Fatalf("read golden %s: %v\nrun with %s=1 to create it\n--- current screen ---\n%s", path, err, updateEnv, got)
	}
	if got+"\n" != string(want) {
		tt.t.Fatalf("golden %s mismatch:\n--- want ---\n%s--- got ---\n%s\n", path, string(want), got)
	}
}

// Close releases the input pipe and waits for the goroutine running the
// cmdio entry point to finish. Safe to call after Result has already drained
// the return value, and safe to call repeatedly.
func (tt *Term) Close() {
	_ = tt.inW.Close()
	select {
	case <-tt.finished:
	case <-time.After(defaultTimeout):
		tt.t.Logf("Close: cmdio call did not finish in %s; raw output:\n%s", defaultTimeout, tt.Raw())
	}
}

// Result waits for the goroutine running the prompt to finish and returns
// the (string, error) it produced. Use on harnesses returned by NewPrompt,
// NewSecret, or NewSelectOrdered.
func (p *PromptTerm) Result() (string, error) {
	r := p.await()
	if r.value == nil {
		return "", r.err
	}
	return r.value.(string), r.err
}

// Result waits for the goroutine running the select to finish and returns
// the (int, error) it produced.
func (s *SelectTerm) Result() (int, error) {
	r := s.await()
	if r.value == nil {
		return 0, r.err
	}
	return r.value.(int), r.err
}

func (tt *Term) await() resultMsg {
	tt.t.Helper()
	select {
	case r := <-tt.done:
		return r
	case <-time.After(defaultTimeout):
		tt.t.Fatalf("cmdio call did not finish in %s; raw output:\n%s", defaultTimeout, tt.Raw())
		return resultMsg{}
	}
}

// drainPending moves whatever bytes the program has written since the last
// drain into the raw buffer and the VT emulator. The emulator's parser is
// fed in UTF-8 boundary-aligned chunks because a multi-byte rune split
// across two writes would otherwise be reported invalid and skip cells.
func (tt *Term) drainPending() {
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

// waitStable blocks until no new bytes have arrived from the program for one
// stabilityWindow, or stabilityMax elapses. Used before snapshotting so
// in-flight redraws don't get captured mid-stream.
func (tt *Term) waitStable() {
	tt.t.Helper()
	deadline := time.Now().Add(stabilityMax)
	tt.drainPending()
	tt.mu.Lock()
	last := tt.raw.Len()
	tt.mu.Unlock()
	for {
		time.Sleep(stabilityWindow)
		tt.drainPending()
		tt.mu.Lock()
		now := tt.raw.Len()
		tt.mu.Unlock()
		if now == last {
			return
		}
		if time.Now().After(deadline) {
			tt.t.Logf("waitStable: output kept changing for %s; snapshotting anyway", stabilityMax)
			return
		}
		last = now
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
