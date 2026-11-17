package cmdio

import (
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
