package run

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindNoResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{},
		},
	}

	_, err := Find(b, "foo")
	assert.ErrorContains(t, err, "bundle defines no resources")
}

func TestFindSingleArg(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
				},
			},
		},
	}

	_, err := Find(b, "foo")
	assert.NoError(t, err)
}

func TestFindSingleArgNotFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
				},
			},
		},
	}

	_, err := Find(b, "bar")
	assert.ErrorContains(t, err, "no such resource: bar")
}

func TestFindSingleArgAmbiguous(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"key": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"key": {},
				},
			},
		},
	}

	_, err := Find(b, "key")
	assert.ErrorContains(t, err, "ambiguous: ")
}

func TestFindSingleArgWithType(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"key": {},
				},
			},
		},
	}

	_, err := Find(b, "jobs.key")
	assert.NoError(t, err)
}

func TestFoo(t *testing.T) {
	w, err := databricks.NewWorkspaceClient(
		&databricks.Config{
			Profile: "DEFAULT",
		},
	)
	require.NoError(t, err)

	runInfo, err := w.Jobs.GetRun(context.TODO(), jobs.GetRun{
		RunId: 19000091,
	})
	require.NoError(t, err)

	// a, err := w.Jobs.GetRunOutput(context.TODO(), jobs.GetRunOutput{
	// 	RunId: 19000091,
	// })
	// require.NoError(t, err)

	result, err := w.Jobs.GetRunOutput(context.TODO(), jobs.GetRunOutput{
		RunId: 19000915,
	})
	require.NoError(t, err)

	fmt.Println("[DEBUG] GetRun job: ", runInfo)
	// fmt.Println("[DEBUG] GetRun a: ", a)
	fmt.Println("[DEBUG] GetRunOutput task: ", result)
	assert.True(t, false)
}
