package aircmd

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// speculativeSubmitEnv turns on submitting before the launch files are uploaded.
// It is off by default and only applies when the user name is cached for the credential.
const speculativeSubmitEnv = "DATABRICKS_AIR_CLI_SPECULATIVE_SUBMIT"

func speculativeSubmitEnabled(ctx context.Context) bool {
	switch strings.ToLower(env.Get(ctx, speculativeSubmitEnv)) {
	case "1", "true":
		return true
	}
	return false
}

// speculationEligible reports whether a run may be submitted before its files
// exist. A code snapshot is staged as a tarball and a usage policy name needs a
// lookup, so those runs keep the normal order.
func speculationEligible(cfg *runConfig) bool {
	return (cfg.CodeSource == nil || cfg.CodeSource.Snapshot == nil) && cfg.UsagePolicyName == nil
}

// speculativeSubmit submits the run while the launch files are still uploading,
// under the home cached for this credential. It takes that path only on a cache
// hit, so no SCIM call precedes it. Writes are then under the cached name, and
// the cache is authoritative for the credential until it expires.
//
// The server checks that command_path exists when it starts the run, about 1.6 s
// after Jobs accepts the submit. Staging is expected to finish well inside that
// window. If validation or staging fails, the run is cancelled. handled is false
// when the launch must take the normal path; the caller then makes a new token,
// because the token may belong to a run that was not used.
func speculativeSubmit(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, items []uploadItem, token string, showProgress bool) (runID int64, runURL string, handled bool, err error) {
	if !speculativeSubmitEnabled(ctx) || !speculationEligible(cfg) {
		return 0, "", false, nil
	}
	cachedName, ok := cachedUserName(ctx, w)
	if !ok || cachedName == "" {
		return 0, "", false, nil
	}
	base := "/Workspace/Users/" + cachedName

	// A remembered dir is reused only under the same home.
	launchIndex := newLaunchIndex(ctx)
	key := launchIndexKey{Host: w.Config.Host, Content: launchContentHash(items)}
	funcDir := launchDirFor(base, cfg)
	recorded := false
	if entry, found := cache.Get[launchIndexEntry](ctx, launchIndex, key); found && entry.Base == base {
		funcDir = entry.Dir
		recorded = true
	}

	// A missing experiment directory is created by the normal path, after Me. The
	// speculative path only checks it, with the helpers the normal path uses.
	if dir := experimentDirectoryOf(cfg); dir != "" {
		status := <-startExperimentDirectoryRead(ctx, w, dir)
		if errors.Is(status.err, apierr.ErrNotFound) {
			return 0, "", false, nil
		}
		if err := ensureExperimentDirectory(ctx, w, dir, status); err != nil {
			return 0, "", true, err
		}
	}

	commandPath := path.Join(funcDir, commandScriptName)
	containers := submittedContainers(cfg, funcDir)
	deps, _ := cfg.inlineDependencies()
	runtimeVersion, _ := cfg.runtimeVersion()
	payload := buildSubmitPayload(cfg, commandPath, dlRuntimeImage(ctx, runtimeVersion), usagePolicyIDLiteral(cfg), snapshotResult{}, deps)
	payload.IdempotencyToken = token
	poolID, priorityClass := computePoolFields(cfg)
	fc, err := filer.NewWorkspaceFilesClient(w, funcDir)
	if err != nil {
		return 0, "", true, err
	}

	type submitOutcome struct {
		runID int64
		err   error
	}
	submitted := make(chan submitOutcome, 1)
	go func() {
		id, err := submitRun(ctx, w, payload, poolID, priorityClass, cfg.unityCatalogImagePath(), containers)
		submitted <- submitOutcome{id, err}
	}()

	group, groupCtx := errgroup.WithContext(ctx)
	var unavailable, staged bool
	var workspaceID int64
	group.Go(func() error {
		var validateErr error
		unavailable, validateErr = preflightValidate(groupCtx, w, cfg, commandPath, containers, token)
		return validateErr
	})
	group.Go(func() error {
		return withSpinner(ctx, showProgress, "Staging run artifacts…", func() error {
			staged = recorded && launchFilesStaged(groupCtx, w, funcDir, items)
			if staged {
				return nil
			}
			_, stageErr := stageRunArtifacts(groupCtx, fc, items, nil)
			return stageErr
		})
	})
	group.Go(func() error {
		workspaceID = workspaceIDForRunLink(groupCtx, w)
		return nil
	})

	// The submit is joined on every path, so a run that was created is used or cancelled.
	groupErr := group.Wait()
	out := <-submitted
	if groupErr != nil {
		if out.err == nil {
			cancelSubmittedRun(ctx, w, out.runID)
		}
		return 0, "", true, groupErr
	}
	if out.err != nil {
		return 0, "", true, out.err
	}
	if !staged {
		cache.Put(ctx, launchIndex, key, launchIndexEntry{Base: base, Dir: funcDir})
	}
	if unavailable && cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, "Warning: server-side config validation was unavailable; continuing with submission.")
	}
	return out.runID, dashboardURL(w.Config.Host, out.runID, workspaceID), true, nil
}

// cancelSubmittedRun cancels a run the launch will not use. The cancel ignores the
// caller's context, so an interrupt cannot leave the run running. A failed cancel is
// reported with the run id.
func cancelSubmittedRun(ctx context.Context, w *databricks.WorkspaceClient, runID int64) {
	if _, err := w.Jobs.CancelRun(context.WithoutCancel(ctx), jobs.CancelRun{RunId: runID}); err != nil && cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, "Warning: could not cancel run "+strconv.FormatInt(runID, 10)+" after a failed launch: "+err.Error())
	}
}

func experimentDirectoryOf(cfg *runConfig) string {
	if cfg.MLflowExperimentDirectory == nil {
		return ""
	}
	return *cfg.MLflowExperimentDirectory
}

// usagePolicyIDLiteral returns the usage policy id the config sets literally, or "".
func usagePolicyIDLiteral(cfg *runConfig) string {
	if cfg.UsagePolicyID != nil {
		return strings.TrimSpace(*cfg.UsagePolicyID)
	}
	return ""
}

// computePoolFields returns the accelerator pool id and priority class, empty when unset.
func computePoolFields(cfg *runConfig) (poolID, priorityClass string) {
	if cfg.Compute.AcceleratorPoolID != nil {
		poolID = *cfg.Compute.AcceleratorPoolID
	}
	if cfg.Compute.PriorityClass != nil {
		priorityClass = *cfg.Compute.PriorityClass
	}
	return poolID, priorityClass
}
