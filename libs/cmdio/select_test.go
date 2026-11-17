package cmdio

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio/teainput"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectModelNoColor(t *testing.T) {
	opts := SelectOptions{Label: "Pick", Items: []Tuple{{"a", "1"}}}

	m, err := newSelectModel(opts, true)
	require.NoError(t, err)
	assert.Contains(t, m.render(), ansiBlue)

	// NO_COLOR drops the blue "?" but keeps bold/underline on the cursor row.
	m, err = newSelectModel(opts, false)
	require.NoError(t, err)
	v := m.render()
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

	assert.True(t, hasSGR(runWithKeys(t, false, "Pick", "\r", run), ansiBlue))

	// NO_COLOR: the prompt still runs, without the blue "?", but the cursor
	// row keeps its bold and underline.
	out := runWithKeys(t, true, "Pick", "\r", run)
	assert.False(t, hasSGR(out, ansiBlue))
	assert.True(t, hasSGR(out, ansiBold))
	assert.True(t, hasSGR(out, ansiUnderline))
}

// A paste is text, not keys: outside search it neither moves the cursor nor
// opens search, and inside search it is appended literally, "/" included.
func TestSelectPaste(t *testing.T) {
	opts := SelectOptions{
		Label: "Pick",
		Items: []Tuple{{"alpha", "1"}, {"beta", "2"}, {"gamma", "3"}},
		Searcher: func(input string, idx int) bool {
			return strings.Contains(strings.ToLower(input), "a")
		},
	}
	m, err := newSelectModel(opts, false)
	require.NoError(t, err)

	next, _ := m.Update(teainput.Paste("j/"))
	m = next.(*selectModel)
	assert.Equal(t, 0, m.cursor, "a pasted j must not move the cursor")
	assert.False(t, m.searchActive, "a pasted / must not open search")
	assert.Empty(t, m.filter)

	next, _ = m.Update(teainput.Press("/"))
	m = next.(*selectModel)
	require.True(t, m.searchActive)

	next, _ = m.Update(teainput.Paste("b/x"))
	m = next.(*selectModel)
	assert.True(t, m.searchActive, "a pasted / inside search is literal")
	assert.Equal(t, "b/x", m.filter)

	next, _ = m.Update(teainput.Paste("y\nz"))
	m = next.(*selectModel)
	assert.Equal(t, "b/xy", m.filter, "a pasted line break ends the pasted text")
}
