package terraform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// tfRunner is the minimal terraform-exec surface used by the wrapper.
// Having an explicit interface keeps tests independent of a real terraform
// binary — the production impl is *tfexec.Terraform; tests inject a fake.
type tfRunner interface {
	Init(ctx context.Context, opts ...tfexec.InitOption) error
	Plan(ctx context.Context, opts ...tfexec.PlanOption) (bool, error)
	Apply(ctx context.Context, opts ...tfexec.ApplyOption) error
	Destroy(ctx context.Context, opts ...tfexec.DestroyOption) error
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
	// Env is the environment map passed to terraform-exec. Populated by New;
	// includes DATABRICKS_HOST/CLIENT_ID/CLIENT_SECRET + cloud-cred passthrough.
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
// directory, and assembles the env-var map used for auth and cloud-cred
// passthrough. The caller is expected to have run SelectTarget first.
func New(ctx context.Context, u *ucm.Ucm) (*Terraform, error) {
	workingDir, err := WorkingDir(u)
	if err != nil {
		return nil, err
	}

	execPath, err := resolveExecPath(ctx, workingDir, hcInstaller{})
	if err != nil {
		return nil, err
	}

	envMap := buildEnv(ctx)

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

// buildEnv assembles the env map passed to terraform-exec.
//
// It starts with the auth variables the databricks terraform provider reads
// natively (DATABRICKS_HOST / DATABRICKS_CLIENT_ID / DATABRICKS_CLIENT_SECRET
// / DATABRICKS_TOKEN / DATABRICKS_CONFIG_PROFILE), then layers on the cloud
// credentials that the underlay resources will eventually need (AWS, Azure,
// GCP), then PATH/HOME/TMPDIR/proxy variables so the subprocess inherits a
// sane environment. Everything flows from the current process env — there is
// no `--profile` resolution here; that is the CLI layer's job.
func buildEnv(ctx context.Context) map[string]string {
	out := map[string]string{}

	passthroughKeys := []string{
		// Databricks auth — consumed by the terraform-provider-databricks.
		// See https://registry.terraform.io/providers/databricks/databricks/latest/docs
		"DATABRICKS_HOST",
		"DATABRICKS_CLIENT_ID",
		"DATABRICKS_CLIENT_SECRET",
		"DATABRICKS_TOKEN",
		"DATABRICKS_CONFIG_PROFILE",
		"DATABRICKS_CONFIG_FILE",
		"DATABRICKS_ACCOUNT_ID",
		"DATABRICKS_AUTH_TYPE",
		"DATABRICKS_METADATA_SERVICE_URL",

		// AWS cloud-underlay credentials. Out-of-scope for M1, but passing
		// them through now keeps the wrapper from re-shaping once AWS
		// resources land.
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
		"AWS_PROFILE",
		"AWS_WEB_IDENTITY_TOKEN_FILE",
		"AWS_ROLE_ARN",
		"AWS_ROLE_SESSION_NAME",

		// Azure cloud-underlay credentials.
		"AZURE_TENANT_ID",
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_FEDERATED_TOKEN_FILE",

		// GCP cloud-underlay credentials.
		"GOOGLE_CREDENTIALS",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GOOGLE_PROJECT",
		"GOOGLE_REGION",

		// Process plumbing.
		"HOME",
		"USERPROFILE",
		"PATH",
		"TF_CLI_CONFIG_FILE",
	}
	for _, k := range passthroughKeys {
		if v, ok := env.Lookup(ctx, k); ok {
			out[k] = v
		}
	}

	// $DATABRICKS_TF_CLI_CONFIG_FILE maps to $TF_CLI_CONFIG_FILE so the
	// VSCode extension's filesystem-mirror config is picked up when it lines
	// up with the provider version we actually use.
	if v, ok := env.Lookup(ctx, CliConfigPathEnv); ok && v != "" {
		if _, err := os.Stat(v); err == nil {
			out["TF_CLI_CONFIG_FILE"] = v
		}
	}

	// Proxy variables — both upper and lower case; terraform-exec is fine
	// with either, but downstream tools on macOS/Linux commonly read the
	// uppercase form.
	for _, v := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"} {
		for _, key := range []string{v, strings.ToLower(v)} {
			if val, ok := env.Lookup(ctx, key); ok {
				out[strings.ToUpper(v)] = val
			}
		}
	}

	// TMPDIR / TMP — let terraform create temp files in a place it can write.
	if runtime.GOOS == "windows" {
		for _, k := range []string{"TMP", "TEMP"} {
			if v, ok := env.Lookup(ctx, k); ok {
				out[k] = v
			}
		}
	} else if v, ok := env.Lookup(ctx, "TMPDIR"); ok {
		out["TMPDIR"] = v
	}

	return out
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
