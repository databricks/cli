package bundle

import (
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
)

type syncFlags struct {
	interval time.Duration
	full     bool
	watch    bool
}

func (f *syncFlags) syncOptionsFromBundle(cmd *cobra.Command, b *bundle.Bundle) (*sync.SyncOptions, error) {
	cacheDir, err := b.CacheDir(cmd.Context())
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	includes, err := b.GetSyncIncludePatterns(cmd.Context())
	if err != nil {
		return nil, fmt.Errorf("cannot get list of sync includes: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:    b.Config.Path,
		RemotePath:   b.Config.Workspace.FilePath,
		Include:      includes,
		Exclude:      b.Config.Sync.Exclude,
		Full:         f.full,
		PollInterval: f.interval,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return &opts, nil
}

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [flags]",
		Short: "Synchronize bundle tree to the workspace",
		Args:  cobra.NoArgs,

		PreRunE: ConfigureBundleWithVariables,
	}

	var f syncFlags
	cmd.Flags().DurationVar(&f.interval, "interval", 1*time.Second, "file system polling interval (for --watch)")
	cmd.Flags().BoolVar(&f.full, "full", false, "perform full synchronization (default is incremental)")
	cmd.Flags().BoolVar(&f.watch, "watch", false, "watch local file system for changes")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// Run initialize phase to make sure paths are set.
		err := bundle.Apply(cmd.Context(), b, phases.Initialize())
		if err != nil {
			return err
		}

		opts, err := f.syncOptionsFromBundle(cmd, b)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		s, err := sync.New(ctx, *opts)
		if err != nil {
			return err
		}

		log.Infof(ctx, "Remote file sync location: %v", opts.RemotePath)

		if f.watch {
			return s.RunContinuous(ctx)
		}

		return s.RunOnce(ctx)
	}

	return cmd
}
