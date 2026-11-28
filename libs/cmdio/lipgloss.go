package cmdio

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// This file owns how cmdio colors styled output: which color profile a writer
// gets (NewRenderer), stripping or downsampling at write time (Renderer.Writer
// and Renderer.Render), and the Color, LightDark and Hyperlink helpers whose
// lipgloss counterparts change shape in v2. Callers still use lipgloss.Style
// and lipgloss.Width directly.
//
// Callers write styled strings through Renderer.Writer because that is where
// lipgloss v2 applies the profile: v2 styles always emit full-color SGR.

// Color is a color value accepted by lipgloss styles: lipgloss.TerminalColor
// on v1, image/color.Color on v2. Palettes and signatures use it instead of the
// concrete lipgloss.Color string type, which v2 turns into a function.
type Color = lipgloss.TerminalColor

// Renderer carries the color profile for output written to a specific writer.
type Renderer struct {
	lg      *lipgloss.Renderer
	profile colorprofile.Profile
}

// NewRenderer returns a renderer targeting w, along with whether w supports
// color. The profile starts from termenv's probe of w and is adjusted where
// cmdio knows better:
//   - when w doesn't support color (NO_COLOR, TERM=dumb, or w is piped or
//     redirected), styles minted from the renderer emit no SGR escapes and
//     output written through Writer has all escape sequences stripped;
//   - when w is an MSYS2 pty pipe (no ConPTY; see conpty.go), which cmdio
//     treats as a terminal but termenv can't recognize, the profile is raised
//     from Ascii to basic ANSI unless CLICOLOR=0 asks for no color
//     (termenv.EnvNoColor). termenv's CI check is not honored there: it only
//     guesses that CI output isn't a terminal, and this one is.
//
// On every other writer termenv's profile stands, including Ascii for
// CLICOLOR=0 or CI on a terminal.
func NewRenderer(ctx context.Context, w io.Writer) (*Renderer, bool) {
	lg := lipgloss.NewRenderer(w)
	if !SupportsColor(ctx, w) {
		lg.SetColorProfile(termenv.Ascii)
		return &Renderer{lg: lg, profile: colorprofile.NoTTY}, false
	}
	if isMSYSPipe(w) && lg.ColorProfile() == termenv.Ascii && !termenv.EnvNoColor() {
		lg.SetColorProfile(termenv.ANSI)
	}
	return &Renderer{lg: lg, profile: colorprofileFromTermenv(lg.ColorProfile())}, true
}

// NewStyle returns a style whose colors are rendered for this renderer.
func (r *Renderer) NewStyle() lipgloss.Style {
	return r.lg.NewStyle()
}

// Profile returns the renderer's color profile. On v1, styles minted from the
// renderer already render at it; Writer and Render only act when color is off.
func (r *Renderer) Profile() colorprofile.Profile {
	return r.profile
}

// Writer wraps w so styled strings written to it match the renderer's profile.
// When color is off it strips every escape sequence. When color is on it
// passes output through unchanged: on v1, styles minted from this renderer
// already render at its profile. (On v2 the writer always downsamples.)
//
// Stripping works per Write call, as colorprofile.Writer does, so an escape
// sequence split across two writes leaks its tail. Write whole styled strings.
func (r *Renderer) Writer(w io.Writer) io.Writer {
	if r.profile != colorprofile.NoTTY {
		return w
	}
	return &profileWriter{w: &colorprofile.Writer{Forward: w, Profile: r.profile}}
}

// Render returns s as it reaches the target through Writer: escapes stripped
// when color is off, unchanged otherwise.
func (r *Renderer) Render(s string) string {
	var b strings.Builder
	_, _ = r.Writer(&b).Write([]byte(s))
	return b.String()
}

// LightDark returns a function picking the light or dark variant of a color,
// mirroring lipgloss v2's LightDark.
func LightDark(isDark bool) func(light, dark Color) Color {
	return func(light, dark Color) Color {
		if isDark {
			return dark
		}
		return light
	}
}

// HasDarkBackground reports whether the terminal on stdout has a dark
// background, queried at most once. It defaults to true when stdout isn't a
// terminal that can be queried, even if stderr is one.
var HasDarkBackground = sync.OnceValue(lipgloss.HasDarkBackground)

// Hyperlink wraps label in an OSC 8 terminal hyperlink to url. Callers decide
// whether the target supports it; output written through a color-less
// Renderer.Writer has the escape stripped.
func Hyperlink(url, label string) string {
	const osc8, st = "\x1b]8;;", "\x1b\\"
	return osc8 + url + st + label + osc8 + st
}

// colorprofileFromTermenv maps the termenv profile lipgloss v1 renders with to
// the colorprofile profile Renderer.Profile reports.
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
