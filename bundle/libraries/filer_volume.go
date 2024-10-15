package libraries

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/filer"
)

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

	if !strings.HasPrefix(artifactPath, "/Volumes/") {
		return nil, "", diag.Errorf("expected artifact_path to start with /Volumes/, got %s", artifactPath)
	}

	parts := strings.Split(artifactPath, "/")
	volumeFormatErr := fmt.Errorf("expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<volume>/..., got %s", artifactPath)

	// Incorrect format.
	if len(parts) < 5 {
		return nil, "", diag.FromErr(volumeFormatErr)
	}

	catalogName := parts[2]
	schemaName := parts[3]
	volumeName := parts[4]

	// Incorrect format.
	if catalogName == "" || schemaName == "" || volumeName == "" {
		return nil, "", diag.FromErr(volumeFormatErr)
	}

	// Check if the UC volume exists in the workspace.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", catalogName, schemaName, volumeName)
	err := w.Files.GetDirectoryMetadataByDirectoryPath(ctx, volumePath)

	// If the volume exists already, directly return the filer for the path to
	// upload the artifacts to.
	if err == nil {
		uploadPath := path.Join(artifactPath, ".internal")
		f, err := filer.NewFilesClient(w, uploadPath)
		return f, uploadPath, diag.FromErr(err)
	}

	diags := diag.Errorf("failed to fetch metadata for the UC volume %s that is configured in the artifact_path: %s", volumePath, err)

	path, locations, ok := findVolumeInBundle(b, catalogName, schemaName, volumeName)
	if !ok {
		return nil, "", diags
	}

	warning := diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   `You might be using a UC volume in your artifact_path that is managed by this bundle but which has not been deployed yet. Please deploy the UC volume in a separate bundle deploy before using it in the artifact_path.`,
		Locations: locations,
		Paths:     []dyn.Path{path},
	}
	return nil, "", diags.Append(warning)
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
