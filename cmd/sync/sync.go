package sync

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/vfs"
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

	opts, err := files.GetSyncOptions(cmd.Context(), bundle.ReadOnly(b))
	if err != nil {
		return nil, fmt.Errorf("cannot get sync options: %w", err)
	}

	opts.Full = f.full
	opts.PollInterval = f.interval
	return opts, nil
}

func (f *syncFlags) syncOptionsFromArgs(cmd *cobra.Command, args []string) (*sync.SyncOptions, error) {
	if len(args) != 2 {
		return nil, flag.ErrHelp
	}

	var outputFunc func(context.Context, <-chan sync.Event, io.Writer)
	switch f.output {
	case flags.OutputText:
		outputFunc = sync.TextOutput
	case flags.OutputJSON:
		outputFunc = sync.JsonOutput
	}

	var outputHandler sync.OutputHandler
	if outputFunc != nil {
		outputHandler = func(ctx context.Context, events <-chan sync.Event) {
			outputFunc(ctx, events, cmd.OutOrStdout())
		}
	}

	opts := sync.SyncOptions{
		LocalRoot: vfs.MustNew(args[0]),
		Paths:     []string{"."},
		Include:   nil,
		Exclude:   nil,

		RemotePath:   args[1],
		Full:         f.full,
		PollInterval: f.interval,

		// We keep existing behavior for VS Code extension where if there is
		// no bundle defined, we store the snapshots in `.databricks`.
		// The sync code will automatically create this directory if it doesn't
		// exist and add it to the `.gitignore` file in the root.
		SnapshotBasePath: filepath.Join(args[0], ".databricks"),
		WorkspaceClient:  root.WorkspaceClient(cmd.Context()),

		OutputHandler: outputHandler,
	}
	return &opts, nil
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sync [flags] SRC DST",
		Short:   "Synchronize a local directory to a workspace directory",
		Args:    root.MaximumNArgs(2),
		GroupID: "development",
	}

	f := syncFlags{
		output: flags.OutputText,
	}
	cmd.Flags().DurationVar(&f.interval, "interval", 1*time.Second, "file system polling interval (for --watch)")
	cmd.Flags().BoolVar(&f.full, "full", false, "perform full synchronization (default is incremental)")
	cmd.Flags().BoolVar(&f.watch, "watch", false, "watch local file system for changes")
	cmd.Flags().Var(&f.output, "output", "type of output format")

	// Wrapper for [root.MustWorkspaceClient] that disables loading authentication configuration from a bundle.
	mustWorkspaceClient := func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.PreRunE = mustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var opts *sync.SyncOptions
		var err error

		//
		// To be uncommented and used once our VS Code extension is bundle aware.
		// Until then, this could interfere with extension usage where a `databricks.yml` file is present.
		// See https://github.com/databricks/cli/pull/207.
		//
		// b := bundle.GetOrNil(cmd.Context())
		// if b != nil {
		// 	// Run initialize phase to make sure paths are set.
		// 	err = bundle.Apply(cmd.Context(), b, phases.Initialize())
		// 	if err != nil {
		// 		return err
		// 	}
		// 	opts, err = syncOptionsFromBundle(cmd, args, b)
		// } else {
		opts, err = f.syncOptionsFromArgs(cmd, args)
		// }
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		s, err := sync.New(ctx, *opts)
		if err != nil {
			return err
		}
		defer s.Close()

		if f.watch {
			err = s.RunContinuous(ctx)
		} else {
			_, err = s.RunOnce(ctx)
		}

		return err
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cmd.SetContext(root.SkipPrompt(cmd.Context()))

		err := mustWorkspaceClient(cmd, args)
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
			wsc := root.WorkspaceClient(cmd.Context())
			return completeRemotePath(cmd.Context(), wsc, toComplete)
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return cmd
}
