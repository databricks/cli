package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterOverrideChangeDescLibraries(t *testing.T) {
	jobLib := []compute.Library{{Whl: "/Workspace/job.whl"}}

	tests := []struct {
		name   string
		path   string
		change ChangeDesc
		action deployplan.ActionType
		reason string
	}{
		{
			name:   "no section with remote libraries is unmanaged",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: nil, Remote: jobLib},
			action: deployplan.Skip,
			reason: deployplan.ReasonUnmanaged,
		},
		{
			name:   "removed section is unmanaged",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: jobLib, New: nil, Remote: jobLib},
			action: deployplan.Skip,
			reason: deployplan.ReasonUnmanaged,
		},
		{
			name:   "empty list stays managed",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: []compute.Library{}, Remote: jobLib},
			action: deployplan.Update,
		},
		{
			name:   "declared libraries stay managed",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: jobLib, Remote: nil},
			action: deployplan.Update,
		},
		{
			// A per-library entry only exists when the config declares a list, so a missing New
			// there is a removal to apply, not an unmanaged section.
			name:   "element removal stays managed",
			path:   "libraries[whl='/Workspace/job.whl']",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: nil, Remote: jobLib[0]},
			action: deployplan.Update,
		},
	}

	r := &ResourceCluster{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := tt.change
			require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath(tt.path), &change, nil))
			assert.Equal(t, tt.action, change.Action)
			assert.Equal(t, tt.reason, change.Reason)
		})
	}
}
