package cmdio_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ttyContext(t *testing.T) context.Context {
	t.Helper()
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	return ctx
}

func noColorContext(t *testing.T) context.Context {
	t.Helper()
	return cmdio.MockDiscard(t.Context())
}

func TestColorHelpersEmitSGRWhenEnabled(t *testing.T) {
	ctx := ttyContext(t)

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Bold", cmdio.Bold(ctx, "id"), "\x1b[1mid\x1b[0m"},
		{"Faint", cmdio.Faint(ctx, "hint"), "\x1b[2mhint\x1b[0m"},
		{"Italic", cmdio.Italic(ctx, "em"), "\x1b[3mem\x1b[0m"},
		{"Underline", cmdio.Underline(ctx, "link"), "\x1b[4mlink\x1b[0m"},
		{"Red", cmdio.Red(ctx, "hello"), "\x1b[31mhello\x1b[0m"},
		{"Green", cmdio.Green(ctx, "ok"), "\x1b[32mok\x1b[0m"},
		{"Yellow", cmdio.Yellow(ctx, "warn"), "\x1b[33mwarn\x1b[0m"},
		{"Blue", cmdio.Blue(ctx, "info"), "\x1b[34minfo\x1b[0m"},
		{"Magenta", cmdio.Magenta(ctx, "trace"), "\x1b[35mtrace\x1b[0m"},
		{"Cyan", cmdio.Cyan(ctx, "debug"), "\x1b[36mdebug\x1b[0m"},
		{"HiBlack", cmdio.HiBlack(ctx, "dim"), "\x1b[90mdim\x1b[0m"},
		{"HiBlue", cmdio.HiBlue(ctx, "APP"), "\x1b[94mAPP\x1b[0m"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestColorHelpersReturnPlainWhenDisabled(t *testing.T) {
	ctx := noColorContext(t)

	assert.Equal(t, "hello", cmdio.Red(ctx, "hello"))
	assert.Equal(t, "warn", cmdio.Yellow(ctx, "warn"))
}

// Mutator/library code paths can run with a bare context (e.g. inside a unit
// test that calls bundle.Apply with t.Context()). The helpers must degrade
// to plain text rather than panic.
func TestColorHelpersDoNotPanicWithoutCmdIO(t *testing.T) {
	ctx := t.Context()

	assert.Equal(t, "hello", cmdio.Red(ctx, "hello"))
	assert.Equal(t, "ok", cmdio.Green(ctx, "ok"))
	assert.Equal(t, "label: ", cmdio.Cyan(ctx, "label: "))
}

func TestWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "hello", 5},
		{"ansi wrapped", "\x1b[1mhello\x1b[0m", 5},
		{"nested ansi", "\x1b[1m\x1b[36mid\x1b[0m", 2},
		{"fullwidth latin", "ＮＡＭＥ", 8},
		{"emoji", "🚀", 2},
		{"ansi wrapped fullwidth", "\x1b[36mＣＬＩ\x1b[0m", 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, cmdio.Width(c.in))
		})
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"pads ascii", "hi", 5, "hi   "},
		{"exact width unchanged", "hello", 5, "hello"},
		{"over width unchanged", "toolong", 3, "toolong"},
		{"measures past ansi", "\x1b[1mhi\x1b[0m", 5, "\x1b[1mhi\x1b[0m   "},
		{"counts wide glyphs", "ＣＬＩ", 8, "ＣＬＩ  "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, cmdio.PadRight(c.in, c.n))
		})
	}
}

func TestPadLeft(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"pads ascii", "hi", 5, "   hi"},
		{"exact width unchanged", "hello", 5, "hello"},
		{"over width unchanged", "toolong", 3, "toolong"},
		{"measures past ansi", "\x1b[1mhi\x1b[0m", 5, "   \x1b[1mhi\x1b[0m"},
		{"counts wide glyphs", "ＣＬＩ", 8, "  ＣＬＩ"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, cmdio.PadLeft(c.in, c.n))
		})
	}
}

func TestNewRendererStripsWhenNoColor(t *testing.T) {
	ctx := noColorContext(t)

	// A bytes.Buffer is never a TTY, so output through the renderer must carry
	// no escapes, whether styles render color themselves or not.
	var buf bytes.Buffer
	r, color := cmdio.NewRenderer(ctx, &buf)
	assert.False(t, color)
	assert.Equal(t, colorprofile.NoTTY, r.Profile())
	assert.Equal(t, "hi", r.Render(r.NewStyle().Bold(true).Render("hi")))

	// Strings styled outside the renderer are stripped at write time too.
	n, err := io.Copy(r.Writer(&buf), strings.NewReader("\x1b[1mbold\x1b[0m "+cmdio.Hyperlink("https://h.test", "link")))
	require.NoError(t, err)
	assert.Equal(t, int64(len("\x1b[1mbold\x1b[0m "+cmdio.Hyperlink("https://h.test", "link"))), n)
	assert.Equal(t, "bold link", buf.String())
}

func TestLightDark(t *testing.T) {
	light, dark := lipgloss.Color("#ffffff"), lipgloss.Color("#000000")
	assert.Equal(t, dark, cmdio.LightDark(true)(light, dark))
	assert.Equal(t, light, cmdio.LightDark(false)(light, dark))
}

func TestNewRendererKeepsTermenvProfile(t *testing.T) {
	ctx := ttyContext(t)

	// cmdio says a FakeTTY supports color, but it isn't an MSYS2 pty pipe, so
	// termenv's own decision (Ascii: it can't see a terminal) stands. The same
	// holds for CLICOLOR=0 or CI on a real terminal. The MSYS2 pipe case is
	// covered in conpty_windows_test.go.
	r, color := cmdio.NewRenderer(ctx, cmdio.FakeTTY(&bytes.Buffer{}))
	assert.True(t, color)
	assert.Equal(t, colorprofile.Ascii, r.Profile())
}

func TestRenderFuncMap(t *testing.T) {
	ctx := ttyContext(t)
	fm := cmdio.RenderFuncMap(ctx)

	for _, name := range []string{"red", "green", "blue", "yellow", "magenta", "cyan", "bold", "faint", "italic", "underline"} {
		_, ok := fm[name].(func(string, ...any) string)
		assert.True(t, ok, "FuncMap missing %q or wrong signature", name)
	}

	red := fm["red"].(func(string, ...any) string)
	assert.Equal(t, "\x1b[31mhi 1\x1b[0m", red("%s %d", "hi", 1))
	assert.Equal(t, "\x1b[31mhi\x1b[0m", red("hi"))
}
