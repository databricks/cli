package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type setRunAs struct {
}

// This mutator does two things:
//
//  1. Sets the run_as field for jobs to the value of the run_as field in the bundle.
//
//  2. Validates the bundle run_as configuration is valid in the context of the bundle.
//     If the run_as user is different from the current deployment user, DABs only
//     supports a subset of resources.
func SetRunAs() bundle.Mutator {
	return &setRunAs{}
}

func (m *setRunAs) Name() string {
	return "SetRunAs"
}

type errUnsupportedResourceTypeForRunAs struct {
	resourceType     string
	resourceLocation dyn.Location
	currentUser      string
	runAsUser        string
}

// TODO(6 March 2024): Link the docs page describing run_as semantics in the error below
// once the page is ready.
func (e errUnsupportedResourceTypeForRunAs) Error() string {
	return fmt.Sprintf("%s are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. Location of the unsupported resource: %s. Current identity: %s. Run as identity: %s", e.resourceType, e.resourceLocation, e.currentUser, e.runAsUser)
}

type errBothSpAndUserSpecified struct {
	spName   string
	spLoc    dyn.Location
	userName string
	userLoc  dyn.Location
}

func (e errBothSpAndUserSpecified) Error() string {
	return fmt.Sprintf("run_as section must specify exactly one identity. A service_principal_name %q is specified at %s. A user_name %q is defined at %s", e.spName, e.spLoc, e.userName, e.userLoc)
}

func validateRunAs(b *bundle.Bundle) error {
	runAs := b.Config.RunAs

	// Error if neither service_principal_name nor user_name are specified
	if runAs.ServicePrincipalName == "" && runAs.UserName == "" {
		return fmt.Errorf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s", b.Config.GetLocation("run_as"))
	}

	// Error if both service_principal_name and user_name are specified
	if runAs.UserName != "" && runAs.ServicePrincipalName != "" {
		return errBothSpAndUserSpecified{
			spName:   runAs.ServicePrincipalName,
			userName: runAs.UserName,
			spLoc:    b.Config.GetLocation("run_as.service_principal_name"),
			userLoc:  b.Config.GetLocation("run_as.user_name"),
		}
	}

	identity := runAs.ServicePrincipalName
	if identity == "" {
		identity = runAs.UserName
	}

	// All resources are supported if the run_as identity is the same as the current deployment identity.
	if identity == b.Config.Workspace.CurrentUser.UserName {
		return nil
	}

	// DLT pipelines do not support run_as in the API.
	if len(b.Config.Resources.Pipelines) > 0 {
		return errUnsupportedResourceTypeForRunAs{
			resourceType:     "pipelines",
			resourceLocation: b.Config.GetLocation("resources.pipelines"),
			currentUser:      b.Config.Workspace.CurrentUser.UserName,
			runAsUser:        identity,
		}
	}

	// DLT model serving endpoints do not support run_as in the API.
	if len(b.Config.Resources.ModelServingEndpoints) > 0 {
		return errUnsupportedResourceTypeForRunAs{
			resourceType:     "model_serving_endpoints",
			resourceLocation: b.Config.GetLocation("resources.model_serving_endpoints"),
			currentUser:      b.Config.Workspace.CurrentUser.UserName,
			runAsUser:        identity,
		}
	}

	return nil
}

func (m *setRunAs) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Mutator is a no-op if run_as is not specified in the bundle
	runAs := b.Config.RunAs
	if runAs == nil {
		return nil
	}

	// Assert the run_as configuration is valid in the context of the bundle
	if err := validateRunAs(b); err != nil {
		return diag.FromErr(err)
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
