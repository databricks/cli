package libraries

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/utils"

	"golang.org/x/sync/errgroup"
)

// The Files API backend has a rate limit of 10 concurrent
// requests and 100 QPS. We limit the number of concurrent requests to 5 to
// avoid hitting the rate limit.
var maxFilesRequestsInFlight = 5

func Upload(libs map[string][]LocationToUpdate) bundle.Mutator {
	return &upload{
		libs: libs,
	}
}

func UploadWithClient(libs map[string][]LocationToUpdate, client filer.Filer) bundle.Mutator {
	return &upload{
		libs:   libs,
		client: client,
	}
}

type upload struct {
	client filer.Filer
	libs   map[string][]LocationToUpdate
}

type LocationToUpdate struct {
	configPath dyn.Path
	location   dyn.Location
}

func (u *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	client, _, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return diags
	}

	// Only set the filer client if it's not already set. We use the client field
	// in the mutator to mock the filer client in testing
	if u.client == nil {
		u.client = client
	}

	sources := utils.SortedKeys(u.libs)

	errs, errCtx := errgroup.WithContext(ctx)
	errs.SetLimit(maxFilesRequestsInFlight)

	for _, source := range sources {
		relPath, err := filepath.Rel(b.SyncRootPath, source)
		if err != nil {
			relPath = source
		} else {
			relPath = filepath.ToSlash(relPath)
		}
		cmdio.LogString(ctx, fmt.Sprintf("Uploading %s...", relPath))
		errs.Go(func() error {
			return UploadFile(errCtx, source, u.client)
		})
	}

	if err := errs.Wait(); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (u *upload) Name() string {
	return "libraries.Upload"
}

// Function to upload file (a library, artifact and etc) to Workspace or UC volume
func UploadFile(ctx context.Context, file string, client filer.Filer) error {
	filename := filepath.Base(file)

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", file, errors.Unwrap(err))
	}
	defer f.Close()

	err = client.Write(ctx, filename, f, filer.OverwriteIfExists, filer.CreateParentDirectories)
	if err != nil {
		return fmt.Errorf("unable to import %s: %w", filename, err)
	}

	log.Infof(ctx, "Upload succeeded")
	return nil
}
