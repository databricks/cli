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

func syncOptionsFromBundle(cmd *cobra.Command, b *bundle.Bundle) (*sync.SyncOptions, error) {
	cacheDir, err := b.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:    b.Config.Path,
		RemotePath:   b.Config.Workspace.FilesPath,
		Full:         full,
		PollInterval: interval,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return &opts, nil
}

var syncCmd = &cobra.Command{
	Use:   "sync [flags]",
	Short: "Synchronize bundle tree to the workspace",
	Args:  cobra.NoArgs,

	PreRunE: ConfigureBundleWithVariables,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// Run initialize phase to make sure paths are set.
		err := bundle.Apply(cmd.Context(), b, phases.Initialize())
		if err != nil {
			return err
		}

		opts, err := syncOptionsFromBundle(cmd, b)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		s, err := sync.New(ctx, *opts)
		if err != nil {
			return err
		}

		log.Infof(ctx, "Remote file sync location: %v", opts.RemotePath)

		if watch {
			return s.RunContinuous(ctx)
		}

		return s.RunOnce(ctx)
	},
}

var interval time.Duration
var full bool
var watch bool

func init() {
	AddCommand(syncCmd)
	syncCmd.Flags().DurationVar(&interval, "interval", 1*time.Second, "file system polling interval (for --watch)")
	syncCmd.Flags().BoolVar(&full, "full", false, "perform full synchronization (default is incremental)")
	syncCmd.Flags().BoolVar(&watch, "watch", false, "watch local file system for changes")
}
