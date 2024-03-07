package mutator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type setRunAs struct {
}

// This mutator does two things:
//  1. It sets the run_as field for jobs to the value of the run_as field in the bundle.
//  2. Validates only supported resource types are defined in the bundle when the run_as
//     identity is different from the current deployment user.
func SetRunAs() bundle.Mutator {
	return &setRunAs{}
}

func (m *setRunAs) Name() string {
	return "SetRunAs"
}

// TODO: Make sure the error message includes line numbers and file path for
// where the faulty resource is located.

// TODO: Add a test that does not allow adding a new resource type without being deliberate
// about whether it belongs to allow list or deny list.

// Resources that satisfy one of the following conditions:
//  1. Allow to set run_as for the resources to a different user from the current
//     deployment user. For example, jobs.
//  2. Does not make sense for these resources to run_as a different user. We do not
//     have plans to add platform side support for `run_as` for these resources.
//     For example, experiments or model serving endpoints.
var allowListForRunAsOther = []string{"jobs", "models", "registered_models", "experiments"}

// Resources that do not allow setting a run_as identity to a different user but
// have plans to add platform side support for `run_as` for these resources at
// some point in the future. For example, pipelines, model serving endpoints or lakeview dashboards.
//
// We expect the allow list and the deny list to form a closure of all resource types
// supported by DABs.
var denyListForRunAsOther = []string{"pipelines", "model_serving_endpoints"}

type errorUnsupportedResourceTypeForRunAs struct {
	resourceType  string
	resourceValue dyn.Value
	currentUser   string
	runAsUser     string
}

// TODO(6 March 2024): This error message is big. We should split this once
// diag.Diagnostics is ready.
// TODO: Does this required any updates to the default template? Probably no.
func (e errorUnsupportedResourceTypeForRunAs) Error() string {
	return fmt.Sprintf("%s are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. List of supported resources: [%s]. Location of unsupported resource: %s. Current identity: %s. Run as identity: %s", e.resourceType, strings.Join(allowListForRunAsOther, ", "), e.resourceValue.Location(), e.currentUser, e.runAsUser)
}

func getRunAsIdentity(runAs dyn.Value) (string, error) {
	// Get service principal name and user name from run_as section
	runAsSp, err := dyn.Get(runAs, "service_principal_name")
	if err != nil && !dyn.IsNoSuchKeyError(err) {
		return "", err
	}
	runAsUser, err := dyn.Get(runAs, "user_name")
	if err != nil && !dyn.IsNoSuchKeyError(err) {
		return "", err
	}

	switch {
	case runAsSp != dyn.InvalidValue && runAsUser != dyn.InvalidValue:
		// TODO: test this case.
		return "", fmt.Errorf("run_as section must specify exactly one identity. Both service_principal_name (%s) and user_name are defined (%s)", runAsSp.Location(), runAsUser.Location())
	case runAsSp != dyn.InvalidValue:
		return runAsSp.MustString(), nil
	case runAsUser != dyn.InvalidValue:
		return runAsUser.MustString(), nil
	default:
		return "", nil
	}
}

func (m *setRunAs) Apply(_ context.Context, b *bundle.Bundle) error {
	// Return early if run_as is not defined in the bundle
	runAs := b.Config.RunAs
	if runAs == nil {
		return nil
	}

	currentUser := b.Config.Workspace.CurrentUser.UserName

	// Assert the run_as configuration is valid
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		// Get run_as from the bundle
		runAs, err := dyn.Get(v, "run_as")

		// If run_as is not defined in the bundle, return early
		if dyn.IsNoSuchKeyError(err) {
			return v, nil
		}
		if err != nil {
			return dyn.InvalidValue, err
		}

		runAsIdentity, err := getRunAsIdentity(runAs)
		if err != nil {
			return dyn.InvalidValue, err
		}

		// If run_as is the same as the current user, return early. All resource
		// types are allowed in this case.
		if runAsIdentity == currentUser {
			return v, nil
		}

		rv, err := dyn.Get(v, "resources")
		if err != nil {
			return dyn.NilValue, err
		}

		r := rv.MustMap()
		for k, v := range r {
			if !slices.Contains(allowListForRunAsOther, k) {
				return dyn.InvalidValue, errorUnsupportedResourceTypeForRunAs{
					resourceType:  k,
					resourceValue: v,
					currentUser:   currentUser,
					runAsUser:     runAsIdentity,
				}
			}
		}
		return v, nil
	})
	if err != nil {
		return err
	}

	// Set run_as for jobs
	for i := range b.Config.Resources.Jobs {
		job := b.Config.Resources.Jobs[i]
		if job.RunAs != nil {
			continue
		}
		job.RunAs = &jobs.JobRunAs{
			ServicePrincipalName: runAs.ServicePrincipalName,
			UserName:             runAs.UserName,
		}
	}

	return nil
}
