package termtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStyled_PlainScreenMatchesString(t *testing.T) {
	e := feed("hello\r\nworld")
	assert.Equal(t, "hello\nworld", e.styledString())
}

func TestStyled_SixteenColorAndReset(t *testing.T) {
	e := feed("\x1b[34m?\x1b[0m Pick")
	assert.Equal(t, "{fg=4}?{/} Pick", e.styledString())
}

func TestStyled_BrightColorsAreEightToFifteen(t *testing.T) {
	e := feed("\x1b[92ma\x1b[0m\x1b[101mb\x1b[0m")
	assert.Equal(t, "{fg=10}a{bg=9}b{/}", e.styledString())
}

func TestStyled_ExtendedColorsKeepTheirForm(t *testing.T) {
	e := feed("\x1b[38;5;208ma\x1b[38;2;143;179;220mb\x1b[48;5;57mc\x1b[0m")
	assert.Equal(t, "{fg=@208}a{fg=#8fb3dc}b{fg=#8fb3dc bg=@57}c{/}", e.styledString())
}

func TestStyled_ColonSeparatedExtendedColor(t *testing.T) {
	e := feed("\x1b[38:5:10mx\x1b[m")
	assert.Equal(t, "{fg=@10}x{/}", e.styledString())
}

func TestStyled_AttributesInFixedOrder(t *testing.T) {
	e := feed("\x1b[7;4;3;2;1;32mx\x1b[m")
	assert.Equal(t, "{fg=2 bold faint italic underline reverse}x{/}", e.styledString())
}

func TestStyled_AttributeResets(t *testing.T) {
	e := feed("\x1b[1;3;4;7;31;42ma\x1b[22;23;24;27;39;49mb")
	assert.Equal(t, "{fg=1 bg=2 bold italic underline reverse}a{/}b", e.styledString())
}

func TestStyled_BlinkStrikeDoubleUnderlineOverline(t *testing.T) {
	e := feed("\x1b[5;9;21;53mx\x1b[0my")
	assert.Equal(t, "{double-underline blink strikethrough overline}x{/}y", e.styledString())
}

func TestStyled_BlinkStrikeDoubleUnderlineOverlineResets(t *testing.T) {
	e := feed("\x1b[5;9;21;53ma\x1b[25mb\x1b[29mc\x1b[24md\x1b[55me")
	assert.Equal(t, "{double-underline blink strikethrough overline}a"+
		"{double-underline strikethrough overline}b"+
		"{double-underline overline}c"+
		"{overline}d{/}e", e.styledString())
}

func TestStyled_UnderlineStylesFromColonSubparameter(t *testing.T) {
	e := feed("\x1b[4:1ma\x1b[4:2mb\x1b[4:3mc\x1b[4:4md\x1b[4:5me\x1b[4:0mf")
	assert.Equal(t, "{underline}a{double-underline}b{curly-underline}c{dotted-underline}d{dashed-underline}e{/}f", e.styledString())
}

func TestStyled_ColonUnderlineDoesNotSwallowNextParameter(t *testing.T) {
	e := feed("\x1b[4:3;1mx\x1b[0m")
	assert.Equal(t, "{bold curly-underline}x{/}", e.styledString())
}

func TestStyled_ColonTrueColorWithAndWithoutColorSpace(t *testing.T) {
	e := feed("\x1b[38:2::143:179:220mx\x1b[38:2:143:179:220my\x1b[48:2::1:2:3mz\x1b[48:5:57mw\x1b[0m")
	assert.Equal(t, "{fg=#8fb3dc}xy{fg=#8fb3dc bg=#010203}z{fg=#8fb3dc bg=@57}w{/}", e.styledString())
}

func TestStyled_InvalidExtendedColorConsumesNothing(t *testing.T) {
	// 38 with an unknown sub-mode is ignored, so 9 and 1 apply as plain codes.
	e := feed("\x1b[31;38;9;1mx\x1b[0m")
	assert.Equal(t, "{fg=1 bold strikethrough}x{/}", e.styledString())
}

func TestStyled_InvalidColonColorKeepsPen(t *testing.T) {
	e := feed("\x1b[31m\x1b[38:9:1mx\x1b[0m")
	assert.Equal(t, "{fg=1}x{/}", e.styledString())
}

func TestStyled_UnderlineColorIsSkipped(t *testing.T) {
	// 58 is not modelled, but both its colon and semicolon forms must consume
	// their values so the attributes after them still apply.
	e := feed("\x1b[58:2::1:2:3;4mx\x1b[58;5;1;1mw\x1b[0m")
	assert.Equal(t, "{underline}x{bold underline}w{/}", e.styledString())
}

func TestStyled_StyleChangeEmitsOneMarker(t *testing.T) {
	e := feed("\x1b[1ma\x1b[32mb\x1b[0mc")
	assert.Equal(t, "{bold}a{fg=2 bold}b{/}c", e.styledString())
}

func TestStyled_SpacesStayInsideSpan(t *testing.T) {
	e := feed("\x1b[1ma b\x1b[0m")
	assert.Equal(t, "{bold}a b{/}", e.styledString())
}

func TestStyled_SpanClosesAtLineEndAndReopens(t *testing.T) {
	e := feed("\x1b[1mab\r\ncd\x1b[0m")
	assert.Equal(t, "{bold}ab{/}\n{bold}cd{/}", e.styledString())
}

func TestStyled_TrailingStyledBlanksAreKept(t *testing.T) {
	e := feed("a\x1b[44m  \x1b[0m   ")
	assert.Equal(t, "a{bg=4}  {/}", e.styledString())
}

func TestStyled_LiteralBraceIsEscaped(t *testing.T) {
	e := feed("{x} \x1b[1m{\x1b[0m")
	assert.Equal(t, "{{x} {bold}{{{/}", e.styledString())
}

func TestStyled_EraseResetsCellStyle(t *testing.T) {
	e := feed("\x1b[31mred\x1b[0m\r\x1b[K")
	assert.Empty(t, e.styledString())
}

func TestStyled_OverwriteTakesNewStyle(t *testing.T) {
	e := feed("\x1b[31mab\x1b[0m\rX")
	assert.Equal(t, "X{fg=1}b{/}", e.styledString())
}

func TestModes_PrivateModesAndActiveAtEnd(t *testing.T) {
	e := feed("\x1b[?25l\x1b[?2004h\x1b[?2004l\x1b[?1049h")
	assert.Equal(t, "-- changes --\n"+
		"?25 reset\n"+
		"?2004 set\n"+
		"?2004 reset\n"+
		"?1049 set\n"+
		"-- active at end --\n"+
		"?25 reset\n"+
		"?1049 set\n", e.modesString())
}

func TestModes_CursorShownAgainIsNotActive(t *testing.T) {
	e := feed("\x1b[?25l\x1b[?25h")
	assert.Equal(t, "-- changes --\n?25 reset\n?25 set\n-- active at end --\n", e.modesString())
}

func TestModes_RepeatsAreKept(t *testing.T) {
	e := feed("\x1b[?25l\x1b[?25l\x1b[?25l")
	assert.Equal(t, "-- changes --\n?25 reset\n?25 reset\n?25 reset\n-- active at end --\n?25 reset\n", e.modesString())
}

func TestModes_KittyKeyboardAndModifyOtherKeys(t *testing.T) {
	e := feed("\x1b[>4;2m\x1b[>1u\x1b[=3;1u\x1b[?5W")
	assert.Equal(t, "-- changes --\n"+
		"modifyOtherKeys 2\n"+
		"kitty-keyboard push 1\n"+
		"kitty-keyboard set 3\n"+
		"DECST8C\n"+
		"-- active at end --\n"+
		"kitty-keyboard stack [1]\n"+
		"modifyOtherKeys 2\n", e.modesString())
}

func TestModes_KittyPopAndModifyOtherKeysReset(t *testing.T) {
	e := feed("\x1b[>1u\x1b[>4;2m\x1b[<1u\x1b[>4m")
	assert.Equal(t, "-- changes --\n"+
		"kitty-keyboard push 1\n"+
		"modifyOtherKeys 2\n"+
		"kitty-keyboard pop 1\n"+
		"modifyOtherKeys 0\n"+
		"-- active at end --\n", e.modesString())
}

func TestModes_PrefixedUDoesNotRestoreCursor(t *testing.T) {
	// "CSI > 1 u" is a kitty keyboard push, not SCO restore-cursor.
	e := feed("\x1b[sab\x1b[>1ucd")
	assert.Equal(t, "abcd", line(e, 0))
	pos(t, e, 0, 4)
}

func TestQueries_Logged(t *testing.T) {
	e := feed("\x1b[?2026$p\x1b[?2027$p\x1b[?u\x1b]11;?\x1b\\\x1b[6n\x1b[c\x1b[>c\x1b[>q")
	assert.Equal(t, "DECRQM ?2026\n"+
		"DECRQM ?2027\n"+
		"kitty-keyboard-flags\n"+
		"OSC 11\n"+
		"CPR\n"+
		"DA1\n"+
		"DA2\n"+
		"XTVERSION\n", e.queriesString())
}

func TestQueries_QueriesDoNotChangeText(t *testing.T) {
	e := feed("a\x1b[?2026$p\x1b]11;?\x07b")
	assert.Equal(t, "ab", line(e, 0))
}

func TestDefaultReply(t *testing.T) {
	assert.Equal(t, "\x1b[?2026;2$y", DefaultReply("DECRQM ?2026"))
	assert.Equal(t, "\x1b[?0u", DefaultReply("kitty-keyboard-flags"))
	assert.Equal(t, "\x1b]11;rgb:0000/0000/0000\x1b\\", DefaultReply("OSC 11"))
	assert.Equal(t, "\x1b[1;1R", DefaultReply("CPR"))
	assert.Empty(t, DefaultReply("XTVERSION"))
}
