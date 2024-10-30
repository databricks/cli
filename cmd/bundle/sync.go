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
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
)

type syncFlags struct {
	interval time.Duration
	full     bool
	watch    bool
	output   flags.Output
}

func (f *syncFlags) syncOptionsFromBundle(cmd *cobra.Command, b *bundle.Bundle) (*sync.SyncOptions, error) {
	opts, err := files.GetSyncOptions(cmd.Context(), bundle.ReadOnly(b))
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

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		// Run initialize phase to make sure paths are set.
		diags = bundle.Apply(ctx, b, phases.Initialize())
		if err := diags.Error(); err != nil {
			return err
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

		if f.watch {
			return s.RunContinuous(ctx)
		}

		_, err = s.RunOnce(ctx)
		return err
	}

	return cmd
}
