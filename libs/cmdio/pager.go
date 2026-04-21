package cmdio

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// pagerPageSize is the number of items rendered between prompts.
const pagerPageSize = 50

// pagerPromptText is shown on stderr between pages.
const pagerPromptText = "[space] more  [enter] all  [q|esc] quit"

// pagerClearLine is the ANSI sequence to return to column 0 and erase the
// current line. Used to remove the prompt before writing the next page.
const pagerClearLine = "\r\x1b[K"

// Key codes we care about when reading single bytes from stdin in raw mode.
const (
	pagerKeyEscape = 0x1b
	pagerKeyCtrlC  = 0x03
)

// startRawStdinKeyReader puts stdin into raw mode and streams keystrokes
// onto the returned channel. Callers must defer restore. Raw mode also
// clears OPOST on Unix, so output written while active needs crlfWriter
// to avoid staircase newlines.
func startRawStdinKeyReader(ctx context.Context) (<-chan byte, func(), error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to enter raw mode on stdin: %w", err)
	}
	restore := func() { _ = term.Restore(fd, oldState) }

	ch := make(chan byte, 16)
	go func() {
		defer close(ch)
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			select {
			case ch <- buf[0]:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, restore, nil
}

// pagerNextKey blocks until a key arrives, the key channel closes, or the
// context is cancelled. Returns ok=false on close or cancellation.
func pagerNextKey(ctx context.Context, keys <-chan byte) (byte, bool) {
	select {
	case k, ok := <-keys:
		return k, ok
	case <-ctx.Done():
		return 0, false
	}
}

// pagerShouldQuit drains any buffered keys non-blockingly and returns true
// if q/Q/esc/Ctrl+C was pressed. A closed channel (stdin EOF) is not a
// quit signal.
func pagerShouldQuit(keys <-chan byte) bool {
	for {
		select {
		case k, ok := <-keys:
			if !ok {
				return false
			}
			if k == 'q' || k == 'Q' || k == pagerKeyEscape || k == pagerKeyCtrlC {
				return true
			}
		default:
			return false
		}
	}
}

// crlfWriter translates outbound '\n' bytes into '\r\n' so output written
// while the TTY is in raw mode (OPOST cleared) still starts at column 0.
// io.Writer semantics are preserved: the returned byte count is the
// number of bytes from p that were consumed, not the (possibly larger)
// number of bytes written to the underlying writer.
type crlfWriter struct {
	w io.Writer
}

func (c crlfWriter) Write(p []byte) (int, error) {
	start := 0
	for i, b := range p {
		if b != '\n' {
			continue
		}
		if i > start {
			if _, err := c.w.Write(p[start:i]); err != nil {
				return start, err
			}
		}
		if _, err := c.w.Write([]byte{'\r', '\n'}); err != nil {
			return i, err
		}
		start = i + 1
	}
	if start < len(p) {
		if _, err := c.w.Write(p[start:]); err != nil {
			return start, err
		}
	}
	return len(p), nil
}
