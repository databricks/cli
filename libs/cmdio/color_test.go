package cmdio_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
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
