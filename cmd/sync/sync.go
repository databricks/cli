package sync

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	stdsync "sync"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

type syncFlags struct {
	// project files polling interval
	interval time.Duration
	full     bool
	watch    bool
	output   flags.Output
}

func (f *syncFlags) syncOptionsFromBundle(cmd *cobra.Command, args []string, b *bundle.Bundle) (*sync.SyncOptions, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("SRC and DST are not configurable in the context of a bundle")
	}

	cacheDir, err := b.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	opts := sync.SyncOptions{
		LocalPath:    b.Config.Path,
		RemotePath:   b.Config.Workspace.FilesPath,
		Full:         f.full,
		PollInterval: f.interval,

		SnapshotBasePath: cacheDir,
		WorkspaceClient:  b.WorkspaceClient(),
	}
	return &opts, nil
}

func (f *syncFlags) syncOptionsFromArgs(cmd *cobra.Command, args []string) (*sync.SyncOptions, error) {
	if len(args) != 2 {
		return nil, flag.ErrHelp
	}

	opts := sync.SyncOptions{
		LocalPath:    args[0],
		RemotePath:   args[1],
		Full:         f.full,
		PollInterval: f.interval,

		// We keep existing behavior for VS Code extension where if there is
		// no bundle defined, we store the snapshots in `.databricks`.
		// The sync code will automatically create this directory if it doesn't
		// exist and add it to the `.gitignore` file in the root.
		SnapshotBasePath: filepath.Join(args[0], ".databricks"),
		WorkspaceClient:  databricks.Must(databricks.NewWorkspaceClient()),
	}
	return &opts, nil
}

func (f *syncFlags) syncOptions(cmd *cobra.Command, args []string) (*sync.SyncOptions, error) {
	// Try to get options from args first. If that fails, try to get them from bundle.
	// If both fail, return the error from args (assuming users are trying to use the args
	// version unless they explicitly use the bundle sync).
	opts, argsErr := f.syncOptionsFromArgs(cmd, args)
	if argsErr == nil {
		return opts, nil
	}
	err := root.MustConfigureBundle(cmd, args)
	if err != nil {
		return nil, argsErr
	}
	b := bundle.GetOrNil(cmd.Context())
	if b != nil {
		// Run initialize phase to make sure paths are set.
		err = bundle.Apply(cmd.Context(), b, phases.Initialize())
		if err != nil {
			return nil, argsErr
		}
		opts, err = f.syncOptionsFromBundle(cmd, args, b)
		if err != nil {
			return nil, argsErr
		}
		return opts, nil
	}
	return nil, argsErr
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [flags] SRC DST",
		Short: "Synchronize a local directory to a workspace directory",
		Args:  cobra.MaximumNArgs(2),
	}

	f := syncFlags{
		output: flags.OutputText,
	}
	cmd.Flags().DurationVar(&f.interval, "interval", 1*time.Second, "file system polling interval (for --watch)")
	cmd.Flags().BoolVar(&f.full, "full", false, "perform full synchronization (default is incremental)")
	cmd.Flags().BoolVar(&f.watch, "watch", false, "watch local file system for changes")
	cmd.Flags().Var(&f.output, "output", "type of output format")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// To be uncommented and used once our VS Code extension is bundle aware.
		// Until then, this could interfere with extension usage where a `databricks.yml` file is present.
		// See https://github.com/databricks/cli/pull/207.
		//
		opts, err := f.syncOptions(cmd, args)

		if err != nil {
			return err
		}

		ctx := cmd.Context()
		s, err := sync.New(ctx, *opts)
		if err != nil {
			return err
		}

		var outputFunc func(context.Context, <-chan sync.Event, io.Writer)
		switch f.output {
		case flags.OutputText:
			outputFunc = textOutput
		case flags.OutputJSON:
			outputFunc = jsonOutput
		}

		var wg stdsync.WaitGroup
		if outputFunc != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				outputFunc(ctx, s.Events(), cmd.OutOrStdout())
			}()
		}

		if f.watch {
			err = s.RunContinuous(ctx)
		} else {
			err = s.RunOnce(ctx)
		}

		s.Close()
		wg.Wait()
		return err
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	}

	return cmd
}
