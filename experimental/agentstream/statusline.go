package agentstream

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"
)

// statusLine overwrites a single line on stderr using ANSI cursor movement.
// Each update ends with \n so the cursor sits on a fresh line. The next update
// moves up one line, erases it, and rewrites. This is more reliable than bare
// \r because it handles line-wrap edge cases.
type statusLine struct {
	w      io.Writer
	active bool
}

// update overwrites the status line with new text (faint, truncated to terminal width).
func (s *statusLine) update(text string) {
	// Replace newlines so multi-sentence thoughts stay on one line.
	text = strings.ReplaceAll(text, "\n", " ")

	width := terminalWidth()
	// Truncate by rune count (not bytes) so multi-byte characters don't wrap.
	runes := []rune(text)
	maxText := width - 2 // account for "> " prefix
	if maxText > 0 && len(runes) > maxText {
		runes = runes[:maxText-3]
		text = string(runes) + "..."
	} else {
		text = string(runes)
	}

	if s.active {
		// Move cursor up one line and erase it before rewriting.
		fmt.Fprint(s.w, "\033[1A\033[2K")
	}
	fmt.Fprintf(s.w, "\033[2m> %s\033[0m\n", text)
	s.active = true
}

// clear removes the status line if one is active.
func (s *statusLine) clear() {
	if s.active {
		fmt.Fprint(s.w, "\033[1A\033[2K")
		s.active = false
	}
}

// terminalWidth returns the terminal width, defaulting to 80.
func terminalWidth() int {
	w, _, err := term.GetSize(2) // fd 2 = stderr
	if err != nil || w <= 0 {
		return 80
	}
	return w
}
