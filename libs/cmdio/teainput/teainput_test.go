package teainput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
		want Event
	}{
		{"rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, Event{Key: "a", Text: "a"}},
		{"alt rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f"), Alt: true}, Event{Key: "alt+f"}},
		{"alt runes", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("fb"), Alt: true}, Event{}},
		{"space", tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}, Event{Key: "space", Text: " "}},
		{"space as rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}, Event{Key: "space", Text: " "}},
		{"alt space", tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" "), Alt: true}, Event{Key: "alt+space"}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, Event{Key: "enter"}},
		{"ctrl+j", tea.KeyMsg{Type: tea.KeyCtrlJ}, Event{Key: "ctrl+j"}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}, Event{Key: "ctrl+c"}},
		{"ctrl+h", tea.KeyMsg{Type: tea.KeyCtrlH}, Event{Key: "ctrl+h"}},
		{"backspace", tea.KeyMsg{Type: tea.KeyBackspace}, Event{Key: "backspace"}},
		{"alt backspace", tea.KeyMsg{Type: tea.KeyBackspace, Alt: true}, Event{Key: "alt+backspace"}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}, Event{Key: "esc"}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}, Event{Key: "tab"}},
		{"alt tab", tea.KeyMsg{Type: tea.KeyTab, Alt: true}, Event{Key: "alt+tab"}},
		{"shift+tab", tea.KeyMsg{Type: tea.KeyShiftTab}, Event{Key: "shift+tab"}},
		{"delete", tea.KeyMsg{Type: tea.KeyDelete}, Event{Key: "delete"}},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, Event{Key: "up"}},
		{"alt enter", tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, Event{Key: "alt+enter"}},
		{"alt ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC, Alt: true}, Event{Key: "alt+ctrl+c"}},
		{"alt left", tea.KeyMsg{Type: tea.KeyLeft, Alt: true}, Event{Key: "alt+left"}},
		// Several runes in one read are text. They carry no key name, so a burst
		// that spells a key name is typed, not pressed.
		{"burst", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jk")}, Event{Text: "jk"}},
		{"burst enter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")}, Event{Text: "enter"}},
		{"burst down", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")}, Event{Text: "down"}},
		{"burst ctrl+c", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+c")}, Event{Text: "ctrl+c"}},
		{"paste", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a b"), Paste: true}, Event{Text: "a b", Paste: true}},
		{"paste enter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter"), Paste: true}, Event{Text: "enter", Paste: true}},
		{"empty paste", tea.KeyMsg{Type: tea.KeyRunes, Paste: true}, Event{Paste: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Decode(tc.msg)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}

	_, ok := Decode(tea.WindowSizeMsg{})
	assert.False(t, ok)
}

func TestPressRoundTrips(t *testing.T) {
	for _, key := range []string{
		"enter", "alt+enter", "ctrl+c", "alt+ctrl+c", "ctrl+j", "ctrl+h", "backspace", "alt+backspace",
		"esc", "tab", "alt+tab", "shift+tab", "delete", "up", "down", "left", "right",
		"space", "alt+space", "q", "Q", "alt+f", "ctrl+n", "ctrl+p",
	} {
		ev, ok := Decode(Press(key))
		assert.True(t, ok)
		assert.Equal(t, key, ev.Key)
	}

	ev, _ := Decode(Type("abc"))
	assert.Equal(t, Event{Text: "abc"}, ev)
	ev, _ = Decode(Paste("x/y"))
	assert.Equal(t, Event{Text: "x/y", Paste: true}, ev)
}
