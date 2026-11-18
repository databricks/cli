package teainput

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func key(k tea.Key) tea.Msg { return tea.KeyPressMsg(k) }

func TestDecode(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
		want Event
	}{
		{"rune", key(tea.Key{Code: 'a', Text: "a"}), Event{Key: "a", Text: "a"}},
		{"shifted rune", key(tea.Key{Code: 'q', ShiftedCode: 'Q', Mod: tea.ModShift, Text: "Q"}), Event{Key: "Q", Text: "Q"}},
		{"alt rune", key(tea.Key{Code: 'f', Mod: tea.ModAlt}), Event{Key: "alt+f"}},
		{"space", key(tea.Key{Code: tea.KeySpace, Text: " "}), Event{Key: "space", Text: " "}},
		{"alt space", key(tea.Key{Code: tea.KeySpace, Mod: tea.ModAlt}), Event{Key: "alt+space"}},
		{"enter", key(tea.Key{Code: tea.KeyEnter}), Event{Key: "enter"}},
		{"shift+enter", key(tea.Key{Code: tea.KeyEnter, Mod: tea.ModShift}), Event{Key: "enter"}},
		{"alt enter", key(tea.Key{Code: tea.KeyEnter, Mod: tea.ModAlt}), Event{Key: "alt+enter"}},
		{"ctrl+enter", key(tea.Key{Code: tea.KeyEnter, Mod: tea.ModCtrl}), Event{Key: "ctrl+enter"}},
		{"ctrl+j", key(tea.Key{Code: 'j', Mod: tea.ModCtrl}), Event{Key: "ctrl+j"}},
		{"ctrl+c", key(tea.Key{Code: 'c', Mod: tea.ModCtrl}), Event{Key: "ctrl+c"}},
		{"backspace", key(tea.Key{Code: tea.KeyBackspace}), Event{Key: "backspace"}},
		{"alt backspace", key(tea.Key{Code: tea.KeyBackspace, Mod: tea.ModAlt}), Event{Key: "alt+backspace"}},
		{"esc", key(tea.Key{Code: tea.KeyEscape}), Event{Key: "esc"}},
		{"tab", key(tea.Key{Code: tea.KeyTab}), Event{Key: "tab"}},
		{"shift+tab", key(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), Event{Key: "shift+tab"}},
		{"delete", key(tea.Key{Code: tea.KeyDelete}), Event{Key: "delete"}},
		{"up", key(tea.Key{Code: tea.KeyUp}), Event{Key: "up"}},
		// Several runes in one read are text. They carry no key name, so a burst
		// that spells a key name is typed, not pressed.
		{"burst", key(tea.Key{Code: tea.KeyExtended, Text: "jk"}), Event{Text: "jk"}},
		{"burst enter", key(tea.Key{Code: tea.KeyExtended, Text: "enter"}), Event{Text: "enter"}},
		{"burst down", key(tea.Key{Code: tea.KeyExtended, Text: "down"}), Event{Text: "down"}},
		{"burst ctrl+c", key(tea.Key{Code: tea.KeyExtended, Text: "ctrl+c"}), Event{Text: "ctrl+c"}},
		{"paste", tea.PasteMsg{Content: "a b"}, Event{Text: "a b", Paste: true}},
		{"paste enter", tea.PasteMsg{Content: "enter"}, Event{Text: "enter", Paste: true}},
		{"empty paste", tea.PasteMsg{}, Event{Paste: true}},
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
	for _, name := range []string{
		"enter", "ctrl+c", "ctrl+j", "ctrl+h", "backspace", "alt+backspace",
		"esc", "tab", "shift+tab", "delete", "up", "down", "left", "right",
		"space", "alt+space", "q", "alt+f", "ctrl+n", "ctrl+p",
	} {
		ev, ok := Decode(Press(name))
		assert.True(t, ok)
		assert.Equal(t, name, ev.Key)
	}

	ev, _ := Decode(Type("abc"))
	assert.Equal(t, Event{Text: "abc"}, ev)
	ev, _ = Decode(Paste("x/y"))
	assert.Equal(t, Event{Text: "x/y", Paste: true}, ev)
}
