package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type validateVolumePath struct {
	bundle.RO
}

func (m *validateVolumePath) Name() string {
	return "validate:volume-path"
}

func (m *validateVolumePath) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	if b.Config.Workspace.RootPath != "" && strings.HasPrefix(b.Config.Workspace.RootPath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("workspace.root_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.RootPath),
			Detail:    "For more information, see https://docs.databricks.com/aws/en/dev-tools/bundles/settings#workspace",
			Locations: b.Config.GetLocations("workspace.root_path"),
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.root_path")},
		})

		// We return early here because we don't want to check the other paths if the root path is invalid
		return diags
	}

	if b.Config.Workspace.FilePath != "" && strings.HasPrefix(b.Config.Workspace.FilePath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("workspace.file_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.FilePath),
			Detail:    "For more information, see https://docs.databricks.com/aws/en/dev-tools/bundles/settings#workspace",
			Locations: b.Config.GetLocations("workspace.file_path"),
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.file_path")},
		})
	}

	if b.Config.Workspace.StatePath != "" && strings.HasPrefix(b.Config.Workspace.StatePath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("workspace.state_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.StatePath),
			Detail:    "For more information, see https://docs.databricks.com/aws/en/dev-tools/bundles/settings#workspace",
			Locations: b.Config.GetLocations("workspace.state_path"),
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.state_path")},
		})
	}

	if b.Config.Workspace.ResourcePath != "" && strings.HasPrefix(b.Config.Workspace.ResourcePath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("workspace.resource_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.ResourcePath),
			Detail:    "For more information, see https://docs.databricks.com/aws/en/dev-tools/bundles/settings#workspace",
			Locations: b.Config.GetLocations("workspace.resource_path"),
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.resource_path")},
		})
	}

	return diags
}

func ValidateVolumePath() bundle.ReadOnlyMutator {
	return &validateVolumePath{}
}
