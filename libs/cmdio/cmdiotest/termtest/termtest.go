// Package termtest drives interactive prompts through an in-process pty and
// feeds the output into a vt10x terminal emulator so tests can assert against
// the rendered screen rather than raw escape sequences.
//
// Typical usage:
//
//	tm := termtest.New(t)
//	defer tm.Close()
//	go func() {
//		_, err := promptInThisGoroutine(tm.Pty()) // uses tm.Pty() as Stdin/Stdout
//		errCh <- err
//	}()
//	tm.WaitFor("Pick a profile:")
//	tm.Golden("01-initial")
//	tm.Type(termtest.KeyDown)
//	tm.Golden("02-after-down")
//	tm.Type(termtest.KeyEnter)
//
// # Golden files
//
// Golden(step) compares the current rendered screen against a file at
// testdata/<TestName>/<step>.golden under the test's package directory.
// The screen is taken from vt10x — what the user sees — not the raw escape
// sequence stream, so the goldens are stable across rendering implementations
// as long as the visible UI matches.
//
// To create or refresh goldens, run with UPDATE_TERMTEST=1.
package termtest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// Common ANSI key sequences. Promptui's underlying readline reads these
// directly from the tty stream, so we feed them as-is.
const (
	KeyEnter     = "\r"
	KeyTab       = "\t"
	KeyBackspace = "\x7f"
	KeyEsc       = "\x1b"
	KeyCtrlC     = "\x03"
	KeyUp        = "\x1b[A"
	KeyDown      = "\x1b[B"
	KeyRight     = "\x1b[C"
	KeyLeft      = "\x1b[D"
	KeyHome      = "\x1b[H"
	KeyEnd       = "\x1b[F"
	KeyDelete    = "\x1b[3~"
	KeyCtrlA     = "\x01"
	KeyCtrlE     = "\x05"
	KeyCtrlU     = "\x15"
	KeyCtrlW     = "\x17"
)

// Default screen size and wait timeout.
const (
	defaultCols    = 120
	defaultRows    = 40
	defaultTimeout = 3 * time.Second
	pollInterval   = 5 * time.Millisecond

	// Stability window for Golden(): wait until no new bytes have arrived for
	// this long before snapshotting. Long enough to absorb a redraw round trip
	// (write keystroke → prompt reads → prompt writes new render → pump reads),
	// short enough that a passing test stays well under a tenth of a second.
	stabilityWindow = 30 * time.Millisecond
	stabilityMax    = 1 * time.Second

	// updateEnv enables overwriting golden files instead of comparing.
	updateEnv = "UPDATE_TERMTEST"
)

// Term wraps a master/slave pty pair and a vt10x terminal emulator. The pump
// goroutine copies bytes from the pty master into both vt10x (for snapshots)
// and an internal raw buffer (for WaitFor substring matching).
type Term struct {
	t   *testing.T
	ptm *os.File
	pts *os.File

	term     vt10x.Terminal
	rawState *term.State

	mu  sync.Mutex
	raw bytes.Buffer

	done chan struct{}
}

// New opens a pty pair sized 120x40 and starts a pump goroutine. The caller
// must call Close when done.
func New(t *testing.T) *Term {
	t.Helper()
	ptm, pts, err := pty.Open()
	require.NoError(t, err, "open pty")

	require.NoError(t, pty.Setsize(ptm, &pty.Winsize{Cols: defaultCols, Rows: defaultRows}))

	// Put the slave into raw mode so the kernel tty discipline does not echo
	// our keystrokes or line-buffer them. chzyer/readline (under promptui)
	// only raw-modes the process's real fd 0, not the Stdin we hand it, so
	// without this our \x1b[ARROW bytes get cooked-mode buffered and the
	// prompt sees them as line-buffered text instead of arrow keys.
	rawState, err := term.MakeRaw(int(pts.Fd()))
	require.NoError(t, err, "make pty slave raw")

	tm := &Term{
		t:        t,
		ptm:      ptm,
		pts:      pts,
		term:     vt10x.New(vt10x.WithSize(defaultCols, defaultRows)),
		rawState: rawState,
		done:     make(chan struct{}),
	}
	go tm.pump()
	return tm
}

// Pty returns the slave end of the pty. Pass it to a prompt as Stdin/Stdout.
// It is a real *os.File, so isatty checks succeed and raw mode works.
func (tm *Term) Pty() *os.File {
	return tm.pts
}

// Type writes the given byte sequence to the master end. Use the Key* constants
// for special keys.
func (tm *Term) Type(seq string) {
	tm.t.Helper()
	_, err := tm.ptm.Write([]byte(seq))
	require.NoError(tm.t, err, "write to pty")
}

// WaitFor blocks until substr appears in the raw output stream, or the default
// timeout elapses. Returns the raw output captured up to that point so callers
// can include it in failure messages.
func (tm *Term) WaitFor(substr string) string {
	tm.t.Helper()
	deadline := time.Now().Add(defaultTimeout)
	for {
		tm.mu.Lock()
		got := tm.raw.String()
		tm.mu.Unlock()
		if strings.Contains(got, substr) {
			return got
		}
		if time.Now().After(deadline) {
			tm.t.Fatalf("WaitFor(%q) timed out after %s\n--- captured raw output ---\n%s", substr, defaultTimeout, got)
		}
		time.Sleep(pollInterval)
	}
}

// Snapshot returns the current vt10x screen contents with trailing whitespace
// trimmed from each line. Empty lines at the bottom are dropped.
func (tm *Term) Snapshot() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	lines := strings.Split(tm.term.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// Raw returns the raw byte stream captured from the pty master, including all
// escape sequences. Useful for debugging when Snapshot output is unexpected.
func (tm *Term) Raw() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.raw.String()
}

// Golden waits for the screen to settle, then compares the rendered output
// against testdata/<TestName>/<step>.golden. With UPDATE_TERMTEST=1 the
// golden file is (re)written instead. The path is relative to the calling
// test's package directory — go test cd's there before running.
//
// Step names are used verbatim as filenames, so prefer ASCII without slashes.
// A "01-", "02-", ... prefix keeps the directory listing in checkpoint order.
func (tm *Term) Golden(step string) {
	tm.t.Helper()
	tm.waitStable()
	got := tm.Snapshot()

	safeName := strings.ReplaceAll(tm.t.Name(), "/", "_")
	path := filepath.Join("testdata", safeName, step+".golden")

	if os.Getenv(updateEnv) != "" {
		require.NoError(tm.t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(tm.t, os.WriteFile(path, []byte(got+"\n"), 0o644))
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		tm.t.Fatalf("read golden %s: %v\nrun with %s=1 to create it\n--- current screen ---\n%s", path, err, updateEnv, got)
	}
	if got+"\n" != string(want) {
		tm.t.Fatalf("golden %s mismatch:\n--- want ---\n%s--- got ---\n%s\n", path, string(want), got)
	}
}

// waitStable blocks until no new bytes have arrived from the pty for one
// stabilityWindow, or stabilityMax elapses. Used before snapshotting so
// in-flight redraws don't get captured mid-stream.
func (tm *Term) waitStable() {
	tm.t.Helper()
	deadline := time.Now().Add(stabilityMax)
	tm.mu.Lock()
	last := tm.raw.Len()
	tm.mu.Unlock()
	for {
		time.Sleep(stabilityWindow)
		tm.mu.Lock()
		now := tm.raw.Len()
		tm.mu.Unlock()
		if now == last {
			return
		}
		if time.Now().After(deadline) {
			tm.t.Logf("waitStable: screen kept changing for %s; snapshotting anyway", stabilityMax)
			return
		}
		last = now
	}
}

// Close releases the pty and stops the pump goroutine.
func (tm *Term) Close() {
	if tm.rawState != nil {
		_ = term.Restore(int(tm.pts.Fd()), tm.rawState)
	}
	_ = tm.pts.Close()
	_ = tm.ptm.Close()
	<-tm.done
}

func (tm *Term) pump() {
	defer close(tm.done)
	buf := make([]byte, 4096)
	for {
		n, err := tm.ptm.Read(buf)
		if n > 0 {
			tm.mu.Lock()
			tm.raw.Write(buf[:n])
			_, _ = tm.term.Write(buf[:n])
			tm.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}
