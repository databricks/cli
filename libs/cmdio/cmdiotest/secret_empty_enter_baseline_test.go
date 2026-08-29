package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretBaseline_EmptyEnter pins Secret's behavior when the user
// presses Enter immediately without typing anything.
func TestSecretBaseline_EmptyEnter(t *testing.T) {
	t.Parallel()
	tm := termtest.NewSecret(t, "Personal access token")
	tm.WaitFor("Personal access token")
	tm.Golden("01-empty")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Empty(t, v)
}
