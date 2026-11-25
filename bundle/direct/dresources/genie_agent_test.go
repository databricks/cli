package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
	"github.com/stretchr/testify/require"
)

// The plan's remote read carries the etag the update is guarded by. The
// config (and so the persisted state) never carries it, so the etag must come
// from the plan entry.
func TestGenieAgentDoUpdateSendsPlannedEtag(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	r := (&ResourceGenieAgent{}).New(m.WorkspaceClient)

	config := &resources.GenieAgentConfig{
		DisplayName: "new",
		WarehouseId: "wh",
	}
	entry := &deployplan.PlanEntry{
		RemoteState: &resources.GenieAgentConfig{Etag: "etag-3"},
	}

	m.GetMockGenieAgentsAPI().EXPECT().
		UpdateGenieAgent(ctx, genieagents.UpdateGenieAgentRequest{
			Etag:       "etag-3",
			GenieAgent: config.GenieAgent,
			Name:       "agents/agent-id",
			UpdateMask: fieldmask.FieldMask{Paths: []string{"*"}},
		}).
		Return(&genieagents.GenieAgent{Name: "agents/agent-id", DisplayName: "new", Etag: "etag-4"}, nil).
		Once()

	remote, err := r.DoUpdate(ctx, "agent-id", config, entry)
	require.NoError(t, err)
	require.Equal(t, "agent-id", remote.GenieAgentId)
	require.Equal(t, "new", remote.DisplayName)
}

// Without a plan-time remote read there is nothing to guard the update with.
func TestGenieAgentDoUpdateWithoutPlannedRemoteSendsNoEtag(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	r := (&ResourceGenieAgent{}).New(m.WorkspaceClient)

	config := &resources.GenieAgentConfig{DisplayName: "new", WarehouseId: "wh"}

	m.GetMockGenieAgentsAPI().EXPECT().
		UpdateGenieAgent(ctx, genieagents.UpdateGenieAgentRequest{
			GenieAgent: config.GenieAgent,
			Name:       "agents/agent-id",
			UpdateMask: fieldmask.FieldMask{Paths: []string{"*"}},
		}).
		Return(&genieagents.GenieAgent{Name: "agents/agent-id"}, nil).
		Once()

	_, err := r.DoUpdate(ctx, "agent-id", config, &deployplan.PlanEntry{})
	require.NoError(t, err)
}

// Server-derived fields must not reach the persisted state or the local diff.
func TestGenieAgentPrepareStateZeroesServerDerivedFields(t *testing.T) {
	input := &resources.GenieAgent{
		GenieAgentId:        "agent-id",
		Name:                "agents/agent-id",
		Etag:                "etag-3",
		EffectiveParentPath: "/Users/someone",
		DisplayName:         "name",
	}

	state := (&ResourceGenieAgent{}).PrepareState(input)
	require.Empty(t, state.Name)
	require.Empty(t, state.Etag)
	require.Empty(t, state.EffectiveParentPath)
	require.Nil(t, state.CreateTime)
	require.Nil(t, state.UpdateTime)
	require.Equal(t, "name", state.DisplayName)
	require.Equal(t, "agent-id", state.GenieAgentId)
}

func TestResponseToGenieAgentConfigParentPath(t *testing.T) {
	for _, tc := range []struct {
		name, echoed, effective, want string
	}{
		{"unmanaged", "", "/Users/moved", ""},
		{"moved", "/Workspace/Users/original", "/Users/moved", "/Workspace/Users/moved"},
		{"prefixed", "/Workspace/Users/original", "/Workspace/Users/moved", "/Workspace/Users/moved"},
		{"no effective path", "/Workspace/Users/original", "", "/Workspace/Users/original"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agent := &genieagents.GenieAgent{ParentPath: tc.echoed, EffectiveParentPath: tc.effective}
			config := responseToGenieAgentConfig(agent)
			require.Equal(t, tc.want, config.ParentPath)
			require.Equal(t, tc.echoed, agent.ParentPath)
		})
	}
}

func TestGenieAgentPrepareStateParentPath(t *testing.T) {
	for _, parentPath := range []string{"", "/Users/someone", "/Workspace/Users/someone"} {
		t.Run(parentPath, func(t *testing.T) {
			state := (&ResourceGenieAgent{}).PrepareState(&resources.GenieAgent{ParentPath: parentPath})
			if parentPath == "" {
				require.Empty(t, state.ParentPath)
			} else {
				require.Equal(t, "/Workspace/Users/someone", state.ParentPath)
			}
		})
	}
}
