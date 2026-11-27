package termtest

import (
	"runtime"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// emulator is a tiny VT-style terminal emulator. It exists to render the
// byte stream bubbletea writes to its output into a 2D screen of plain text
// so tests can match against `screen text` goldens.
//
// Scope is deliberately narrow. bubbletea v1's standard renderer only moves
// the cursor with CUU, EL, CUP and CRLF (its IL and DECSTBM live in the
// deprecated insertTop, which cmdio does not use). The other cell and cursor
// sequences handled here (REP, ECH, ICH, DCH, IRM, IL, DL, SU, SD, DECSTBM,
// RI, CHT, CBT) are the ones the cell renderer of charm.land/bubbletea/v2
// v2.1.1 emits through github.com/charmbracelet/ultraviolet
// (terminal_renderer.go, terminal_renderer_hardscroll.go). SGR styling,
// alt-screen switches, charsets and OSC are recognised by the underlying ansi
// parser but do not affect cell text. Styles, modes and queries are recorded
// alongside the text for the styled, modes and queries goldens (see
// observe.go), without changing what String returns.
type emulator struct {
	width, height int
	grid          [][]rune
	row, col      int

	// Saved cursor (DECSC/DECRC; ESC 7 / ESC 8).
	savedRow, savedCol int

	// Scroll region rows (DECSTBM), inclusive.
	top, bottom int

	// insert is IRM (CSI 4 h): printed runes shift the rest of the line right.
	insert bool

	// lastRune is the most recently printed rune, repeated by REP. Zero
	// until something is printed.
	lastRune rune

	// mapNewline makes LF also return to column 0, like a cooked tty's ONLCR
	// output translation. bubbletea v2 assumes that mapping whenever its input
	// is not a TTY, except on Windows (charm.land/bubbletea/v2 v2.1.1
	// tea.go:1107 "mapNl := runtime.GOOS != \"windows\" && p.ttyInput == nil";
	// ultraviolet terminal_renderer.go:1645-1649). termtest feeds input
	// through a pipe, so it follows the same rule. v1 always writes CRLF, so
	// the setting does not affect it.
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
	e := &emulator{width: width, height: height, mapNewline: runtime.GOOS != "windows"}
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
	e.top, e.bottom = 0, e.height-1
	e.insert = false
	e.lastRune = 0
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

// handlePrint writes a printable rune at the cursor and advances. A rune past
// the right margin is dropped rather than wrapped; the prompts we render fit
// easily inside 120x40, which is what termtest.New configures. In insert mode
// (IRM) the rest of the line shifts right first.
func (e *emulator) handlePrint(r rune) {
	if e.col >= e.width {
		// At the right margin: drop the rune rather than wrapping. Real
		// terminals would wrap; the prompts we test never reach the edge.
		return
	}
	if e.inBounds() {
		if e.insert {
			e.insertCells(1)
		}
		e.grid[e.row][e.col] = r
		e.styles[e.row][e.col] = e.pen
	}
	e.lastRune = r
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
	case 0x0a: // LF: move down a line (scrolling at the bottom margin), then to column 0 if mapped.
		e.index()
		if e.mapNewline {
			e.col = 0
		}
	case 0x0d: // CR: start of line.
		e.col = 0
	}
}

// handleCsi handles the CSI sequences emitted by bubbletea's renderer.
// Only commands that affect cell positions or contents matter. SGR ('m') and
// the mode and query sequences do not move the cursor or change cell text,
// except IRM ("CSI 4 h"), which changes how later prints write; observeCsi
// records the ones the goldens pin.
func (e *emulator) handleCsi(cmd ansi.Cmd, params ansi.Params) {
	// Prefixed and intermediate CSIs are terminal queries and mode switches
	// (DEC private modes, DECRQM "CSI ? 2026 $ p", kitty keyboard
	// "CSI > 1 u" / "CSI < u" / "CSI ? u", modifyOtherKeys "CSI > 4;2 m"),
	// never cursor or cell operations. Without this check "CSI > 1 u" would
	// be read as SCO restore-cursor.
	e.observeCsi(cmd, params)
	if cmd.Prefix() != 0 || cmd.Intermediate() != 0 {
		return
	}
	n, _, _ := params.Param(0, 1)
	switch cmd.Final() {
	case 'b': // REP: repeat the last printed rune n times.
		// A real terminal ignores REP before anything has been printed.
		if e.lastRune == 0 {
			break
		}
		for range n {
			e.handlePrint(e.lastRune)
		}
	case 'X': // ECH: erase n cells from the cursor without moving it.
		if e.inBounds() {
			for i := e.col; i < min(e.col+n, e.width); i++ {
				e.clearCell(e.row, i)
			}
		}
	case '@': // ICH: insert n blank cells at the cursor.
		if e.inBounds() {
			e.insertCells(n)
		}
	case 'P': // DCH: delete n cells at the cursor, shifting the rest left.
		if e.inBounds() {
			e.deleteCells(n)
		}
	case 'L': // IL: insert n blank lines at the cursor row.
		if e.row >= e.top && e.row <= e.bottom {
			e.scrollDown(e.row, n)
			e.col = 0
		}
	case 'M': // DL: delete n lines at the cursor row.
		if e.row >= e.top && e.row <= e.bottom {
			e.scrollUp(e.row, n)
			e.col = 0
		}
	case 'S': // SU: scroll the region up n lines.
		e.scrollUp(e.top, n)
	case 'T': // SD: scroll the region down n lines.
		// The five-parameter form is xterm mouse highlight tracking.
		if len(params) <= 1 {
			e.scrollDown(e.top, n)
		}
	case 'r': // DECSTBM: set scroll region (1-based, inclusive); homes the cursor.
		top, _, _ := params.Param(0, 1)
		bottom, _, _ := params.Param(1, e.height)
		top, bottom = clamp(top-1, 0, e.height-1), clamp(bottom-1, 0, e.height-1)
		if top < bottom {
			e.top, e.bottom = top, bottom
			e.row, e.col = 0, 0
		}
	case 'h', 'l': // SM / RM: only IRM (mode 4) affects cell text.
		for i := range params {
			if params[i].Param(0) == 4 {
				e.insert = cmd.Final() == 'h'
			}
		}
	case 'I': // CHT: forward n tab stops.
		for range n {
			e.handleControl(0x09)
		}
	case 'Z': // CBT: back n tab stops.
		for range n {
			e.col = max(((e.col+7)/8-1)*8, 0)
		}
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
		e.savedRow, e.savedCol = e.row, e.col
	case 'u': // SCO restore cursor
		e.row, e.col = e.savedRow, e.savedCol
		// All other CSI commands (SGR 'm', etc.) are no-ops for cell text.
	}
}

// handleEsc handles the ESC sequences that matter for cell positions:
// cursor save/restore (ESC 7 / ESC 8) and reverse index (ESC M).
func (e *emulator) handleEsc(cmd ansi.Cmd) {
	if cmd.Intermediate() != 0 {
		return // charset designations etc.
	}
	switch cmd.Final() {
	case 'M': // RI: up one line, scrolling down at the top margin.
		if e.row == e.top {
			e.scrollDown(e.top, 1)
		} else if e.row > 0 {
			e.row--
		}
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

// index moves the cursor down one row, scrolling the region up when the
// cursor sits on its bottom margin.
func (e *emulator) index() {
	switch {
	case e.row == e.bottom:
		e.scrollUp(e.top, 1)
	case e.row+1 < e.height:
		e.row++
	}
}

// insertCells shifts the cells from the cursor right by n, dropping cells
// pushed past the right margin, and blanks the gap.
func (e *emulator) insertCells(n int) {
	line, styles := e.grid[e.row], e.styles[e.row]
	n = min(n, e.width-e.col)
	copy(line[e.col+n:], line[e.col:e.width-n])
	copy(styles[e.col+n:], styles[e.col:e.width-n])
	for i := e.col; i < e.col+n; i++ {
		e.clearCell(e.row, i)
	}
}

// deleteCells removes n cells at the cursor, shifting the rest of the line
// left and blanking the right margin.
func (e *emulator) deleteCells(n int) {
	line, styles := e.grid[e.row], e.styles[e.row]
	n = min(n, e.width-e.col)
	copy(line[e.col:], line[e.col+n:])
	copy(styles[e.col:], styles[e.col+n:])
	for i := e.width - n; i < e.width; i++ {
		e.clearCell(e.row, i)
	}
}

// scrollUp removes n lines at row from, shifting the lines below it up to
// the bottom margin and blanking the bottom n lines of the region.
func (e *emulator) scrollUp(from, n int) {
	n = min(n, e.bottom-from+1)
	region, styles := e.grid[from:e.bottom+1], e.styles[from:e.bottom+1]
	copy(region, region[n:])
	copy(styles, styles[n:])
	for i := len(region) - n; i < len(region); i++ {
		region[i] = blankRow(e.width)
		styles[i] = make([]cellStyle, e.width)
	}
}

// scrollDown inserts n blank lines at row from, shifting the lines below it
// down and dropping those pushed past the bottom margin.
func (e *emulator) scrollDown(from, n int) {
	n = min(n, e.bottom-from+1)
	region, styles := e.grid[from:e.bottom+1], e.styles[from:e.bottom+1]
	copy(region[n:], region)
	copy(styles[n:], styles)
	for i := range n {
		region[i] = blankRow(e.width)
		styles[i] = make([]cellStyle, e.width)
	}
}

// blankRow returns a row of width spaces.
func blankRow(width int) []rune {
	row := make([]rune, width)
	for i := range row {
		row[i] = ' '
	}
	return row
}

func (e *emulator) inBounds() bool {
	return e.row >= 0 && e.row < e.height && e.col >= 0 && e.col < e.width
}

func clamp(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
