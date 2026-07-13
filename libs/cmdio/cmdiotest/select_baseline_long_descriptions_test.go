package cmdiotest_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_LongDescriptions pins Select's behavior when item Ids
// are long enough to potentially overflow the terminal width. The active row
// uses the Tuple template "{{.Name | bold}} ({{.Id|faint}})", so the long Id
// is only rendered on the active line; non-active rows show only the Name.
func TestSelectBaseline_LongDescriptions(t *testing.T) {
	t.Parallel()
	items := []cmdio.Tuple{
		{Name: "short", Id: "this-is-a-very-long-resource-identifier-that-exceeds-typical-width-1234567890"},
		{Name: "medium-length-name", Id: "another-extremely-long-id-string-with-lots-of-content-aaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{Name: "x", Id: "yet-another-long-identifier-with-quite-a-bit-of-text-bbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}

	tm := termtest.NewSelectOrdered(t, items, "Pick a resource")
	tm.WaitFor("Pick a resource")
	tm.WaitFor("short")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyDown)
	tm.Golden("02-second-active")

	tm.Type(termtest.KeyDown)
	tm.Golden("03-third-active")

	tm.Type(termtest.KeyEnter)

	id, err := tm.Result()
	require.NoError(t, err, "raw output: %q", tm.Raw())
	assert.Equal(t, items[2].Id, id, "snapshot:\n%s", tm.Snapshot())
}
