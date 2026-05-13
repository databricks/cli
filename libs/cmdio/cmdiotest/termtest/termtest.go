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
// matches. To create or refresh goldens, pass -update to `go test`
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
)

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

	// typeInterKeyDelay is the pause [Term.Type] waits after writing each
	// keystroke to the input pipe. Real terminals deliver keystrokes with
	// ~100 ms between them; pipe writes from back-to-back Type calls would
	// otherwise land in a single bubbletea Read and be parsed ambiguously
	// (a lone `\x1b` followed by `a` parses as Alt+a, not Esc then 'a').
	// 20 ms matches the ESC-disambiguation timeouts neovim and tmux use
	// for the same problem.
	typeInterKeyDelay = 20 * time.Millisecond
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
	t *testing.T

	inW *os.File
	inR *os.File

	done     chan resultMsg[T]
	finished chan struct{}

	mu      sync.Mutex
	out     bytes.Buffer
	term    *emulator
	raw     bytes.Buffer
	pending []byte
}

// NewPrompt runs [cmdio.RunPrompt] in a goroutine and returns a harness for
// it.
func NewPrompt(t *testing.T, opts cmdio.PromptOptions) *Term[string] {
	t.Helper()
	return run(t, func(ctx context.Context) (string, error) {
		return cmdio.RunPrompt(ctx, opts)
	})
}

// NewSecret runs [cmdio.Secret] in a goroutine and returns a harness for it.
func NewSecret(t *testing.T, label string) *Term[string] {
	t.Helper()
	return run(t, func(ctx context.Context) (string, error) {
		return cmdio.Secret(ctx, label)
	})
}

// NewSelect runs [cmdio.RunSelect] in a goroutine and returns a harness for
// it.
func NewSelect(t *testing.T, opts cmdio.SelectOptions) *Term[int] {
	t.Helper()
	return run(t, func(ctx context.Context) (int, error) {
		return cmdio.RunSelect(ctx, opts)
	})
}

// NewSelectOrdered runs [cmdio.SelectOrdered] in a goroutine and returns a
// harness for it.
func NewSelectOrdered(t *testing.T, items []cmdio.Tuple, label string) *Term[string] {
	t.Helper()
	return run(t, func(ctx context.Context) (string, error) {
		return cmdio.SelectOrdered(ctx, items, label)
	})
}

func run[T any](t *testing.T, fn func(ctx context.Context) (T, error)) *Term[T] {
	inR, inW, err := os.Pipe()
	require.NoError(t, err, "open input pipe")
	tt := &Term[T]{
		t:        t,
		inW:      inW,
		inR:      inR,
		done:     make(chan resultMsg[T], 1),
		finished: make(chan struct{}),
		term:     newEmulator(cmdio.TestTerminalWidth, cmdio.TestTerminalHeight),
	}
	io := cmdio.NewTestIO(inR, io.Discard, &syncWriter{mu: &tt.mu, buf: &tt.out})
	ctx := cmdio.InContext(t.Context(), io)
	go func() {
		v, err := fn(ctx)
		_ = inR.Close()
		tt.done <- resultMsg[T]{value: v, err: err}
		close(tt.finished)
	}()
	t.Cleanup(tt.Close)
	return tt
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

// Type writes a string into the program's stdin. The string can be ordinary
// text or one of the Key* constants — they're all just bytes the user would
// have typed on a real terminal. Returns after [typeInterKeyDelay] so a
// follow-up Type lands in a separate bubbletea Read; see the constant's
// comment for why that matters.
func (tt *Term[T]) Type(s string) {
	tt.t.Helper()
	_, err := tt.inW.Write([]byte(s))
	require.NoError(tt.t, err, "write to stdin pipe")
	time.Sleep(typeInterKeyDelay)
}

// WaitFor blocks until substr appears in the raw output stream, or the
// default timeout elapses. Returns the captured output for use in failure
// messages.
func (tt *Term[T]) WaitFor(substr string) string {
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
	return strings.Join(lines, "\n")
}

// Raw returns the raw byte stream captured from the program, including all
// escape sequences. Useful for debugging when Snapshot output is unexpected.
func (tt *Term[T]) Raw() string {
	tt.drainPending()
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.raw.String()
}

// Golden waits for the rendered screen to match
// testdata/<TestName>/<step>.golden, then asserts they're equal. With
// the testdiff -update flag set ([testdiff.OverwriteMode]) the golden
// file is (re)written from the settled snapshot instead. Step names are
// used verbatim as filenames; prefer "01-", "02-", … to keep listings
// in checkpoint order.
func (tt *Term[T]) Golden(step string) {
	tt.t.Helper()
	safeName := strings.ReplaceAll(tt.t.Name(), "/", "_")
	path := filepath.Join("testdata", safeName, step+".golden")

	if testdiff.OverwriteMode {
		time.Sleep(updateSettleWait)
		got := tt.Snapshot() + "\n"
		require.NoError(tt.t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(tt.t, os.WriteFile(path, []byte(got), 0o644))
		return
	}

	wantBytes, err := os.ReadFile(path)
	if err != nil {
		tt.t.Fatalf("read golden %s: %v\nrun with -update to create it\n--- current screen ---\n%s", path, err, tt.Snapshot())
	}
	want := string(wantBytes)
	got := tt.waitForMatch(want) + "\n"
	testdiff.AssertEqualTexts(tt.t, path, "actual", want, got)
}

// waitForMatch polls the rendered screen until it (with a trailing
// newline appended, matching how goldens are stored on disk) equals
// want, then returns. If defaultTimeout elapses without a match, returns the
// last snapshot seen so the caller can produce a diff against it.
func (tt *Term[T]) waitForMatch(want string) string {
	tt.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		got := tt.Snapshot()
		if got+"\n" == want {
			return got
		}
		if time.Now().After(deadline) {
			return got
		}
		time.Sleep(pollInterval)
	}
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
