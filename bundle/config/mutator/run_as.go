package mutator

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
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
//  2. Validates that the bundle run_as configuration is valid in the context of the bundle.
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

func (e errUnsupportedResourceTypeForRunAs) Error() string {
	return fmt.Sprintf("%s are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. Please refer to the documentation at https://docs.databricks.com/dev-tools/bundles/run-as.html for more details. Location of the unsupported resource: %s. Current identity: %s. Run as identity: %s", e.resourceType, e.resourceLocation, e.currentUser, e.runAsUser)
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
	neitherSpecifiedErr := fmt.Errorf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s", b.Config.GetLocation("run_as"))
	// Error if neither service_principal_name nor user_name are specified, but the
	// run_as section is present.
	if b.Config.Value().Get("run_as").Kind() == dyn.KindNil {
		return neitherSpecifiedErr
	}
	// Error if one or both of service_principal_name and user_name are specified,
	// but with empty values.
	if b.Config.RunAs.ServicePrincipalName == "" && b.Config.RunAs.UserName == "" {
		return neitherSpecifiedErr
	}

	// Error if both service_principal_name and user_name are specified
	runAs := b.Config.RunAs
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
	// TODO: maybe oly make this check only fail when owner != runas
	if len(b.Config.Resources.Pipelines) > 0 {
		return errUnsupportedResourceTypeForRunAs{
			resourceType:     "pipelines",
			resourceLocation: b.Config.GetLocation("resources.pipelines"),
			currentUser:      b.Config.Workspace.CurrentUser.UserName,
			runAsUser:        identity,
		}
	}

	// Model serving endpoints do not support run_as in the API.
	if len(b.Config.Resources.ModelServingEndpoints) > 0 {
		return errUnsupportedResourceTypeForRunAs{
			resourceType:     "model_serving_endpoints",
			resourceLocation: b.Config.GetLocation("resources.model_serving_endpoints"),
			currentUser:      b.Config.Workspace.CurrentUser.UserName,
			runAsUser:        identity,
		}
	}

	// Monitors do not support run_as in the API.
	if len(b.Config.Resources.QualityMonitors) > 0 {
		return errUnsupportedResourceTypeForRunAs{
			resourceType:     "quality_monitors",
			resourceLocation: b.Config.GetLocation("resources.quality_monitors"),
			currentUser:      b.Config.Workspace.CurrentUser.UserName,
			runAsUser:        identity,
		}
	}

	return nil
}

func setRunAsForJobs(b *bundle.Bundle) {
	runAs := b.Config.RunAs
	if runAs == nil {
		return
	}

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
}

// Legacy behavior of run_as for DLT pipelines. Available under the experimental.use_run_as_legacy flag.
// Only available to unblock customers stuck due to breaking changes in https://github.com/databricks/cli/pull/1233
func setPipelineOwnersToRunAsIdentity(b *bundle.Bundle) {
	runAs := b.Config.RunAs
	if runAs == nil {
		return
	}

	me := b.Config.Workspace.CurrentUser.UserName
	// If user deploying the bundle and the one defined in run_as are the same
	// Do not add IS_OWNER permission. Current user is implied to be an owner in this case.
	// Otherwise, it will fail due to this bug https://github.com/databricks/terraform-provider-databricks/issues/2407
	if runAs.UserName == me || runAs.ServicePrincipalName == me {
		return
	}

	for i := range b.Config.Resources.Pipelines {
		pipeline := b.Config.Resources.Pipelines[i]
		pipeline.Permissions = slices.DeleteFunc(pipeline.Permissions, func(p resources.Permission) bool {
			return (runAs.ServicePrincipalName != "" && p.ServicePrincipalName == runAs.ServicePrincipalName) ||
				(runAs.UserName != "" && p.UserName == runAs.UserName)
		})
		pipeline.Permissions = append(pipeline.Permissions, resources.Permission{
			Level:                "IS_OWNER",
			ServicePrincipalName: runAs.ServicePrincipalName,
			UserName:             runAs.UserName,
		})
	}
}

func (m *setRunAs) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Mutator is a no-op if run_as is not specified in the bundle
	if b.Config.Value().Get("run_as").Kind() == dyn.KindInvalid {
		return nil
	}

	if b.Config.Experimental != nil && b.Config.Experimental.UseLegacyRunAs {
		setPipelineOwnersToRunAsIdentity(b)
		setRunAsForJobs(b)
		return diag.Diagnostics{
			{
				Severity: diag.Warning,
				Summary:  "You are using the legacy mode of run_as. The support for this mode is experimental and might be removed in a future release of the CLI. In order to run the DLT pipelines in your DAB as the run_as user this mode changes the owners of the pipelines to the run_as identity, which requires the user deploying the bundle to be a workspace admin, and also a Metastore admin if the pipeline target is in UC.",
				Path:     dyn.MustPathFromString("experimental.use_legacy_run_as"),
				Location: b.Config.GetLocation("experimental.use_legacy_run_as"),
			},
		}
	}

	// Assert the run_as configuration is valid in the context of the bundle
	if err := validateRunAs(b); err != nil {
		return diag.FromErr(err)
	}

	setRunAsForJobs(b)
	return nil
}
