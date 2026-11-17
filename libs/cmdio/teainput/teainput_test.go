package teainput_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/cli/libs/cmdio/teainput"
	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
		want teainput.Event
	}{
		{"rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, teainput.Event{Key: "a", Text: "a"}},
		{"alt rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f"), Alt: true}, teainput.Event{Key: "alt+f"}},
		{"alt runes", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("fb"), Alt: true}, teainput.Event{}},
		{"space", tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}, teainput.Event{Key: "space", Text: " "}},
		{"space as rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}, teainput.Event{Key: "space", Text: " "}},
		{"alt space", tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" "), Alt: true}, teainput.Event{Key: "alt+space"}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, teainput.Event{Key: "enter"}},
		{"ctrl+j", tea.KeyMsg{Type: tea.KeyCtrlJ}, teainput.Event{Key: "ctrl+j"}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}, teainput.Event{Key: "ctrl+c"}},
		{"ctrl+h", tea.KeyMsg{Type: tea.KeyCtrlH}, teainput.Event{Key: "ctrl+h"}},
		{"backspace", tea.KeyMsg{Type: tea.KeyBackspace}, teainput.Event{Key: "backspace"}},
		{"alt backspace", tea.KeyMsg{Type: tea.KeyBackspace, Alt: true}, teainput.Event{Key: "alt+backspace"}},
		{"alt ctrl+h", tea.KeyMsg{Type: tea.KeyCtrlH, Alt: true}, teainput.Event{Key: "alt+ctrl+h"}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}, teainput.Event{Key: "esc"}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}, teainput.Event{Key: "tab"}},
		{"alt tab", tea.KeyMsg{Type: tea.KeyTab, Alt: true}, teainput.Event{Key: "alt+tab"}},
		{"shift+tab", tea.KeyMsg{Type: tea.KeyShiftTab}, teainput.Event{Key: "shift+tab"}},
		{"delete", tea.KeyMsg{Type: tea.KeyDelete}, teainput.Event{Key: "delete"}},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, teainput.Event{Key: "up"}},
		{"alt enter", tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, teainput.Event{Key: "alt+enter"}},
		{"alt ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC, Alt: true}, teainput.Event{Key: "alt+ctrl+c"}},
		{"alt left", tea.KeyMsg{Type: tea.KeyLeft, Alt: true}, teainput.Event{Key: "alt+left"}},
		// Several runes in one read are text. They carry no key name, so a burst
		// that spells a key name is typed, not pressed.
		{"burst", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jk")}, teainput.Event{Text: "jk"}},
		{"burst enter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")}, teainput.Event{Text: "enter"}},
		{"burst down", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")}, teainput.Event{Text: "down"}},
		{"burst ctrl+c", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+c")}, teainput.Event{Text: "ctrl+c"}},
		{"paste", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a b"), Paste: true}, teainput.Event{Text: "a b", Paste: true}},
		{"paste enter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter"), Paste: true}, teainput.Event{Text: "enter", Paste: true}},
		{"empty paste", tea.KeyMsg{Type: tea.KeyRunes, Paste: true}, teainput.Event{Paste: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := teainput.Decode(tc.msg)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}

	_, ok := teainput.Decode(tea.WindowSizeMsg{})
	assert.False(t, ok)
}
