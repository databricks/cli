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
	// Define paths to check and their corresponding config field names
	pathChecks := []struct {
		path       string
		configName string
	}{
		{b.Config.Workspace.RootPath, "workspace.root_path"},
		{b.Config.Workspace.FilePath, "workspace.file_path"},
		{b.Config.Workspace.StatePath, "workspace.state_path"},
		{b.Config.Workspace.ResourcePath, "workspace.resource_path"},
	}

	// Check each path
	for _, check := range pathChecks {
		if check.path != "" && strings.HasPrefix(check.path, "/Volumes/") {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("%s %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", check.configName, check.path),
				Detail:    "For more information, see https://docs.databricks.com/aws/en/dev-tools/bundles/settings#workspace",
				Locations: b.Config.GetLocations(check.configName),
				Paths:     []dyn.Path{dyn.MustPathFromString(check.configName)},
			})

			// Return early for root path validation
			if check.configName == "workspace.root_path" {
				return diags
			}
		}
	}

	return diags
}

func ValidateVolumePath() bundle.ReadOnlyMutator {
	return &validateVolumePath{}
}
