package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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
		configPath *structpath.PathNode
	}{
		{b.Config.Workspace.RootPath, structpath.NewPath(nil, "workspace", "root_path")},
		{b.Config.Workspace.FilePath, structpath.NewPath(nil, "workspace", "file_path")},
		{b.Config.Workspace.StatePath, structpath.NewPath(nil, "workspace", "state_path")},
		{b.Config.Workspace.ResourcePath, structpath.NewPath(nil, "workspace", "resource_path")},
	}

	// Check each path
	for _, check := range pathChecks {
		if check.path != "" && strings.HasPrefix(check.path, "/Volumes/") {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("%s %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", check.configPath, check.path),
				Detail:    "For more information, see https://docs.databricks.com/aws/en/dev-tools/bundles/settings#workspace",
				Locations: b.Config.GetLocationsOf(check.configPath),
				Paths:     []*structpath.PathNode{check.configPath},
			})

			// Return early for root path validation
			if check.configPath.String() == "workspace.root_path" {
				return diags
			}
		}
	}

	return diags
}

func ValidateVolumePath() bundle.ReadOnlyMutator {
	return &validateVolumePath{}
}
