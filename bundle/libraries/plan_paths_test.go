package libraries

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
)

func TestPlanUploadReferences(t *testing.T) {
	const up = "/Workspace/Users/u/.bundle/b/default/artifacts/.internal"
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.updated": {
			Action: deployplan.Update,
			NewState: &structvar.StructVarJSON{Value: []byte(`{"tasks":[{"libraries":[
				{"whl":"` + up + `/a-1.0-py3-none-any.whl"},
				{"whl":"` + up + `/b-1.0-py3-none-any.whl[test]"},
				{"whl":"` + up + `/nested/c-1.0-py3-none-any.whl"},
				{"whl":"/Workspace/elsewhere/d-1.0-py3-none-any.whl"}],
				"parameters":["--wheel=` + up + `/g-1.0-py3-none-any.whl", "%pip install ` + up + `/h-1.0-py3-none-any.whl; python_version>'3'"]}]}`)},
			RemoteState: map[string]any{"whl": up + "/old-0.9-py3-none-any.whl"},
		},
		"resources.pipelines.unchanged": {
			Action:      deployplan.Skip,
			RemoteState: map[string]any{"environment": map[string]any{"dependencies": []any{up + "/e-1.0-py3-none-any.whl"}}},
		},
		"resources.clusters.unchanged": {
			Action: deployplan.Skip,
			RemoteState: map[string]any{"libraries": []any{
				map[string]any{"whl": up + "/installed-by-run-1.0-py3-none-any.whl"},
			}},
			Changes: deployplan.Changes{"libraries": {
				Action: deployplan.Skip,
				Reason: deployplan.ReasonUnmanaged,
				Remote: []any{map[string]any{"whl": up + "/installed-by-run-1.0-py3-none-any.whl"}},
			}},
		},
		"resources.jobs.deleted": {
			Action:      deployplan.Delete,
			RemoteState: map[string]any{"whl": up + "/f-1.0-py3-none-any.whl"},
		},
	}}

	planned, unchanged := planUploadReferences(plan, up)
	assert.Equal(t, []string{
		"a-1.0-py3-none-any.whl",
		"b-1.0-py3-none-any.whl",
		"g-1.0-py3-none-any.whl",
		"h-1.0-py3-none-any.whl",
	}, planned)
	assert.Equal(t, []string{"e-1.0-py3-none-any.whl"}, unchanged)
}

func TestHasPatchedName(t *testing.T) {
	names := []string{"pkg-1.2.3.dev4+1577836800000000000-py3-none-any.whl"}
	assert.True(t, hasPatchedName("pkg-1.2.3.dev4-py3-none-any.whl", names))
	assert.True(t, hasPatchedName("pkg-1.2.3.dev4+gabc-py3-none-any.whl", names))
	assert.False(t, hasPatchedName("pkg-1.2.3-py3-none-any.whl", names))
	assert.False(t, hasPatchedName("other-1.2.3.dev4-py3-none-any.whl", names))
}
