package resourcemutator

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// maxAppThumbnailBytes matches the limit of the Apps UI and of the databricks_app
// Terraform resource, so all clients accept the same images.
const maxAppThumbnailBytes = 250_000

type configureAppThumbnail struct{}

// ConfigureAppThumbnail reads the image at resources.apps.*.thumbnail_path, checks that it
// is a PNG or JPEG within the size limit, and stores it base64-encoded in thumbnail.
func ConfigureAppThumbnail() bundle.Mutator {
	return &configureAppThumbnail{}
}

func (c configureAppThumbnail) Name() string {
	return "ConfigureAppThumbnail"
}

func (c configureAppThumbnail) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			pathValue := v.Get("thumbnail_path")
			thumbnailPath, ok := pathValue.AsString()
			if !ok || thumbnailPath == "" {
				return v, nil
			}

			contents, err := b.SyncRoot.ReadFile(thumbnailPath)
			if err != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("failed to read app thumbnail %s: %s", thumbnailPath, err),
					Locations: pathValue.Locations(),
				})
				return v, nil
			}

			// The Apps API accepts only PNG and JPEG, detected by their file signatures.
			contentType := http.DetectContentType(contents)
			if contentType != "image/png" && contentType != "image/jpeg" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("app thumbnail %s must be a PNG or JPEG image, got %s", thumbnailPath, contentType),
					Locations: pathValue.Locations(),
				})
				return v, nil
			}

			if len(contents) > maxAppThumbnailBytes {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("app thumbnail %s is %d bytes, larger than the limit of %d bytes", thumbnailPath, len(contents), maxAppThumbnailBytes),
					Locations: pathValue.Locations(),
				})
				return v, nil
			}

			return dyn.Set(v, "thumbnail", dyn.V(base64.StdEncoding.EncodeToString(contents)))
		})
	})

	return diags.Extend(diag.FromErr(err))
}
