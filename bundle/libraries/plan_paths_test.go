package libraries

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
)

func TestPlanUploadReferences(t *testing.T) {
	const up = "/Workspace/Users/u/.bundle/b/default/artifacts/.internal"
	owners := uploadOwners{
		libraries: map[string]map[string][]string{
			"resources.jobs.updated": {
				"tasks.libraries.whl":                    {"a-1.0-py3-none-any.whl", "b-*.whl"},
				"tasks.libraries.jar":                    {"*.jar"},
				"tasks.ai_runtime_task.code_source_path": {"code.tar.gz"},
			},
			"resources.pipelines.updated":   {"environment.dependencies": {"p-1.0-py3-none-any.whl"}},
			"resources.pipelines.unchanged": {"environment.dependencies": {"e-*.whl"}},
			"resources.clusters.policy":     {"libraries.jar": {"*.jar"}},
		},
		artifacts: []string{"g-*.whl"},
	}
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.updated": {
			Action: deployplan.Update,
			NewState: &structvar.StructVarJSON{Value: []byte(`{"tasks":[{"libraries":[
				{"whl":"` + up + `/a-1.0+1577836800000000000-py3-none-any.whl"},
				{"whl":"` + up + `/b-1.0-py3-none-any.whl[test]"},
				{"whl":"` + up + `/nested/c-1.0-py3-none-any.whl"},
				{"whl":"/Workspace/elsewhere/d-1.0-py3-none-any.whl"},
				{"jar":"` + up + `/app-1.0.jar"}],
				"ai_runtime_task":{"code_source_path":"` + up + `/code.tar.gz"},
				"notebook_task":{"base_parameters":{"jar":"` + up + `/result.jar"}},
				"parameters":["--wheel=` + up + `/g-1.0-py3-none-any.whl", "%pip install ` + up + `/h-1.0-py3-none-any.whl; python_version>'3'", "` + up + `/result.json"]}]}`)},
			RemoteState: map[string]any{"whl": up + "/old-0.9-py3-none-any.whl"},
		},
		"resources.jobs.explicit": {
			Action:   deployplan.Update,
			NewState: &structvar.StructVarJSON{Value: []byte(`{"tasks":[{"libraries":[{"jar":"` + up + `/shared.jar"}]}]}`)},
		},
		"resources.pipelines.updated": {
			Action:   deployplan.Create,
			NewState: &structvar.StructVarJSON{Value: []byte(`{"environment":{"dependencies":["` + up + `/p-1.0-py3-none-any.whl", "` + up + `/pkg"]}}`)},
		},
		"resources.pipelines.unchanged": {
			Action:      deployplan.Skip,
			RemoteState: map[string]any{"environment": map[string]any{"dependencies": []any{up + "/e-1.0-py3-none-any.whl"}}},
		},
		"resources.clusters.unmanaged": {
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
		// With a cluster policy, a library a job run installed is a remote addition, even
		// though the cluster declares a library of its own.
		"resources.clusters.policy": {
			Action: deployplan.Skip,
			RemoteState: map[string]any{"libraries": []any{
				map[string]any{"jar": up + "/own.jar"},
				map[string]any{"jar": up + "/old.jar"},
			}},
			Changes: deployplan.Changes{
				"libraries[jar='" + up + "/old.jar']": {
					Action: deployplan.Skip,
					Reason: deployplan.ReasonRemoteAddition,
					Remote: map[string]any{"jar": up + "/old.jar"},
				},
				// A skipped change whose configured value also names the file: the
				// configuration declares own.jar, so it stays required.
				"libraries[jar='" + up + "/own.jar']": {
					Action: deployplan.Skip,
					Reason: deployplan.ReasonRemoteAlreadySet,
					New:    map[string]any{"jar": up + "/own.jar"},
					Remote: map[string]any{"jar": up + "/own.jar"},
				},
			},
		},
		"resources.jobs.deleted": {
			Action:      deployplan.Delete,
			RemoteState: map[string]any{"whl": up + "/f-1.0-py3-none-any.whl"},
		},
	}}

	required, optional := planUploadReferences(plan, up, false, owners)
	assert.Equal(t, []string{
		"a-1.0+1577836800000000000-py3-none-any.whl",
		"app-1.0.jar",
		"b-1.0-py3-none-any.whl",
		"code.tar.gz",
		"g-1.0-py3-none-any.whl",
		"p-1.0-py3-none-any.whl",
	}, required)
	assert.Equal(t, []string{
		"e-1.0-py3-none-any.whl",
		"h-1.0-py3-none-any.whl",
		"own.jar",
		"pkg",
		"result.jar",
		"result.json",
		"shared.jar",
	}, optional)

	// With artifact cleanup, unchanged resources need their own files as well.
	required, optional = planUploadReferences(plan, up, true, owners)
	assert.Contains(t, required, "e-1.0-py3-none-any.whl")
	assert.Contains(t, required, "own.jar")
	// A library only the remote has is not the bundle's, so it is neither required nor
	// uploaded, as in plain deploy.
	assert.NotContains(t, required, "old.jar")
	assert.NotContains(t, optional, "old.jar")
}

func TestMatchesUploadName(t *testing.T) {
	assert.True(t, matchesUploadName([]string{"pkg-1.0-py3-none-any.whl"}, "pkg-1.0+1577836800000000000-py3-none-any.whl"))
	assert.True(t, matchesUploadName([]string{"*.jar"}, "app-1.0.jar"))
	assert.False(t, matchesUploadName([]string{"*.jar"}, "app-1.0.whl"))
	assert.False(t, matchesUploadName(nil, "app-1.0.jar"))
}

func TestHasPatchedName(t *testing.T) {
	names := []string{"pkg-1.2.3.dev4+1577836800000000000-py3-none-any.whl"}
	assert.True(t, hasPatchedName("pkg-1.2.3.dev4-py3-none-any.whl", names))
	assert.True(t, hasPatchedName("pkg-1.2.3.dev4+gabc-py3-none-any.whl", names))
	assert.False(t, hasPatchedName("pkg-1.2.3-py3-none-any.whl", names))
	assert.False(t, hasPatchedName("other-1.2.3.dev4-py3-none-any.whl", names))
}
