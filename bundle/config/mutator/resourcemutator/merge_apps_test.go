package resourcemutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
)

func TestMergeApps(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"foo": {
						App: apps.App{
							Name: "foo",
							Resources: []apps.AppResource{
								{
									Name: "job1",
									Job: &apps.AppResourceJob{
										Id:         "1234",
										Permission: "CAN_MANAGE_RUN",
									},
								},
								{
									Name: "sql1",
									SqlWarehouse: &apps.AppResourceSqlWarehouse{
										Id:         "5678",
										Permission: "CAN_USE",
									},
								},
								{
									Name: "job1",
									Job: &apps.AppResourceJob{
										Id:         "1234",
										Permission: "CAN_MANAGE",
									},
								},
								{
									Name: "sql1",
									Job: &apps.AppResourceJob{
										Id:         "9876",
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

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeApps())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Apps["foo"]

	assert.Len(t, j.Resources, 2)
	assert.Equal(t, "job1", j.Resources[0].Name)
	assert.Equal(t, "sql1", j.Resources[1].Name)

	assert.Equal(t, "CAN_MANAGE", string(j.Resources[0].Job.Permission))

	assert.Nil(t, j.Resources[1].SqlWarehouse)
	assert.Equal(t, "CAN_MANAGE", string(j.Resources[1].Job.Permission))
}
