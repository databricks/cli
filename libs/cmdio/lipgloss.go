package cmdio

import (
	"context"
	"image/color"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

// This file owns how cmdio colors styled output: which color profile a writer
// gets (NewRenderer), stripping or downsampling at write time (Renderer.Writer
// and Renderer.Render), and the Color, LightDark and Hyperlink helpers.
// Callers still use lipgloss.Style and lipgloss.Width directly.
//
// Callers write styled strings through Renderer.Writer because that is where
// the profile is applied: lipgloss v2 styles always emit full-color SGR.
// Bubble Tea programs get a profile via tea.WithColorProfile (see
// teaprogram.go).

// Color is a color value accepted by lipgloss styles. Any image/color.Color
// works; lipgloss.Color returns one.
type Color = color.Color

// Renderer carries the color profile for output written to a specific writer.
type Renderer struct {
	profile colorprofile.Profile
}

// NewRenderer returns a renderer targeting w, along with whether w supports
// color. When it does not (NO_COLOR, TERM=dumb, or w is piped or redirected)
// output written through Writer has all escape sequences stripped. Otherwise
// the profile is the one lipgloss v1 rendered with; see colorDepth.
func NewRenderer(ctx context.Context, w io.Writer) (*Renderer, bool) {
	if !SupportsColor(ctx, w) {
		return &Renderer{profile: colorprofile.NoTTY}, false
	}
	return &Renderer{profile: colorDepth(w)}, true
}

// colorDepth returns the color depth for a writer cmdio already decided
// supports color. lipgloss v2 no longer picks one, and colorprofile's rules
// differ from termenv's (it ignores CLICOLOR=0 and CI, and maps tmux and
// COLORTERM=yes differently), so cmdio keeps asking termenv, which lipgloss v1
// used, to keep colors where they were:
//   - termenv's profile for w, read from the process environment. That is
//     Ascii for CLICOLOR=0 or CI unless CLICOLOR_FORCE is set, and Ascii when
//     termenv can't see a terminal, e.g. TERM unset on Linux and macOS;
//   - raised from Ascii to basic ANSI when w is an MSYS2 pty pipe (no ConPTY;
//     see conpty.go), which cmdio treats as a terminal but termenv can't
//     recognize, unless CLICOLOR=0 asks for no color. termenv's CI check is
//     not honored there: it only guesses that CI output isn't a terminal,
//     and this one is.
func colorDepth(w io.Writer) colorprofile.Profile {
	return colorDepthFor(termenvEnviron{os.Environ()}, isTermenvTTY(w), isMSYSPipe(w))
}

// colorDepthFor is colorDepth for an environment and the two facts it needs
// about the writer, so tests can vary them.
func colorDepthFor(environ termenv.Environ, tty, msysPipe bool) colorprofile.Profile {
	// termenv's terminal check treats CI as "not a terminal" (termenv
	// v0.16.0 termenv.go:32). WithTTY skips that check, so apply it here.
	if environ.Getenv("CI") != "" {
		tty = false
	}
	o := termenv.NewOutput(io.Discard, termenv.WithEnvironment(environ), termenv.WithTTY(tty))
	p := colorprofileFromTermenv(o.EnvColorProfile())
	if msysPipe && p == colorprofile.Ascii && !o.EnvNoColor() {
		return colorprofile.ANSI
	}
	return p
}

// isTermenvTTY reports whether termenv considers w a terminal: only an
// *os.File for which isatty says so. MSYS2 pty pipes and fakes are not.
func isTermenvTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isatty.IsTerminal(f.Fd()) //nolint:forbidigo // mirrors termenv's own check
}

// termenvEnviron adapts a KEY=VALUE list to termenv.Environ.
type termenvEnviron struct{ vars []string }

func (e termenvEnviron) Environ() []string { return e.vars }

func (e termenvEnviron) Getenv(key string) string {
	for _, kv := range e.vars {
		if k, v, ok := strings.Cut(kv, "="); ok && k == key {
			return v
		}
	}
	return ""
}

// colorprofileFromTermenv maps a termenv profile to the colorprofile one
// lipgloss v2 and Bubble Tea use.
func colorprofileFromTermenv(p termenv.Profile) colorprofile.Profile {
	switch p {
	case termenv.TrueColor:
		return colorprofile.TrueColor
	case termenv.ANSI256:
		return colorprofile.ANSI256
	case termenv.ANSI:
		return colorprofile.ANSI
	default:
		return colorprofile.Ascii
	}
}

// NewStyle returns a style to be written through this renderer's Writer.
func (r *Renderer) NewStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

// Profile returns the color profile output is downsampled to.
func (r *Renderer) Profile() colorprofile.Profile {
	return r.profile
}

// Writer wraps w so styled strings written to it are downsampled to the
// renderer's profile; when color is off it strips every escape sequence.
//
// Downsampling works per Write call, as colorprofile.Writer does, so an escape
// sequence split across two writes leaks its tail. Write whole styled strings.
func (r *Renderer) Writer(w io.Writer) io.Writer {
	return &profileWriter{w: &colorprofile.Writer{Forward: w, Profile: r.profile}}
}

// Render downsamples an already-styled string to the renderer's profile.
func (r *Renderer) Render(s string) string {
	var b strings.Builder
	_, _ = r.Writer(&b).Write([]byte(s))
	return b.String()
}

// LightDark returns a function picking the light or dark variant of a color.
// It is lipgloss.LightDark; callers go through cmdio so palettes don't depend
// on the lipgloss major version.
func LightDark(isDark bool) func(light, dark Color) Color {
	return lipgloss.LightDark(isDark)
}

// Hyperlink wraps label in an OSC 8 terminal hyperlink to url. Callers decide
// whether the target supports it; output written through a color-less
// Renderer.Writer has the escape stripped.
func Hyperlink(url, label string) string {
	const osc8, st = "\x1b]8;;", "\x1b\\"
	return osc8 + url + st + label + osc8 + st
}

// profileWriter reports the caller's byte count on success. colorprofile.Writer
// returns the length of the rewritten output, which io.Copy and friends would
// treat as a short write whenever escapes were stripped.
type profileWriter struct {
	w io.Writer
}

func (p *profileWriter) Write(b []byte) (int, error) {
	if _, err := p.w.Write(b); err != nil {
		return 0, err
	}
	return len(b), nil
}
