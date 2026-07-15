package aircmd

// air run submits a training workload by converting the run YAML to a Databricks
// Asset Bundle and driving `bundle deploy` + `bundle run` in-process. This is the
// ONLY submit path: per the AIR-CLI/DABs agreement with Pieter Noordhuis (DECO
// requires AIR to integrate with DABs and submit through the public Jobs API, not
// an internal proxy), `air run` is a high-level wrapper of
// [convert train.yaml -> databricks.yml -> deploy -> run].
//
// The bundle is a persistent Jobs resource, so `air list` finds it via the indexed
// jobs-list path and re-runs anchor to a standing job. It reuses the exportbundle.go
// converter verbatim, so the deployed ai_runtime_task — including env vars via the
// common Jobs env-var API — is exactly what `air export-bundle` emits.
//
// IN-PROCESS, NOT SHELL-OUT: deploy and run call the bundle libraries directly
// (cmd/bundle/utils.ProcessBundle + bundle/run), the same entry points the
// `databricks bundle` commands use and the pattern the pipelines CLI established
// (cmd/pipelines). There is no child `databricks` process, so the deploy always
// uses this build's ai_runtime_task-aware bundle schema — an older `databricks` on
// PATH would drop the unknown ai_runtime_task field and deploy a task-less job.
//
// KNOWN TRADEOFF (design doc): the deployed job is PERSISTENT and is NOT swept by
// the Jobs ephemeral-job GC (JobsSoftDeletion only sweeps EPHEMERAL/WORKFLOW
// types). Runs of distinct experiments accumulate distinct jobs, drifting toward
// the per-workspace saved-jobs cap. Job reuse (a stable per-experiment bundle name,
// so re-runs update one job) mitigates it; a cleanup/TTL story is future work.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// dabsTarget is the bundle target air deploys/runs against. `dev` mode gives DABs
// dev-mode (per-user name prefix + isolation), which matches AIR's per-user runs
// and keeps one user's jobs from colliding with another's on the same experiment.
const dabsTarget = "dev"

// submitWorkload converts the run config to a bundle, deploys it (creating/updating
// a persistent job), then triggers a run. It returns the new run_id and its
// dashboard URL. The whole flow is in-process: deploy reuses the bundle deploy
// orchestration (state, sync, engine) via ProcessBundle; run calls Jobs.RunNow on
// the deployed job so the run_id is returned directly (fire-and-return, no wait).
func submitWorkload(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath, idempotencyKey string) (int64, string, error) {
	// The converter's gate rejects configs a bundle can't represent faithfully
	// (docker_image, usage_policy, git-pinned code_source, $CODE_SOURCE_PATH
	// commands). Fail fast with its actionable message before writing or deploying.
	if err := checkBundleConvertible(cfg); err != nil {
		return 0, "", err
	}

	// Idempotency does not map onto deploy+run (two calls, not one tokened submit).
	// Note it rather than silently honoring it; job identity (stable bundle name)
	// is the DABs-native dedupe mechanism.
	if idempotencyKey != "" || cfg.IdempotencyToken != nil {
		cmdio.LogString(ctx, "note: --idempotency-key is ignored on the DABs path (deploy+run is not a single idempotent call)")
	}

	bundleRoot, cleanup, err := writeBundleProject(cfg, configPath)
	if err != nil {
		return 0, "", err
	}
	defer cleanup()

	// Surface the generated artifact's path (transparency): the user can inspect
	// what we deploy on their behalf. The dir is temporary (auto-removed on exit).
	// For the full contents without deploying, use `air run --dry-run`.
	cmdio.LogString(ctx, "Generated bundle (temporary): "+filepath.Join(bundleRoot, "databricks.yml"))

	// deploy: creates/updates the persistent job. This is the step that adds
	// latency vs the old ephemeral submit (~2-3s warm, ~12s first-of-session).
	cmdio.LogString(ctx, "Deploying bundle (creates a persistent job)...")
	b, err := deployBundle(ctx, w, bundleRoot)
	if err != nil {
		return 0, "", fmt.Errorf("bundle deploy: %w", err)
	}

	// run: RunNow on the just-deployed job, returning as soon as the run is created
	// (no wait), matching the old submit path's fire-and-return.
	cmdio.LogString(ctx, "Triggering run...")
	runID, err := runDeployedJob(ctx, w, b, cfg.ExperimentName)
	if err != nil {
		return 0, "", fmt.Errorf("bundle run: %w", err)
	}

	cmdio.LogString(ctx, "note: this run created a PERSISTENT job (not auto-GC'd like an ephemeral run). Use `air list` to find it.")

	dashboardURL := strings.TrimRight(w.Config.Host, "/") + "/jobs/runs/" + strconv.FormatInt(runID, 10)
	return runID, dashboardURL, nil
}

// renderBundle produces the exact databricks.yml the run path deploys: the
// exportbundle.go converter output plus the dev targets block the run path appends.
// It touches no filesystem and does not deploy, so `air run --dry-run` can show the
// user the artifact we'd generate on their behalf (transparency), and
// writeBundleProject reuses it so preview and real run never diverge.
func renderBundle(cfg *runConfig, configPath string) (string, error) {
	if err := checkBundleConvertible(cfg); err != nil {
		return "", err
	}
	body, err := marshalBundle(convertToBundle(cfg), configPath)
	if err != nil {
		return "", err
	}
	// The converter's emitted YAML has no targets block (export-bundle leaves that
	// to the user); the run path needs one so deploy is non-interactive. Append a
	// dev target — the host is resolved from the CLI profile at deploy time.
	return string(body) + bundleTargetsBlock(), nil
}

// writeBundleProject renders databricks.yml (via renderBundle) plus the command.sh
// the bundle syncs, into a temp bundle root. Returns the root and a cleanup func.
//
// TODO(air): reuse the code-snapshot staging (snapshot.go) so the user's code tree
// lands in the bundle folder for `bundle sync` to upload. This writes command.sh
// from cfg.Command; wiring code_source snapshotting into the synced folder is the
// remaining piece before code_source runs are supported on this path.
func writeBundleProject(cfg *runConfig, configPath string) (string, func(), error) {
	body, err := renderBundle(cfg, configPath)
	if err != nil {
		return "", func() {}, err
	}
	bundleRoot, err := os.MkdirTemp("", "air-dabs-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(bundleRoot) }

	if err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(body), 0o600); err != nil {
		cleanup()
		return "", func() {}, err
	}

	// command.sh: the entrypoint the task's command_path points at (synced by deploy).
	if cfg.Command != nil {
		if err := os.WriteFile(filepath.Join(bundleRoot, bundleCommandScript), []byte(*cfg.Command), 0o600); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	return bundleRoot, cleanup, nil
}

// bundleTargetsBlock is the minimal dev target appended to the converted YAML so
// `bundle deploy` runs non-interactively. dev mode gives per-user name prefix +
// isolation. The workspace host comes from the active CLI profile, so it's omitted
// here (the bundle resolves it from auth at deploy time).
func bundleTargetsBlock() string {
	return "\n# Appended by `air run` (host resolved from your CLI profile at deploy time):\ntargets:\n  " + dabsTarget + ":\n    mode: development\n    default: true\n"
}

// deployBundle deploys the bundle rooted at bundleRoot in-process and returns the
// configured *bundle.Bundle (with resource IDs populated, so the caller can resolve
// the deployed job). It reuses cmd/bundle/utils.ProcessBundle — the exact
// orchestration `databricks bundle deploy` runs (load, validate, build, sync,
// deploy) — via a synthetic carrier command:
//
//   - air's own cobra command carries none of the flags ProcessBundle reads
//     (--var, --target, --profile, --output), so we build a throwaway command that
//     declares them and seed the context (logdiag + auth profile + bundle root).
//   - the bundle root is steered to bundleRoot with DATABRICKS_BUNDLE_ROOT on the
//     context (env.Set is context-scoped, not a real env var), so MustConfigureBundle
//     loads our generated databricks.yml instead of walking cwd.
//   - the direct engine (DATABRICKS_BUNDLE_ENGINE=direct) avoids the Terraform
//     provider/registry dependency; it is the GA default on new CLIs and set
//     explicitly here so the deploy is self-contained.
func deployBundle(ctx context.Context, w *databricks.WorkspaceClient, bundleRoot string) (*bundle.Bundle, error) {
	cmd := newBundleCarrierCommand(ctx, w, bundleRoot)

	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		FastValidate: true,
		Build:        true,
		Deploy:       true,
		// air's command context already has logdiag initialized (root's
		// PersistentPreRunE); re-initializing panics ("InitContext twice"). Same as
		// the pipelines CLI run path.
		SkipInitContext: true,
	})
	if err != nil {
		return nil, err
	}
	if b == nil || logdiag.HasError(cmd.Context()) {
		return nil, errors.New("bundle deploy failed")
	}
	return b, nil
}

// newBundleCarrierCommand builds the synthetic cobra command deployBundle hands to
// ProcessBundle. It declares the flags ProcessBundle/configureBundle read and seeds
// the context so the deploy runs with air's resolved auth against the generated
// bundle root. The command is never added to a tree and never Execute()d — it only
// carries flags + context into the bundle libraries.
func newBundleCarrierCommand(ctx context.Context, w *databricks.WorkspaceClient, bundleRoot string) *cobra.Command {
	cmd := &cobra.Command{Use: "air-bundle-deploy"}

	// Flags ProcessBundleRet (and the bundle root loader) read by name.
	cmd.Flags().StringSlice("var", nil, "")
	cmd.Flags().StringP("target", "t", dabsTarget, "")
	cmd.Flags().StringP("profile", "p", "", "")
	outputFlag := flags.OutputText
	cmd.Flags().Var(&outputFlag, "output", "")

	// Forward air's active profile so the bundle authenticates the same way air
	// did, instead of falling back to the default profile.
	if w.Config.Profile != "" {
		_ = cmd.Flags().Set("profile", w.Config.Profile)
	}

	// logdiag is how the bundle libraries report diagnostics; ProcessBundle asserts
	// it is set up (we pass SkipInitContext=false, so it initializes, but the auth
	// profile + bundle root must be seeded before that on the same context).
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "direct")
	cmd.SetContext(ctx)
	return cmd
}

// runDeployedJob triggers a run of the just-deployed job and returns the new run_id
// without waiting for completion. It resolves the job by its bundle resource key
// (the experiment name) to get the server-assigned job_id that deploy populated,
// then calls Jobs.RunNow. AIR runs take no run parameters, so RunNow needs only the
// job_id. (We call RunNow directly rather than the bundle runner's NoWait path,
// which returns nil instead of the run_id.)
func runDeployedJob(ctx context.Context, w *databricks.WorkspaceClient, b *bundle.Bundle, experimentName string) (int64, error) {
	ref, err := bundleresources.Lookup(b, experimentName, isRunnableJob)
	if err != nil {
		return 0, fmt.Errorf("locating the deployed job: %w", err)
	}
	job, ok := ref.Resource.(*resources.Job)
	if !ok {
		return 0, fmt.Errorf("deployed resource %q is not a job", experimentName)
	}
	jobID, err := strconv.ParseInt(job.ID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("deployed job has no valid id (deploy may have failed): %q", job.ID)
	}

	wait, err := w.Jobs.RunNow(ctx, jobs.RunNow{JobId: jobID})
	if err != nil {
		return 0, err
	}
	return wait.RunId, nil
}

// isRunnableJob filters resources.Lookup to job resources, so an experiment name
// that also matched another resource type can't be selected. AIR bundles only ever
// contain one job, so this is a safety guard rather than a disambiguator.
func isRunnableJob(ref bundleresources.Reference) bool {
	_, ok := ref.Resource.(*resources.Job)
	return ok
}
