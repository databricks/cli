package aircmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/spf13/cobra"
)

// convert_to_dabs turns an AIR CLI run YAML into a Databricks Asset Bundle so a
// workload authored for `air run` can be deployed and managed as a bundle.
//
// The emitted bundle is schema-valid: `databricks bundle validate` accepts it,
// and `databricks bundle deploy` reproduces the same ai_runtime_task workload the
// CLI would submit. Two properties make that work without any manual upload step:
//
//   - The DABs `ai_runtime_task` is the SDK jobs.AiRuntimeTask (strict schema:
//     experiment + deployments[].{command_path,compute} + code_source_path).
//     Framework concerns (retries, timeout) live on the surrounding task, not in
//     ai_runtime_task — so they are emitted as task-level fields.
//   - `bundle deploy` uploads a *local* code_source_path / command_path itself
//     (collectLocalLibraries → aiRuntimeCodeSourcePattern), so we point those at
//     local files and let deploy do the uploading. The user runs one deploy.
//
// The command writes the code launch artifacts (command.sh, and — since the task
// proto carries no inline env/secrets/parameters — the env_vars.json /
// secret_env_vars.json / hyperparameters.yaml sidecars the server-side launcher
// reads) into the code_source directory, exactly mirroring the CLI's own launch
// layout so the deployed workload behaves identically to `air run`.

// dabsTargetName is the single default target emitted; a development-mode target
// is the conventional starting point for a generated bundle.
const dabsTargetName = "dev"

func newConvertToDabsCommand() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "convert-to-dabs <yaml_path>",
		Args:  root.ExactArgs(1),
		Short: "Convert an AIR run YAML into a Databricks Asset Bundle",
		Long: `Convert an AIR CLI run YAML config into a Databricks Asset Bundle (DABs).

The emitted bundle can be deployed with the standard DABs workflow:

    databricks bundle validate
    databricks bundle deploy

bundle deploy uploads the code source and launch scripts for you, so no manual
upload step is required. This command performs a purely local translation and
does not contact the workspace.`,
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the bundle into (default: an ephemeral temp directory keyed on the experiment name)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		yamlPath := args[0]

		cfg, err := loadRunConfig(yamlPath)
		if err != nil {
			return err
		}

		// Default to an ephemeral, deterministic scratch dir so repeated runs are
		// idempotent (no leftover bundle to clear) and the user's project stays
		// clean. It's cleared and rewritten each run. An explicit --output-dir is
		// treated as a location the user owns and is only created, not wiped.
		dir := outputDir
		ephemeral := dir == ""
		if ephemeral {
			dir = filepath.Join(os.TempDir(), "databricks-air-dabs", cfg.ExperimentName)
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to reset %s: %w", dir, err)
			}
		}

		written, err := writeBundle(ctx, cfg, yamlPath, dir, ephemeral)
		if err != nil {
			return err
		}

		printConvertNextSteps(ctx, dir, written, bundleResourceKey(cfg.ExperimentName))
		return nil
	}

	return cmd
}

// codeSourceTarballName is the bundle-local filename of the code archive. The
// ai_runtime_task code_source_path points at it; bundle deploy uploads the single
// file and the backend unpacks it on each node (code_source_path is an archive,
// not a directory — deploy's uploader only handles files, so a bare dir fails).
const codeSourceTarballName = "code_source.tar.gz"

// convertToDabs builds the DABs bundle value and the loose launch artifacts for a
// run config. It reads only what the run path's buildArtifacts reads, so the
// mapping is unit-testable in isolation. Returns the bundle root as a
// map[string]dyn.Value (ready for yamlsaver) and the loose artifacts (command.sh +
// env/secret/param sidecars) to write at the bundle root; the code_source itself
// is packaged into a tarball separately by writeBundle.
func convertToDabs(ctx context.Context, cfg *runConfig, configPath string) (map[string]dyn.Value, []uploadItem, error) {
	// idempotency_token is intentionally not mapped: it dedups a single runs/submit
	// call, which has no analogue for a persistent, repeatedly-runnable bundle job.
	//
	// usage_policy_name resolution is not ported (mirrors the submit path), and
	// docker images have no ai_runtime_task representation yet.
	if cfg.UsagePolicyName != nil {
		return nil, nil, errors.New("usage_policy_name is not yet supported by convert-to-dabs")
	}
	if cfg.Environment != nil && cfg.Environment.DockerImage != nil {
		return nil, nil, errors.New("environment.docker_image is not yet supported by convert-to-dabs")
	}
	// remote_volume points the code archive at a UC Volume. bundle deploy uploads
	// code_source_path to the bundle's artifact path, not an arbitrary Volume, so a
	// converted bundle can't honor it — reject rather than silently drop it.
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil && cfg.CodeSource.Snapshot.RemoteVolume != nil {
		return nil, nil, errors.New("code_source.snapshot.remote_volume is not supported by convert-to-dabs; bundle deploy manages the artifact upload location")
	}

	artifacts, err := buildArtifacts(cfg, configPath)
	if err != nil {
		return nil, nil, err
	}

	root := buildBundleValue(ctx, cfg)
	return root, artifacts, nil
}

// nv builds a dyn.Value at "position" n: yamlsaver orders a map's keys by their
// Location line, so assigning ascending n values fixes the emitted key order.
// It routes through dyn.V so nested Go maps/slices are converted recursively,
// then stamps the ordering location.
func nv(v any, n int) dyn.Value {
	return dyn.V(v).WithLocations([]dyn.Location{{Line: n}})
}

// localBundlePath renders a bundle-relative path with a leading "./" so bundle
// deploy classifies it as a local artifact to upload (see IsLibraryLocal). It is
// built from path.Join (forward slashes) so the emitted YAML is identical across
// operating systems.
func localBundlePath(p string) string {
	return "./" + p
}

// buildBundleValue assembles the bundle root as an ordered map[string]dyn.Value.
// The command script and the code tarball are bundle-local files (emitted
// "./"-prefixed) so `bundle deploy` uploads them.
func buildBundleValue(ctx context.Context, cfg *runConfig) map[string]dyn.Value {
	name := cfg.ExperimentName

	// ai_runtime_task: experiment + one deployment (command_path + compute) +
	// code_source_path. Only the fields the strict schema allows.
	//
	// Paths are emitted "./"-prefixed so bundle deploy treats them as LOCAL and
	// uploads them: libraries.IsLibraryLocal classifies a bare, extensionless path
	// as a PyPI package name (not a local file) and skips it, which would deploy a
	// path the backend can't resolve. The "./" prefix forces local classification.
	deployment := map[string]dyn.Value{
		"command_path": nv(localBundlePath(commandScriptName), 1),
		"compute": nv(map[string]dyn.Value{
			"accelerator_type":  nv(cfg.Compute.AcceleratorType, 1),
			"accelerator_count": nv(cfg.Compute.NumAccelerators, 2),
		}, 2),
	}

	aiRuntimeTask := map[string]dyn.Value{
		"experiment":  nv(name, 1),
		"deployments": nv([]dyn.Value{dyn.V(deployment)}, 2),
	}
	line := 3
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		aiRuntimeTask["code_source_path"] = nv(localBundlePath(codeSourceTarballName), line)
		line++
	}
	if cfg.MLflowRunName != nil {
		aiRuntimeTask["mlflow_run"] = nv(*cfg.MLflowRunName, line)
		line++
	}
	if cfg.MLflowExperimentDirectory != nil {
		aiRuntimeTask["mlflow_experiment_directory"] = nv(*cfg.MLflowExperimentDirectory, line)
	}

	// Task wrapper: task_key + framework fields (retries/timeout) + env key +
	// the ai_runtime_task. Framework fields live here per the schema, not inside
	// ai_runtime_task.
	task := map[string]dyn.Value{
		"task_key":        nv(name, 1),
		"environment_key": nv(aiRuntimeEnvironmentKey, 2),
	}
	taskLine := 3
	if cfg.MaxRetries != nil {
		task["max_retries"] = nv(*cfg.MaxRetries, taskLine)
		taskLine++
	}
	if cfg.TimeoutMinutes != nil {
		task["timeout_seconds"] = nv(cfg.timeoutSeconds(), taskLine)
		taskLine++
	}
	task["ai_runtime_task"] = nv(aiRuntimeTask, taskLine)

	// environments[]: version + inline dependencies (a requirements.yaml file
	// ships as a launch artifact instead, so only inline deps are inlined here).
	// Resolve the version through the same path `air run` uses (config, else env
	// override, else the default channel) so a config without an explicit version
	// still pins the version the workload would have run with — not an empty spec.
	runtimeVersion, _ := cfg.runtimeVersion()
	envSpec := map[string]dyn.Value{
		"environment_version": nv(dlRuntimeImage(ctx, runtimeVersion), 1),
	}
	if deps, ok := cfg.inlineDependencies(); ok && len(deps) > 0 {
		depVals := make([]dyn.Value, len(deps))
		for i, d := range deps {
			depVals[i] = dyn.V(d)
		}
		envSpec["dependencies"] = nv(depVals, 2)
	}
	environment := map[string]dyn.Value{
		"environment_key": nv(aiRuntimeEnvironmentKey, 1),
		"spec":            nv(envSpec, 2),
	}

	job := map[string]dyn.Value{
		"name":         nv(name, 1),
		"tasks":        nv([]dyn.Value{dyn.V(task)}, 2),
		"environments": nv([]dyn.Value{dyn.V(environment)}, 3),
	}
	// usage_policy_id is an already-resolved budget policy id, so it maps directly
	// to the job's budget_policy_id. (usage_policy_name needs server-side resolution
	// and is rejected in convertToDabs.)
	if cfg.UsagePolicyID != nil {
		job["budget_policy_id"] = nv(*cfg.UsagePolicyID, 4)
	}
	if perms := buildPermissionsValue(cfg.Permissions); perms.Kind() != dyn.KindInvalid {
		job["permissions"] = nv(perms.MustSequence(), 5)
	}

	rootValue := map[string]dyn.Value{
		"bundle": nv(map[string]dyn.Value{
			"name": nv(name, 1),
		}, 1),
		"targets": nv(map[string]dyn.Value{
			dabsTargetName: nv(map[string]dyn.Value{
				"mode":    nv("development", 1),
				"default": nv(true, 2),
			}, 1),
		}, 2),
		"resources": nv(map[string]dyn.Value{
			"jobs": nv(map[string]dyn.Value{
				bundleResourceKey(name): nv(job, 1),
			}, 1),
		}, 3),
	}
	return rootValue
}

// bundleResourceKey derives a job resource key from the experiment name. The key
// is emitted as an unquoted YAML map key, and DABs' strict loader rejects a key
// that parses as a non-string scalar (a purely numeric name like "12345" -> !!int,
// or "true"/"null"). experiment_name allows exactly [alphanumeric, -, _], so the
// only unsafe keys are those that YAML types as int/float/bool/null; prefix those
// with "job_" to force a string key. The human-facing name/experiment fields keep
// the original value (yamlsaver quotes them as scalar string values).
func bundleResourceKey(name string) string {
	switch strings.ToLower(name) {
	case "true", "false", "null":
		return "job_" + name
	}
	if _, err := strconv.ParseFloat(name, 64); err == nil {
		return "job_" + name
	}
	return name
}

// buildPermissionsValue maps run-config permissions to DABs job permissions
// (level → principal). Returns an invalid value when there are none.
func buildPermissionsValue(perms []permission) dyn.Value {
	if len(perms) == 0 {
		return dyn.InvalidValue
	}
	out := make([]dyn.Value, 0, len(perms))
	for _, p := range perms {
		m := map[string]dyn.Value{"level": nv(p.Level, 1)}
		switch {
		case p.UserName != nil:
			m["user_name"] = nv(*p.UserName, 2)
		case p.GroupName != nil:
			m["group_name"] = nv(*p.GroupName, 2)
		case p.ServicePrincipalName != nil:
			m["service_principal_name"] = nv(*p.ServicePrincipalName, 2)
		}
		out = append(out, dyn.V(m))
	}
	return dyn.V(out)
}

// writeBundle writes the bundle into dir: databricks.yml, the loose launch
// artifacts (command.sh + env/secret/param sidecars), and — when the config has a
// code_source — a code_source.tar.gz of the resolved root_path. All the referenced
// files are bundle-local so `bundle deploy` uploads them. force allows overwriting
// an existing dir (used for the ephemeral scratch dir, which is reset each run).
// Returns the relative paths written, for the next-steps message.
func writeBundle(ctx context.Context, cfg *runConfig, configPath, dir string, overwrite bool) ([]string, error) {
	root, artifacts, err := convertToDabs(ctx, cfg, configPath)
	if err != nil {
		return nil, err
	}

	// Restrict perms: the bundle carries env_vars.json (literal env var values) and
	// lands in a predictable location (the ephemeral /tmp path or a user dir), so
	// keep it owner-only rather than world-readable.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	// checkCollision guards every emitted file with one consistent message. On the
	// ephemeral path force is true (the dir was just wiped, so nothing collides);
	// for an explicit --output-dir we refuse to clobber files the user owns.
	checkCollision := func(name string) error {
		if overwrite {
			return nil
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return fmt.Errorf("%s already exists in %s; use a different --output-dir or remove it", name, dir)
		}
		return nil
	}
	writeFile := func(name string, data []byte) error {
		if err := checkCollision(name); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, name), data, 0o600)
	}

	if err := checkCollision("databricks.yml"); err != nil {
		return nil, err
	}
	bundlePath := filepath.Join(dir, "databricks.yml")
	// force=true: we've already run the collision check above, so SaveAsYAML's own
	// (stale, --force-referencing) guard must not fire.
	if err := yamlsaver.NewSaver().SaveAsYAML(root, bundlePath, true); err != nil {
		return nil, err
	}
	written := []string{"databricks.yml"}

	// Loose launch artifacts (command.sh + sidecars) at the bundle root.
	for _, item := range artifacts {
		if err := writeFile(item.name, item.data); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", item.name, err)
		}
		written = append(written, item.name)
	}

	// Code source: package the resolved root_path into a single archive next to the
	// bundle. code_source_path points at this tarball; the backend unpacks it.
	// Reuse the snapshot packager so a pinned git commit/branch is archived exactly
	// as `air run` would (git archive of the commit), not the dirty working tree.
	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		snap := cfg.CodeSource.Snapshot
		repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
		if err != nil {
			return nil, err
		}
		plan, err := resolveSnapshotPlan(ctx, newGitRepo(repoPath), snap.Git, snap.IncludePaths)
		if err != nil {
			return nil, err
		}
		if err := checkCollision(codeSourceTarballName); err != nil {
			return nil, err
		}
		// The output path must be absolute: the git-archive packager runs git with
		// `-C repoPath`, so a relative -o would resolve against the repo dir, not the
		// bundle dir (`git archive -o out/... ` -> repoPath/out/...). plain-tar uses
		// the CWD and would tolerate a relative path, but absolute is correct for both.
		tarball, err := filepath.Abs(filepath.Join(dir, codeSourceTarballName))
		if err != nil {
			return nil, err
		}
		if err := packageSnapshot(ctx, repoPath, plan, tarball); err != nil {
			return nil, err
		}
		written = append(written, codeSourceTarballName)
	}

	return written, nil
}

// printConvertNextSteps tells the user what was written and the exact deploy
// sequence, since the value of the command is a one-command deploy afterwards.
//
// It also spells out cleanup. This matters specifically for AIR users: `air run`
// submits an ephemeral runs/submit workload that the platform reaps on its own,
// whereas `bundle deploy` creates a *persistent* job that lingers until explicitly
// destroyed — DABs has no automatic GC. A user migrating from `air run` will not
// expect a durable resource, so we call out `bundle destroy` explicitly.
func printConvertNextSteps(ctx context.Context, dir string, written []string, jobKey string) {
	cmdio.LogString(ctx, fmt.Sprintf("Wrote a Databricks Asset Bundle to %s:", dir))
	for _, w := range written {
		cmdio.LogString(ctx, "  "+w)
	}
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "To deploy and run this workload as a bundle:")
	cmdio.LogString(ctx, "  1. cd "+dir)
	cmdio.LogString(ctx, "  2. databricks bundle validate")
	cmdio.LogString(ctx, "  3. databricks bundle deploy")
	cmdio.LogString(ctx, "  4. databricks bundle run "+jobKey)
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "bundle deploy uploads the code source and launch scripts automatically.")
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Unlike `air run` (which submits an ephemeral run), bundle deploy creates a")
	cmdio.LogString(ctx, "persistent job that is not garbage-collected. When you are done, remove the")
	cmdio.LogString(ctx, "job and its uploaded files with:")
	cmdio.LogString(ctx, "  databricks bundle destroy")
}
