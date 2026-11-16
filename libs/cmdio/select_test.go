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
