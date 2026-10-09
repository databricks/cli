package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type applySourceLinkedDeploymentPreset struct{}

func targetPresetPath(target string) *structpath.PathNode {
	return structpath.NewPath(nil, "targets", target, "presets", "source_linked_deployment")
}

// Apply source-linked deployment preset
func ApplySourceLinkedDeploymentPreset() *applySourceLinkedDeploymentPreset {
	return &applySourceLinkedDeploymentPreset{}
}

func (m *applySourceLinkedDeploymentPreset) Name() string {
	return "ApplySourceLinkedDeploymentPreset"
}

func (m *applySourceLinkedDeploymentPreset) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if config.IsExplicitlyDisabled(b.Config.Presets.SourceLinkedDeployment) {
		return nil
	}

	var diags diag.Diagnostics
	isDatabricksWorkspace := dbr.RunsOnRuntime(ctx) && strings.HasPrefix(b.SyncRootPath, "/Workspace/")
	target := b.Config.Bundle.Target

	if config.IsExplicitlyEnabled((b.Config.Presets.SourceLinkedDeployment)) {
		if !isDatabricksWorkspace {
			path := targetPresetPath(target)
			diags = diags.Append(
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "source-linked deployment is available only in the Databricks Workspace",
					Paths:     []*structpath.PathNode{path},
					Locations: b.Config.GetLocations("presets.source_linked_deployment"),
				},
			)

			disabled := false
			b.Config.Presets.SourceLinkedDeployment = &disabled
			return diags
		}

		b.Metrics.AddBoolValue("source_linked_set_for_non_development", b.Config.Bundle.Mode != config.Development)

		if b.Config.Bundle.Mode != config.Development {
			path := targetPresetPath(target)
			diags = diags.Append(
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "source-linked deployment in non-development mode is deprecated and will not be supported in a future release",
					Paths:     []*structpath.PathNode{path},
					Locations: b.Config.GetLocations("presets.source_linked_deployment"),
				},
			)
		}
	}

	if isDatabricksWorkspace && b.Config.Bundle.Mode == config.Development {
		enabled := true
		b.Config.Presets.SourceLinkedDeployment = &enabled
	}

	// This mutator runs before workspace paths are defaulted so it's safe to check for the user-defined value
	if b.Config.Workspace.FilePath != "" && config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		path := structpath.MustParsePath("workspace.file_path")
		diags = diags.Append(
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "workspace.file_path setting will be ignored in source-linked deployment mode",
				Detail:   "In source-linked deployment files are not copied to the destination and resources use source files instead",
				Paths:    []*structpath.PathNode{path},
			},
		)
	}

	return diags
}
