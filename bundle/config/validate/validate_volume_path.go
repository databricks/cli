package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
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
			Severity: diag.Error,
			Summary:  fmt.Sprintf("workspace.root_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.RootPath),
		})
	}

	if b.Config.Workspace.FilePath != "" && strings.HasPrefix(b.Config.Workspace.FilePath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("workspace.file_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.FilePath),
		})
	}

	if b.Config.Workspace.StatePath != "" && strings.HasPrefix(b.Config.Workspace.StatePath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("workspace.state_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.StatePath),
		})
	}

	if b.Config.Workspace.ResourcePath != "" && strings.HasPrefix(b.Config.Workspace.ResourcePath, "/Volumes/") {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("workspace.resource_path %s starts with /Volumes. /Volumes can only be used with workspace.artifact_path.", b.Config.Workspace.ResourcePath),
		})
	}

	return diags
}

func ValidateVolumePath() bundle.ReadOnlyMutator {
	return &validateVolumePath{}
}
