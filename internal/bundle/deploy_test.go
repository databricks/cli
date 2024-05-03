package bundle

import (
	"errors"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleDeployUcSchema(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "uc_schema", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {a
		destroyBundle(t, ctx, bundleRoot)
	})

	// Assert the schema is created
	// TODO: Skip this test on non-uc workspaces. Need a new filter function for it?
	schemaName := "main.test-schema-" + uniqueId
	schema, err := w.Schemas.GetByFullName(ctx, schemaName)
	require.NoError(t, err)
	assert.Equal(t, schema.FullName, schemaName)
	assert.Equal(t, schema.Comment, "This schema was created from DABs")

	// Assert the pipeline is created, and it uses the specified schema
	pipelineName := "test-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(ctx, pipelineName)
	require.NoError(t, err)
	assert.Equal(t, pipeline.Name, pipelineName)
	id := pipeline.PipelineId

	// Assert the pipeline uses the schema
	i, err := w.Pipelines.GetByPipelineId(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, i.Spec.Catalog, "main")
	assert.Equal(t, i.Spec.Target, "test-schema-"+uniqueId)

	// Assert the schema is deleted
	_, err = w.Schemas.GetByFullName(ctx, schemaName)
	apiErr := &apierr.APIError{}
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "SCHEMA_DOES_NOT_EXIST", apiErr.ErrorCode)
}
