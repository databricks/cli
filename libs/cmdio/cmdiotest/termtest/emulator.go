package termtest

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// emulator is a tiny VT-style terminal emulator. It exists to render the
// byte stream bubbletea writes to its output into a 2D screen of plain text
// so tests can match against `screen text` goldens.
//
// Scope is deliberately narrow: bubbletea's stock renderer (which is what
// the cmdio prompt/select models run through) emits a small subset of CSI
// sequences — cursor moves, erase-line, set/reset-mode for cursor
// visibility — plus printable runes, CR, LF, and BS. Alt-screen switches,
// scroll regions, charsets, DCS, and friends are recognised by the
// underlying ansi parser but intentionally ignored: they do not affect cell
// text. SGR styling, DEC private modes, keyboard protocol switches and
// terminal queries are recorded alongside the text for the styled, modes
// and queries goldens (see observe.go), without changing what String
// returns. Cursor movement covers the usual CUU/CUD/CUP family and the
// unprefixed SCO save/restore pair (CSI s and CSI u); prefixed forms such as
// the kitty keyboard CSI ? u and CSI > n u do not move the cursor.
type emulator struct {
	width, height int
	grid          [][]rune
	row, col      int

	// Saved cursor (DECSC/DECRC; ESC 7 / ESC 8).
	savedRow, savedCol int

	// mapNewline makes LF also return to column 0, like a cooked tty's
	// ONLCR. Off by default: bubbletea v1 writes CRLF itself.
	mapNewline bool

	// Observation state for the styled, modes and queries goldens. None of
	// it affects cell text (see observe.go).
	styles          [][]cellStyle
	pen             cellStyle
	privateModes    map[int]bool
	modeEvents      []modeEvent
	kittyStack      []int
	modifyOtherKeys int
	queries         []string

	parser *ansi.Parser
}

func newEmulator(width, height int) *emulator {
	e := &emulator{width: width, height: height}
	e.reset()
	e.parser = ansi.NewParser()
	e.parser.SetHandler(ansi.Handler{
		Print:     e.handlePrint,
		Execute:   e.handleControl,
		HandleCsi: e.handleCsi,
		HandleEsc: e.handleEsc,
		HandleOsc: e.handleOsc,
	})
	return e
}

// reset clears the screen, homes the cursor, and resets the pen and the
// private modes.
func (e *emulator) reset() {
	e.grid = make([][]rune, e.height)
	e.styles = make([][]cellStyle, e.height)
	for i := range e.grid {
		e.grid[i] = make([]rune, e.width)
		e.styles[i] = make([]cellStyle, e.width)
		for j := range e.grid[i] {
			e.grid[i][j] = ' '
		}
	}
	e.pen = cellStyle{}
	e.privateModes = map[int]bool{}
	e.row, e.col = 0, 0
	e.savedRow, e.savedCol = 0, 0
}

// Write feeds bytes into the parser.
func (e *emulator) Write(p []byte) (int, error) {
	e.parser.Parse(p)
	return len(p), nil
}

// String returns the screen contents, joined with "\n".
func (e *emulator) String() string {
	var b strings.Builder
	for i, line := range e.grid {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(string(line))
	}
	return b.String()
}

// handlePrint writes a printable rune at the cursor and advances. Off-screen
// writes are dropped (no scroll, no wrap — the prompts we render fit easily
// inside 120x40, which is what termtest.New configures).
func (e *emulator) handlePrint(r rune) {
	if e.col >= e.width {
		// At the right margin: drop the rune rather than wrapping. Real
		// terminals would wrap; the prompts we test never reach the edge.
		return
	}
	if e.inBounds() {
		e.grid[e.row][e.col] = r
		e.styles[e.row][e.col] = e.pen
	}
	e.col++
}

// handleControl handles C0 control characters: BS, HT, LF, CR.
func (e *emulator) handleControl(b byte) {
	switch b {
	case 0x08: // BS
		if e.col > 0 {
			e.col--
		}
	case 0x09: // HT: next tab stop (every 8 columns).
		stop := ((e.col / 8) + 1) * 8
		e.col = min(stop, e.width-1)
	case 0x0a: // LF: move down one line, no scroll.
		if e.row+1 < e.height {
			e.row++
		}
		if e.mapNewline {
			e.col = 0
		}
	case 0x0d: // CR: start of line.
		e.col = 0
	}
}

// handleCsi handles the CSI sequences emitted by bubbletea's renderer.
// Only commands that affect cell positions or contents matter. SGR ('m') and
// the mode and query sequences do not move the cursor or change cell text;
// observeCsi records the ones the goldens pin.
func (e *emulator) handleCsi(cmd ansi.Cmd, params ansi.Params) {
	e.observeCsi(cmd, params)
	n, _, _ := params.Param(0, 1)
	switch cmd.Final() {
	case 'A': // CUU: cursor up
		e.row = max(e.row-n, 0)
	case 'B', 'e': // CUD / VPR: cursor down
		e.row = min(e.row+n, e.height-1)
	case 'C', 'a': // CUF / HPR: cursor forward
		e.col = min(e.col+n, e.width-1)
	case 'D': // CUB: cursor back
		e.col = max(e.col-n, 0)
	case 'E': // CNL: next line
		e.row = min(e.row+n, e.height-1)
		e.col = 0
	case 'F': // CPL: previous line
		e.row = max(e.row-n, 0)
		e.col = 0
	case 'G', '`': // CHA / HPA: cursor to column n (1-based)
		e.col = clamp(n-1, 0, e.width-1)
	case 'd': // VPA: cursor to row n (1-based)
		e.row = clamp(n-1, 0, e.height-1)
	case 'H', 'f': // CUP / HVP: cursor position (row;col, 1-based)
		row, _, _ := params.Param(0, 1)
		col, _, _ := params.Param(1, 1)
		e.row = clamp(row-1, 0, e.height-1)
		e.col = clamp(col-1, 0, e.width-1)
	case 'J': // ED: erase in display
		n, _, _ := params.Param(0, 0)
		e.eraseDisplay(n)
	case 'K': // EL: erase in line
		n, _, _ := params.Param(0, 0)
		e.eraseLine(n)
	case 's': // SCO save cursor
		if cmd.Prefix() == 0 {
			e.savedRow, e.savedCol = e.row, e.col
		}
	case 'u': // SCO restore cursor; prefixed forms are kitty keyboard sequences.
		if cmd.Prefix() == 0 {
			e.row, e.col = e.savedRow, e.savedCol
		}
		// All other CSI commands (SGR 'm', mode set/reset 'h'/'l', cursor
		// show/hide '?25h/l', etc.) are no-ops for cell text.
	}
}

// handleEsc handles ESC sequences: only the C1-equivalent cursor save/restore
// (ESC 7 / ESC 8) matter for cell positions.
func (e *emulator) handleEsc(cmd ansi.Cmd) {
	switch cmd.Final() {
	case '7': // DECSC: save cursor
		e.savedRow, e.savedCol = e.row, e.col
	case '8': // DECRC: restore cursor
		e.row, e.col = e.savedRow, e.savedCol
	}
}

// eraseLine implements CSI K with parameter n: 0 = to end, 1 = from start,
// 2 = whole line.
func (e *emulator) eraseLine(n int) {
	if !e.inBounds() {
		return
	}
	line := e.grid[e.row]
	switch n {
	case 0:
		for i := e.col; i < e.width; i++ {
			e.clearCell(e.row, i)
		}
	case 1:
		for i := 0; i <= e.col && i < e.width; i++ {
			e.clearCell(e.row, i)
		}
	case 2:
		for i := range line {
			e.clearCell(e.row, i)
		}
	}
}

// eraseDisplay implements CSI J with parameter n: 0 = to end of screen,
// 1 = from start, 2 = whole screen.
func (e *emulator) eraseDisplay(n int) {
	switch n {
	case 0:
		e.eraseLine(0)
		for r := e.row + 1; r < e.height; r++ {
			for c := range e.grid[r] {
				e.clearCell(r, c)
			}
		}
	case 1:
		for r := range e.row {
			for c := range e.grid[r] {
				e.clearCell(r, c)
			}
		}
		e.eraseLine(1)
	case 2, 3:
		for r := range e.grid {
			for c := range e.grid[r] {
				e.clearCell(r, c)
			}
		}
	}
}

// clearCell blanks one cell and resets its style. Erased cells take the
// default style rather than the pen's background. The pins do not cover an
// erase under a colored background, so this is a modelling choice, not a
// measured property of bubbletea.
func (e *emulator) clearCell(r, c int) {
	e.grid[r][c] = ' '
	e.styles[r][c] = cellStyle{}
}

func (e *emulator) inBounds() bool {
	return e.row >= 0 && e.row < e.height && e.col >= 0 && e.col < e.width
}

func clamp(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
