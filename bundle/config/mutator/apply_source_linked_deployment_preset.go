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

	target := b.Config.Bundle.Target

	// Immutable bundles deploy files to a content-addressed snapshot. Source-linked
	// deployment would instead point ${workspace.file_path} at the mutable sync root,
	// so a single resource could mix snapshot paths with source-tree paths that
	// change without a deploy. The two modes are mutually exclusive.
	if b.IsImmutableFolder() {
		if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
			path := dyn.NewPath(dyn.Key("targets"), dyn.Key(target), dyn.Key("presets"), dyn.Key("source_linked_deployment"))
			return diag.Diagnostics{{
				Severity:  diag.Error,
				Summary:   "presets.source_linked_deployment cannot be enabled when experimental.immutable_folder is enabled",
				Paths:     []dyn.Path{path},
				Locations: b.Config.GetLocations(path[2:].String()),
			}}
		}
		// Do not auto-enable source-linked deployment for development mode.
		return nil
	}

	var diags diag.Diagnostics
	isDatabricksWorkspace := dbr.RunsOnRuntime(ctx) && strings.HasPrefix(b.SyncRootPath, "/Workspace/")

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

		b.Metrics.AddBoolValue("source_linked_set_for_non_development", b.Config.Bundle.Mode != config.Development)

		if b.Config.Bundle.Mode != config.Development {
			path := dyn.NewPath(dyn.Key("targets"), dyn.Key(target), dyn.Key("presets"), dyn.Key("source_linked_deployment"))
			diags = diags.Append(
				diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "source-linked deployment in non-development mode is deprecated and will not be supported in a future release",
					Paths: []dyn.Path{
						path,
					},
					Locations: b.Config.GetLocations(path[2:].String()),
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
		path := dyn.NewPath(dyn.Key("workspace"), dyn.Key("file_path"))
		diags = diags.Append(
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "workspace.file_path setting will be ignored in source-linked deployment mode",
				Detail:   "In source-linked deployment files are not copied to the destination and resources use source files instead",
				Paths: []dyn.Path{
					path,
				},
				Locations: b.Config.GetLocations(path.String()),
			},
		)
	}

	return diags
}
