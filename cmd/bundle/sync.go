package bundle

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
)

type syncFlags struct {
	interval time.Duration
	full     bool
	watch    bool
	output   flags.Output
	dryRun   bool
}

func (f *syncFlags) syncOptionsFromBundle(cmd *cobra.Command, b *bundle.Bundle) (*sync.SyncOptions, error) {
	opts, err := files.GetSyncOptions(cmd.Context(), b)
	if err != nil {
		return nil, fmt.Errorf("cannot get sync options: %w", err)
	}

	if f.output != "" {
		var outputFunc func(context.Context, <-chan sync.Event, io.Writer)
		switch f.output {
		case flags.OutputText:
			outputFunc = sync.TextOutput
		case flags.OutputJSON:
			outputFunc = sync.JsonOutput
		}
		if outputFunc != nil {
			opts.OutputHandler = func(ctx context.Context, c <-chan sync.Event) {
				outputFunc(ctx, c, cmd.OutOrStdout())
			}
		}
	}

	opts.Full = f.full
	opts.PollInterval = f.interval
	opts.DryRun = f.dryRun
	return opts, nil
}

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [flags]",
		Short: "Synchronize bundle tree to the workspace",
		Args:  root.NoArgs,
	}

	var f syncFlags
	cmd.Flags().DurationVar(&f.interval, "interval", 1*time.Second, "file system polling interval (for --watch)")
	cmd.Flags().BoolVar(&f.full, "full", false, "perform full synchronization (default is incremental)")
	cmd.Flags().BoolVar(&f.watch, "watch", false, "watch local file system for changes")
	cmd.Flags().Var(&f.output, "output", "type of the output format")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "simulate sync execution without making actual changes")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Run initialize phase to make sure paths are set.
		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts, err := f.syncOptionsFromBundle(cmd, b)
		if err != nil {
			return err
		}

		s, err := sync.New(ctx, *opts)
		if err != nil {
			return err
		}
		defer s.Close()

		log.Infof(ctx, "Remote file sync location: %v", opts.RemotePath)

		if opts.DryRun {
			log.Warnf(ctx, "Running in dry-run mode. No actual changes will be made.")
		}

		if f.watch {
			return s.RunContinuous(ctx)
		}

		_, err = s.RunOnce(ctx)
		return err
	}

	return cmd
}
