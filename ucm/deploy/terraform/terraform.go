package terraform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// tfRunner is the minimal terraform-exec surface used by the wrapper.
// Having an explicit interface keeps tests independent of a real terraform
// binary — the production impl is *tfexec.Terraform; tests inject a fake.
type tfRunner interface {
	Init(ctx context.Context, opts ...tfexec.InitOption) error
	Plan(ctx context.Context, opts ...tfexec.PlanOption) (bool, error)
	ShowPlanFile(ctx context.Context, planPath string, opts ...tfexec.ShowOption) (*tfjson.Plan, error)
	Apply(ctx context.Context, opts ...tfexec.ApplyOption) error
	Destroy(ctx context.Context, opts ...tfexec.DestroyOption) error
	Import(ctx context.Context, address, id string, opts ...tfexec.ImportOption) error
	StateRm(ctx context.Context, address string, opts ...tfexec.StateRmCmdOption) error
	SetEnv(env map[string]string) error
}

// tfRunnerFactory builds a tfRunner given a working dir and exec path. Split
// so tests can swap out the real binary for a fake.
type tfRunnerFactory func(workingDir, execPath string) (tfRunner, error)

// defaultRunnerFactory is the production factory — returns a real
// *tfexec.Terraform bound to the given workingDir+execPath.
func defaultRunnerFactory(workingDir, execPath string) (tfRunner, error) {
	return tfexec.NewTerraform(workingDir, execPath)
}

// lockerFactory constructs a Locker scoped to the target-specific state
// directory. Overridable by tests (so Apply/Destroy can hand a contending
// Locker a shared in-memory filer).
type lockerFactory func(ctx context.Context, u *ucm.Ucm, user string) (*lock.Locker, error)

// Terraform is the top-level terraform-engine wrapper. One instance per
// ucm.Ucm; calls to Render/Init/Plan/Apply/Destroy drive the underlying
// tfRunner in sequence. The Terraform value itself is safe to reuse across
// calls — Init is idempotent.
type Terraform struct {
	// ExecPath is the absolute path of the terraform binary.
	ExecPath string
	// WorkingDir is where main.tf.json, the plan artefact, and the state
	// backend config live.
	WorkingDir string
	// Env is the environment map passed to terraform-exec. Populated by New
	// from buildEnv — auth.Env(authCfg) + inheritEnvVars + temp/proxy
	// passthrough + DATABRICKS_CLI_PATH absolute-ization.
	Env map[string]string

	runner         tfRunner
	runnerFactory  tfRunnerFactory
	installer      Installer
	lockerFactory  lockerFactory
	user           string
	lockTargetDir  string
	initialized    bool
	lastPlanPath   string
	lastPlanExists bool
}

// New wires up a Terraform for the given ucm. It resolves (and if necessary
// downloads via hc-install) the terraform binary, computes the working
// directory, and assembles the env-var map used for auth and process
// plumbing. The caller is expected to have run SelectTarget first.
func New(ctx context.Context, u *ucm.Ucm) (*Terraform, error) {
	workingDir, err := WorkingDir(u)
	if err != nil {
		return nil, err
	}

	execPath, err := resolveExecPath(ctx, workingDir, hcInstaller{})
	if err != nil {
		return nil, err
	}

	authCfg, err := resolveAuthConfig(u)
	if err != nil {
		return nil, err
	}
	envMap, err := buildEnv(ctx, u, authCfg)
	if err != nil {
		return nil, err
	}

	user, lockDir := lockIdentity(ctx, u)

	return &Terraform{
		ExecPath:      execPath,
		WorkingDir:    workingDir,
		Env:           envMap,
		runnerFactory: defaultRunnerFactory,
		installer:     hcInstaller{},
		lockerFactory: defaultLockerFactory,
		user:          user,
		lockTargetDir: lockDir,
	}, nil
}

// resolveExecPath returns an absolute path to a usable terraform binary.
// Preference order:
//  1. DATABRICKS_TF_EXEC_PATH (validated by exec.LookPath).
//  2. <workingDir>/bin/<terraform binary> if already present.
//  3. Download via hc-install into <workingDir>/bin.
func resolveExecPath(ctx context.Context, workingDir string, installer Installer) (string, error) {
	if p, ok := env.Lookup(ctx, ExecPathEnv); ok && p != "" {
		abs, err := exec.LookPath(p)
		if err != nil {
			return "", fmt.Errorf("locate %s=%q: %w", ExecPathEnv, p, err)
		}
		log.Debugf(ctx, "Using terraform at %s (from %s)", filepath.ToSlash(abs), ExecPathEnv)
		return abs, nil
	}

	binDir := filepath.Join(workingDir, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return "", fmt.Errorf("create terraform bin dir %s: %w", binDir, err)
	}

	existing := filepath.Join(binDir, product.Terraform.BinaryName())
	if _, err := os.Stat(existing); err == nil {
		log.Debugf(ctx, "Using terraform at %s", filepath.ToSlash(existing))
		return existing, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat %s: %w", existing, err)
	}

	tv, _, err := GetTerraformVersion(ctx)
	if err != nil {
		return "", err
	}
	log.Infof(ctx, "Downloading terraform %s to %s", tv.String(), filepath.ToSlash(binDir))
	path, err := installer.Install(ctx, binDir, tv)
	if err != nil {
		return "", fmt.Errorf("install terraform: %w", err)
	}
	return path, nil
}

// resolveAuthConfig resolves the workspace client for u and returns its SDK
// config. The resolved config is the canonical snapshot of which auth method
// fired (profile vs env vs OAuth cache) — buildEnv materializes it into
// DATABRICKS_* env vars so the terraform subprocess inherits the same auth
// regardless of how the parent CLI got there. Mirrors bundle.AuthEnv (see
// bundle/bundle.go).
func resolveAuthConfig(u *ucm.Ucm) (*config.Config, error) {
	if u == nil {
		return nil, nil
	}
	w, err := u.WorkspaceClientE()
	if err != nil {
		return nil, fmt.Errorf("resolve ucm auth for terraform: %w", err)
	}
	return w.Config, nil
}

// buildEnv assembles the env map passed to terraform-exec. Mirrors the
// order DAB's bundle/deploy/terraform/init.go uses:
//
//  1. auth.Env(authCfg) seeds DATABRICKS_* from the resolved SDK config
//     so --profile and OAuth-cache selections win over parent-env state.
//  2. inheritEnvVars copies envCopy, OIDC tokens, Azure DevOps SYSTEM_*,
//     and DATABRICKS_TF_CLI_CONFIG_FILE (version-gated → TF_CLI_CONFIG_FILE).
//  3. setTempDirEnvVars sets TMP/TEMP/TMPDIR, falling back to
//     localStateDir("tmp") on Windows to dodge MAX_PATH.
//  4. setProxyEnvVars forwards HTTP_PROXY / HTTPS_PROXY / NO_PROXY.
//  5. resolveDatabricksCliPath absolute-izes DATABRICKS_CLI_PATH so the
//     terraform subprocess can find the parent CLI from .databricks/ucm/...
//
// Cloud-cred env vars (AWS/Azure/GCP) are intentionally NOT forwarded
// — UCM strictly mirrors DAB here. Revisit when UCM gains resources
// that need them.
func buildEnv(ctx context.Context, u *ucm.Ucm, authCfg *config.Config) (map[string]string, error) {
	out := map[string]string{}

	if authCfg != nil {
		for k, v := range auth.Env(authCfg) {
			out[k] = v
		}
	}

	if err := inheritEnvVars(ctx, out); err != nil {
		return nil, err
	}
	if err := setTempDirEnvVars(ctx, out, u); err != nil {
		return nil, err
	}
	if err := setProxyEnvVars(ctx, out); err != nil {
		return nil, err
	}

	// Pre-seed DATABRICKS_CLI_PATH from the parent env so
	// resolveDatabricksCliPath has something to absolute-ize when
	// authCfg is nil. inheritEnvVars's envCopy allow-list
	// intentionally omits this key — it needs to be processed (made
	// absolute), not forwarded verbatim.
	if v, ok := env.Lookup(ctx, "DATABRICKS_CLI_PATH"); ok && v != "" {
		out["DATABRICKS_CLI_PATH"] = v
	}
	resolveDatabricksCliPath(out)

	return out, nil
}

// lockIdentity returns the (user, lockTargetDir) pair used to construct a
// Locker for Apply/Destroy. user identifies the holder in the on-the-wire
// lock record; lockTargetDir is the state dir whose race we are resolving.
//
// The lock dir follows the same `.databricks/ucm/<target>/state` convention
// U4 will use for terraform state. Keeping the path client-derived (rather
// than pulling it from a not-yet-wired Ucm field) lets U5 ship ahead of U4.
func lockIdentity(ctx context.Context, u *ucm.Ucm) (string, string) {
	user := env.Get(ctx, "USER")
	if user == "" {
		user = env.Get(ctx, "USERNAME")
	}
	if user == "" {
		user = "ucm"
	}
	lockDir := filepath.Join(u.RootPath, filepath.FromSlash(cacheDirName), u.Config.Ucm.Target, "state")
	return user, lockDir
}

// ensureRunner lazily binds a tfRunner to the Terraform wrapper. Split so
// Init can re-use it and tests can bypass the factory by pre-populating the
// runner field.
func (t *Terraform) ensureRunner(_ context.Context) error {
	if t.runner != nil {
		return nil
	}
	factory := t.runnerFactory
	if factory == nil {
		factory = defaultRunnerFactory
	}
	r, err := factory(t.WorkingDir, t.ExecPath)
	if err != nil {
		return fmt.Errorf("init terraform-exec: %w", err)
	}
	if err := r.SetEnv(t.Env); err != nil {
		return fmt.Errorf("set terraform env: %w", err)
	}
	t.runner = r
	return nil
}
