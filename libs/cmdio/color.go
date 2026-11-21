package cmdio

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/charmbracelet/lipgloss"
)

// SGR (Select Graphic Rendition) escapes; see
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR
const (
	ansiReset     = "\x1b[0m"
	ansiBold      = "\x1b[1m"
	ansiFaint     = "\x1b[2m"
	ansiItalic    = "\x1b[3m"
	ansiUnderline = "\x1b[4m"
	ansiRed       = "\x1b[31m"
	ansiGreen     = "\x1b[32m"
	ansiYellow    = "\x1b[33m"
	ansiBlue      = "\x1b[34m"
	ansiMagenta   = "\x1b[35m"
	ansiCyan      = "\x1b[36m"
	ansiHiBlack   = "\x1b[90m"
	ansiHiBlue    = "\x1b[94m"
	ansiBoldGreen = "\x1b[32;1m"
	ansiBoldBlue  = "\x1b[34;1m"
)

// colorEnabled reports whether ctx permits colorized output. Returns false
// when no cmdIO is attached so the helpers can be called from contexts
// without one (e.g. mutator unit tests) without panicking.
func colorEnabled(ctx context.Context) bool {
	c, ok := ctx.Value(cmdIOKey).(*cmdIO)
	if !ok {
		return false
	}
	return c.capabilities.SupportsStdoutColor()
}

func render(ctx context.Context, code, msg string) string {
	if !colorEnabled(ctx) {
		return msg
	}
	return code + msg + ansiReset
}

// Bold renders msg with increased intensity (bold).
func Bold(ctx context.Context, msg string) string { return render(ctx, ansiBold, msg) }

// Faint renders msg with decreased intensity (faint, dim).
func Faint(ctx context.Context, msg string) string { return render(ctx, ansiFaint, msg) }

// Italic renders msg in italic.
func Italic(ctx context.Context, msg string) string { return render(ctx, ansiItalic, msg) }

// Underline renders msg underlined.
func Underline(ctx context.Context, msg string) string { return render(ctx, ansiUnderline, msg) }

// Red renders msg in red.
func Red(ctx context.Context, msg string) string { return render(ctx, ansiRed, msg) }

// Green renders msg in green.
func Green(ctx context.Context, msg string) string { return render(ctx, ansiGreen, msg) }

// Yellow renders msg in yellow.
func Yellow(ctx context.Context, msg string) string { return render(ctx, ansiYellow, msg) }

// Blue renders msg in blue.
func Blue(ctx context.Context, msg string) string { return render(ctx, ansiBlue, msg) }

// Magenta renders msg in magenta.
func Magenta(ctx context.Context, msg string) string { return render(ctx, ansiMagenta, msg) }

// Cyan renders msg in cyan.
func Cyan(ctx context.Context, msg string) string { return render(ctx, ansiCyan, msg) }

// HiBlack renders msg in bright black (gray).
func HiBlack(ctx context.Context, msg string) string { return render(ctx, ansiHiBlack, msg) }

// HiBlue renders msg in bright blue.
func HiBlue(ctx context.Context, msg string) string { return render(ctx, ansiHiBlue, msg) }

// RenderFuncMap returns a template.FuncMap with color helpers bound to ctx.
// Templates use the printf-style signature (`{{ green "%d" .JobId }}`) so the
// helpers accept a format string + args.
func RenderFuncMap(ctx context.Context) template.FuncMap {
	return template.FuncMap{
		"red":       templateColor(ctx, ansiRed),
		"green":     templateColor(ctx, ansiGreen),
		"blue":      templateColor(ctx, ansiBlue),
		"yellow":    templateColor(ctx, ansiYellow),
		"magenta":   templateColor(ctx, ansiMagenta),
		"cyan":      templateColor(ctx, ansiCyan),
		"bold":      templateColor(ctx, ansiBold),
		"faint":     templateColor(ctx, ansiFaint),
		"italic":    templateColor(ctx, ansiItalic),
		"underline": templateColor(ctx, ansiUnderline),
	}
}

func templateColor(ctx context.Context, code string) func(string, ...any) string {
	return func(format string, a ...any) string {
		msg := format
		if len(a) > 0 {
			msg = fmt.Sprintf(format, a...)
		}
		return render(ctx, code, msg)
	}
}

// Width returns the visible cell width of s. ANSI color escapes (such as those
// emitted by the helpers above) are ignored, and wide glyphs like CJK
// characters and emoji are counted as two cells. Use this instead of len() or
// utf8.RuneCountInString when aligning columns of rendered text.
func Width(s string) int {
	return lipgloss.Width(s)
}

// PadRight returns s padded with trailing spaces to a visible width of n (see
// Width). Because it measures the rendered string, cells already wrapped by the
// color helpers stay aligned. Strings at or beyond width n are returned as-is.
func PadRight(s string, n int) string {
	if pad := n - Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// PadLeft returns s padded with leading spaces to a visible width of n (see
// Width), right-aligning the rendered content. Strings at or beyond width n are
// returned as-is.
func PadLeft(s string, n int) string {
	if pad := n - Width(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}
