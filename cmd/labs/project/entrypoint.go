package project

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/logger"
	"github.com/spf13/cobra"
)

type Entrypoint struct {
	*Project

	RequireRunningCluster bool `yaml:"require_running_cluster,omitempty"`
	IsUnauthenticated     bool `yaml:"is_unauthenticated,omitempty"`
	IsAccountLevel        bool `yaml:"is_account_level,omitempty"`
	IsBundleAware         bool `yaml:"is_bundle_aware,omitempty"`
}

var (
	ErrNoLoginConfig      = errors.New("no login configuration found")
	ErrMissingClusterID   = errors.New("missing a cluster compatible with Databricks Connect")
	ErrMissingWarehouseID = errors.New("missing a SQL warehouse")
	ErrNotInTTY           = errors.New("not in an interactive terminal")
)

func (e *Entrypoint) NeedsCluster() bool {
	if e.Installer == nil {
		return false
	}
	if e.Installer.RequireDatabricksConnect && e.Installer.MinRuntimeVersion == "" {
		e.Installer.MinRuntimeVersion = "13.1"
	}
	return e.Installer.MinRuntimeVersion != ""
}

func (e *Entrypoint) NeedsWarehouse() bool {
	if e.Installer == nil {
		return false
	}
	return len(e.Installer.WarehouseTypes) != 0
}

func (e *Entrypoint) Prepare(cmd *cobra.Command) (map[string]string, error) {
	ctx := cmd.Context()
	libDir := e.EffectiveLibDir()
	environment := map[string]string{
		"DATABRICKS_CLI_VERSION":     build.GetInfo().Version,
		"DATABRICKS_LABS_CACHE_DIR":  e.CacheDir(),
		"DATABRICKS_LABS_CONFIG_DIR": e.ConfigDir(),
		"DATABRICKS_LABS_STATE_DIR":  e.StateDir(),
		"DATABRICKS_LABS_LIB_DIR":    libDir,
	}
	if e.IsPythonProject() {
		e.preparePython(ctx, environment)
	}
	cfg, err := e.validLogin(cmd)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	// cleanup auth profile and config file location,
	// so that we don't confuse SDKs
	cfg.Profile = ""
	cfg.ConfigFile = ""
	var varNames []string
	for k, v := range e.environmentFromConfig(cfg) {
		environment[k] = v
		varNames = append(varNames, k)
	}
	if e.NeedsCluster() && e.RequireRunningCluster {
		err = e.ensureRunningCluster(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("running cluster: %w", err)
		}
	}
	log.Debugf(ctx, "Passing down environment variables: %s", strings.Join(varNames, ", "))
	return environment, nil
}

func (e *Entrypoint) preparePython(ctx context.Context, environment map[string]string) {
	venv := e.virtualEnvPath(ctx)
	environment["PATH"] = e.joinPaths(filepath.Join(venv, "bin"), env.Get(ctx, "PATH"))

	// PYTHONPATH extends the standard lookup locations for module files. It follows the same structure as
	// the shell's PATH, where you specify one or more directory paths separated by the appropriate delimiter
	// (such as colons for Unix or semicolons for Windows). If a directory listed in PYTHONPATH doesn't exist,
	// it is disregarded without any notifications.
	//
	// Beyond regular directories, individual entries in PYTHONPATH can point to zipfiles that contain pure
	// Python modules in either their source or compiled forms. It's important to note that extension modules
	// cannot be imported from zipfiles.
	//
	// The initial search path varies depending on your installation but typically commences with the
	// prefix/lib/pythonversion path (as indicated by PYTHONHOME). This default path is always included
	// in PYTHONPATH.
	//
	// An extra directory can be included at the beginning of the search path, coming before PYTHONPATH,
	// as explained in the Interface options section. You can control the search path from within a Python
	// script using the sys.path variable.
	//
	// Here we are also supporting the "src" layout for python projects.
	//
	// See https://docs.python.org/3/using/cmdline.html#envvar-PYTHONPATH
	libDir := e.EffectiveLibDir()
	// The intention for every install is to be sandboxed - not dependent on anything else than Python binary.
	// Having ability to override PYTHONPATH in the mix will break this assumption. Need strong evidence that
	// this is really needed.
	environment["PYTHONPATH"] = e.joinPaths(libDir, filepath.Join(libDir, "src"))
}

func (e *Entrypoint) ensureRunningCluster(ctx context.Context, cfg *config.Config) error {
	feedback := cmdio.Spinner(ctx)
	defer close(feedback)
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return fmt.Errorf("workspace client: %w", err)
	}
	// TODO: add in-progress callback to EnsureClusterIsRunning() in SDK
	feedback <- "Ensuring the cluster is running..."
	err = w.Clusters.EnsureClusterIsRunning(ctx, cfg.ClusterID)
	if err != nil {
		return fmt.Errorf("ensure running: %w", err)
	}
	return nil
}

func (e *Entrypoint) joinPaths(paths ...string) string {
	return strings.Join(paths, string(os.PathListSeparator))
}

func (e *Entrypoint) envAwareConfig(ctx context.Context) (*config.Config, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return nil, err
	}
	return &config.Config{
		ConfigFile: filepath.Join(home, ".databrickscfg"),
		Loaders: []config.Loader{
			env.NewConfigLoader(ctx),
			config.ConfigAttributes,
			config.ConfigFile,
		},
	}, nil
}

func (e *Entrypoint) envAwareConfigWithProfile(ctx context.Context, profile string) (*config.Config, error) {
	cfg, err := e.envAwareConfig(ctx)
	if err != nil {
		return nil, err
	}
	cfg.Profile = profile
	return cfg, nil
}

func (e *Entrypoint) getLoginConfig(cmd *cobra.Command) (*loginConfig, *config.Config, error) {
	ctx := cmd.Context()
	// it's okay for this config file not to exist, because some environments,
	// like GitHub Actions, don't (need) to have it. There's a small downside of
	// a warning log message from within Go SDK.
	profileOverride := e.profileOverride(cmd)
	if profileOverride != "" {
		log.Infof(ctx, "Overriding login profile: %s", profileOverride)
		cfg, err := e.envAwareConfigWithProfile(ctx, profileOverride)
		if err != nil {
			return nil, nil, err
		}
		return &loginConfig{}, cfg, nil
	}
	lc, err := e.loadLoginConfig(ctx)
	isNoLoginConfig := errors.Is(err, fs.ErrNotExist)
	defaultConfig, err := e.envAwareConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	if isNoLoginConfig && !e.IsBundleAware && e.isAuthConfigured(defaultConfig) {
		log.Debugf(ctx, "Login is configured via environment variables")
		return &loginConfig{}, defaultConfig, nil
	}
	if isNoLoginConfig && !e.IsBundleAware {
		return nil, nil, ErrNoLoginConfig
	}
	if e.IsAccountLevel {
		log.Debugf(ctx, "Using account-level login profile: %s", lc.AccountProfile)
		cfg, err := e.envAwareConfigWithProfile(ctx, lc.AccountProfile)
		if err != nil {
			return nil, nil, err
		}
		return lc, cfg, nil
	}
	if e.IsBundleAware {
		ctx := cmd.Context()
		b := root.TryConfigureBundle(cmd)
		if b != nil {
			log.Infof(ctx, "Using login configuration from Databricks Asset Bundle")
			return &loginConfig{}, b.WorkspaceClient().Config, nil
		}
	}
	log.Debugf(ctx, "Using workspace-level login profile: %s", lc.WorkspaceProfile)
	cfg, err := e.envAwareConfigWithProfile(ctx, lc.WorkspaceProfile)
	if err != nil {
		return nil, nil, err
	}
	return lc, cfg, nil
}

func (e *Entrypoint) validLogin(cmd *cobra.Command) (*config.Config, error) {
	if e.IsUnauthenticated {
		return &config.Config{}, nil
	}
	lc, cfg, err := e.getLoginConfig(cmd)
	if err != nil {
		return nil, fmt.Errorf("login config: %w", err)
	}
	err = cfg.EnsureResolved()
	if err != nil {
		return nil, err
	}
	ctx := cmd.Context()
	logger.Debugf(ctx, "Resolved login: %s", config.ConfigAttributes.DebugString(cfg))
	// merge ~/.databrickscfg and ~/.databricks/labs/x/config/login.json when
	// it comes to project-specific configuration
	if e.NeedsCluster() && cfg.ClusterID == "" {
		cfg.ClusterID = lc.ClusterID
	}
	if e.NeedsWarehouse() && cfg.WarehouseID == "" {
		cfg.WarehouseID = lc.WarehouseID
	}
	// there's a lot of end-user friction for projects, that require account-level commands.
	// this is mainly related to the fact, that, as of January 2024, workspace administrators
	// do not necessarily have access to call account-level APIs. There are ongoing discussions
	// on how to best implement this on a platform level.
	//
	// Current temporary workaround is creating dummy ~/.databrickscfg profile with `account_id`
	// field, though it doesn't really remove the end-user friction, hence we don't require
	// an account profile during installation (anymore) and just prompt for it, when context
	// does require it. This also means that we always prompt for account-level commands, unless
	// users specify a `--profile` flag.
	isACC := cfg.IsAccountClient()
	if e.IsAccountLevel && cfg.Profile == "" {
		if !cmdio.IsPromptSupported(ctx) {
			return nil, root.ErrCannotConfigureAuth
		}
		replaceCfg, err := e.envAwareConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("replace config: %w", err)
		}
		err = lc.askAccountProfile(ctx, replaceCfg)
		if err != nil {
			return nil, fmt.Errorf("account: %w", err)
		}
		err = replaceCfg.EnsureResolved()
		if err != nil {
			return nil, fmt.Errorf("resolve: %w", err)
		}
		return replaceCfg, nil
	} else if e.IsAccountLevel && !isACC {
		return nil, databricks.ErrNotAccountClient
	}
	if e.NeedsCluster() && !isACC && cfg.ClusterID == "" {
		return nil, ErrMissingClusterID
	}
	if e.NeedsWarehouse() && !isACC && cfg.WarehouseID == "" {
		return nil, ErrMissingWarehouseID
	}
	return cfg, nil
}

func (e *Entrypoint) environmentFromConfig(cfg *config.Config) map[string]string {
	env := map[string]string{}
	for _, a := range config.ConfigAttributes {
		if a.IsZero(cfg) {
			continue
		}
		for _, ev := range a.EnvVars {
			env[ev] = a.GetString(cfg)
		}
	}
	return env
}

func (e *Entrypoint) isAuthConfigured(cfg *config.Config) bool {
	r := &http.Request{Header: http.Header{}}
	err := cfg.Authenticate(r.WithContext(context.Background()))
	return err == nil
}
