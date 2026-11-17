package teainput

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// namedKeys maps the non-printable keystroke names used by [Event.Key] to key
// codes.
var namedKeys = map[string]rune{
	"enter":     tea.KeyEnter,
	"tab":       tea.KeyTab,
	"backspace": tea.KeyBackspace,
	"esc":       tea.KeyEscape,
	"space":     tea.KeySpace,
	"delete":    tea.KeyDelete,
	"insert":    tea.KeyInsert,
	"up":        tea.KeyUp,
	"down":      tea.KeyDown,
	"left":      tea.KeyLeft,
	"right":     tea.KeyRight,
	"home":      tea.KeyHome,
	"end":       tea.KeyEnd,
	"pgup":      tea.KeyPgUp,
	"pgdown":    tea.KeyPgDown,
}

var modifierPrefixes = []struct {
	prefix string
	mod    tea.KeyMod
}{
	{"ctrl+", tea.ModCtrl},
	{"alt+", tea.ModAlt},
	{"shift+", tea.ModShift},
}

// Press builds the message for a key press, named like [Event.Key]
// (e.g. "enter", "ctrl+c", "alt+backspace", "space", "q"). It exists so model
// tests don't construct version-specific key messages. It panics on a name
// that is neither a known key nor a single rune, so a misspelled key fails
// the test instead of being typed as text.
func Press(key string) tea.Msg {
	var mod tea.KeyMod
	for _, p := range modifierPrefixes {
		if rest, ok := strings.CutPrefix(key, p.prefix); ok && rest != "" {
			mod |= p.mod
			key = rest
		}
	}

	if code, ok := namedKeys[key]; ok {
		k := tea.Key{Code: code, Mod: mod}
		if code == tea.KeySpace && mod == 0 {
			k.Text = " "
		}
		return tea.KeyPressMsg(k)
	}

	if utf8.RuneCountInString(key) != 1 {
		panic(fmt.Sprintf("teainput.Press: unknown key %q", key))
	}
	r, _ := utf8.DecodeRuneInString(key)
	k := tea.Key{Code: r, Mod: mod}
	if mod&^tea.ModShift == 0 {
		k.Text = key
	}
	return tea.KeyPressMsg(k)
}

// Type builds the message for typing text as a single event, like a burst of
// fast typing.
func Type(text string) tea.Msg {
	r, _ := utf8.DecodeRuneInString(text)
	if utf8.RuneCountInString(text) > 1 {
		r = tea.KeyExtended
	}
	return tea.KeyPressMsg(tea.Key{Code: r, Text: text})
}

// Paste builds the message for a bracketed paste of text.
func Paste(text string) tea.Msg {
	return tea.PasteMsg{Content: text}
}
