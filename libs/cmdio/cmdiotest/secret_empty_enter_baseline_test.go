package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
)

// TestSecretBaseline_EmptyEnter pins Secret's behavior when the user
// presses Enter immediately without typing anything.
func TestSecretBaseline_EmptyEnter(t *testing.T) {
	tm := termtest.NewSecret(t, "Personal access token")
	tm.WaitFor("Personal access token")
	tm.Golden("01-empty")

	tm.Type(termtest.KeyEnter)

	v, err := tm.Result()
	t.Logf("value: %q", v)
	t.Logf("error: %v", err)
}
