package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Terminals with the kitty keyboard protocol or modifyOtherKeys enabled send
// Shift+Enter as its own key; prompt and select must still submit on it.
func TestPrompt_ShiftEnterSubmits(t *testing.T) {
	for name, seq := range map[string]string{
		"kitty":           "\x1b[13;2u",
		"modifyOtherKeys": "\x1b[27;2;13~",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tm := termtest.NewPrompt(t, cmdio.PromptOptions{Label: "Name"})
			tm.WaitFor("Name")
			tm.Type("ab")
			tm.WaitFor("ab")
			tm.Type(seq)
			v, err := tm.Result()
			require.NoError(t, err, "raw output: %q", tm.Raw())
			assert.Equal(t, "ab", v)
		})
	}
}

func TestSelect_ShiftEnterSubmits(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSelectOrdered(t, []cmdio.Tuple{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
	}, "Pick")
	tm.WaitFor("alpha")
	tm.Type(termtest.KeyDown)
	tm.Type("\x1b[13;2u")
	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, "b", v)
}
