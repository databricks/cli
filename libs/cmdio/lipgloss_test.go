package cmdio

import (
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
)

// These mirror what NewRenderer decided on lipgloss v1, where termenv picked
// the profile (see TestNewRendererKeepsTermenvProfile and the MSYS2 cases in
// conpty_windows_test.go).
func TestColorDepth(t *testing.T) {
	cases := []struct {
		name     string
		env      []string
		tty      bool
		msysPipe bool
		want     colorprofile.Profile
	}{
		{"not a terminal", []string{"TERM=xterm-256color"}, false, false, colorprofile.Ascii},
		{"16-color terminal", []string{"TERM=xterm"}, true, false, colorprofile.ANSI},
		{"256-color terminal", []string{"TERM=xterm-256color"}, true, false, colorprofile.ANSI256},
		{"COLORTERM upgrades to truecolor", []string{"TERM=xterm-256color", "COLORTERM=truecolor"}, true, false, colorprofile.TrueColor},
		{"COLORTERM=yes is 256 colors", []string{"TERM=xterm", "COLORTERM=yes"}, true, false, colorprofile.ANSI256},
		{"tmux keeps truecolor", []string{"TERM=screen-256color", "COLORTERM=truecolor", "TERM_PROGRAM=tmux"}, true, false, colorprofile.TrueColor},
		{"screen is 256 colors", []string{"TERM=screen-256color", "COLORTERM=truecolor"}, true, false, colorprofile.ANSI256},
		{"TERM unset", nil, true, false, colorprofile.Ascii},
		{"CLICOLOR=0", []string{"TERM=xterm-256color", "CLICOLOR=0"}, true, false, colorprofile.Ascii},
		{"CLICOLOR=0 with CLICOLOR_FORCE", []string{"TERM=xterm-256color", "CLICOLOR=0", "CLICOLOR_FORCE=1"}, true, false, colorprofile.ANSI256},
		{"CLICOLOR_FORCE=0 doesn't force", []string{"TERM=xterm-256color", "CLICOLOR=0", "CLICOLOR_FORCE=0"}, true, false, colorprofile.Ascii},
		{"CI", []string{"TERM=xterm-256color", "CI=1"}, true, false, colorprofile.Ascii},
		{"CI with CLICOLOR_FORCE", []string{"TERM=xterm-256color", "CI=1", "CLICOLOR_FORCE=1"}, true, false, colorprofile.ANSI},
		{"CI unset", []string{"TERM=xterm-256color", "CI="}, true, false, colorprofile.ANSI256},
		{"MSYS2 pty pipe is raised to ANSI", []string{"TERM=xterm"}, false, true, colorprofile.ANSI},
		{"MSYS2 pty pipe ignores CI", []string{"TERM=xterm", "CI=1"}, false, true, colorprofile.ANSI},
		{"MSYS2 pty pipe honors CLICOLOR=0", []string{"TERM=xterm", "CLICOLOR=0"}, false, true, colorprofile.Ascii},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.tty && runtime.GOOS == "windows" {
				t.Skip("termenv ignores TERM and COLORTERM on Windows")
			}
			assert.Equal(t, c.want, colorDepthFor(termenvEnviron{c.env}, c.tty, c.msysPipe))
		})
	}
}

func TestTeaColorProfile(t *testing.T) {
	// Bubble Tea downsamples a program's whole view; template color helpers
	// emit basic ANSI whenever the stream supports color, so that is the floor.
	assert.GreaterOrEqual(t, teaColorProfile(true, &strings.Builder{}), colorprofile.ANSI)
	// Without color, bold and the other attributes stay.
	assert.Equal(t, colorprofile.Ascii, teaColorProfile(false, &strings.Builder{}))
}

func TestWriterDownsamplesToProfile(t *testing.T) {
	r := &Renderer{profile: colorprofile.ANSI}
	link := Hyperlink("https://h.test", "docs")
	styled := "\x1b[38;2;143;179;220mlabel\x1b[m " + link

	var b strings.Builder
	_, err := r.Writer(&b).Write([]byte(styled))
	assert.NoError(t, err)
	// Truecolor SGR becomes the nearest 16-color code; the hyperlink survives.
	assert.Equal(t, "\x1b[94mlabel\x1b[m "+link, b.String())
}
