package direct

import (
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySkipCallsWaitAfterResumeOnlyForNonterminalJobRun(t *testing.T) {
	for name, tc := range map[string]struct {
		state    *jobs.RunState
		wantGets int32
	}{
		"running": {
			state:    &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			wantGets: 2,
		},
		"terminal success": {
			state: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateTerminated,
				ResultState:    jobs.RunResultStateSuccess,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var gets atomic.Int32
			server := testserver.New(t)
			server.Handle("GET", "/api/2.2/jobs/runs/get", func(testserver.Request) any {
				gets.Add(1)
				return jobs.Run{RunId: 123, JobId: 456, State: &jobs.RunState{
					LifeCycleState: jobs.RunLifeCycleStateTerminated,
					ResultState:    jobs.RunResultStateSuccess,
				}}
			})
			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:  server.URL,
				Token: "testtoken",
			})
			require.NoError(t, err)
			adapter, err := dresources.NewAdapter((*dresources.ResourceJobRun)(nil), "job_runs", client)
			require.NoError(t, err)

			const resourceKey = "resources.job_runs.my_run"
			plan := deployplan.NewPlanDirect()
			plan.Plan[resourceKey] = &deployplan.PlanEntry{
				Action:      deployplan.Skip,
				RemoteState: &dresources.JobRunRemote{RunId: 123, State: tc.state},
			}
			b := &DeploymentBundle{
				Adapters: map[string]*dresources.Adapter{"job_runs": adapter},
				Plan:     plan,
			}
			require.NoError(t, b.StateDB.Open(
				t.Context(), filepath.Join(t.TempDir(), "resources.json"),
				dstate.WithRecovery(false), dstate.WithWrite(true),
				dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{},
			))
			require.NoError(t, b.StateDB.SaveState(t.Context(), resourceKey, "123", &dresources.JobRunState{}, nil))

			b.Apply(t.Context(), client, plan, false)
			_, err = b.StateDB.Finalize(t.Context())
			require.NoError(t, err)

			assert.Equal(t, tc.wantGets, gets.Load())
		})
	}
}
