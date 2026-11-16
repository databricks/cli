package serving_endpoints

import (
	"bytes"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
)

func entity(name string, deployment serving.ServedModelStateDeployment, msg string) serving.ServedEntityOutput {
	return serving.ServedEntityOutput{
		Name:  name,
		State: &serving.ServedModelState{Deployment: deployment, DeploymentStateMessage: msg},
	}
}

func TestEndpointProgressMessage(t *testing.T) {
	inProgress := &serving.EndpointState{ConfigUpdate: serving.EndpointStateConfigUpdateInProgress}

	tests := []struct {
		name     string
		endpoint *serving.ServingEndpointDetailed
		want     string
	}{
		{
			name: "pending config entity message",
			endpoint: &serving.ServingEndpointDetailed{
				State: inProgress,
				PendingConfig: &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{
					entity("iris-1", serving.ServedModelStateDeploymentDeploymentCreating, "Container image creation in progress"),
				}},
			},
			want: "iris-1: Container image creation in progress",
		},
		{
			name: "multiple entities, deployment state when message is empty",
			endpoint: &serving.ServingEndpointDetailed{
				State: inProgress,
				PendingConfig: &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{
					entity("a-1", serving.ServedModelStateDeploymentDeploymentCreating, "Provisioning"),
					entity("b-2", serving.ServedModelStateDeploymentDeploymentReady, ""),
				}},
			},
			want: "a-1: Provisioning; b-2: DEPLOYMENT_READY",
		},
		{
			name: "falls back to active config when nothing is pending",
			endpoint: &serving.ServingEndpointDetailed{
				State: &serving.EndpointState{ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating},
				Config: &serving.EndpointCoreConfigOutput{ServedEntities: []serving.ServedEntityOutput{
					entity("iris-1", serving.ServedModelStateDeploymentDeploymentReady, "Served entity creation succeeded"),
				}},
			},
			want: "iris-1: Served entity creation succeeded",
		},
		{
			name: "no entity state keeps the old message",
			endpoint: &serving.ServingEndpointDetailed{
				State:         inProgress,
				PendingConfig: &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{{Name: "iris-1"}}},
			},
			want: "current status: IN_PROGRESS",
		},
		{
			name:     "no state at all",
			endpoint: &serving.ServingEndpointDetailed{},
			want:     "current status: ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, endpointProgressMessage(tt.endpoint))
		})
	}
}

func TestWaitProgressPrintsChangesWithoutTTY(t *testing.T) {
	// Buffers are not TTYs, so this is the agent/CI case where no spinner renders.
	var stdout, stderr bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, nil, &stdout, &stderr, "", ""))

	p := newWaitProgress(ctx)
	for _, msg := range []string{"Container image creation pending", "Container image creation pending", "Provisioning"} {
		p.report(ctx, &serving.ServingEndpointDetailed{
			PendingConfig: &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{
				entity("iris-1", serving.ServedModelStateDeploymentDeploymentCreating, msg),
			}},
		})
	}
	p.report(ctx, &serving.ServingEndpointDetailed{
		State: &serving.EndpointState{ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating},
		Config: &serving.EndpointCoreConfigOutput{ServedEntities: []serving.ServedEntityOutput{
			entity("iris-1", serving.ServedModelStateDeploymentDeploymentReady, "Served entity creation succeeded"),
		}},
	})
	p.close()

	assert.Equal(t, "iris-1: Container image creation pending (0s)\niris-1: Provisioning (0s)\n", stderr.String())
	assert.Empty(t, stdout.String())
}

func TestWaitProgressExplain(t *testing.T) {
	waitErr := errors.New("failed to reach NOT_UPDATING, got UPDATE_FAILED: current status: UPDATE_FAILED")

	t.Run("adds entity state and log commands", func(t *testing.T) {
		// Shape of a real endpoint whose container build failed.
		p := &waitProgress{endpoint: &serving.ServingEndpointDetailed{
			Name:  "my-endpoint",
			State: &serving.EndpointState{ConfigUpdate: serving.EndpointStateConfigUpdateUpdateFailed},
			PendingConfig: &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{
				entity("bad_model-1", serving.ServedModelStateDeploymentDeploymentFailed,
					"Container build failed. Cause: user_pip_resolution. See Build Logs for details."),
			}},
		}}
		err := p.explain("my-endpoint", waitErr)
		assert.ErrorIs(t, err, waitErr)
		assert.Equal(t, waitErr.Error()+`
bad_model-1: Container build failed. Cause: user_pip_resolution. See Build Logs for details.
  build logs:  databricks serving-endpoints build-logs my-endpoint bad_model-1
  server logs: databricks serving-endpoints logs my-endpoint bad_model-1`, err.Error())
	})

	t.Run("no poll result keeps the error", func(t *testing.T) {
		p := &waitProgress{}
		assert.Equal(t, waitErr, p.explain("my-endpoint", waitErr))
	})

	t.Run("entities without state keep the error", func(t *testing.T) {
		p := &waitProgress{endpoint: &serving.ServingEndpointDetailed{
			PendingConfig: &serving.EndpointPendingConfig{ServedEntities: []serving.ServedEntityOutput{{Name: "m-1"}}},
		}}
		assert.Equal(t, waitErr, p.explain("my-endpoint", waitErr))
	})
}
