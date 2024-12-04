package libraries

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func extractVolumeFromPath(artifactPath string) (string, string, string, error) {
	if !IsVolumesPath(artifactPath) {
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

// This function returns a filer for ".internal" folder inside the directory configured
// at `workspace.artifact_path`.
// This function also checks if the UC volume exists in the workspace and then:
//  1. If the UC volume exists in the workspace:
//     Returns a filer for the UC volume.
//  2. If the UC volume does not exist in the workspace but is (with high confidence) defined in
//     the bundle configuration:
//     Returns an error and a warning that instructs the user to deploy the
//     UC volume before using it in the artifact path.
//  3. If the UC volume does not exist in the workspace and is not defined in the bundle configuration:
//     Returns an error.
func filerForVolume(ctx context.Context, b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	artifactPath := b.Config.Workspace.ArtifactPath
	w := b.WorkspaceClient()

	catalogName, schemaName, volumeName, err := extractVolumeFromPath(artifactPath)
	if err != nil {
		return nil, "", diag.Diagnostics{
			{
				Severity:  diag.Error,
				Summary:   err.Error(),
				Locations: b.Config.GetLocations("workspace.artifact_path"),
				Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
			},
		}
	}

	// Check if the UC volume exists in the workspace.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", catalogName, schemaName, volumeName)
	err = w.Files.GetDirectoryMetadataByDirectoryPath(ctx, volumePath)

	// If the volume exists already, directly return the filer for the path to
	// upload the artifacts to.
	if err == nil {
		uploadPath := path.Join(artifactPath, InternalDirName)
		f, err := filer.NewFilesClient(w, uploadPath)
		return f, uploadPath, diag.FromErr(err)
	}

	baseErr := diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("unable to determine if volume at %s exists: %s", volumePath, err),
		Locations: b.Config.GetLocations("workspace.artifact_path"),
		Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
	}

	if errors.Is(err, apierr.ErrNotFound) {
		// Since the API returned a 404, the volume does not exist.
		// Modify the error message to provide more context.
		baseErr.Summary = fmt.Sprintf("volume %s does not exist: %s", volumePath, err)

		// If the volume is defined in the bundle, provide a more helpful error diagnostic,
		// with more details and location information.
		path, locations, ok := findVolumeInBundle(b, catalogName, schemaName, volumeName)
		if !ok {
			return nil, "", diag.Diagnostics{baseErr}
		}
		baseErr.Detail = `You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.`
		baseErr.Paths = append(baseErr.Paths, path)
		baseErr.Locations = append(baseErr.Locations, locations...)
	}

	return nil, "", diag.Diagnostics{baseErr}
}

func findVolumeInBundle(b *bundle.Bundle, catalogName, schemaName, volumeName string) (dyn.Path, []dyn.Location, bool) {
	volumes := b.Config.Resources.Volumes
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
		pathString := fmt.Sprintf("resources.volumes.%s", k)
		return dyn.MustPathFromString(pathString), b.Config.GetLocations(pathString), true
	}
	return nil, nil, false
}
