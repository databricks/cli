package teainput_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio/teainput"
	"github.com/stretchr/testify/assert"
)

func TestPressRoundTrips(t *testing.T) {
	for _, name := range []string{
		"enter", "ctrl+c", "ctrl+j", "ctrl+h", "backspace", "alt+backspace",
		"esc", "tab", "shift+tab", "delete", "up", "down", "left", "right",
		"space", "alt+space", "q", "alt+f", "ctrl+n", "ctrl+p",
	} {
		ev, ok := teainput.Decode(teainput.Press(name))
		assert.True(t, ok)
		assert.Equal(t, name, ev.Key)
	}

	ev, _ := teainput.Decode(teainput.Type("abc"))
	assert.Equal(t, teainput.Event{Text: "abc"}, ev)
	ev, _ = teainput.Decode(teainput.Paste("x/y"))
	assert.Equal(t, teainput.Event{Text: "x/y", Paste: true}, ev)
}

func TestPressPanicsOnUnknownName(t *testing.T) {
	assert.PanicsWithValue(t, `teainput.Press: unknown key "pageup"`, func() { teainput.Press("pageup") })
}
