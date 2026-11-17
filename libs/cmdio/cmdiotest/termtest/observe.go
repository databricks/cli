package termtest

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// cellStyle is the SGR state of one screen cell. The zero value is the
// terminal's default style. Colors keep the form they were sent in: a 16-color
// index as a number, a 256-color index with an `@` prefix, or truecolor as
// `#rrggbb`. The color depth a program picks is part of what the styled
// goldens pin.
type cellStyle struct {
	fg, bg        string
	bold          bool
	faint         bool
	italic        bool
	underline     string // "", "single", "double", "curly", "dotted" or "dashed"
	blink         bool
	reverse       bool
	strikethrough bool
	overline      bool
}

// marker renders the style as the space-separated attribute list used inside
// a `{...}` marker, in a fixed order.
func (s cellStyle) marker() string {
	var parts []string
	if s.fg != "" {
		parts = append(parts, "fg="+s.fg)
	}
	if s.bg != "" {
		parts = append(parts, "bg="+s.bg)
	}
	for _, a := range []struct {
		on   bool
		name string
	}{
		{s.bold, "bold"},
		{s.faint, "faint"},
		{s.italic, "italic"},
		{s.underline != "" && s.underline != "single", s.underline + "-underline"},
		{s.underline == "single", "underline"},
		{s.blink, "blink"},
		{s.reverse, "reverse"},
		{s.strikethrough, "strikethrough"},
		{s.overline, "overline"},
	} {
		if a.on {
			parts = append(parts, a.name)
		}
	}
	return strings.Join(parts, " ")
}

// underlineStyles maps the subparameter of SGR 4:n to an underline style.
var underlineStyles = map[int]string{1: "single", 2: "double", 3: "curly", 4: "dotted", 5: "dashed"}

// applySGR updates the pen from the parameters of an SGR (CSI ... m)
// sequence. A colon group such as 4:3 or 38:2::r:g:b arrives as one group,
// with the sub-parameters packed with HasMore; semicolon forms such as
// 38;2;r;g;b arrive as separate single-value groups.
func (e *emulator) applySGR(params ansi.Params) {
	if len(params) == 0 {
		e.pen = cellStyle{}
		return
	}
	groups := sgrGroups(params)
	for i := 0; i < len(groups); i++ {
		g := groups[i]
		if len(g) > 1 {
			e.applyColonSGR(g)
			continue
		}
		switch v := g[0]; {
		case v == 0:
			e.pen = cellStyle{}
		case v == 1:
			e.pen.bold = true
		case v == 2:
			e.pen.faint = true
		case v == 3:
			e.pen.italic = true
		case v == 4:
			e.pen.underline = "single"
		case v == 5:
			e.pen.blink = true
		case v == 7:
			e.pen.reverse = true
		case v == 9:
			e.pen.strikethrough = true
		case v == 21:
			e.pen.underline = "double"
		case v == 22:
			e.pen.bold, e.pen.faint = false, false
		case v == 23:
			e.pen.italic = false
		case v == 24:
			e.pen.underline = ""
		case v == 25:
			e.pen.blink = false
		case v == 27:
			e.pen.reverse = false
		case v == 29:
			e.pen.strikethrough = false
		case v >= 30 && v <= 37:
			e.pen.fg = strconv.Itoa(v - 30)
		case v == 39:
			e.pen.fg = ""
		case v >= 40 && v <= 47:
			e.pen.bg = strconv.Itoa(v - 40)
		case v == 49:
			e.pen.bg = ""
		case v >= 90 && v <= 97:
			e.pen.fg = strconv.Itoa(v - 90 + 8)
		case v >= 100 && v <= 107:
			e.pen.bg = strconv.Itoa(v - 100 + 8)
		case v == 53:
			e.pen.overline = true
		case v == 55:
			e.pen.overline = false
		case v == 38 || v == 48 || v == 58:
			// The color follows as separate values: 38;5;n or 38;2;r;g;b.
			// When they do not form a color, nothing is consumed and the
			// following values apply as plain SGR codes, as in xterm.
			if c, n, ok := extendedColor(singleValues(groups[i+1:])); ok {
				i += n
				e.setColor(v, c)
			}
		}
	}
}

// sgrGroups splits SGR parameters into groups: a group is one parameter
// followed by its colon sub-parameters, if any.
func sgrGroups(params ansi.Params) [][]int {
	var groups [][]int
	var cur []int
	for _, p := range params {
		cur = append(cur, p.Param(0))
		if !p.HasMore() {
			groups = append(groups, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		groups = append(groups, cur)
	}
	return groups
}

// singleValues returns the values of the leading single-value groups, which
// are the semicolon-separated parameters of an extended color.
func singleValues(groups [][]int) []int {
	var vals []int
	for _, g := range groups {
		if len(g) != 1 {
			break
		}
		vals = append(vals, g[0])
	}
	return vals
}

// applyColonSGR applies a colon group: 4:n sets the underline style, and
// 38/48/58 take an extended color in the colon form.
func (e *emulator) applyColonSGR(g []int) {
	switch g[0] {
	case 4:
		if style, ok := underlineStyles[g[1]]; ok {
			e.pen.underline = style
		} else if g[1] == 0 {
			e.pen.underline = ""
		}
	case 38, 48, 58:
		if c, ok := colonColor(g[1:]); ok {
			e.setColor(g[0], c)
		}
	}
}

// setColor sets the foreground (38) or background (48). Underline colors (58)
// are not modelled; their values are skipped by the callers.
func (e *emulator) setColor(code int, c string) {
	switch code {
	case 38:
		e.pen.fg = c
	case 48:
		e.pen.bg = c
	}
}

// extendedColor decodes the semicolon form of a 38, 48 or 58 SGR code, given
// the values after it. It returns the color, how many values it consumed, and
// false if the values do not form a color.
func extendedColor(rest []int) (string, int, bool) {
	if len(rest) == 0 {
		return "", 0, false
	}
	switch rest[0] {
	case 5:
		if len(rest) >= 2 {
			return "@" + strconv.Itoa(rest[1]), 2, true
		}
	case 2:
		if len(rest) >= 4 {
			return rgbColor(rest[1], rest[2], rest[3]), 4, true
		}
	}
	return "", 0, false
}

// colonColor decodes the sub-parameters after 38, 48 or 58 in the colon form:
// 5:n, or 2:r:g:b and 2::r:g:b (with an optional color space id). It returns
// false if they do not form a color.
func colonColor(sub []int) (string, bool) {
	if len(sub) == 0 {
		return "", false
	}
	switch sub[0] {
	case 5:
		if len(sub) >= 2 {
			return "@" + strconv.Itoa(sub[1]), true
		}
	case 2:
		switch {
		case len(sub) >= 5:
			return rgbColor(sub[2], sub[3], sub[4]), true
		case len(sub) >= 4:
			return rgbColor(sub[1], sub[2], sub[3]), true
		}
	}
	return "", false
}

func rgbColor(r, g, b int) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// modeEvent is one terminal mode change seen in the output stream.
type modeEvent struct {
	name  string
	value string
}

func (m modeEvent) String() string {
	if m.value == "" {
		return m.name
	}
	return m.name + " " + m.value
}

// observeCsi records the CSI sequences that do not move the cursor but that
// the modes, queries and styles goldens pin. It runs before the cell-level
// handling in handleCsi.
func (e *emulator) observeCsi(cmd ansi.Cmd, params ansi.Params) {
	prefix, inter, final := cmd.Prefix(), cmd.Intermediate(), cmd.Final()
	first, _, _ := params.Param(0, 0)
	switch {
	case prefix == 0 && inter == 0 && final == 'm':
		e.applySGR(params)
	case prefix == '?' && inter == 0 && (final == 'h' || final == 'l'):
		for _, p := range params {
			n := p.Param(0)
			set := final == 'h'
			e.privateModes[n] = set
			e.recordMode(modeEvent{name: "?" + strconv.Itoa(n), value: setWord(set)})
		}
	case prefix == '?' && inter == '$' && final == 'p':
		e.recordQuery("DECRQM ?" + strconv.Itoa(first))
	case prefix == '?' && inter == 0 && final == 'u':
		e.recordQuery("kitty-keyboard-flags")
	case prefix == '>' && inter == 0 && final == 'u':
		e.kittyStack = append(e.kittyStack, first)
		e.recordMode(modeEvent{name: "kitty-keyboard", value: "push " + strconv.Itoa(first)})
	case prefix == '<' && inter == 0 && final == 'u':
		n := max(first, 1)
		e.kittyStack = e.kittyStack[:max(len(e.kittyStack)-n, 0)]
		e.recordMode(modeEvent{name: "kitty-keyboard", value: "pop " + strconv.Itoa(n)})
	// The mode parameter (1 set, 2 OR, 3 AND-NOT) is not decoded: the log
	// records the flags only.
	case prefix == '=' && inter == 0 && final == 'u':
		e.recordMode(modeEvent{name: "kitty-keyboard", value: "set " + strconv.Itoa(first)})
	case prefix == '>' && inter == 0 && final == 'm':
		if first == 4 {
			level, _, _ := params.Param(1, 0)
			e.modifyOtherKeys = level
			e.recordMode(modeEvent{name: "modifyOtherKeys", value: strconv.Itoa(level)})
		}
	case prefix == '?' && inter == 0 && final == 'W' && first == 5:
		e.recordMode(modeEvent{name: "DECST8C"})
	case prefix == 0 && inter == 0 && final == 'n' && first == 6:
		e.recordQuery("CPR")
	case prefix == 0 && inter == 0 && final == 'c':
		e.recordQuery("DA1")
	case prefix == '>' && inter == 0 && final == 'c':
		e.recordQuery("DA2")
	case prefix == '>' && inter == 0 && final == 'q':
		e.recordQuery("XTVERSION")
	}
}

// handleOsc records OSC queries (a "?" argument) such as OSC 11, the
// background color query.
func (e *emulator) handleOsc(cmd int, data []byte) {
	if strings.HasSuffix(string(data), ";?") {
		e.recordQuery("OSC " + strconv.Itoa(cmd))
	}
}

// recordMode keeps every change, including an immediate repeat, so a program
// that writes the same mode twice shows up in the golden.
func (e *emulator) recordMode(ev modeEvent) {
	e.modeEvents = append(e.modeEvents, ev)
}

func (e *emulator) recordQuery(q string) {
	e.queries = append(e.queries, q)
}

// styledString renders the screen like String but with inline style
// markers: `{attrs}` opens a styled span, `{/}` returns to the default style,
// and a change between two non-default styles emits a single new opening
// marker. Every line closes its span before the newline. A literal `{` is
// written as `{{`. Trailing default-styled blanks are trimmed and trailing
// empty lines dropped, matching Snapshot.
func (e *emulator) styledString() string {
	lines := make([]string, len(e.grid))
	for r, row := range e.grid {
		end := len(row)
		for end > 0 && (row[end-1] == ' ' || row[end-1] == '\t') && e.styles[r][end-1] == (cellStyle{}) {
			end--
		}
		var b strings.Builder
		cur := cellStyle{}
		for c := range end {
			st := e.styles[r][c]
			if st != cur {
				if st == (cellStyle{}) {
					b.WriteString("{/}")
				} else {
					b.WriteString("{" + st.marker() + "}")
				}
				cur = st
			}
			if row[c] == '{' {
				b.WriteString("{{")
			} else {
				b.WriteRune(row[c])
			}
		}
		if cur != (cellStyle{}) {
			b.WriteString("{/}")
		}
		lines[r] = b.String()
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// modesString renders the mode-change log and the modes still active, i.e.
// differing from a terminal's power-on state (cursor visible, everything else
// off).
func (e *emulator) modesString() string {
	var b strings.Builder
	b.WriteString("-- changes --\n")
	for _, ev := range e.modeEvents {
		b.WriteString(ev.String() + "\n")
	}
	b.WriteString("-- active at end --\n")
	keys := make([]int, 0, len(e.privateModes))
	for k := range e.privateModes {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		set := e.privateModes[k]
		// The cursor (?25) is visible by default; every other mode is off.
		if (k == 25 && !set) || (k != 25 && set) {
			fmt.Fprintf(&b, "?%d %s\n", k, setWord(set))
		}
	}
	if len(e.kittyStack) > 0 {
		fmt.Fprintf(&b, "kitty-keyboard stack %v\n", e.kittyStack)
	}
	if e.modifyOtherKeys != 0 {
		fmt.Fprintf(&b, "modifyOtherKeys %d\n", e.modifyOtherKeys)
	}
	return b.String()
}

// queriesString renders the queries seen in the output stream, one per line.
func (e *emulator) queriesString() string {
	var b strings.Builder
	for _, q := range e.queries {
		b.WriteString(q + "\n")
	}
	return b.String()
}

// setWord is how a mode change is written in the mode log.
func setWord(set bool) string {
	if set {
		return "set"
	}
	return "reset"
}
