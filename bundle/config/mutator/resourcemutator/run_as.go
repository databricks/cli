package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

type setRunAs struct{}

// This mutator does two things:
//
//  1. Sets the run_as field of resources that support it to the value of the run_as field in the bundle.
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

func reportRunAsNotSupported(resourceType string, location diag.Location, currentUser, runAsUser string) diag.Diagnostics {
	return diag.Diagnostics{{
		Summary: fmt.Sprintf("%s do not support a setting a run_as user that is different from the owner.\n"+
			"Current identity: %s. Run as identity: %s.\n"+
			"See https://docs.databricks.com/dev-tools/bundles/run-as.html to learn more about the run_as property.", resourceType, currentUser, runAsUser),
		Locations: []diag.Location{location},
		Severity:  diag.Error,
	}}
}

func validateRunAs(b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	identityCount := 0
	runAs := b.Config.RunAs
	if runAs != nil {
		for _, identity := range []string{runAs.UserName, runAs.ServicePrincipalName, runAs.GroupName} {
			if identity != "" {
				identityCount++
			}
		}
	}
	if identityCount != 1 {
		return diag.Diagnostics{{
			Summary:   "run_as section must specify exactly one non-empty identity: user_name, service_principal_name, or group_name",
			Locations: []diag.Location{b.Config.GetLocation("run_as")},
			Severity:  diag.Error,
		}}
	}

	identity := runAs.ServicePrincipalName
	if identity == "" {
		identity = runAs.UserName
	}

	// All resources are supported if the run_as identity is the same as the current deployment identity.
	if runAs.GroupName == "" && identity == b.Config.Workspace.CurrentUser.UserName {
		return diags
	}

	if runAs.GroupName != "" {
		identity = fmt.Sprintf("group %q", runAs.GroupName)
	}

	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			if validator, ok := r.(resources.RunAsValidator); ok {
				if err := validator.ValidateRunAs(runAs); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	// Errors are reported in AllResources order.
	for _, group := range b.Config.Resources.AllResources() {
		resourceType := group.Description.PluralName
		declarer, _ := config.ResourceTypeAs[resources.RunAsDeclarer](resourceType)
		if declarer.RunAsSupport() != resources.RunAsRejected {
			continue
		}

		if rejecter, ok := declarer.(resources.RunAsConditionalRejecter); ok {
			for key, r := range group.Resources {
				if !r.(resources.RunAsConditionalRejecter).RejectsRunAs() {
					continue
				}
				diags = diags.Extend(reportRunAsNotSupported(
					rejecter.RunAsRejectedSubject(),
					b.Config.GetLocationOf(structpath.NewPath(nil, "resources", resourceType, key)),
					b.Config.Workspace.CurrentUser.UserName,
					identity,
				))
			}
			continue
		}

		if len(group.Resources) == 0 {
			continue
		}
		diags = diags.Extend(reportRunAsNotSupported(
			resourceType,
			b.Config.GetLocation("resources."+resourceType),
			b.Config.Workspace.CurrentUser.UserName,
			identity,
		))
	}

	return diags
}

func applyRunAs(b *bundle.Bundle) {
	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			if receiver, ok := r.(resources.RunAsReceiver); ok {
				receiver.ApplyRunAs(b.Config.RunAs)
			}
		}
	}
}

func applyLegacyRunAs(b *bundle.Bundle) {
	runAs := b.Config.RunAs
	if runAs == nil {
		return
	}

	currentUser := b.Config.Workspace.CurrentUser.UserName
	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			if receiver, ok := r.(resources.RunAsLegacyReceiver); ok {
				receiver.ApplyLegacyRunAs(runAs, currentUser)
			}
		}
	}
}

func (m *setRunAs) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Track the use of the legacy run_as mode.
	b.Metrics.AddBoolValue("experimental.use_legacy_run_as", b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.UseLegacyRunAs))

	// Track whether top level run_as is set.
	b.Metrics.AddBoolValue("run_as_set", b.Config.Value().Get("run_as").Kind() != dyn.KindInvalid)

	// Mutator is a no-op if run_as is not specified in the bundle
	if b.Config.Value().Get("run_as").Kind() == dyn.KindInvalid {
		return nil
	}

	// User has opted to use the legacy behavior of run_as with the
	// experimental.use_legacy_run_as flag.
	if b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.UseLegacyRunAs) {
		if b.Config.Value().Get("run_as").Get("group_name").Kind() != dyn.KindInvalid {
			return diag.Errorf("run_as.group_name is not supported with experimental.use_legacy_run_as; disable experimental.use_legacy_run_as to use a group identity")
		}
		applyLegacyRunAs(b)
		return diag.Diagnostics{
			{
				Severity:  diag.Warning,
				Summary:   "You are using the legacy mode of run_as. The support for this mode is experimental and might be removed in a future release of the CLI. In order to run the pipelines in your DABs project as the run_as user this mode changes the owners of the pipelines to the run_as identity, which requires the user deploying the bundle to be a workspace admin, and also a Metastore admin if the pipeline target is in UC.",
				Paths:     structpath.NewPathSlice("experimental", "use_legacy_run_as"),
				Locations: b.Config.GetLocations("experimental.use_legacy_run_as"),
			},
		}
	}

	// Assert the run_as configuration is valid in the context of the bundle
	diags := validateRunAs(b)
	if diags.HasError() {
		return diags
	}

	applyRunAs(b)
	return nil
}
