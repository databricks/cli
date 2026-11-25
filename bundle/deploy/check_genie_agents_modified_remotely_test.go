package deploy

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckGenieAgentsModifiedRemotely(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		deployedEtag, remoteEtag                      string
		isPlan, force, removed, firstDeploy, noAgents bool
		getError                                      error
		wantSeverity                                  diag.Severity
		wantSummary                                   string
	}{
		{name: "unchanged", deployedEtag: "v1", remoteEtag: "v1"},
		{name: "plan warns", deployedEtag: "v1", remoteEtag: "v2", isPlan: true, wantSeverity: diag.Warning, wantSummary: `genie agent "agent" has been modified remotely`},
		{name: "deploy refuses", deployedEtag: "v1", remoteEtag: "v2", wantSeverity: diag.Error, wantSummary: `genie agent "agent" has been modified remotely`},
		{name: "force bypasses", deployedEtag: "v1", force: true},
		{name: "old state has no etag"},
		{name: "first deployment", firstDeploy: true},
		{name: "removed resource", deployedEtag: "v1", removed: true},
		{name: "no agents", noAgents: true},
		{name: "missing remote", deployedEtag: "v1", getError: apierr.ErrResourceDoesNotExist},
		{name: "get failure", deployedEtag: "v1", getError: errors.New("failure"), wantSeverity: diag.Error, wantSummary: `failed to get genie agent "agent"`},
		{name: "get failure during plan", deployedEtag: "v1", isPlan: true, getError: errors.New("failure"), wantSeverity: diag.Error, wantSummary: `failed to get genie agent "agent"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				BundleRootPath: t.TempDir(),
				Config: config.Root{
					Bundle:    config.Bundle{Target: "test", Force: tc.force},
					Resources: config.Resources{GenieAgents: map[string]*resources.GenieAgent{"agent": {DisplayName: "My Agent"}}},
				},
			}
			if tc.noAgents {
				b.Config.Resources.GenieAgents = nil
			} else if tc.removed {
				b.Config.Resources.GenieAgents = map[string]*resources.GenieAgent{"other": {}}
			}
			data := dstate.NewDatabase("lineage", 1)
			if !tc.firstDeploy {
				state, err := json.Marshal(map[string]string{"etag": tc.deployedEtag})
				require.NoError(t, err)
				data.State["resources.genie_agents.agent"] = dstate.ResourceEntry{ID: "id1", State: state}
			}
			data.State["resources.genie_agents.agent.permissions"] = dstate.ResourceEntry{ID: "id1", State: json.RawMessage(`{"etag":"permissions"}`)}
			data.State["resources.jobs.job"] = dstate.ResourceEntry{ID: "job-id", State: json.RawMessage(`{}`)}
			b.DeploymentBundle.StateDB.OpenWithData(filepath.Join(t.TempDir(), "resources.json"), data)
			// Only the remote API is mocked; the check reads real deployment state.
			m := mocks.NewMockWorkspaceClient(t)
			if tc.deployedEtag != "" && !tc.force && !tc.removed && !tc.noAgents && !tc.firstDeploy {
				m.GetMockGenieAgentsAPI().EXPECT().GetGenieAgent(mock.Anything, genieagents.GetGenieAgentRequest{Name: "agents/id1"}).
					Return(&genieagents.GenieAgent{Etag: tc.remoteEtag}, tc.getError).Once()
			}
			b.SetWorkpaceClient(m.WorkspaceClient)
			diags := bundle.Apply(t.Context(), b, CheckGenieAgentsModifiedRemotely(tc.isPlan))
			if tc.wantSummary == "" {
				assert.Empty(t, diags)
			} else {
				require.Len(t, diags, 1)
				assert.Equal(t, tc.wantSummary, diags[0].Summary)
				assert.Equal(t, tc.wantSeverity, diags[0].Severity)
			}
		})
	}
}
