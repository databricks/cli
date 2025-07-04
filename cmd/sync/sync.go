package sync

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/vfs"
	"github.com/spf13/cobra"
)

type syncFlags struct {
	// project files polling interval
	interval    time.Duration
	full        bool
	watch       bool
	output      flags.Output
	exclude     []string
	include     []string
	dryRun      bool
	excludeFrom string
	includeFrom string
}

func readPatternsFile(filePath string) ([]string, error) {
	if filePath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read exclude-from file: %w", err)
	}

	var patterns []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading exclude-from file: %w", err)
	}

	return patterns, nil
}

func (f *syncFlags) syncOptionsFromBundle(cmd *cobra.Command, args []string, b *bundle.Bundle) (*sync.SyncOptions, error) {
	if len(args) > 0 {
		return nil, errors.New("SRC and DST are not configurable in the context of a bundle")
	}

	opts, err := files.GetSyncOptions(cmd.Context(), b)
	if err != nil {
		return nil, fmt.Errorf("cannot get sync options: %w", err)
	}

	excludePatterns, err := readPatternsFile(f.excludeFrom)
	if err != nil {
		return nil, err
	}

	includePatterns, err := readPatternsFile(f.includeFrom)
	if err != nil {
		return nil, err
	}

	opts.Full = f.full
	opts.PollInterval = f.interval
	opts.WorktreeRoot = b.WorktreeRoot
	opts.Exclude = append(opts.Exclude, f.exclude...)
	opts.Exclude = append(opts.Exclude, excludePatterns...)
	opts.Include = append(opts.Include, f.include...)
	opts.Include = append(opts.Include, includePatterns...)
	opts.DryRun = f.dryRun
	return opts, nil
}

func (f *syncFlags) syncOptionsFromArgs(cmd *cobra.Command, args []string) (*sync.SyncOptions, error) {
	if err := root.ExactArgs(2)(cmd, args); err != nil {
		return nil, err
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

	ctx := cmd.Context()
	client := cmdctx.WorkspaceClient(ctx)

	if f.dryRun {
		log.Warnf(ctx, "Running in dry-run mode. No actual changes will be made.")
	}

	excludePatterns, err := readPatternsFile(f.excludeFrom)
	if err != nil {
		return nil, err
	}

	includePatterns, err := readPatternsFile(f.includeFrom)
	if err != nil {
		return nil, err
	}

	localRoot := vfs.MustNew(args[0])
	info, err := git.FetchRepositoryInfo(ctx, localRoot.Native(), client)
	if err != nil {
		log.Warnf(ctx, "Failed to read git info: %s", err)
	}

	var worktreeRoot vfs.Path

	if info.WorktreeRoot == "" {
		worktreeRoot = localRoot
	} else {
		worktreeRoot = vfs.MustNew(info.WorktreeRoot)
	}

	opts := sync.SyncOptions{
		WorktreeRoot: worktreeRoot,
		LocalRoot:    localRoot,
		Paths:        []string{"."},
		Include:      append(f.include, includePatterns...),
		Exclude:      append(f.exclude, excludePatterns...),

		RemotePath:   args[1],
		Full:         f.full,
		PollInterval: f.interval,

		// We keep existing behavior for VS Code extension where if there is
		// no bundle defined, we store the snapshots in `.databricks`.
		// The sync code will automatically create this directory if it doesn't
		// exist and add it to the `.gitignore` file in the root.
		SnapshotBasePath: filepath.Join(args[0], ".databricks"),
		WorkspaceClient:  client,

		OutputHandler: outputHandler,
		DryRun:        f.dryRun,
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
	cmd.Flags().StringSliceVar(&f.exclude, "exclude", nil, "patterns to exclude from sync (can be specified multiple times)")
	cmd.Flags().StringSliceVar(&f.include, "include", nil, "patterns to include in sync (can be specified multiple times)")
	cmd.Flags().StringVar(&f.excludeFrom, "exclude-from", "", "file containing patterns to exclude from sync (one pattern per line)")
	cmd.Flags().StringVar(&f.includeFrom, "include-from", "", "file containing patterns to include to sync (one pattern per line)")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "simulate sync execution without making actual changes")

	// Wrapper for [root.MustWorkspaceClient] that disables loading authentication configuration from a bundle.
	mustWorkspaceClient := func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.PreRunE = mustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var opts *sync.SyncOptions
		var err error

		opts, err = f.syncOptionsFromArgs(cmd, args)
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
			wsc := cmdctx.WorkspaceClient(cmd.Context())
			return completeRemotePath(cmd.Context(), wsc, toComplete)
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return cmd
}
