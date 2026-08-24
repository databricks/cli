package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/configsync"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	envlib "github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

func newConfigRemoteSyncCommand() *cobra.Command {
	var save bool
	var selectIDs []string
	var statePath string

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
  databricks bundle config-remote-sync --select-ids jobs:123456789 --save

  # Read the deployment state from an explicit workspace location
  databricks bundle config-remote-sync --state-path /Workspace/Shared/.bundle/my_bundle/dev/state`,
		Hidden: true, // Used by DABs in the Workspace only
	}

	cmd.Flags().BoolVar(&save, "save", false, "Write updated config files to disk")
	cmd.Flags().StringSliceVar(&selectIDs, "select-ids", nil, "Sync only the given resources, each as <type>:<id> (e.g. jobs:123456789). Can be repeated or comma-separated.")
	cmd.Flags().StringVar(&statePath, "state-path", "", "Absolute workspace path of the deployment state folder to read, overriding workspace.state_path. Use when the state does not live under the path this command resolves by default, e.g. because the bundle was deployed by another user.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS == "windows" {
			return errors.New("config-remote-sync command is not supported on Windows")
		}

		if err := validateStatePathFlag(statePath); err != nil {
			return err
		}

		// Scope the local state cache to the state folder being read. The cache is
		// otherwise keyed on bundle root and target alone, so another deployment's
		// state would land in the default location and be picked up as this bundle's
		// own by later commands, including deploy.
		if statePath != "" {
			cmd.SetContext(envlib.Set(cmd.Context(), env.TempDirVariable, stateCacheDir(statePath)))
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

		_, _, err := utils.ProcessBundleRet(cmd, utils.ProcessOptions{
			ReadState:  true,
			Build:      true,
			AlwaysPull: true,
			InitFunc: func(b *bundle.Bundle) {
				b.SkipLocalFileValidation = true
			},
			// Applied after phases.Initialize so the dev-mode uniqueness check does not
			// reject a state folder this command only reads: it belongs to the
			// deployment being synced, not to whoever runs the command.
			PostInitFunc: func(ctx context.Context, b *bundle.Bundle) error {
				if statePath == "" {
					return nil
				}
				// Assigned through a mutator so the value also lands in the dynamic
				// config tree. Later phases convert dyn->typed on entry, which would
				// otherwise restore the default before the state snapshot is read.
				bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) {
					b.Config.Workspace.StatePath = normalizeStatePath(statePath)
				})
				return nil
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

				changes, err := configsync.ExtractChanges(ctx, b, plan, stateDesc.Engine)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryDetectChangesFailed
					return fmt.Errorf("failed to extract changes: %w", err)
				}
				stats.CollectChangeStats(ctx, changes)

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
					changes = configsync.FilterChanges(changes, selected)
				}

				// Loaded once and shared: ResolveChanges uses it to skip changes whose
				// parent is a variable reference, RestoreVariableReferences to restore refs.
				preResolved := configsync.LoadPreResolvedConfig(ctx, b)

				fieldChanges, skipped, err := configsync.ResolveChanges(ctx, b, changes, preResolved)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryResolveFailed
					return fmt.Errorf("failed to resolve field changes: %w", err)
				}
				stats.SkippedChangesCount = int64(skipped)

				if err := configsync.RestoreVariableReferences(ctx, b, fieldChanges, preResolved, &stats.Restore); err != nil {
					log.Warnf(ctx, "variable restoration skipped: %v", err)
				}

				files, err := configsync.ApplyChangesToYAML(ctx, b, fieldChanges)
				if err != nil {
					stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryYamlApplyFailed
					return fmt.Errorf("failed to generate YAML files: %w", err)
				}
				stats.FilesChangedCount = int64(len(files))

				if save {
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

// validateStatePathFlag rejects a --state-path this command cannot resolve to one
// deployment's state folder. "~" is refused rather than expanded: it resolves to the
// home of whoever runs the command, which is the resolution this flag exists to override.
func validateStatePathFlag(statePath string) error {
	switch {
	case statePath == "":
		return nil
	case strings.HasPrefix(statePath, "~"):
		return fmt.Errorf("--state-path must be an absolute workspace path, got %q: pass the path of the deployment to sync, not one relative to the current user's home", statePath)
	case !strings.HasPrefix(statePath, "/"):
		return fmt.Errorf("--state-path must be an absolute workspace path, got %q", statePath)
	case strings.HasPrefix(statePath, "/Volumes/"):
		return fmt.Errorf("--state-path does not support Volumes paths, got %q", statePath)
	}
	return nil
}

// stateCacheDir returns the local cache directory to use for an overridden state path.
// It is keyed on the state folder so state read from another deployment can never occupy
// the location this bundle caches its own state in, and stays outside the bundle tree so
// a plain deploy in the same directory cannot pick it up. Deterministic, so repeat runs
// against the same state folder still reuse the cache.
func stateCacheDir(statePath string) string {
	sum := sha256.Sum256([]byte(normalizeStatePath(statePath)))
	return filepath.Join(os.TempDir(), "databricks-bundle-state", hex.EncodeToString(sum[:8]))
}

// normalizeStatePath applies the /Workspace prefixing that PrependWorkspacePrefix gives
// a configured state_path. That mutator has already run by the time the flag is applied.
func normalizeStatePath(statePath string) string {
	if strings.HasPrefix(statePath, "/Workspace/") {
		return statePath
	}
	return "/Workspace" + statePath
}
