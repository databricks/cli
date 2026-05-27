package cmdio

import (
	"context"
	"fmt"
	"text/template"
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

// Bold renders msg in bold.
func Bold(ctx context.Context, msg string) string { return render(ctx, ansiBold, msg) }

// Red renders msg in red.
func Red(ctx context.Context, msg string) string { return render(ctx, ansiRed, msg) }

// Green renders msg in green.
func Green(ctx context.Context, msg string) string { return render(ctx, ansiGreen, msg) }

// Yellow renders msg in yellow.
func Yellow(ctx context.Context, msg string) string { return render(ctx, ansiYellow, msg) }

// Blue renders msg in blue.
func Blue(ctx context.Context, msg string) string { return render(ctx, ansiBlue, msg) }

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
		"red":     templateColor(ctx, ansiRed),
		"green":   templateColor(ctx, ansiGreen),
		"blue":    templateColor(ctx, ansiBlue),
		"yellow":  templateColor(ctx, ansiYellow),
		"magenta": templateColor(ctx, ansiMagenta),
		"cyan":    templateColor(ctx, ansiCyan),
		"bold":    templateColor(ctx, ansiBold),
		"italic":  templateColor(ctx, ansiItalic),
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
