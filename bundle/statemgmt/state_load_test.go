package statemgmt

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateToBundleMarksResourcesOnlyInStateDeleted(t *testing.T) {
	types := declaredResourceTypes()
	state := ExportedResourcesMap{}
	for _, rt := range types {
		state[stateKey(rt, "removed")] = ResourceState{ID: stateID(rt, "removed")}
	}

	root := config.Root{}
	require.NoError(t, StateToBundle(t.Context(), state, &root))

	loaded := resourcesByKey(&root)
	for _, rt := range types {
		t.Run(rt.Key, func(t *testing.T) {
			group := loaded[rt.Key]
			assert.ElementsMatch(t, []string{"removed"}, slices.Collect(maps.Keys(group)))
			assertResource(t, group, rt, "removed", stateID(rt, "removed"), resources.ModifiedStatusDeleted)
		})
	}
}

func TestStateToBundleMarksResourcesOnlyInConfigCreated(t *testing.T) {
	types := declaredResourceTypes()
	root := configWith(t, types, "added")

	require.NoError(t, StateToBundle(t.Context(), nil, root))

	loaded := resourcesByKey(root)
	for _, rt := range types {
		t.Run(rt.Key, func(t *testing.T) {
			group := loaded[rt.Key]
			assert.ElementsMatch(t, []string{"added"}, slices.Collect(maps.Keys(group)))
			assertResource(t, group, rt, "added", "", resources.ModifiedStatusCreated)
		})
	}
}

func TestStateToBundleMergesStateIntoConfig(t *testing.T) {
	types := declaredResourceTypes()
	root := configWith(t, types, "deployed", "added")
	state := ExportedResourcesMap{}
	for _, rt := range types {
		state[stateKey(rt, "deployed")] = ResourceState{ID: stateID(rt, "deployed")}
		state[stateKey(rt, "removed")] = ResourceState{ID: stateID(rt, "removed")}
	}

	require.NoError(t, StateToBundle(t.Context(), state, root))

	loaded := resourcesByKey(root)
	for _, rt := range types {
		t.Run(rt.Key, func(t *testing.T) {
			group := loaded[rt.Key]
			assert.ElementsMatch(t, []string{"deployed", "removed", "added"}, slices.Collect(maps.Keys(group)))
			assertResource(t, group, rt, "deployed", stateID(rt, "deployed"), "")
			assertResource(t, group, rt, "removed", stateID(rt, "removed"), resources.ModifiedStatusDeleted)
			assertResource(t, group, rt, "added", "", resources.ModifiedStatusCreated)
		})
	}
}

// The tests above start from zero values, so they cannot see whether configured fields
// survive. This one checks that they do, for a resource that is deployed and for one that is not.
func TestStateToBundleKeepsConfiguredFields(t *testing.T) {
	config := config.Root{
		Resources: config.Resources{
			Apps: map[string]*resources.App{
				"test_app": {
					App: apps.App{
						Name: "test_app",
					},
				},
				"test_app_new": {
					App: apps.App{
						Name: "test_app_new",
					},
				},
			},
			SecretScopes: map[string]*resources.SecretScope{
				"test_secret_scope": {
					Name: "test_secret_scope",
				},
				"test_secret_scope_new": {
					Name: "test_secret_scope_new",
				},
			},
		},
	}
	state := ExportedResourcesMap{
		"resources.apps.test_app":                       {ID: "test_app"},
		"resources.apps.test_app_old":                   {ID: "test_app_old"},
		"resources.secret_scopes.test_secret_scope":     {ID: "test_secret_scope"},
		"resources.secret_scopes.test_secret_scope_old": {ID: "test_secret_scope_old"},
	}
	err := StateToBundle(t.Context(), state, &config)
	assert.NoError(t, err)

	assert.Equal(t, "test_app", config.Resources.Apps["test_app"].Name)
	assert.Empty(t, config.Resources.Apps["test_app"].ModifiedStatus)
	assert.Equal(t, "test_app_old", config.Resources.Apps["test_app_old"].ID)
	assert.Empty(t, config.Resources.Apps["test_app_old"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.Apps["test_app_old"].ModifiedStatus)
	assert.Equal(t, "test_app_new", config.Resources.Apps["test_app_new"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.Apps["test_app_new"].ModifiedStatus)

	assert.Equal(t, "test_secret_scope", config.Resources.SecretScopes["test_secret_scope"].Name)
	assert.Empty(t, config.Resources.SecretScopes["test_secret_scope"].ModifiedStatus)
	assert.Equal(t, "test_secret_scope_old", config.Resources.SecretScopes["test_secret_scope_old"].ID)
	assert.Empty(t, config.Resources.SecretScopes["test_secret_scope_old"].Name)
	assert.Equal(t, resources.ModifiedStatusDeleted, config.Resources.SecretScopes["test_secret_scope_old"].ModifiedStatus)
	assert.Equal(t, "test_secret_scope_new", config.Resources.SecretScopes["test_secret_scope_new"].Name)
	assert.Equal(t, resources.ModifiedStatusCreated, config.Resources.SecretScopes["test_secret_scope_new"].ModifiedStatus)
}

// declaredResourceTypes returns the resource types that a bundle declares under resources:.
// Internal types are left out: the deploy pipeline creates them and they have no ID or
// modified status to load.
func declaredResourceTypes() []config.ResourceType {
	internal := map[string]bool{}
	for field := range reflect.TypeFor[config.Resources]().Fields() {
		if field.Tag.Get("bundle") == "internal" {
			internal[structtag.JSONTag(field.Tag.Get("json")).Name()] = true
		}
	}
	return slices.DeleteFunc(config.AllResourceTypes(), func(rt config.ResourceType) bool {
		return internal[rt.Key]
	})
}

func stateKey(rt config.ResourceType, name string) string {
	return "resources." + rt.Key + "." + name
}

// stateID is unique per type and name, so that an ID loaded into the wrong resource is caught.
func stateID(rt config.ResourceType, name string) string {
	return rt.Key + "/" + name
}

// configWith returns a configuration with a zero-value resource of every type in types under
// each of names.
func configWith(t *testing.T, types []config.ResourceType, names ...string) *config.Root {
	t.Helper()
	groups := dyn.NewMapping()
	for _, rt := range types {
		group := dyn.NewMapping()
		for _, name := range names {
			group.SetLoc(name, nil, dyn.V(dyn.NewMapping()))
		}
		groups.SetLoc(rt.Key, nil, dyn.V(group))
	}

	root := &config.Root{}
	require.NoError(t, convert.ToTyped(root, dyn.V(map[string]dyn.Value{"resources": dyn.V(groups)})))
	return root
}

func resourcesByKey(root *config.Root) map[string]map[string]config.ConfigResource {
	byKey := map[string]map[string]config.ConfigResource{}
	for _, group := range root.Resources.AllResources() {
		byKey[group.Description.PluralName] = group.Resources
	}
	return byKey
}

// assertResource checks that the resource under name is a zero value of type rt apart from its
// ID and modified status, which StateToBundle sets. An empty id or status means the field is unset.
func assertResource(t *testing.T, group map[string]config.ConfigResource, rt config.ResourceType, name, id string, status resources.ModifiedStatus) {
	t.Helper()
	res, ok := group[name]
	if !assert.True(t, ok, "resources.%s.%s is missing", rt.Key, name) {
		return
	}

	want, err := convert.FromTyped(rt.New(), dyn.NilValue)
	require.NoError(t, err)
	if id != "" {
		want, err = dyn.Set(want, "id", dyn.V(id))
		require.NoError(t, err)
	}
	if status != "" {
		want, err = dyn.Set(want, "modified_status", dyn.V(status))
		require.NoError(t, err)
	}

	got, err := convert.FromTyped(res, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, want.AsAny(), got.AsAny(), "resources.%s.%s", rt.Key, name)
}

func TestApplyStateLoadsDeployedState(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{
			Dashboards: map[string]*resources.Dashboard{
				"dash": {},
			},
			GenieSpaces: map[string]*resources.GenieSpace{
				"space": {},
			},
			JobRuns: map[string]*resources.JobRun{
				"run":          {},
				"run_no_state": {ResolvedJobID: 5},
			},
		},
	}}

	state := ExportedResourcesMap{
		"resources.dashboards.dash":             {ID: "d1", ETag: "v1"},
		"resources.dashboards.dash.permissions": {ID: "d1", ETag: "v2"},
		"resources.dashboards.deleted":          {ID: "d2", ETag: "v3"},
		"resources.genie_spaces.space":          {ID: "g1", ETag: "v4"},
		"resources.job_runs.run":                {ID: "r1", JobID: 123},
		"resources.job_runs.run_no_state":       {ID: "r2"},
	}

	diags := applyState(t.Context(), b, state, nil)
	require.NoError(t, diags.Error())

	assert.Equal(t, "v1", b.Config.Resources.Dashboards["dash"].Etag)
	assert.Equal(t, "v3", b.Config.Resources.Dashboards["deleted"].Etag)
	assert.Empty(t, b.Config.Resources.GenieSpaces["space"].Etag)
	assert.Equal(t, int64(123), b.Config.Resources.JobRuns["run"].ResolvedJobID)
	assert.Equal(t, int64(5), b.Config.Resources.JobRuns["run_no_state"].ResolvedJobID)
}
