package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/require"
)

func TestExperimentDoUpdateSetsConfiguredTags(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	resource := (&ResourceExperiment{}).New(client)
	ctx := t.Context()
	id, _, err := resource.DoCreate(ctx, &ml.CreateExperiment{
		Name: "experiment-tags",
		Tags: []ml.ExperimentTag{{Key: "mlflow.workload_creation_policy_id", Value: "old-policy"}},
	})
	require.NoError(t, err)
	before, err := resource.DoRead(ctx, id)
	require.NoError(t, err)

	_, err = resource.DoUpdate(ctx, id, &ml.CreateExperiment{
		Name: "experiment-tags",
		Tags: []ml.ExperimentTag{
			{Key: "mlflow.workload_creation_policy_id", Value: "new-policy"},
			{Key: "added-tag", Value: "value"},
		},
	}, &deployplan.PlanEntry{
		RemoteState: before,
		Changes: deployplan.Changes{
			"tags": {Action: deployplan.Update},
		},
	})
	require.NoError(t, err)

	after, err := resource.DoRead(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, after.ExperimentId)
	tags := make(map[string]string, len(after.Tags))
	for _, tag := range after.Tags {
		tags[tag.Key] = tag.Value
	}
	require.Equal(t, "new-policy", tags["mlflow.workload_creation_policy_id"])
	require.Equal(t, "value", tags["added-tag"])
}

func TestExperimentDoUpdateRejectsTagRemovalBeforeRename(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	resource := (&ResourceExperiment{}).New(client)
	ctx := t.Context()
	id, _, err := resource.DoCreate(ctx, &ml.CreateExperiment{
		Name: "original-name",
		Tags: []ml.ExperimentTag{
			{Key: "keep", Value: "value"},
			{Key: "remove", Value: "value"},
		},
	})
	require.NoError(t, err)
	before, err := resource.DoRead(ctx, id)
	require.NoError(t, err)

	_, err = resource.DoUpdate(ctx, id, &ml.CreateExperiment{
		Name: "new-name",
		Tags: []ml.ExperimentTag{{Key: "keep", Value: "value"}},
	}, &deployplan.PlanEntry{
		RemoteState: before,
		Changes: deployplan.Changes{
			"name":               {Action: deployplan.Update},
			"tags[key='remove']": {Action: deployplan.Update, Old: ml.ExperimentTag{Key: "remove", Value: "value"}},
		},
	})
	require.ErrorContains(t, err, `cannot remove experiment tag "remove"`)

	after, err := resource.DoRead(ctx, id)
	require.NoError(t, err)
	require.Equal(t, before.Name, after.Name)
	require.Equal(t, before.Tags, after.Tags)
}
