package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
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
//   - code_source_path points at a *local directory* inside the bundle. The
//     deploy-time aicode.PackageAndUpload mutator (bundle/config/mutator/aicode)
//     owns the snapshotting: at `bundle deploy` it packages that directory into a
//     content-addressed tarball, uploads it, and rewrites code_source_path to the
//     remote archive. convert-to-dabs' job is only to *stage the source bytes* as
//     that directory — copying the working tree, or materializing a pinned git
//     commit into it — never to build the tarball itself.
//
// command.sh (and — since the task proto carries no inline env/secrets/parameters —
// the env_vars.json / secret_env_vars.json / hyperparameters.yaml sidecars the
// server-side launcher reads) are written at the bundle root and uploaded by
// deploy, mirroring the CLI's own launch layout. requirements.yaml is NOT emitted:
// the aicode.SynthesizeRequirements mutator derives it from the job's
// environments[] spec at deploy time, so convert folds the dependency set into that
// spec instead.

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

	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the bundle into (default: a <experiment>-bundle folder next to the input YAML). Accepts an absolute or relative path.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		yamlPath := args[0]

		cfg, err := loadRunConfig(yamlPath)
		if err != nil {
			return err
		}

		// Default the bundle next to the input YAML (a <experiment>-bundle subfolder
		// so the generated files don't scatter across the user's project dir). An
		// explicit --output-dir — absolute or relative — overrides. We never write to
		// a temp dir: the bundle is the user's artifact to keep, deploy, and manage.
		dir := outputDir
		if dir == "" {
			dir = filepath.Join(filepath.Dir(yamlPath), cfg.ExperimentName+"-bundle")
		}

		written, err := writeBundle(ctx, cfg, yamlPath, dir)
		if err != nil {
			return err
		}

		printConvertNextSteps(ctx, dir, written, bundleResourceKey(cfg.ExperimentName), conversionNotes(cfg))
		return nil
	}

	return cmd
}

// codeSourceDirName is the bundle-local directory the code_source is staged into.
// ai_runtime_task.code_source_path points at it; the deploy-time aicode mutator
// packages the directory into a tarball and uploads it (see the file header). A
// fixed name (rather than the source's basename) keeps the emitted databricks.yml
// deterministic regardless of where the user's code lives.
const codeSourceDirName = "code_source"

// convertToDabs builds the DABs bundle value and the loose launch artifacts for a
// run config. It reads only what the run path's buildArtifacts reads, so the
// mapping is unit-testable in isolation. Returns the bundle root as a
// map[string]dyn.Value (ready for yamlsaver) and the loose artifacts (command.sh +
// env/secret/param sidecars) to write at the bundle root; the code_source directory
// is materialized separately by writeBundle.
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
	// Drop requirements.yaml: the deploy-time aicode.SynthesizeRequirements mutator
	// regenerates it from the job's environments[] spec (which convert populates with
	// the same dependency set), so emitting it here would be redundant and could drift.
	artifacts = slices.DeleteFunc(artifacts, func(it uploadItem) bool {
		return it.name == requirementsName
	})

	root := buildBundleValue(ctx, cfg, configPath)
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
// command.sh and the code_source directory are bundle-local (emitted "./"-prefixed)
// so `bundle deploy` uploads/packages them.
func buildBundleValue(ctx context.Context, cfg *runConfig, configPath string) map[string]dyn.Value {
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
		// A local directory (not a "./"-prefixed file): the aicode mutator recognizes
		// a directory code_source_path, packages it, and rewrites this field at deploy.
		aiRuntimeTask["code_source_path"] = nv(localBundlePath(codeSourceDirName), line)
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

	// environments[]: version + the dependency set. The aicode.SynthesizeRequirements
	// mutator regenerates requirements.yaml from this spec at deploy time, so the full
	// dependency set (whether authored inline or in a requirements file) must live
	// here — convert emits no requirements.yaml of its own. Resolve the version
	// through the same path `air run` uses (config, else env override, else the
	// default channel) so a config without an explicit version still pins the version
	// the workload would have run with — not an empty spec.
	envVersion, deps := bundleEnvironmentDeps(ctx, cfg, configPath)
	envSpec := map[string]dyn.Value{
		"environment_version": nv(envVersion, 1),
	}
	if len(deps) > 0 {
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

// bundleEnvironmentDeps resolves the runtime version and the flattened dependency
// list to emit in the bundle's environments[] spec. The aicode mutator synthesizes
// requirements.yaml from that spec at deploy, so the whole set must be here —
// whether the user authored dependencies inline or pointed at a requirements file.
// A requirements file is read and its non-comment, non-blank lines are inlined; the
// version, when the file carries one, wins over the config/default version. Any read
// error is best-effort ignored (writeBundle/buildArtifacts surface real problems);
// convert falls back to inline deps so the spec is never silently wrong.
func bundleEnvironmentDeps(ctx context.Context, cfg *runConfig, configPath string) (version string, deps []string) {
	cfgVersion, _ := cfg.runtimeVersion()
	version = dlRuntimeImage(ctx, cfgVersion)

	if inline, ok := cfg.inlineDependencies(); ok {
		return version, inline
	}

	reqPath, ok := cfg.requirementsFile()
	if !ok {
		return version, nil
	}
	if !filepath.IsAbs(reqPath) {
		reqPath = filepath.Join(filepath.Dir(configPath), reqPath)
	}
	data, err := os.ReadFile(reqPath)
	if err != nil {
		return version, nil
	}
	// The requirements file is the same requirements.yaml shape the run path reads
	// (version + dependencies), so parse it as such and inline the dependency lines.
	var doc requirementsDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return version, nil
	}
	if doc.Version != "" {
		version = dlRuntimeImage(ctx, doc.Version)
	}
	return version, doc.Dependencies
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
// code_source — a code_source/ directory holding the staged source tree. All the
// referenced files are bundle-local; `bundle deploy` uploads the launch artifacts
// and the aicode mutator packages+uploads the code_source directory. It refuses to
// overwrite existing files so a re-run can't silently clobber a bundle the user has
// edited. Returns the relative paths written, for the next-steps message.
func writeBundle(ctx context.Context, cfg *runConfig, configPath, dir string) ([]string, error) {
	root, artifacts, err := convertToDabs(ctx, cfg, configPath)
	if err != nil {
		return nil, err
	}

	// Restrict perms: the bundle carries env_vars.json (literal env var values), so
	// keep the dir owner-only rather than world-readable.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	// Refuse to clobber an existing file, with one consistent message. A user
	// re-running convert into the same dir gets a clear error rather than a silent
	// overwrite of edits they may have made.
	checkCollision := func(name string) error {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return fmt.Errorf("%s already exists in %s; use --output-dir or remove it", name, dir)
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
	// guard (which references a --force flag this command doesn't have) can't fire.
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

	// Code source: stage the resolved root_path into the bundle's code_source/ dir.
	// The aicode mutator packages+uploads it at deploy; convert only lays down the
	// bytes. A pinned git commit/branch is materialized from the archived commit (not
	// the dirty working tree), matching what `air run` would submit; otherwise the
	// working tree is copied, honoring .gitignore just like the run path's plain-tar.
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
		if err := checkCollision(codeSourceDirName); err != nil {
			return nil, err
		}
		if err := materializeCodeSource(ctx, repoPath, plan, filepath.Join(dir, codeSourceDirName)); err != nil {
			return nil, err
		}
		written = append(written, codeSourceDirName+"/")
	}

	return written, nil
}

// materializeCodeSource stages the snapshot described by plan into destDir (the
// bundle's code_source/ directory). It packages the source with the existing
// snapshot packagers — git archive for a pinned commit, gitignore-aware plain tar
// for a working tree — into a temporary tarball, then extracts it and moves its
// single top-level directory into destDir. Going through the packagers (rather than
// a raw copy) reuses their exact commit-pin and .gitignore/include-path handling, so
// the staged tree matches what `air run` would submit; the deploy-time aicode mutator
// then re-packages destDir into the uploaded, content-addressed archive.
func materializeCodeSource(ctx context.Context, repoPath string, plan snapshotPlan, destDir string) error {
	staging, err := os.MkdirTemp("", "air-convert-code-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	// Absolute tarball path: the git-archive packager runs git with `-C repoPath`, so
	// a relative -o would resolve against the repo dir. Both packagers accept absolute.
	tarball, err := filepath.Abs(filepath.Join(staging, "code_source.tar.gz"))
	if err != nil {
		return err
	}
	// git archive for a pinned commit (deterministic, ignores the dirty tree),
	// gitignore-aware plain tar for a working tree — the same split the run path uses.
	dirName := filepath.Base(repoPath)
	if plan.mode == modeGitArchive {
		err = createGitArchiveSnapshot(ctx, newGitRepo(repoPath), plan.commitSHA, tarball, dirName, plan.includePaths)
	} else {
		err = createPlainTarball(ctx, repoPath, tarball, plan.includePaths)
	}
	if err != nil {
		return err
	}

	// The archive's single top-level entry is the source directory's basename (both
	// packagers preserve it). Extract into a scratch dir, then move that top-level
	// directory to the fixed destDir name so the emitted code_source_path is stable.
	extractDir := filepath.Join(staging, "extract")
	if err := os.MkdirAll(extractDir, 0o700); err != nil {
		return err
	}
	if err := extractTarball(ctx, tarball, extractDir); err != nil {
		return err
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return fmt.Errorf("unexpected code_source archive layout: expected a single top-level directory, got %d entries", len(entries))
	}
	if err := os.Rename(filepath.Join(extractDir, entries[0].Name()), destDir); err != nil {
		return err
	}
	return nil
}

// extractTarball unpacks a gzipped tarball into destDir via `tar`, mirroring the
// snapshot packagers' reliance on the system tar (so symlink/permission handling is
// identical to what the run path produced when it built the archive).
func extractTarball(ctx context.Context, tarball, destDir string) error {
	cmd := exec.CommandContext(ctx, "tar", "-xzf", tarball, "-C", destDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("failed to extract code_source archive: %w: %s", err, msg)
		}
		return fmt.Errorf("failed to extract code_source archive: %w", err)
	}
	return nil
}

// printConvertNextSteps tells the user what was written and the exact deploy
// sequence, since the value of the command is a one-command deploy afterwards.
//
// It also spells out cleanup. This matters specifically for AIR users: `air run`
// submits an ephemeral runs/submit workload that the platform reaps on its own,
// whereas `bundle deploy` creates a *persistent* job that lingers until explicitly
// destroyed — DABs has no automatic GC. A user migrating from `air run` will not
// expect a durable resource, so we call out `bundle destroy` explicitly.
func printConvertNextSteps(ctx context.Context, dir string, written []string, jobKey string, notes []string) {
	cmdio.LogString(ctx, fmt.Sprintf("Wrote a Databricks Asset Bundle to %s:", dir))
	for _, w := range written {
		cmdio.LogString(ctx, "  "+w)
	}

	// Notes surface anything the user should know: fields we transformed or dropped,
	// and values they may need to fill in. Migrating users otherwise can't tell what
	// silently changed between their run YAML and the bundle.
	if len(notes) > 0 {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Notes:")
		for _, n := range notes {
			cmdio.LogString(ctx, "  - "+n)
		}
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

// conversionNotes lists what the conversion transformed, staged out-of-band, or
// could not represent natively — so a user migrating from `air run` can see what
// changed between their run YAML and the emitted bundle, and what they may still
// need to fill in. Best-effort: git resolution errors are ignored here (writeBundle
// surfaces them), so a note is only emitted when the state is unambiguous.
func conversionNotes(cfg *runConfig) []string {
	var notes []string

	if cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil {
		snap := cfg.CodeSource.Snapshot
		if snap.Git != nil {
			notes = append(notes, "code_source.git was pinned in the run YAML; the pinned commit was materialized into "+
				"the code_source/ directory. Re-run convert-to-dabs to re-materialize a different revision.")
		}
		notes = append(notes, "code_source was staged into the code_source/ directory; bundle deploy packages and uploads it. "+
			"Re-run convert-to-dabs to re-stage after code changes.")
	}

	// env vars / secrets have no native ai_runtime_task field yet, so they ride as
	// sidecar files the server-side launcher reads (same as `air run`).
	if len(cfg.EnvVariables) > 0 {
		notes = append(notes, "env_variables were written to env_vars.json (no native bundle field yet); they are uploaded with the code and applied at run time.")
	}
	if len(cfg.Secrets) > 0 {
		notes = append(notes, "secrets were written to secret_env_vars.json (no native bundle field yet); they are resolved at run time.")
	}
	if len(cfg.Parameters) > 0 {
		notes = append(notes, "parameters were written to hyperparameters.yaml; they are not a native bundle field and are passed through to the workload.")
	}

	return notes
}
