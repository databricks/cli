package bundle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/configsync"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

func newConfigRemoteSyncCommand() *cobra.Command {
	var save bool
	var selectIDs []string

	cmd := &cobra.Command{
		Use:   "config-remote-sync",
		Short: "[EXPERIMENTAL] Sync remote resource changes to bundle configuration",
		Long: `[EXPERIMENTAL] Compares deployed state with current remote state and generates updated configuration files.

When --save is specified, writes updated YAML files to disk.
Otherwise, outputs diff without modifying files.

IMPORTANT: This is an experimental feature and is subject to change. Windows is not yet supported.

Examples:
  # Show diff without saving
  databricks bundle config-remote-sync

  # Show diff and save to files
  databricks bundle config-remote-sync --save

  # Restrict the sync to a single resource by its type and deployed resource ID
  databricks bundle config-remote-sync --select-ids jobs:123456789 --save`,
		Hidden: true, // Used by DABs in the Workspace only
	}

	cmd.Flags().BoolVar(&save, "save", false, "Write updated config files to disk")
	cmd.Flags().StringSliceVar(&selectIDs, "select-ids", nil, "Sync only the given resources, each as <type>:<id> (e.g. jobs:123456789). Can be repeated or comma-separated.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS == "windows" {
			return errors.New("config-remote-sync command is not supported on Windows")
		}

		stats := configsync.Stats{Save: save}

		// Emit telemetry on every exit path, including failures inside
		// ProcessBundleRet before PostStateFunc runs. Skip when no auth config
		// was resolved: without it the upload at the end of the command
		// lifecycle has no workspace to send to.
		defer func() {
			if cmdctx.HasConfigUsed(cmd.Context()) {
				stats.LogTelemetry(cmd.Context())
			}
		}()

		// Populated by PostStateFunc on success; rendered after the run so an
		// error raised anywhere in ProcessBundleRet takes the same output path.
		var (
			files   []configsync.FileChange
			changes configsync.Changes
		)

		_, _, err := utils.ProcessBundleRet(cmd, utils.ProcessOptions{
			ReadState:  true,
			Build:      true,
			AlwaysPull: true,
			InitFunc: func(b *bundle.Bundle) {
				b.SkipLocalFileValidation = true
			},
			PostStateFunc: func(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error {
				stats.Engine = stateDesc.Engine
				stats.CollectStateStats(stateDesc)

				// Open the deployment state once and reuse it for both planning and
				// selector resolution (avoids reading the terraform snapshot twice).
				deployBundle, err := configsync.OpenDeploymentState(ctx, b, stateDesc.Engine)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryDetectChangesFailed
					if errors.Is(err, configsync.ErrStateSnapshotNotFound) {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryStateNotFound
					}
					return err
				}

				plan, err := deployBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), &b.Config)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryDetectChangesFailed
					return fmt.Errorf("failed to detect changes: %w", err)
				}

				detected, err := configsync.ExtractChanges(ctx, b, plan, stateDesc.Engine)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryDetectChangesFailed
					return fmt.Errorf("failed to extract changes: %w", err)
				}
				stats.CollectChangeStats(ctx, detected)

				// Record the ids present in state and the ids requested: on failure
				// they are what classifies the miss.
				stats.CollectStateIDs(slices.Collect(maps.Keys(configsync.IndexDeployedResources(&deployBundle.StateDB))))

				if len(selectIDs) > 0 {
					stats.CollectSelectedIDs(selectIDs)
					// Filter after planning, never before: the plan must cover every
					// resource so ${resources.*} references resolve; only the emitted
					// changes are restricted to the selected resources.
					selected, err := configsync.ResolveResourceSelectors(ctx, &deployBundle.StateDB, selectIDs)
					if err != nil {
						return err
					}
					detected = configsync.FilterChanges(detected, selected)
				}

				// Loaded once and shared: ResolveChanges uses it to skip changes whose
				// parent is a variable reference, RestoreVariableReferences to restore refs.
				preResolved := configsync.LoadPreResolvedConfig(ctx, b)

				fieldChanges, skipped, err := configsync.ResolveChanges(ctx, b, detected, preResolved)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryResolveFailed
					return fmt.Errorf("failed to resolve field changes: %w", err)
				}
				stats.SkippedChangesCount = int64(skipped)

				if err := configsync.RestoreVariableReferences(ctx, b, fieldChanges, preResolved, &stats.Restore); err != nil {
					log.Warnf(ctx, "variable restoration skipped: %v", err)
				}

				applied, err := configsync.ApplyChangesToYAML(ctx, b, fieldChanges)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryYamlApplyFailed
					return fmt.Errorf("failed to generate YAML files: %w", err)
				}
				stats.FilesChangedCount = int64(len(applied))

				if save {
					if err := configsync.SaveFiles(ctx, b, applied); err != nil {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategorySaveFailed
						return fmt.Errorf("failed to save files: %w", err)
					}
					stats.FilesWrittenCount = int64(len(applied))
				}

				files = applied
				changes = detected
				return nil
			},
		})

		return configsync.WriteResult(cmd.OutOrStdout(), root.OutputType(cmd) == flags.OutputJSON, &stats, files, changes, err)
	}

	return cmd
}
