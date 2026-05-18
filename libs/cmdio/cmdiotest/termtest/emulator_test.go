package termtest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testCols and testRows match a classic VT100 default. The emulator's
// behavior at margins, after wraps, and around screen-clearing should be
// independent of the exact dimensions; tests bake these in so failures
// point at the same coordinates every run.
const (
	testCols = 80
	testRows = 25
)

// blankLine is testCols spaces — the trimmed equivalent of an empty row.
var blankLine = strings.Repeat(" ", testCols)

// run constructs a fresh 80x25 emulator, feeds it the byte stream, and
// returns it. Tests then assert on e.String(), e.row, e.col, etc.
func feed(input string) *emulator {
	e := newEmulator(testCols, testRows)
	_, _ = e.Write([]byte(input))
	return e
}

// line returns the i-th row of e.String() with trailing spaces stripped, so
// tests can use short expected strings.
func line(e *emulator, i int) string {
	rows := strings.Split(e.String(), "\n")
	return strings.TrimRight(rows[i], " ")
}

// pos asserts the cursor sits at (row, col).
func pos(t *testing.T, e *emulator, row, col int) {
	t.Helper()
	assert.Equal(t, row, e.row, "row")
	assert.Equal(t, col, e.col, "col")
}

func TestEmulator_PrintPlain(t *testing.T) {
	e := feed("hello")
	assert.Equal(t, "hello", line(e, 0))
	pos(t, e, 0, 5)
}

func TestEmulator_PrintUTF8(t *testing.T) {
	e := feed("café✔█")
	// café is 4 runes, ✔ and █ are each 1 rune; 6 cells total.
	assert.Equal(t, "café✔█", line(e, 0))
	pos(t, e, 0, 6)
}

func TestEmulator_PrintDropsAtRightMargin(t *testing.T) {
	// Fill the row exactly to width, then one more rune that should be
	// dropped (the emulator deliberately does not wrap; prompts we render
	// fit easily inside 80x25).
	in := strings.Repeat("x", testCols) + "Y"
	e := feed(in)
	assert.Equal(t, strings.Repeat("x", testCols), line(e, 0))
	// Cursor sits past the right margin but didn't wrap.
	assert.Equal(t, testCols, e.col)
	assert.Equal(t, 0, e.row)
}

func TestEmulator_Backspace(t *testing.T) {
	e := feed("abc\b")
	pos(t, e, 0, 2)
	// Backspace doesn't erase — the rune at col 2 ('c') is still there.
	assert.Equal(t, "abc", line(e, 0))
}

func TestEmulator_BackspaceAtCol0IsNoop(t *testing.T) {
	e := feed("\b")
	pos(t, e, 0, 0)
}

func TestEmulator_TabAdvancesToNextEighthColumn(t *testing.T) {
	cases := []struct {
		col      int
		expected int
	}{
		{0, 8},
		{1, 8},
		{7, 8},
		{8, 16},
		{9, 16},
		{63, 64},
		{64, 72},
		{72, testCols - 1}, // clamped: stop would be 80, the last valid col is 79
		{testCols - 1, testCols - 1},
	}
	for _, tc := range cases {
		e := newEmulator(testCols, testRows)
		e.col = tc.col
		_, _ = e.Write([]byte{0x09})
		assert.Equal(t, tc.expected, e.col, "from col %d", tc.col)
	}
}

func TestEmulator_LineFeed(t *testing.T) {
	e := feed("ab\ncd")
	// LF moves down one line; it does not return to col 0.
	assert.Equal(t, "ab", line(e, 0))
	assert.Equal(t, "  cd", line(e, 1))
	pos(t, e, 1, 4)
}

func TestEmulator_LineFeedDoesNotScroll(t *testing.T) {
	// Park at the bottom row and try to LF off the end.
	e := newEmulator(testCols, testRows)
	e.row = testRows - 1
	_, _ = e.Write([]byte{0x0a})
	pos(t, e, testRows-1, 0) // stays at last row
}

func TestEmulator_CarriageReturn(t *testing.T) {
	e := feed("hello\rWORLD")
	assert.Equal(t, "WORLD", line(e, 0))
	pos(t, e, 0, 5)
}

func TestEmulator_CRLFThenPrint(t *testing.T) {
	e := feed("line1\r\nline2")
	assert.Equal(t, "line1", line(e, 0))
	assert.Equal(t, "line2", line(e, 1))
	pos(t, e, 1, 5)
}

func TestEmulator_CursorUp(t *testing.T) {
	// CUU only changes the row; the column is preserved.
	e := feed("a\r\nb\r\nc\x1b[2A")
	pos(t, e, 0, 1)
}

func TestEmulator_CursorUpClampsAtTop(t *testing.T) {
	e := feed("\x1b[99A")
	pos(t, e, 0, 0)
}

func TestEmulator_CursorDownAndVPR(t *testing.T) {
	for _, final := range []byte{'B', 'e'} {
		e := feed("\x1b[3" + string(final))
		pos(t, e, 3, 0)
	}
}

func TestEmulator_CursorDownClampsAtBottom(t *testing.T) {
	e := feed("\x1b[99B")
	pos(t, e, testRows-1, 0)
}

func TestEmulator_CursorForwardAndHPR(t *testing.T) {
	for _, final := range []byte{'C', 'a'} {
		e := feed("\x1b[5" + string(final))
		pos(t, e, 0, 5)
	}
}

func TestEmulator_CursorBack(t *testing.T) {
	e := feed("hello\x1b[3D")
	pos(t, e, 0, 2)
}

func TestEmulator_CursorNextLine(t *testing.T) {
	e := feed("hello\x1b[1E")
	pos(t, e, 1, 0) // CNL is down + col 0
}

func TestEmulator_CursorPreviousLine(t *testing.T) {
	e := feed("\n\nx\x1b[2F")
	pos(t, e, 0, 0)
}

func TestEmulator_CursorToColumn(t *testing.T) {
	// CHA 'G' and HPA '`' are equivalent.
	for _, final := range []byte{'G', '`'} {
		e := feed("hello\x1b[3" + string(final))
		pos(t, e, 0, 2) // 1-based: col 3 → index 2
	}
}

func TestEmulator_CursorToColumnDefaultsToOne(t *testing.T) {
	e := feed("hello\x1b[G")
	pos(t, e, 0, 0)
}

func TestEmulator_CursorToRow(t *testing.T) {
	e := feed("\x1b[5d")
	pos(t, e, 4, 0) // 1-based
}

func TestEmulator_CursorPosition(t *testing.T) {
	cases := []struct {
		seq string
		row int
		col int
	}{
		{"\x1b[H", 0, 0},     // default to (1,1)
		{"\x1b[5;10H", 4, 9}, // 1-based input → 0-based indices
		{"\x1b[5;10f", 4, 9}, // HVP, same semantics
	}
	for _, tc := range cases {
		e := feed(tc.seq)
		assert.Equal(t, tc.row, e.row, "row for %q", tc.seq)
		assert.Equal(t, tc.col, e.col, "col for %q", tc.seq)
	}
}

func TestEmulator_CursorPositionClamps(t *testing.T) {
	// Way out of bounds — should clamp to (height-1, width-1).
	e := feed("\x1b[999;999H")
	pos(t, e, testRows-1, testCols-1)
}

func TestEmulator_EraseLineToEnd(t *testing.T) {
	e := feed("hello world\x1b[7G\x1b[0K")
	// Cursor moves to col 7 (1-based) = index 6. EL 0 wipes from col 6 to
	// end of line.
	assert.Equal(t, "hello", line(e, 0)) // 'w' at index 6 is gone
}

func TestEmulator_EraseLineFromStart(t *testing.T) {
	e := feed("hello world\x1b[7G\x1b[1K")
	// Cursor at col 7 (1-based) = index 6, where 'w' sits. EL 1 wipes
	// columns 0..6 inclusive, taking 'w' with it. Result: 7 spaces, then
	// "orld".
	assert.Equal(t, "       orld", line(e, 0))
}

func TestEmulator_EraseLineWhole(t *testing.T) {
	e := feed("hello\x1b[2K")
	assert.Equal(t, "", line(e, 0))
}

func TestEmulator_EraseLineDefaultsToZero(t *testing.T) {
	// CSI K with no arg behaves like CSI 0 K.
	e := feed("hello\x1b[3G\x1b[K")
	assert.Equal(t, "he", line(e, 0))
}

func TestEmulator_EraseDisplayToEnd(t *testing.T) {
	// CR-LF resets col so each row starts at col 0; otherwise LF alone
	// would carry the cursor column from the previous line.
	e := feed("aaa\r\nbbb\r\nccc\x1b[2;2H\x1b[0J")
	// Cursor at (1,1) inside "bbb"; ED 0 clears (1,1) → end. Row 0 intact,
	// row 1 keeps the first char, rows 2+ blank.
	assert.Equal(t, "aaa", line(e, 0))
	assert.Equal(t, "b", line(e, 1))
	assert.Equal(t, "", line(e, 2))
}

func TestEmulator_EraseDisplayFromStart(t *testing.T) {
	e := feed("aaa\r\nbbb\r\nccc\x1b[2;2H\x1b[1J")
	// Cursor at (1,1). ED 1 clears start → (1,1). Row 0 blank, row 1 has
	// space at cols 0..1 and 'b' at col 2, row 2 intact.
	assert.Equal(t, "", line(e, 0))
	assert.Equal(t, "  b", line(e, 1))
	assert.Equal(t, "ccc", line(e, 2))
}

func TestEmulator_EraseDisplayWhole(t *testing.T) {
	for _, n := range []string{"2", "3"} {
		e := feed("aaa\nbbb\x1b[" + n + "J")
		assert.Equal(t, "", line(e, 0))
		assert.Equal(t, "", line(e, 1))
	}
}

func TestEmulator_SaveAndRestoreCursorSCO(t *testing.T) {
	// CSI s saves; CSI u restores. Cursor moves in between, restore brings
	// it back.
	e := feed("\x1b[5;10H\x1b[s\x1b[1;1H\x1b[u")
	pos(t, e, 4, 9)
}

func TestEmulator_SaveAndRestoreCursorDEC(t *testing.T) {
	// ESC 7 saves; ESC 8 restores.
	e := feed("\x1b[5;10H\x1b7\x1b[1;1H\x1b8")
	pos(t, e, 4, 9)
}

func TestEmulator_RestoreWithoutSaveGoesToOrigin(t *testing.T) {
	// savedRow/savedCol default to 0,0 — restore on a fresh emulator parks
	// at the origin.
	e := feed("\x1b[5;10H\x1b[u")
	pos(t, e, 0, 0)
}

func TestEmulator_SGRIsIgnored(t *testing.T) {
	// Color and style codes must not move the cursor or write cells.
	e := feed("\x1b[1;31;42mhello\x1b[0m")
	assert.Equal(t, "hello", line(e, 0))
	pos(t, e, 0, 5)
}

func TestEmulator_DECPrivateModesAreIgnored(t *testing.T) {
	// Show/hide cursor, bracketed paste, mouse modes — all no-ops here.
	e := feed("\x1b[?25l\x1b[?2004hhello\x1b[?2004l\x1b[?25h")
	assert.Equal(t, "hello", line(e, 0))
	pos(t, e, 0, 5)
}

func TestEmulator_OSCIsIgnored(t *testing.T) {
	// OSC 0 (set window title) terminated by BEL — must not corrupt the
	// screen or the cursor.
	e := feed("\x1b]0;title\x07after")
	assert.Equal(t, "after", line(e, 0))
	pos(t, e, 0, 5)
}

func TestEmulator_PrintAfterCursorMoveOverwrites(t *testing.T) {
	// Move into the middle of an existing line and write — the rune at
	// that cell is replaced.
	e := feed("hello\x1b[1;3Hp")
	assert.Equal(t, "hepl", line(e, 0)[:4])
	pos(t, e, 0, 3)
}

func TestEmulator_ResetClearsGridAndCursor(t *testing.T) {
	e := feed("hello")
	e.reset()
	for i := range testRows {
		assert.Equal(t, blankLine, strings.Split(e.String(), "\n")[i], "row %d", i)
	}
	pos(t, e, 0, 0)
}

func TestEmulator_WriteIsAdditive(t *testing.T) {
	// Bytes split across Write calls should produce the same result as one
	// concatenated call — escape sequences must not lose state at the boundary.
	e := newEmulator(testCols, testRows)
	_, _ = e.Write([]byte("\x1b[5;"))
	_, _ = e.Write([]byte("10Hhello"))
	pos(t, e, 4, 14) // (4,9) + "hello"
	assert.Equal(t, "         hello", line(e, 4))
}
