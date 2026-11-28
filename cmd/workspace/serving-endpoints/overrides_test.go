package serving_endpoints

import (
	"bytes"
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
