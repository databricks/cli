package direct

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInitForApplyRehydratesRemoteState(t *testing.T) {
	var b DeploymentBundle
	require.NoError(t, b.StateDB.Open(filepath.Join(t.TempDir(), "state.json")))

	newStateJSON, err := structvar.NewStructVar(&dresources.GrantsState{
		SecurableType: "catalog",
		FullName:      "main.foo",
		EmbeddedSlice: []catalog.PrivilegeAssignment{
			{
				Principal:  "alice",
				Privileges: []catalog.Privilege{catalog.PrivilegeUseCatalog},
			},
		},
	}, nil).ToJSON()
	require.NoError(t, err)

	wantRemoteState := &dresources.GrantsState{
		SecurableType: "catalog",
		FullName:      "main.foo",
		EmbeddedSlice: []catalog.PrivilegeAssignment{
			{
				Principal:  "alice",
				Privileges: []catalog.Privilege{catalog.PrivilegeUseCatalog},
			},
			{
				Principal:  "bob",
				Privileges: []catalog.Privilege{catalog.PrivilegeSelect},
			},
		},
	}

	plan := deployplan.NewPlanDirect()
	plan.Plan["resources.catalogs.foo.grants"] = &deployplan.PlanEntry{
		Action:      deployplan.Update,
		ID:          "catalog/main.foo",
		NewState:    newStateJSON,
		RemoteState: wantRemoteState,
	}

	loadedPlan := roundTripPlan(t, plan)
	require.IsType(t, map[string]any{}, loadedPlan.Plan["resources.catalogs.foo.grants"].RemoteState)

	m := mocks.NewMockWorkspaceClient(t)
	err = b.InitForApply(t.Context(), m.WorkspaceClient, loadedPlan)
	require.NoError(t, err)

	sv, ok := b.StateCache.Load("resources.catalogs.foo.grants")
	require.True(t, ok)
	require.IsType(t, &dresources.GrantsState{}, sv.Value)

	gotRemoteState, ok := loadedPlan.Plan["resources.catalogs.foo.grants"].RemoteState.(*dresources.GrantsState)
	require.True(t, ok)
	require.Equal(t, wantRemoteState.SecurableType, gotRemoteState.SecurableType)
	require.Equal(t, wantRemoteState.FullName, gotRemoteState.FullName)
	require.Len(t, gotRemoteState.EmbeddedSlice, len(wantRemoteState.EmbeddedSlice))
	for i, want := range wantRemoteState.EmbeddedSlice {
		require.Equal(t, want.Principal, gotRemoteState.EmbeddedSlice[i].Principal)
		require.Equal(t, want.Privileges, gotRemoteState.EmbeddedSlice[i].Privileges)
	}
}

func TestTryImportGrantsErrorsOnUnexpectedReadFailure(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockGrantsAPI().EXPECT().Get(mock.Anything, catalog.GetGrantRequest{
		FullName:      "main.foo",
		MaxResults:    0,
		PageToken:     "",
		Principal:     "",
		SecurableType: "catalog",
	}).Return(nil, &apierr.APIError{
		StatusCode: 429,
		Message:    "rate limited",
	}).Once()

	adapter, err := dresources.NewAdapter((*dresources.ResourceGrants)(nil), "catalogs.grants", m.WorkspaceClient)
	require.NoError(t, err)

	id, remoteState, imported, err := tryImportGrants(t.Context(), "resources.catalogs.foo.grants", adapter, &dresources.GrantsState{
		SecurableType: "catalog",
		FullName:      "main.foo",
	})
	require.Error(t, err)
	require.Empty(t, id)
	require.Nil(t, remoteState)
	require.False(t, imported)
}

func roundTripPlan(t *testing.T, plan *deployplan.Plan) *deployplan.Plan {
	t.Helper()

	path := filepath.Join(t.TempDir(), "plan.json")
	data, err := json.Marshal(plan)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, data, 0o600))

	loadedPlan, err := deployplan.LoadPlanFromFile(path)
	require.NoError(t, err)

	return loadedPlan
}
