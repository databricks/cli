package loader

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessInclude(t *testing.T) {
	b := &bundle.Bundle{
		RootPath: "testdata",
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	m := ProcessInclude(filepath.Join(b.RootPath, "host.yml"), "host.yml")
	assert.Equal(t, "ProcessInclude(host.yml)", m.Name())

	// Assert the host value prior to applying the mutator
	assert.Equal(t, "foo", b.Config.Workspace.Host)

	// Apply the mutator and assert that the host value has been updated
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

func TestResourceNames(t *testing.T) {
	names := []string{}
	typ := reflect.TypeOf(config.Resources{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTags := strings.Split(field.Tag.Get("json"), ",")
		singularName := strings.TrimSuffix(jsonTags[0], "s")
		names = append(names, singularName)
	}

	// Assert the contents of the two lists are equal. Please add the singular
	// name of your resource to resourceNames global if you are adding a new
	// resource.
	assert.Equal(t, len(resourceTypes), len(names))
	for _, name := range names {
		assert.Contains(t, resourceTypes, name)
	}
}

func TestValidateFileFormat(t *testing.T) {
	onlyJob := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job1": {},
			},
		},
		Targets: map[string]*config.Target{
			"target1": {
				Resources: &config.Resources{
					Jobs: map[string]*resources.Job{
						"job1": {},
					},
				},
			},
		},
	}
}

// func TestGatherResources(t *testing.T) {
// 	// TODO: Add location tests?
// 	tcases := []struct {
// 		name      string
// 		resources config.Resources
// 		targets   map[string]*config.Target
// 		filenames map[string]string
// 		expected  map[string]resource
// 	}{
// 		{
// 			name:      "empty",
// 			resources: config.Resources{},
// 			expected:  map[string]resource{},
// 		},
// 		{
// 			name: "one job",
// 			resources: config.Resources{
// 				Jobs: map[string]*resources.Job{
// 					"job1": {},
// 				},
// 			},
// 			// TODO CONTINUE: Setting file names for the resources defined here.
// 			// and testing they are correctly aggregated.
// 			expected: map[string]resource{
// 				"job1": {
// 					typ: "job",
// 					p:   dyn.MustPathFromString("resources.jobs.job1"),
// 				},
// 			},
// 		},
// 		{
// 			name: "three jobs",
// 			resources: config.Resources{
// 				Jobs: map[string]*resources.Job{
// 					"job1": {},
// 					"job2": {},
// 					"job3": {},
// 				},
// 			},
// 			expected: []resource{
// 				{"job1", "job"},
// 				{"job2", "job"},
// 				{"job3", "job"},
// 			},
// 		},
// 		{
// 			name:      "only one experiment target",
// 			resources: config.Resources{},
// 			targets: map[string]*config.Target{
// 				"target1": {
// 					Resources: &config.Resources{
// 						Experiments: map[string]*resources.MlflowExperiment{
// 							"experiment1": {},
// 						},
// 					},
// 				},
// 			},
// 			expected: []resource{
// 				{"experiment1", "experiment"},
// 			},
// 		},
// 		{
// 			name: "multiple resources",
// 			resources: config.Resources{
// 				Jobs: map[string]*resources.Job{
// 					"job1": {},
// 				},
// 				Pipelines: map[string]*resources.Pipeline{
// 					"pipeline1": {},
// 					"pipeline2": {},
// 				},
// 				Experiments: map[string]*resources.MlflowExperiment{
// 					"experiment1": {},
// 				},
// 			},
// 			targets: map[string]*config.Target{
// 				"target1": {
// 					Resources: &config.Resources{
// 						ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
// 							"model_serving_endpoint1": {},
// 						},
// 					},
// 				},
// 				"target2": {
// 					Resources: &config.Resources{
// 						RegisteredModels: map[string]*resources.RegisteredModel{
// 							"registered_model1": {},
// 						},
// 					},
// 				},
// 			},
// 			expected: []resource{
// 				{"job1", "job"},
// 				{"pipeline1", "pipeline"},
// 				{"pipeline2", "pipeline"},
// 				{"experiment1", "experiment"},
// 				{"model_serving_endpoint1", "model_serving_endpoint"},
// 				{"registered_model1", "registered_model"},
// 			},
// 		},
// 	}

// 	for _, tc := range tcases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			b := &bundle.Bundle{}

// 			bundle.ApplyFunc(context.Background(), b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
// 				b.Config.Resources = tc.resources
// 				b.Config.Targets = tc.targets
// 				return nil
// 			})

// 			res, err := gatherResources(&b.Config)
// 			require.NoError(t, err)

// 			assert.Equal(t, tc.expected, res)
// 		})
// 	}

// }
