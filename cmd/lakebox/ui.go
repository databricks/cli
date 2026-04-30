package lakebox

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Single accent color throughout. Bold for emphasis. Dim for metadata.
const (
	rs   = "\033[0m"  // reset
	bo   = "\033[1m"  // bold
	dm   = "\033[2m"  // dim
	cyan = "\033[36m" // accent
)

func isTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return false
		}
		return fi.Mode()&os.ModeCharDevice != 0
	}
	return false
}

// spinner shows a braille spinner like Claude Code.
type spinner struct {
	w       io.Writer
	msg     string
	done    chan struct{}
	once    sync.Once
	started time.Time
}

func spin(w io.Writer, msg string) *spinner {
	s := &spinner{w: w, msg: msg, done: make(chan struct{}), started: time.Now()}
	if isTTY(w) {
		go s.run()
	} else {
		fmt.Fprintf(w, "* %s\n", msg)
	}
	return s
}

func (s *spinner) run() {
	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			elapsed := time.Since(s.started).Truncate(time.Second)
			fmt.Fprintf(s.w, "\r  %s%s%s %s%s%s %s(%s)%s  ",
				cyan, frames[i%len(frames)], rs,
				bo, s.msg, rs,
				dm, elapsed, rs)
			i++
		}
	}
}

func (s *spinner) ok(msg string) {
	s.once.Do(func() {
		close(s.done)
		if isTTY(s.w) {
			fmt.Fprintf(s.w, "\r\033[K  %s✓%s %s\n", cyan, rs, msg)
		} else {
			fmt.Fprintf(s.w, "✓ %s\n", msg)
		}
	})
}

func (s *spinner) fail(msg string) {
	s.once.Do(func() {
		close(s.done)
		if isTTY(s.w) {
			fmt.Fprintf(s.w, "\r\033[K  %s✗%s %s\n", cyan, rs, msg)
		} else {
			fmt.Fprintf(s.w, "✗ %s\n", msg)
		}
	})
}

// --- Consistent output primitives ---

// status formats a status string with the accent color.
func status(s string) string {
	switch strings.ToLower(s) {
	case "running":
		return cyan + "running" + rs
	case "stopped":
		return dm + "stopped" + rs
	case "creating":
		return cyan + bo + "creating…" + rs
	default:
		return dm + strings.ToLower(s) + rs
	}
}

// field prints "  label  value"
func field(w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %s%-10s%s %s\n", dm, label, rs, value)
}

// ok prints "  ✓ message"
func ok(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s✓%s %s\n", cyan, rs, msg)
}

// warn prints "  ! message"
func warn(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %s!%s %s\n", cyan, rs, msg)
}

// blank prints an empty line.
func blank(w io.Writer) {
	fmt.Fprintln(w)
}

// accent wraps text in the accent color.
func accent(s string) string {
	return cyan + s + rs
}

// bold wraps text in bold.
func bold(s string) string {
	return bo + s + rs
}

// dim wraps text in dim.
func dim(s string) string {
	return dm + s + rs
}
