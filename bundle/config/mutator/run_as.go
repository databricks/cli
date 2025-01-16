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

type setRunAs struct{}

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

func reportRunAsNotSupported(resourceType string, location dyn.Location, currentUser, runAsUser string) diag.Diagnostics {
	return diag.Diagnostics{{
		Summary: fmt.Sprintf("%s do not support a setting a run_as user that is different from the owner.\n"+
			"Current identity: %s. Run as identity: %s.\n"+
			"See https://docs.databricks.com/dev-tools/bundles/run-as.html to learn more about the run_as property.", resourceType, currentUser, runAsUser),
		Locations: []dyn.Location{location},
		Severity:  diag.Error,
	}}
}

func validateRunAs(b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	neitherSpecifiedErr := diag.Diagnostics{{
		Summary:   "run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified",
		Locations: []dyn.Location{b.Config.GetLocation("run_as")},
		Severity:  diag.Error,
	}}

	// Fail fast if neither service_principal_name nor user_name are specified, but the
	// run_as section is present.
	if b.Config.Value().Get("run_as").Kind() == dyn.KindNil {
		return neitherSpecifiedErr
	}

	// Fail fast if one or both of service_principal_name and user_name are specified,
	// but with empty values.
	runAs := b.Config.RunAs
	if runAs.ServicePrincipalName == "" && runAs.UserName == "" {
		return neitherSpecifiedErr
	}

	if runAs.UserName != "" && runAs.ServicePrincipalName != "" {
		diags = diags.Extend(diag.Diagnostics{{
			Summary:   "run_as section cannot specify both user_name and service_principal_name",
			Locations: []dyn.Location{b.Config.GetLocation("run_as")},
			Severity:  diag.Error,
		}})
	}

	identity := runAs.ServicePrincipalName
	if identity == "" {
		identity = runAs.UserName
	}

	// All resources are supported if the run_as identity is the same as the current deployment identity.
	if identity == b.Config.Workspace.CurrentUser.UserName {
		return diags
	}

	// DLT pipelines do not support run_as in the API.
	if len(b.Config.Resources.Pipelines) > 0 {
		diags = diags.Extend(reportRunAsNotSupported(
			"pipelines",
			b.Config.GetLocation("resources.pipelines"),
			b.Config.Workspace.CurrentUser.UserName,
			identity,
		))
	}

	// Model serving endpoints do not support run_as in the API.
	if len(b.Config.Resources.ModelServingEndpoints) > 0 {
		diags = diags.Extend(reportRunAsNotSupported(
			"model_serving_endpoints",
			b.Config.GetLocation("resources.model_serving_endpoints"),
			b.Config.Workspace.CurrentUser.UserName,
			identity,
		))
	}

	// Monitors do not support run_as in the API.
	if len(b.Config.Resources.QualityMonitors) > 0 {
		diags = diags.Extend(reportRunAsNotSupported(
			"quality_monitors",
			b.Config.GetLocation("resources.quality_monitors"),
			b.Config.Workspace.CurrentUser.UserName,
			identity,
		))
	}

	// Dashboards do not support run_as in the API.
	if len(b.Config.Resources.Dashboards) > 0 {
		diags = diags.Extend(reportRunAsNotSupported(
			"dashboards",
			b.Config.GetLocation("resources.dashboards"),
			b.Config.Workspace.CurrentUser.UserName,
			identity,
		))
	}

	// Apps do not support run_as in the API.
	if len(b.Config.Resources.Apps) > 0 {
		diags = diags.Extend(reportRunAsNotSupported(
			"apps",
			b.Config.GetLocation("resources.apps"),
			b.Config.Workspace.CurrentUser.UserName,
			identity,
		))
	}

	return diags
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
				Severity:  diag.Warning,
				Summary:   "You are using the legacy mode of run_as. The support for this mode is experimental and might be removed in a future release of the CLI. In order to run the DLT pipelines in your DAB as the run_as user this mode changes the owners of the pipelines to the run_as identity, which requires the user deploying the bundle to be a workspace admin, and also a Metastore admin if the pipeline target is in UC.",
				Paths:     []dyn.Path{dyn.MustPathFromString("experimental.use_legacy_run_as")},
				Locations: b.Config.GetLocations("experimental.use_legacy_run_as"),
			},
		}
	}

	// Assert the run_as configuration is valid in the context of the bundle
	diags := validateRunAs(b)
	if diags.HasError() {
		return diags
	}

	setRunAsForJobs(b)
	return nil
}
