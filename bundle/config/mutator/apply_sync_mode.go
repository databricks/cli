package mutator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

var (
	syncModePath               = structpath.MustParsePath("workspace.sync_mode")
	sourceLinkedDeploymentPath = structpath.MustParsePath("presets.source_linked_deployment")
	immutableFolderPath        = structpath.MustParsePath("experimental.immutable_folder")
)

type applySyncMode struct{}

// ApplySyncMode validates workspace.sync_mode and warns about the deprecated settings it replaces.
// It also checks that in-place is supported and makes it the default for development deployments
// from the Databricks Workspace.
func ApplySyncMode() *applySyncMode {
	return &applySyncMode{}
}

func (m *applySyncMode) Name() string {
	return "ApplySyncMode"
}

func (m *applySyncMode) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := warnDeprecatedSyncMode(b)

	// Variable references are rejected too: the sync mode is needed before variables are resolved.
	if mode := b.Config.Workspace.SyncMode; mode != "" && !slices.Contains(mode.Values(), mode) {
		return diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("invalid value %q for workspace.sync_mode (expected one of %v)", mode, mode.Values()),
			Paths:     []*structpath.PathNode{syncModePath},
			Locations: b.Config.GetLocationsOf(syncModePath),
		})
	}

	// An explicit sync mode other than in-place opts out of the in-place default below.
	mode := b.SyncMode()
	if mode != "" && mode != config.SyncModeInPlace {
		return diags
	}

	isDatabricksWorkspace := dbr.RunsOnRuntime(ctx) && strings.HasPrefix(b.SyncRootPath, "/Workspace/")

	if mode == config.SyncModeInPlace {
		// In-place is set by workspace.sync_mode or the deprecated presets.source_linked_deployment.
		path := syncModePath
		if b.Config.Workspace.SyncMode == "" {
			path = sourceLinkedDeploymentPath
		}

		if !isDatabricksWorkspace {
			diags = diags.Append(
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "source-linked deployment is available only in the Databricks Workspace",
					Paths:     []*structpath.PathNode{path},
					Locations: b.Config.GetLocationsOf(path),
				},
			)

			// workspace.sync_mode takes precedence over the deprecated setting as well.
			b.Config.Workspace.SyncMode = config.SyncModeOverwrite
			return diags
		}

		b.Metrics.AddBoolValue("source_linked_set_for_non_development", b.Config.Bundle.Mode != config.Development)

		if b.Config.Bundle.Mode != config.Development {
			diags = diags.Append(
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "source-linked deployment in non-development mode is deprecated and will not be supported in a future release",
					Paths:     []*structpath.PathNode{path},
					Locations: b.Config.GetLocationsOf(path),
				},
			)
		}
	}

	if isDatabricksWorkspace && b.Config.Bundle.Mode == config.Development {
		b.Config.Workspace.SyncMode = config.SyncModeInPlace
	}

	// This mutator runs before workspace paths are defaulted so it's safe to check for the user-defined value
	if b.Config.Workspace.FilePath != "" && b.IsSourceLinkedDeployment() {
		path := structpath.NewPath(nil, "workspace", "file_path")
		diags = diags.Append(
			diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   "workspace.file_path setting will be ignored in source-linked deployment mode",
				Detail:    "In source-linked deployment files are not copied to the destination and resources use source files instead",
				Paths:     []*structpath.PathNode{path},
				Locations: b.Config.GetLocationsOf(path),
			},
		)
	}

	return diags
}

// warnDeprecatedSyncMode warns about the deprecated settings that workspace.sync_mode replaces.
// They keep working, with the precedence of [bundle.Bundle.SyncMode].
func warnDeprecatedSyncMode(b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// override is the highest-precedence setting that is set so far; it overrides the ones below it.
	var override *structpath.PathNode
	if b.Config.Workspace.SyncMode != "" {
		override = syncModePath
	}

	if b.Config.Experimental != nil && b.Config.Experimental.ImmutableFolder {
		diags = diags.Append(deprecatedSyncModeWarning(b, immutableFolderPath, override))
		if override == nil {
			override = immutableFolderPath
		}
	}

	if b.Config.Presets.SourceLinkedDeployment != nil {
		diags = diags.Append(deprecatedSyncModeWarning(b, sourceLinkedDeploymentPath, override))
	}

	return diags
}

// deprecatedSyncModeWarning warns about the deprecated setting at path. A non-nil override is
// the setting that takes precedence over it.
func deprecatedSyncModeWarning(b *bundle.Bundle, path, override *structpath.PathNode) diag.Diagnostic {
	summary := fmt.Sprintf("%s is deprecated, please set workspace.sync_mode to %q instead", path, b.SyncMode())
	if override != nil {
		summary = fmt.Sprintf("%s is deprecated and ignored because %s is set, please remove it", path, override)
	}
	return diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   summary,
		Paths:     []*structpath.PathNode{path},
		Locations: b.Config.GetLocationsOf(path),
	}
}
