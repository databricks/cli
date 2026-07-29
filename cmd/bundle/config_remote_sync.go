package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/configsync"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

func retrySourceChanges(ctx context.Context, attempt func() error) error {
	delay := 10 * time.Millisecond
	for {
		err := attempt()
		if !errors.Is(err, configsync.ErrSourceChanged) || errors.Is(err, configsync.ErrSourceRecoveryRequired) {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, time.Second)
	}
}

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
		baseContext := cmd.Context()

		// Emit telemetry on every exit path, including failures inside
		// ProcessBundleRet before PostStateFunc runs. Skip when no auth config
		// was resolved: without it the upload at the end of the command
		// lifecycle has no workspace to send to.
		defer func() {
			if cmdctx.HasConfigUsed(cmd.Context()) {
				stats.LogTelemetry(cmd.Context())
			}
		}()

		runAttempt := func() error {
			var sourceSnapshot *configsync.SourceSnapshot
			var sourceSnapshotErr error

			_, _, err := utils.ProcessBundleRet(cmd, utils.ProcessOptions{
				ReadState:  true,
				Build:      true,
				AlwaysPull: true,
				InitFunc: func(b *bundle.Bundle) {
					b.SkipLocalFileValidation = true
					sourceSnapshot, sourceSnapshotErr = configsync.CaptureSourceSnapshot(cmd.Context(), b)
				},
				PostInitFunc: func(_ context.Context, _ *bundle.Bundle) error {
					if sourceSnapshotErr != nil {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryResolveFailed
						return fmt.Errorf("failed to capture source configuration: %w", sourceSnapshotErr)
					}
					return nil
				},
				PostStateFunc: func(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error {
					stats.Engine = stateDesc.Engine

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

					changes, err := configsync.ExtractChanges(ctx, b, plan, stateDesc.Engine)
					if err != nil {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryDetectChangesFailed
						return fmt.Errorf("failed to extract changes: %w", err)
					}
					stats.CollectChangeStats(ctx, changes)

					if len(selectIDs) > 0 {
						// Filter after planning, never before: the plan must cover every
						// resource so ${resources.*} references resolve; only the emitted
						// changes are restricted to the selected resources.
						selected, err := configsync.ResolveResourceSelectors(ctx, &deployBundle.StateDB, selectIDs)
						if err != nil {
							return err
						}
						changes = configsync.FilterChanges(changes, selected)
					}

					fieldChanges, err := configsync.ResolveChangesFromSnapshot(ctx, b, changes, sourceSnapshot)
					if err != nil {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryResolveFailed
						return fmt.Errorf("failed to resolve field changes: %w", err)
					}

					if err := configsync.RestoreVariableReferences(ctx, b, fieldChanges, &stats.Restore); err != nil {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryResolveFailed
						return fmt.Errorf("failed to restore variable references: %w", err)
					}

					files, err := configsync.ApplyChangesToYAML(ctx, b, fieldChanges)
					if err != nil {
						stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryYamlApplyFailed
						return fmt.Errorf("failed to generate YAML files: %w", err)
					}
					stats.FilesChangedCount = int64(len(files))

					if save {
						if err := sourceSnapshot.Validate(); err != nil {
							stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategorySaveFailed
							return fmt.Errorf("failed to validate source files: %w", err)
						}
						if err := configsync.SaveFiles(ctx, b, files); err != nil {
							stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategorySaveFailed
							return fmt.Errorf("failed to save files: %w", err)
						}
						stats.FilesWrittenCount = int64(len(files))
					}

					var result []byte
					if root.OutputType(cmd) == flags.OutputJSON {
						diffOutput := &configsync.DiffOutput{
							Files:   files,
							Changes: changes,
						}
						result, err = json.MarshalIndent(diffOutput, "", "  ")
						if err != nil {
							stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryOutputFailed
							return fmt.Errorf("failed to marshal output: %w", err)
						}
					} else if root.OutputType(cmd) == flags.OutputText {
						result = []byte(configsync.FormatTextOutput(changes))
					}

					out := cmd.OutOrStdout()
					_, _ = out.Write(result)
					_, _ = out.Write([]byte{'\n'})
					return nil
				},
			})
			return err
		}

		// A source conflict invalidates both the plan and its YAML source mapping.
		// Reload the bundle and remote state together so the unattended sync
		// lands the edit on the newest source instead of returning stale.
		err := retrySourceChanges(baseContext, func() error {
			stats = configsync.Stats{Save: save}
			cmd.SetContext(baseContext)
			return runAttempt()
		})
		if err != nil {
			if stats.ErrorCategory == "" {
				stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryBundleLoadFailed
			}
			stats.ErrorMessage = telemetry.ScrubErrorMessage(err.Error())
		}
		return err
	}

	return cmd
}
