package cmdio

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectModelNoColor(t *testing.T) {
	opts := SelectOptions{Label: "Pick", Items: []Tuple{{"a", "1"}}}

	m, err := newSelectModel(opts, true)
	require.NoError(t, err)
	assert.Contains(t, m.View(), ansiBlue)

	// NO_COLOR drops the blue "?" but keeps bold/underline on the cursor row.
	m, err = newSelectModel(opts, false)
	require.NoError(t, err)
	v := m.View()
	assert.NotContains(t, v, ansiBlue)
	assert.Contains(t, v, ansiBold)
	assert.Contains(t, v, ansiUnderline)
}

func TestRunSelectNoColor(t *testing.T) {
	opts := SelectOptions{Label: "Pick", Items: []Tuple{{"a", "1"}, {"b", "2"}}}
	run := func(c *cmdIO) error {
		_, err := RunSelect(InContext(t.Context(), c), opts)
		return err
	}

	assert.Contains(t, runWithKeys(t, false, "Pick", "\r", run), ansiBlue)

	// NO_COLOR: the prompt still runs, without the blue "?", but the cursor
	// row keeps its bold and underline.
	out := runWithKeys(t, true, "Pick", "\r", run)
	assert.NotContains(t, out, ansiBlue)
	assert.Contains(t, out, ansiBold)
	assert.Contains(t, out, ansiUnderline)
}

func TestSelectModelEscapesControlCharacters(t *testing.T) {
	const raw = "a\x1b]0;t\x07\x1b[2J\rb"
	const escaped = `a\u001b]0;t\u0007\u001b[2J\rb`
	opts := SelectOptions{
		Label: "Pick " + raw,
		// An item the caller styled itself keeps its colors.
		Items: []Tuple{{ansiGreen + "●" + ansiReset + " first " + raw, "1"}, {"second " + raw, "2"}},
		// Templates built from a value, as bundle init does with a description.
		Selected: "Chosen " + raw + ": {{.Name}}",
	}
	m, err := newSelectModel(opts, false)
	require.NoError(t, err)

	for _, submitted := range []bool{false, true} {
		m.submitted = submitted
		v := m.View()
		assert.NotContains(t, v, "\x1b]")
		assert.NotContains(t, v, "\x1b[2J")
		assert.NotContains(t, v, "\x07")
		assert.NotContains(t, v, "\r")
		assert.Contains(t, v, ansiGreen+"●"+ansiReset+" first "+escaped)
		assert.NotContains(t, v, "template error")
		if submitted {
			assert.Contains(t, v, "Chosen "+escaped)
		} else {
			assert.Contains(t, v, "Pick "+escaped)
			assert.Contains(t, v, "second "+escaped)
		}
	}
}

func TestSelectModelEscapesControlCharactersInTemplateErrors(t *testing.T) {
	// The error quotes the action that failed as it is written in the template,
	// here with a color code and a clear-screen sequence in a string.
	const failing = "{{index \"a\x1b[31m\x1b[2J\" 99}}"
	const escaped = `<index "a\u001b[31m\u001b[2J" 99>`
	opts := SelectOptions{
		Label:         "Pick",
		Items:         []Tuple{{"first", "1"}, {"second", "2"}},
		LabelTemplate: failing,
		Active:        failing,
		Inactive:      failing,
		Selected:      failing,
	}
	m, err := newSelectModel(opts, true)
	require.NoError(t, err)

	v := m.View()
	assert.NotContains(t, v, "\x1b[31m")
	assert.NotContains(t, v, "\x1b[2J")
	assert.Contains(t, v, "[label template error: ")
	// The label, the active item and the inactive item.
	assert.Equal(t, 3, strings.Count(v, escaped))

	m.submitted = true
	v = m.View()
	assert.NotContains(t, v, "\x1b[31m")
	assert.Contains(t, v, "[selected template error: ")
	assert.Equal(t, 1, strings.Count(v, escaped))
}

func TestPromptFunctionsStyleEachLine(t *testing.T) {
	// A style left on at the end of a line is reset there when the prompt is
	// written, so each line of a multi-line value gets the style by itself.
	bold := promptFuncMap(false)["bold"].(func(any) string)
	assert.Equal(t, ansiBold+"one"+ansiReset, bold("one"))
	assert.Equal(t, ansiBold+"one"+ansiReset+"\n"+ansiBold+"two"+ansiReset, bold("one\ntwo"))

	opts := SelectOptions{Label: "Pick", Items: []Tuple{{"one\ntwo", "1"}}}
	m, err := newSelectModel(opts, false)
	require.NoError(t, err)
	// The active item is underlined; it is shown as the tuple it is.
	assert.Contains(t, m.View(), ansiUnderline+"{one"+ansiReset+"\n"+ansiUnderline+"two 1}"+ansiReset)
}
