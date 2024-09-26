package mutator

import (
	"context"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allResourceTypes(t *testing.T) []string {
	// Compute supported resource types based on the `Resources{}` struct.
	r := &config.Resources{}
	rv, err := convert.FromTyped(r, dyn.NilValue)
	require.NoError(t, err)
	normalized, _ := convert.Normalize(r, rv, convert.IncludeMissingFields)
	resourceTypes := []string{}
	for _, k := range normalized.MustMap().Keys() {
		resourceTypes = append(resourceTypes, k.MustString())
	}
	slices.Sort(resourceTypes)

	// Assert the total list of resource supported, as a sanity check that using
	// the dyn library gives us the correct list of all resources supported. Please
	// also update this check when adding a new resource
	require.Equal(t, []string{
		"clusters",
		"experiments",
		"jobs",
		"model_serving_endpoints",
		"models",
		"pipelines",
		"quality_monitors",
		"registered_models",
		"schemas",
	},
		resourceTypes,
	)

	return resourceTypes
}

func TestRunAsWorksForAllowedResources(t *testing.T) {
	config := config.Root{
		Workspace: config.Workspace{
			CurrentUser: &config.User{
				User: &iam.User{
					UserName: "alice",
				},
			},
		},
		RunAs: &jobs.JobRunAs{
			UserName: "bob",
		},
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job_one": {
					JobSettings: &jobs.JobSettings{
						Name: "foo",
					},
				},
				"job_two": {
					JobSettings: &jobs.JobSettings{
						Name: "bar",
					},
				},
				"job_three": {
					JobSettings: &jobs.JobSettings{
						Name: "baz",
					},
				},
			},
			Models: map[string]*resources.MlflowModel{
				"model_one": {},
			},
			RegisteredModels: map[string]*resources.RegisteredModel{
				"registered_model_one": {},
			},
			Experiments: map[string]*resources.MlflowExperiment{
				"experiment_one": {},
			},
		},
	}

	b := &bundle.Bundle{
		Config: config,
	}

	diags := bundle.Apply(context.Background(), b, SetRunAs())
	assert.NoError(t, diags.Error())

	for _, job := range b.Config.Resources.Jobs {
		assert.Equal(t, "bob", job.RunAs.UserName)
	}
}

func TestRunAsErrorForUnsupportedResources(t *testing.T) {
	// Bundle "run_as" has two modes of operation, each with a different set of
	// resources that are supported.
	// Cases:
	//   1. When the bundle "run_as" identity is same as the current deployment
	//      identity. In this case all resources are supported.
	//   2. When the bundle "run_as" identity is different from the current
	//      deployment identity. In this case only a subset of resources are
	//      supported. This subset of resources are defined in the allow list below.
	//
	// To be a part of the allow list, the resource must satisfy one of the following
	// two conditions:
	//   1. The resource supports setting a run_as identity to a different user
	//      from the owner/creator of the resource. For example, jobs.
	//   2. Run as semantics do not apply to the resource. We do not plan to add
	//      platform side support for `run_as` for these resources. For example,
	//      experiments or registered models.
	//
	// Any resource that is not on the allow list cannot be used when the bundle
	// run_as is different from the current deployment user. "bundle validate" must
	// return an error if such a resource has been defined, and the run_as identity
	// is different from the current deployment identity.
	//
	// Action Item: If you are adding a new resource to DABs, please check in with
	// the relevant owning team whether the resource should be on the allow list or (implicitly) on
	// the deny list. Any resources that could have run_as semantics in the future
	// should be on the deny list.
	// For example: Teams for pipelines, model serving endpoints or Lakeview dashboards
	// are planning to add platform side support for `run_as` for these resources at
	// some point in the future. These resources are (implicitly) on the deny list, since
	// they are not on the allow list below.
	allowList := []string{
		"clusters",
		"jobs",
		"models",
		"registered_models",
		"experiments",
		"schemas",
	}

	base := config.Root{
		Workspace: config.Workspace{
			CurrentUser: &config.User{
				User: &iam.User{
					UserName: "alice",
				},
			},
		},
		RunAs: &jobs.JobRunAs{
			UserName: "bob",
		},
	}

	v, err := convert.FromTyped(base, dyn.NilValue)
	require.NoError(t, err)

	// Define top level resources key in the bundle configuration.
	// This is not part of the typed configuration, so we need to add it manually.
	v, err = dyn.Set(v, "resources", dyn.V(map[string]dyn.Value{}))
	require.NoError(t, err)

	for _, rt := range allResourceTypes(t) {
		// Skip allowed resources
		if slices.Contains(allowList, rt) {
			continue
		}

		// Add an instance of the resource type that is not on the allow list to
		// the bundle configuration.
		nv, err := dyn.SetByPath(v, dyn.NewPath(dyn.Key("resources"), dyn.Key(rt)), dyn.V(map[string]dyn.Value{
			"foo": dyn.V(map[string]dyn.Value{
				"path": dyn.V("bar"),
			}),
		}))
		require.NoError(t, err)

		// Get back typed configuration from the newly created invalid bundle configuration.
		r := &config.Root{}
		err = convert.ToTyped(r, nv)
		require.NoError(t, err)

		// Assert this invalid bundle configuration fails validation.
		b := &bundle.Bundle{
			Config: *r,
		}
		diags := bundle.Apply(context.Background(), b, SetRunAs())
		assert.Equal(t, diags.Error().Error(), errUnsupportedResourceTypeForRunAs{
			resourceType:     rt,
			resourceLocation: dyn.Location{},
			currentUser:      "alice",
			runAsUser:        "bob",
		}.Error(), "expected run_as with a different identity than the current deployment user to not supported for resources of type: %s", rt)
	}
}
