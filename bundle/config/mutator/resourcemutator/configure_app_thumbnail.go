package resourcemutator

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// maxAppThumbnailBytes matches the limit of the Apps UI and of the databricks_app
// Terraform resource, so all clients accept the same images.
const maxAppThumbnailBytes = 250_000

type configureAppThumbnail struct{}

// ConfigureAppThumbnail reads the image at resources.apps.*.thumbnail_path, checks that it
// is a PNG or JPEG within the size limit, and stores it base64-encoded in thumbnail_content.
func ConfigureAppThumbnail() bundle.Mutator {
	return &configureAppThumbnail{}
}

func (c configureAppThumbnail) Name() string {
	return "ConfigureAppThumbnail"
}

func (c configureAppThumbnail) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.Apps)) {
		app := b.Config.Resources.Apps[name]
		if app == nil || app.ThumbnailPath == "" {
			continue
		}
		thumbnailPath := app.ThumbnailPath
		locations := b.Config.GetLocationsOf(structpath.NewPath(nil, "resources", "apps", name, "thumbnail_path"))

		contents, err := b.SyncRoot.ReadFile(thumbnailPath)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to read app thumbnail %s: %s", thumbnailPath, err),
				Locations: locations,
			})
			continue
		}

		// The Apps API accepts only PNG and JPEG, detected by their file signatures.
		contentType := http.DetectContentType(contents)
		if contentType != "image/png" && contentType != "image/jpeg" {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("app thumbnail %s must be a PNG or JPEG image, got %s", thumbnailPath, contentType),
				Locations: locations,
			})
			continue
		}

		if len(contents) > maxAppThumbnailBytes {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("app thumbnail %s is %d bytes, larger than the limit of %d bytes", thumbnailPath, len(contents), maxAppThumbnailBytes),
				Locations: locations,
			})
			continue
		}

		app.ThumbnailContent = base64.StdEncoding.EncodeToString(contents)
	}

	return diags
}
