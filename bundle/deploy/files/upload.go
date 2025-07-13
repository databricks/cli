package files

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/sync"
)

func Upload(
	ctx context.Context,
	opts *sync.SyncOptions,
	sourceLinked bool,
) ([]fileset.File, error) {
	if sourceLinked {
		cmdio.LogString(ctx, "Source-linked deployment is enabled. Deployed resources reference the source files in your working tree instead of separate copies.")
		return nil, nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("Uploading bundle files to %s...", opts.RemotePath))
	sync, err := sync.New(ctx, *opts)
	if err != nil {
		return nil, err
	}
	defer sync.Close()

	files, err := sync.RunOnce(ctx)
	if err != nil {
		return nil, err
	}

	return files, nil
}
