package cmdiotest_test

import (
	"fmt"
	"maps"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// The pin tests record what the interactive components do today, including
// behavior that is accidental, so that a change of the TUI libraries shows
// up as golden diffs. They pin; they don't judge.

// spinnerFrames are the braille frames cmdio's spinners cycle through.
const spinnerFrames = "⣾⣽⣻⢿⡿⣟⣯⣷"

// blankSpinnerFrames replaces every spinner frame with "*", so a golden of a
// running spinner doesn't depend on which frame was on screen.
func blankSpinnerFrames(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(spinnerFrames, r) {
			return '*'
		}
		return r
	}, s)
}

// countSpinnerFrames returns how many distinct spinner frames appear in s.
func countSpinnerFrames(s string) int {
	seen := map[rune]bool{}
	for _, r := range s {
		if strings.ContainsRune(spinnerFrames, r) {
			seen[r] = true
		}
	}
	return len(seen)
}

// colorEnv is one environment of the color matrix, set in the command's
// context (libs/env), where cmdio reads NO_COLOR and TERM. The map also sets
// CI, CLICOLOR, CLICOLOR_FORCE, COLORTERM and TERM_PROGRAM there (an empty
// value reads as unset). lipgloss, termenv and bubbletea read the process
// environment instead, so these rows don't change what they see.
// TestColorPin_Swatch explains why the rows are kept.
type colorEnv struct {
	name string
	env  map[string]string
}

func colorEnvs() []colorEnv {
	base := map[string]string{
		"TERM":           "xterm-256color",
		"COLORTERM":      "",
		"NO_COLOR":       "",
		"CLICOLOR":       "",
		"CLICOLOR_FORCE": "",
		"CI":             "",
		"TERM_PROGRAM":   "",
	}
	with := func(overrides map[string]string) map[string]string {
		m := maps.Clone(base)
		maps.Copy(m, overrides)
		return m
	}
	return []colorEnv{
		{"term-unset", with(map[string]string{"TERM": ""})},
		{"term-xterm-256color", with(nil)},
		{"term-screen", with(map[string]string{"TERM": "screen"})},
		{"term-vt100", with(map[string]string{"TERM": "vt100"})},
		{"term-dumb", with(map[string]string{"TERM": "dumb"})},
		{"ci", with(map[string]string{"CI": "1"})},
		{"clicolor-0", with(map[string]string{"CLICOLOR": "0"})},
		{"no-color", with(map[string]string{"NO_COLOR": "1"})},
		{"tmux-truecolor", with(map[string]string{"TERM": "tmux-256color", "COLORTERM": "truecolor"})},
		{"colorterm-yes", with(map[string]string{"COLORTERM": "yes"})},
	}
}

// colorEnvNamed returns the environment of the color matrix called name.
func colorEnvNamed(name string) map[string]string {
	for _, ce := range colorEnvs() {
		if ce.name == name {
			return ce.env
		}
	}
	panic("no color env " + name)
}

// pinItems is the item list the select pins use.
func pinItems() []cmdio.Tuple {
	return []cmdio.Tuple{
		{Name: "alpha", Id: "id-01"},
		{Name: "beta", Id: "id-02"},
		{Name: "gamma", Id: "id-03"},
	}
}

// quoted renders bytes as one escaped Go string, so goldens of raw
// input or output stay readable.
func quoted(b []byte) string {
	if len(b) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("%q", b)
}

// typeKeys writes each key as its own read: it waits for the program to
// settle after every write, so two keys never reach it as one burst. Use
// [termtest.Term.TypeBurst] to test a burst on purpose.
func typeKeys[T any](tm *termtest.Term[T], keys ...string) {
	for _, k := range keys {
		tm.Type(k)
		tm.Settle()
	}
}
