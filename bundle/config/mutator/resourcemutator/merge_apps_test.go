package resourcemutator_test

import (
	"github.com/databricks/cli/libs/utils"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/stretchr/testify/assert"
)

func TestMergeApps(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"foo": {
						App: apps.App{
							Name: new("foo"),
							Resources: []apps.AppResource{
								{
									Name: new("job1"),
									Job: &apps.AppResourceJob{
										Id:         new("1234"),
										Permission: "CAN_MANAGE_RUN",
									},
								},
								{
									Name: new("sql1"),
									SqlWarehouse: &apps.AppResourceSqlWarehouse{
										Id:         new("5678"),
										Permission: "CAN_USE",
									},
								},
								{
									Name: new("job1"),
									Job: &apps.AppResourceJob{
										Id:         new("1234"),
										Permission: "CAN_MANAGE",
									},
								},
								{
									Name: new("sql1"),
									Job: &apps.AppResourceJob{
										Id:         new("9876"),
										Permission: "CAN_MANAGE",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, resourcemutator.MergeApps())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Apps["foo"]

	assert.Len(t, j.Resources, 2)
	assert.Equal(t, "job1", utils.Deref(j.Resources[0].Name))
	assert.Equal(t, "sql1", utils.Deref(j.Resources[1].Name))

	assert.Equal(t, "CAN_MANAGE", string(j.Resources[0].Job.Permission))

	assert.Nil(t, j.Resources[1].SqlWarehouse)
	assert.Equal(t, "CAN_MANAGE", string(j.Resources[1].Job.Permission))
}
