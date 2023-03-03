package sync

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/sync"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

func syncOptionsFromBundle(cmd *cobra.Command, args []string, b *bundle.Bundle) (*sync.SyncOptions, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("SRC and DST are not configurable in the context of a bundle")
	}

	cacheDir, err := b.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:    b.Config.Path,
		RemotePath:   b.Config.Workspace.FilePath.Workspace,
		Full:         full,
		PollInterval: interval,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return &opts, nil
}

func syncOptionsFromArgs(cmd *cobra.Command, args []string) (*sync.SyncOptions, error) {
	if len(args) != 2 {
		return nil, flag.ErrHelp
	}

	opts := sync.SyncOptions{
		LocalPath:    args[0],
		RemotePath:   args[1],
		Full:         full,
		PollInterval: interval,

		// We keep existing behavior for VS Code extension where if there is
		// no bundle defined, we store the snapshots in `.databricks`.
		// The sync code will automatically create this directory if it doesn't
		// exist and add it to the `.gitignore` file in the root.
		SnapshotBasePath: filepath.Join(args[0], ".databricks"),
		WorkspaceClient:  databricks.Must(databricks.NewWorkspaceClient()),
	}
	return &opts, nil
}

var syncCmd = &cobra.Command{
	Use:   "sync [flags] SRC DST",
	Short: "Synchronize a local directory to a workspace directory",
	Args:  cobra.MaximumNArgs(2),

	// PreRunE: root.TryConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		var opts *sync.SyncOptions
		var err error

		//
		// To be uncommented and used once our VS Code extension is bundle aware.
		// Until then, this could interfere with extension usage where a `bundle.yml` file is present.
		// See https://github.com/databricks/bricks/pull/207.
		//
		// b := bundle.GetOrNil(cmd.Context())
		// if b != nil {
		// 	// Run initialize phase to make sure paths are set.
		// 	err = bundle.Apply(cmd.Context(), b, []bundle.Mutator{
		// 		phases.Initialize(),
		// 	})
		// 	if err != nil {
		// 		return err
		// 	}
		// 	opts, err = syncOptionsFromBundle(cmd, args, b)
		// } else {
		opts, err = syncOptionsFromArgs(cmd, args)
		// }
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		s, err := sync.New(ctx, *opts)
		if err != nil {
			return err
		}

		switch output {
		case flags.OutputText:
			go logOutput(ctx, s.Events())
		case flags.OutputJSON:
			go jsonOutput(ctx, s.Events(), cmd.OutOrStdout())
		}

		if watch {
			return s.RunContinuous(ctx)
		}

		return s.RunOnce(ctx)
	},

	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		err := root.TryConfigureBundle(cmd, args)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// No completion in the context of a bundle.
		// Source and destination paths are taken from bundle configuration.
		b := bundle.GetOrNil(cmd.Context())
		if b != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		switch len(args) {
		case 0:
			return nil, cobra.ShellCompDirectiveFilterDirs
		case 1:
			wsc, err := databricks.NewWorkspaceClient()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return completeRemotePath(cmd.Context(), wsc, toComplete)
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

// project files polling interval
var interval time.Duration
var full bool
var watch bool
var output flags.Output = flags.OutputText

func init() {
	root.RootCmd.AddCommand(syncCmd)
	syncCmd.Flags().DurationVar(&interval, "interval", 1*time.Second, "file system polling interval (for --watch)")
	syncCmd.Flags().BoolVar(&full, "full", false, "perform full synchronization (default is incremental)")
	syncCmd.Flags().BoolVar(&watch, "watch", false, "watch local file system for changes")
	syncCmd.Flags().Var(&output, "output", "type of output to produce (default is text)")
}
