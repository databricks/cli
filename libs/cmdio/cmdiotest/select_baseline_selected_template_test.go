package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_SelectedTemplate pins the post-Enter rendering of
// [cmdio.RunSelect] when a non-empty Selected template is provided.
//
// cmdio.Select / cmdio.SelectOrdered set HideSelected:true, so the Selected
// branch is only reachable via RunSelect. Real callers that hit it:
// cmd/auth/profile_picker.go, libs/databrickscfg/profile/select.go,
// libs/databrickscfg/cfgpickers/clusters.go. Without this test, breaking the
// post-submit render or the Selected template behavior goes undetected.
func TestSelectBaseline_SelectedTemplate(t *testing.T) {
	type item struct {
		Name string
		Id   string
	}
	items := []item{
		{Name: "alpha", Id: "a"},
		{Name: "beta", Id: "b"},
		{Name: "gamma", Id: "c"},
	}

	tm := termtest.NewSelect(t, cmdio.SelectOptions{
		Label:    "Pick one",
		Items:    items,
		Active:   `> {{ .Name }} ({{ .Id }})`,
		Inactive: `  {{ .Name }} ({{ .Id }})`,
		Selected: `Chose: {{ .Name }} ({{ .Id }})`,
	})
	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyDown)
	tm.Golden("02-after-down")

	tm.Type(termtest.KeyEnter)

	idx, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, 1, idx, "snapshot:\n%s", tm.Snapshot())

	// Pin the rendered Selected template. This is the only test that asserts
	// the post-Enter frame; if promptui ever stops rendering Selected, or the
	// trailing newline / cursor handling changes, this golden catches it.
	tm.Golden("03-after-enter")
}
