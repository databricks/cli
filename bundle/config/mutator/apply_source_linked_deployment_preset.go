package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type applySourceLinkedDeploymentPreset struct{}

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
			path := dyn.NewPath(dyn.Key("targets"), dyn.Key(target), dyn.Key("presets"), dyn.Key("source_linked_deployment"))
			diags = diags.Append(
				diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "source-linked deployment is available only in the Databricks Workspace",
					Paths: []dyn.Path{
						path,
					},
					Locations: b.Config.GetLocations(path[2:].String()),
				},
			)

			disabled := false
			b.Config.Presets.SourceLinkedDeployment = &disabled
			return diags
		}
	}

	if isDatabricksWorkspace && b.Config.Bundle.Mode == config.Development {
		enabled := true
		b.Config.Presets.SourceLinkedDeployment = &enabled
	}

	if len(b.Config.Resources.Apps) > 0 && config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		path := dyn.NewPath(dyn.Key("targets"), dyn.Key(target), dyn.Key("presets"), dyn.Key("source_linked_deployment"))
		diags = diags.Append(
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "source-linked deployment is not supported for apps",
				Paths: []dyn.Path{
					path,
				},
				Locations: b.Config.GetLocations(path[2:].String()),
			},
		)

		return diags
	}

	if b.Config.Workspace.FilePath != "" && config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		path := dyn.NewPath(dyn.Key("targets"), dyn.Key(target), dyn.Key("workspace"), dyn.Key("file_path"))

		diags = diags.Append(
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "workspace.file_path setting will be ignored in source-linked deployment mode",
				Paths: []dyn.Path{
					path[2:],
				},
				Locations: b.Config.GetLocations(path[2:].String()),
			},
		)
	}

	return diags
}
