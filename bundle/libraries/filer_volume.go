package libraries

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/filer"
)

func ExtractVolumeFromPath(artifactPath string) (string, string, string, error) {
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

func filerForVolume(ctx context.Context, b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	w := b.WorkspaceClient()
	uploadPath := path.Join(b.Config.Workspace.ArtifactPath, InternalDirName)
	f, err := filer.NewFilesClient(w, uploadPath)
	return f, uploadPath, diag.FromErr(err)
}

func FindVolumeInBundle(r config.Root, catalogName, schemaName, volumeName string) (dyn.Path, []dyn.Location, bool) {
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
		pathString := fmt.Sprintf("resources.volumes.%s", k)
		return dyn.MustPathFromString(pathString), r.GetLocations(pathString), true
	}
	return nil, nil, false
}
