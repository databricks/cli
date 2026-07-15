package aircmd

// air run submits a training workload by converting the run YAML to a Databricks
// Asset Bundle and driving deploy + run in-process: convert train.yaml ->
// databricks.yml -> deploy -> run. It reuses the exportbundle.go converter, so the
// deployed ai_runtime_task is exactly what `air export-bundle` emits.
//
// Deploy and run call the bundle libraries directly (cmd/bundle/utils.ProcessBundle
// + Jobs.RunNow), not a child `databricks` process. Shelling out would risk an
// older `databricks` on PATH dropping the unknown ai_runtime_task field and
// deploying a task-less job.
//
// The deployed job is persistent and is NOT swept by the Jobs ephemeral-job GC, so
// runs of distinct experiments accumulate distinct jobs, drifting toward the
// per-workspace saved-jobs cap. A cleanup/TTL story is future work.

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	if idempotencyKey != "" || cfg.IdempotencyToken != nil {
		cmdio.LogString(ctx, "note: --idempotency-key is ignored on the DABs path (deploy+run is not a single idempotent call)")
	}

	bundleRoot, cleanup, err := writeBundleProject(ctx, cfg, configPath)
	if err != nil {
		return 0, "", err
	}
	defer cleanup()

	// Surface the generated artifact's path so the user can inspect what is
	// deployed. The dir is temporary; `air run --dry-run` shows the full contents.
	cmdio.LogString(ctx, "Generated bundle (temporary): "+filepath.Join(bundleRoot, "databricks.yml"))

	cmdio.LogString(ctx, "Deploying bundle (creates a persistent job)...")
	b, err := deployBundle(ctx, w, bundleRoot)
	if err != nil {
		return 0, "", fmt.Errorf("bundle deploy: %w", err)
	}

	// RunNow on the just-deployed job, returning as soon as the run is created.
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
// converter output plus the dev targets block. It touches no filesystem and does
// not deploy, so `air run --dry-run` can show the artifact; writeBundleProject
// reuses it so preview and real run never diverge.
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

// writeBundleProject renders databricks.yml (via renderBundle) plus command.sh and,
// for a code_source snapshot, the user's code tree, into a temp bundle root. Deploy
// uploads the whole root as an immutable-folder snapshot. Returns the root and a
// cleanup func.
func writeBundleProject(ctx context.Context, cfg *runConfig, configPath string) (string, func(), error) {
	body, err := renderBundle(cfg, configPath)
	if err != nil {
		return "", func() {}, err
	}
	bundleRoot, err := os.MkdirTemp("", "air-dabs-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(bundleRoot) }

	// Copy the code_source working tree into the bundle root first, so a stray
	// command.sh / databricks.yml in the user's tree can't shadow ours below.
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		if err := stageCodeSource(ctx, cfg.CodeSource.Snapshot, configPath, bundleRoot); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}

	if err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(body), 0o600); err != nil {
		cleanup()
		return "", func() {}, err
	}

	// command.sh: the entrypoint the task's command_path points at.
	if cfg.Command != nil {
		if err := os.WriteFile(filepath.Join(bundleRoot, bundleCommandScript), []byte(*cfg.Command), 0o600); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	return bundleRoot, cleanup, nil
}

// stageCodeSource copies the snapshot's working tree into dest so deploy uploads it
// as part of the immutable-folder snapshot. When include_paths is set, only those
// paths (files or directories, relative to root_path) are copied; otherwise the
// whole tree is copied. checkBundleConvertible has already rejected git and
// remote_volume snapshots, so this only handles the local working tree.
func stageCodeSource(ctx context.Context, snap *snapshotSourceConfig, configPath, dest string) error {
	root, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return err
	}
	if len(snap.IncludePaths) == 0 {
		return copyTree(root, dest)
	}
	for _, rel := range snap.IncludePaths {
		if err := copyTree(filepath.Join(root, rel), filepath.Join(dest, rel)); err != nil {
			return err
		}
	}
	return nil
}

// copyTree recursively copies the file or directory at src to dst, preserving the
// relative layout and file modes. Symlinks are skipped.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case info.Mode()&os.ModeSymlink != 0:
			return nil
		default:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return copyFile(p, target, info.Mode())
		}
	})
}

// copyFile copies a single regular file from src to dst with the given mode.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// bundleTargetsBlock is the minimal dev target appended to the converted YAML so
// `bundle deploy` runs non-interactively. dev mode gives per-user name prefix +
// isolation. The workspace host comes from the active CLI profile, so it's omitted
// here (the bundle resolves it from auth at deploy time).
func bundleTargetsBlock() string {
	return "\n# Appended by `air run` (host resolved from your CLI profile at deploy time):\ntargets:\n  " + dabsTarget + ":\n    mode: development\n    default: true\n"
}

// deployBundle deploys the bundle rooted at bundleRoot in-process and returns the
// configured *bundle.Bundle with resource IDs populated, so the caller can resolve
// the deployed job. It reuses ProcessBundle (load, validate, build, sync, deploy)
// via a synthetic carrier command (see newBundleCarrierCommand).
//
// SkipInitContext: air's command context already has logdiag initialized (root's
// PersistentPreRunE); re-initializing panics ("InitContext twice").
func deployBundle(ctx context.Context, w *databricks.WorkspaceClient, bundleRoot string) (*bundle.Bundle, error) {
	cmd := newBundleCarrierCommand(ctx, w, bundleRoot)

	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		FastValidate:    true,
		Build:           true,
		Deploy:          true,
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

// newBundleCarrierCommand builds a throwaway cobra command for ProcessBundle. air's
// own command carries none of the flags ProcessBundle reads (--var, --target,
// --profile, --output), so this declares them and seeds the context with air's auth
// profile and the bundle root. DATABRICKS_BUNDLE_ROOT (context-scoped via env.Set)
// points the loader at the generated databricks.yml instead of walking cwd; the
// direct engine avoids the Terraform provider/registry dependency. The command is
// never added to a tree or Execute()d.
func newBundleCarrierCommand(ctx context.Context, w *databricks.WorkspaceClient, bundleRoot string) *cobra.Command {
	cmd := &cobra.Command{Use: "air-bundle-deploy"}

	cmd.Flags().StringSlice("var", nil, "")
	cmd.Flags().StringP("target", "t", dabsTarget, "")
	cmd.Flags().StringP("profile", "p", "", "")
	outputFlag := flags.OutputText
	cmd.Flags().Var(&outputFlag, "output", "")

	// Forward air's active profile so the bundle authenticates the same way air did.
	if w.Config.Profile != "" {
		_ = cmd.Flags().Set("profile", w.Config.Profile)
	}

	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "direct")
	cmd.SetContext(ctx)
	return cmd
}

// runDeployedJob triggers a run of the just-deployed job and returns the new run_id
// without waiting for completion. It resolves the job by its bundle resource key
// (the experiment name) to get the job_id deploy populated, then calls Jobs.RunNow.
// AIR runs take no run parameters. RunNow is called directly rather than via the
// bundle runner's NoWait path, which returns nil instead of the run_id.
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

// isRunnableJob filters resources.Lookup to job resources.
func isRunnableJob(ref bundleresources.Reference) bool {
	_, ok := ref.Resource.(*resources.Job)
	return ok
}
