package aircmd

// PROTOTYPE — Option D: `air run --via-dabs` submits the workload as a Databricks
// Asset Bundle (a persistent Jobs resource: deploy then run) instead of the
// ephemeral runs/submit path in runsubmit.go.
//
// Design (per the "AIR CLI + DABs" design meeting, Jul 14): `air run` becomes a
// high-level caller of [convert -> bundle deploy -> bundle run]. This reuses the
// exportbundle.go converter (train.yaml -> databricks.yml) verbatim, so the
// generated ai_runtime_task — including env vars via the common Jobs env-var API —
// is exactly what `air export-bundle` produces and what `air run` would submit.
//
// It is gated behind --via-dabs (and AIR_VIA_DABS=1); the ephemeral path stays the
// default and untouched. This is additive.
//
// PROTOTYPE SCOPE / shell-out choice: deploy+run are driven by shelling out to the
// `databricks bundle` CLI, which is the literal "wrapper around DABs" the meeting
// described and keeps this diff legible. A production version would instead call
// cmd/bundle/utils.ProcessBundle(...) in-process (see TODO(prototype) below) to
// avoid the child-process dependency. The behavior (persistent job, deploy
// latency, no ephemeral-job GC) is identical either way — the point being to
// exercise the real DABs path end to end.
//
// KNOWN GAP (called out in the design doc): the deployed job is PERSISTENT and is
// NOT swept by the Jobs ephemeral-job GC (JobsSoftDeletion only sweeps
// EPHEMERAL/WORKFLOW types). Every `air run --via-dabs` leaves a job behind. A real
// impl must choose a reuse/cleanup policy (stable per-experiment bundle so re-runs
// update one job, or `bundle destroy` after run). The prototype leaves the job and
// logs the tradeoff.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
)

// viaDABSEnv opts into the DABs submit path without the flag (flag wins).
const viaDABSEnv = "AIR_VIA_DABS"

// dabsTarget is the bundle target the prototype deploys/runs against. A real impl
// would let the user pick (or default per-workspace); "dev" gets DABs dev-mode
// (per-user name prefix + isolation), which suits AIR's per-user runs.
const dabsTarget = "dev"

// submitViaDABS is the Option-D analogue of submitWorkload (runsubmit.go), with the
// identical signature so run.go can pick either at the call site. It converts the
// run config to a bundle, writes the project (databricks.yml + command.sh + code),
// then deploys and runs it. Returns the run_id + dashboard URL.
func submitViaDABS(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, configPath, idempotencyKey string) (int64, string, error) {
	// The converter's gate rejects configs a bundle can't represent faithfully
	// (e.g. git-pinned code_source, $CODE_SOURCE_PATH commands). Fail fast with its
	// actionable message before we write or deploy anything.
	if err := checkBundleConvertible(cfg); err != nil {
		return 0, "", err
	}

	// Idempotency doesn't map onto deploy+run (two calls, not one tokened submit);
	// note rather than silently honor it. A real impl dedupes on bundle/job identity.
	if idempotencyKey != "" || cfg.IdempotencyToken != nil {
		cmdio.LogString(ctx, "note: --idempotency-key is ignored on the DABs path (deploy+run is not a single idempotent call)")
	}

	root, cleanup, err := writeBundleProject(cfg, configPath)
	if err != nil {
		return 0, "", err
	}
	defer cleanup()

	// Surface the generated artifact's path (transparency): the user can inspect
	// what we deploy on their behalf. The dir is temporary (auto-removed on exit) —
	// labeled as such so we don't imply a durable, user-managed location. For the
	// full contents without deploying, use `air run --dry-run --via-dabs`.
	cmdio.LogString(ctx, "Generated bundle (temporary): "+filepath.Join(root, "databricks.yml"))

	// deploy: creates/updates the persistent job. This is the step that adds
	// latency vs the ephemeral submit (measured ~2-3s warm, ~12s first-of-session).
	cmdio.LogString(ctx, "Deploying bundle (creates a persistent job)...")
	if err := runBundle(ctx, w, root, "deploy"); err != nil {
		return 0, "", fmt.Errorf("bundle deploy: %w", err)
	}

	// run: RunNow on the deployed job. --no-wait returns as soon as the run starts,
	// matching the ephemeral path's fire-and-return (we don't block on completion).
	cmdio.LogString(ctx, "Triggering run...")
	if err := runBundle(ctx, w, root, "run", cfg.ExperimentName, "--no-wait"); err != nil {
		return 0, "", fmt.Errorf("bundle run: %w", err)
	}

	cmdio.LogString(ctx, "note: this run created a PERSISTENT job (not auto-GC'd like an ephemeral run). Use `air list` to find it.")

	// TODO(prototype): parse run_id + dashboard URL from `databricks bundle run
	// --output json`. For now return 0 and the jobs page; the run is live either way.
	return 0, workspaceJobsURL(w), nil
}

// renderBundle produces the exact databricks.yml the run path would deploy: the
// exportbundle.go converter output plus the dev targets block the run path appends.
// It touches no filesystem and does not deploy, so `air run --dry-run --via-dabs`
// can show the user the artifact we'd generate on their behalf (transparency), and
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
// and code the bundle syncs, into a temp bundle root. Returns the root and a
// cleanup func.
//
// TODO(prototype): reuse the ephemeral path's snapshot/upload staging so the exact
// code tree + command.sh land in the bundle folder. This sketch writes command.sh
// from cfg.Command; wiring code_source snapshotting through the inherited
// bundle/config/mutator/aicode mutator is the remaining piece.
func writeBundleProject(cfg *runConfig, configPath string) (string, func(), error) {
	body, err := renderBundle(cfg, configPath)
	if err != nil {
		return "", func() {}, err
	}
	root, err := os.MkdirTemp("", "air-dabs-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }

	if err := os.WriteFile(filepath.Join(root, "databricks.yml"), []byte(body), 0o600); err != nil {
		cleanup()
		return "", func() {}, err
	}

	// command.sh: the entrypoint the task's command_path points at (synced by deploy).
	if cfg.Command != nil {
		if err := os.WriteFile(filepath.Join(root, bundleCommandScript), []byte(*cfg.Command), 0o600); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	return root, cleanup, nil
}

// bundleTargetsBlock is the minimal dev target appended to the converted YAML so
// `bundle deploy` runs non-interactively. dev mode gives per-user name prefix +
// isolation. The workspace host comes from the active CLI profile, so it's omitted
// here (the bundle resolves it from auth at deploy time).
func bundleTargetsBlock() string {
	// The comment supersedes the export-bundle header's "add a targets block"
	// note: on the run path we append this dev target for you (host resolved from
	// your CLI profile at deploy time), so `air run --via-dabs` needs no manual edit.
	return "\n# Appended by `air run --via-dabs` (host resolved from your CLI profile at deploy time):\ntargets:\n  " + dabsTarget + ":\n    mode: development\n    default: true\n"
}

// runBundle shells out to `databricks bundle <args...> -t <target>` in the bundle
// root. Uses the direct engine (no Terraform / no registry.terraform.io dep — GA
// default on new CLIs, set explicitly for older ones). Auth is inherited from the
// same profile/env the parent air command resolved.
//
// TODO(prototype): replace with an in-process cmd/bundle/utils.ProcessBundle call
// so there's no dependency on a `databricks` binary on PATH.
func runBundle(ctx context.Context, w *databricks.WorkspaceClient, root string, args ...string) error {
	full := append([]string{"bundle"}, args...)
	full = append(full, "-t", dabsTarget)
	cmd := exec.CommandContext(ctx, "databricks", full...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"DATABRICKS_BUNDLE_ENGINE=direct",
		"DATABRICKS_HOST="+w.Config.Host,
	)
	cmd.Stdout = os.Stderr // keep our stdout clean for the JSON envelope
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func workspaceJobsURL(w *databricks.WorkspaceClient) string {
	host := w.Config.Host
	for len(host) > 0 && host[len(host)-1] == '/' {
		host = host[:len(host)-1]
	}
	return host + "/jobs"
}

// resolveViaDABS decides whether to take the DABs submit path: the --via-dabs flag
// or AIR_VIA_DABS=1. Kept here so run.go's wiring stays a one-liner.
func resolveViaDABS(ctx context.Context, flag bool) bool {
	return flag || env.Get(ctx, viaDABSEnv) == "1"
}
