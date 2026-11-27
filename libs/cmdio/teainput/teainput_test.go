package teainput_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/databricks/cli/libs/cmdio/teainput"
	"github.com/stretchr/testify/assert"
)

func key(k tea.Key) tea.Msg { return tea.KeyPressMsg(k) }

func TestDecode(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
		want teainput.Event
	}{
		{"rune", key(tea.Key{Code: 'a', Text: "a"}), teainput.Event{Key: "a", Text: "a"}},
		{"shifted rune", key(tea.Key{Code: 'q', ShiftedCode: 'Q', Mod: tea.ModShift, Text: "Q"}), teainput.Event{Key: "Q", Text: "Q"}},
		{"alt rune", key(tea.Key{Code: 'f', Mod: tea.ModAlt}), teainput.Event{Key: "alt+f"}},
		{"space", key(tea.Key{Code: tea.KeySpace, Text: " "}), teainput.Event{Key: "space", Text: " "}},
		{"alt space", key(tea.Key{Code: tea.KeySpace, Mod: tea.ModAlt}), teainput.Event{Key: "alt+space"}},
		{"enter", key(tea.Key{Code: tea.KeyEnter}), teainput.Event{Key: "enter"}},
		{"shift+enter", key(tea.Key{Code: tea.KeyEnter, Mod: tea.ModShift}), teainput.Event{Key: "enter"}},
		{"alt enter", key(tea.Key{Code: tea.KeyEnter, Mod: tea.ModAlt}), teainput.Event{Key: "alt+enter"}},
		{"ctrl+enter", key(tea.Key{Code: tea.KeyEnter, Mod: tea.ModCtrl}), teainput.Event{Key: "ctrl+enter"}},
		{"ctrl+j", key(tea.Key{Code: 'j', Mod: tea.ModCtrl}), teainput.Event{Key: "ctrl+j"}},
		{"ctrl+c", key(tea.Key{Code: 'c', Mod: tea.ModCtrl}), teainput.Event{Key: "ctrl+c"}},
		{"backspace", key(tea.Key{Code: tea.KeyBackspace}), teainput.Event{Key: "backspace"}},
		{"alt backspace", key(tea.Key{Code: tea.KeyBackspace, Mod: tea.ModAlt}), teainput.Event{Key: "alt+backspace"}},
		{"esc", key(tea.Key{Code: tea.KeyEscape}), teainput.Event{Key: "esc"}},
		{"tab", key(tea.Key{Code: tea.KeyTab}), teainput.Event{Key: "tab"}},
		{"shift+tab", key(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), teainput.Event{Key: "shift+tab"}},
		{"delete", key(tea.Key{Code: tea.KeyDelete}), teainput.Event{Key: "delete"}},
		{"up", key(tea.Key{Code: tea.KeyUp}), teainput.Event{Key: "up"}},
		// Several runes in one read are text. They carry no key name, so a burst
		// that spells a key name is typed, not pressed.
		{"burst", key(tea.Key{Code: tea.KeyExtended, Text: "jk"}), teainput.Event{Text: "jk"}},
		{"burst enter", key(tea.Key{Code: tea.KeyExtended, Text: "enter"}), teainput.Event{Text: "enter"}},
		{"burst down", key(tea.Key{Code: tea.KeyExtended, Text: "down"}), teainput.Event{Text: "down"}},
		{"burst ctrl+c", key(tea.Key{Code: tea.KeyExtended, Text: "ctrl+c"}), teainput.Event{Text: "ctrl+c"}},
		{"paste", tea.PasteMsg{Content: "a b"}, teainput.Event{Text: "a b", Paste: true}},
		{"paste enter", tea.PasteMsg{Content: "enter"}, teainput.Event{Text: "enter", Paste: true}},
		{"empty paste", tea.PasteMsg{}, teainput.Event{Paste: true}},
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
