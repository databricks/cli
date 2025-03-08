package validate

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type validateArtifactPath struct{ bundle.RO }

func ValidateArtifactPath() bundle.ReadOnlyMutator {
	return &validateArtifactPath{}
}

func (v *validateArtifactPath) Name() string {
	return "validate:artifact_paths"
}

func extractVolumeFromPath(artifactPath string) (string, string, string, error) {
	if !libraries.IsVolumesPath(artifactPath) {
		return "", "", "", fmt.Errorf("expected artifact_path to start with /Volumes/, got %s", artifactPath)
	}

	parts := strings.Split(artifactPath, "/")
	volumeFormatErr := fmt.Errorf("expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<volume>/..., got %s", artifactPath)

	// Incorrect format.
	if len(parts) < 5 {
		return "", "", "", volumeFormatErr
	}

	catalogName := parts[2]
	schemaName := parts[3]
	volumeName := parts[4]

	// Incorrect format.
	if catalogName == "" || schemaName == "" || volumeName == "" {
		return "", "", "", volumeFormatErr
	}

	return catalogName, schemaName, volumeName, nil
}

func findVolumeInBundle(r config.Root, catalogName, schemaName, volumeName string) (dyn.Path, []dyn.Location, bool) {
	volumes := r.Resources.Volumes
	for k, v := range volumes {
		if v.CatalogName != catalogName || v.Name != volumeName {
			continue
		}
		// UC schemas can be defined in the bundle itself, and thus might be interpolated
		// at runtime via the ${resources.schemas.<name>} syntax. Thus we match the volume
		// definition if the schema name is the same as the one in the bundle, or if the
		// schema name is interpolated.
		// We only have to check for ${resources.schemas...} references because any
		// other valid reference (like ${var.foo}) would have been interpolated by this point.
		p, ok := dynvar.PureReferenceToPath(v.SchemaName)
		isSchemaDefinedInBundle := ok && p.HasPrefix(dyn.Path{dyn.Key("resources"), dyn.Key("schemas")})
		if v.SchemaName != schemaName && !isSchemaDefinedInBundle {
			continue
		}
		pathString := "resources.volumes." + k
		return dyn.MustPathFromString(pathString), r.GetLocations(pathString), true
	}
	return nil, nil, false
}

func (v *validateArtifactPath) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// We only validate UC Volumes paths right now.
	if !libraries.IsVolumesPath(b.Config.Workspace.ArtifactPath) {
		return nil
	}

	wrapErrorMsg := func(s string) diag.Diagnostics {
		return diag.Diagnostics{
			{
				Summary:   s,
				Severity:  diag.Error,
				Locations: b.Config.GetLocations("workspace.artifact_path"),
				Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
			},
		}
	}

	catalogName, schemaName, volumeName, err := extractVolumeFromPath(b.Config.Workspace.ArtifactPath)
	if err != nil {
		return wrapErrorMsg(err.Error())
	}
	volumeFullName := fmt.Sprintf("%s.%s.%s", catalogName, schemaName, volumeName)
	w := b.WorkspaceClient()
	_, err = w.Volumes.ReadByName(ctx, volumeFullName)

	if errors.Is(err, apierr.ErrPermissionDenied) {
		return wrapErrorMsg(fmt.Sprintf("cannot access volume %s: %s", volumeFullName, err))
	}
	if errors.Is(err, apierr.ErrNotFound) {
		path, locations, ok := findVolumeInBundle(b.Config, catalogName, schemaName, volumeName)
		if !ok {
			return wrapErrorMsg(fmt.Sprintf("volume %s does not exist", volumeFullName))
		}

		// If the volume is defined in the bundle, provide a more helpful error diagnostic,
		// with more details and location information.
		return diag.Diagnostics{{
			Summary:  fmt.Sprintf("volume %s does not exist", volumeFullName),
			Severity: diag.Error,
			Detail: `You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.`,
			Locations: slices.Concat(b.Config.GetLocations("workspace.artifact_path"), locations),
			Paths:     append([]dyn.Path{dyn.MustPathFromString("workspace.artifact_path")}, path),
		}}

	}
	if err != nil {
		return wrapErrorMsg(fmt.Sprintf("cannot read volume %s: %s", volumeFullName, err))
	}
	return nil
}
