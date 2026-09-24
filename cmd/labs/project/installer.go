package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/cmd/labs/unpack"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

const ownerRWXworldRX = 0o755

type whTypes []sql.EndpointInfoWarehouseType

type hook struct {
	*Entrypoint              `yaml:",inline"`
	Script                   string  `yaml:"script"`
	RequireDatabricksConnect bool    `yaml:"require_databricks_connect,omitempty"`
	MinRuntimeVersion        string  `yaml:"min_runtime_version,omitempty"`
	WarehouseTypes           whTypes `yaml:"warehouse_types,omitempty"`
	Extras                   string  `yaml:"extras,omitempty"`
}

func (h *hook) RequireRunningCluster() bool {
	if h.Entrypoint == nil {
		return false
	}
	return h.Entrypoint.RequireRunningCluster
}

func (h *hook) HasPython() bool {
	return strings.HasSuffix(h.Script, ".py")
}

func (h *hook) runHook(cmd *cobra.Command) error {
	if h.Script == "" {
		return nil
	}
	ctx := cmd.Context()
	envs, err := h.Prepare(cmd)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	libDir := h.EffectiveLibDir()
	var args []string
	if strings.HasSuffix(h.Script, ".py") {
		args = append(args, h.virtualEnvPython(ctx))
	}
	return process.Forwarded(ctx,
		append(args, h.Script),
		cmd.InOrStdin(),
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
		process.WithDir(libDir),
		process.WithEnvs(envs))
}

type installer struct {
	*Project
	version string

	// command instance is used for:
	// - auth profile flag override
	// - standard input, output, and error streams
	cmd            *cobra.Command
	offlineInstall bool
}

func (i *installer) Install(ctx context.Context) error {
	err := i.ensureInstallFoldersExist()
	if err != nil {
		return fmt.Errorf("folders: %w", err)
	}
	i.folder, err = PathInLabs(ctx, i.Name)
	if err != nil {
		return err
	}
	w, err := i.login(ctx)
	if err != nil && errors.Is(err, profile.ErrNoConfiguration) {
		cfg, err := i.Installer.envAwareConfig(ctx)
		if err != nil {
			return err
		}
		w, err = databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err != nil {
			return fmt.Errorf("no ~/.databrickscfg: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	if i.offlineInstall {
		if _, err := os.Stat(i.LibDir()); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("no local installation found: %w", err)
		}
		return i.completeInstall(ctx, w)
	}

	err = i.installDownloadedLibrary(ctx, w)
	if err != nil {
		return err
	}
	return nil
}

func (i *installer) installDownloadedLibrary(ctx context.Context, w *databricks.WorkspaceClient) (err error) {
	libDir := i.LibDir()
	stagingDir, err := os.MkdirTemp(filepath.Dir(libDir), ".lib-staging-")
	if err != nil {
		return fmt.Errorf("create library staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	if err := i.downloadLibrary(ctx, stagingDir); err != nil {
		return fmt.Errorf("lib: %w", err)
	}
	if err := validateLibrary(stagingDir); err != nil {
		return fmt.Errorf("validate staged library: %w", err)
	}

	txn := newLibraryTransaction(libDir, stagingDir)
	if err := txn.swap(); err != nil {
		return fmt.Errorf("swap library: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, txn.rollback())
		}
	}()

	if err = i.completeInstall(ctx, w); err != nil {
		return err
	}
	if err = txn.commit(); err != nil {
		return fmt.Errorf("commit library installation: %w", err)
	}
	return nil
}

func (i *installer) completeInstall(ctx context.Context, w *databricks.WorkspaceClient) error {
	if err := i.setupPythonVirtualEnvironment(ctx, w); err != nil {
		return fmt.Errorf("python: %w", err)
	}
	if err := i.recordVersion(ctx); err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	if err := i.runInstallHook(ctx); err != nil {
		return fmt.Errorf("installer: %w", err)
	}
	return nil
}

func (i *installer) ensureInstallFoldersExist() error {
	dirs := []string{i.CacheDir(), i.ConfigDir(), i.StateDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, ownerRWXworldRX); err != nil {
			return fmt.Errorf("folder %s: %w", dir, err)
		}
	}
	return nil
}

func validateLibrary(libDir string) error {
	info, err := os.Stat(filepath.Join(libDir, "labs.yml"))
	if err != nil {
		return fmt.Errorf("stat labs.yml: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("labs.yml is not a regular file")
	}
	return nil
}

type libraryTransaction struct {
	libDir     string
	stagingDir string
	backupDir  string
	hadLibrary bool
	rename     func(string, string) error
	removeAll  func(string) error
}

func newLibraryTransaction(libDir, stagingDir string) *libraryTransaction {
	return &libraryTransaction{
		libDir:     libDir,
		stagingDir: stagingDir,
		backupDir:  stagingDir + "-backup",
		rename:     os.Rename,
		removeAll:  os.RemoveAll,
	}
}

func (t *libraryTransaction) swap() error {
	_, err := os.Stat(t.libDir)
	switch {
	case err == nil:
		t.hadLibrary = true
		if err := t.rename(t.libDir, t.backupDir); err != nil {
			return fmt.Errorf("back up current library: %w", err)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("stat current library: %w", err)
	}
	if err := t.rename(t.stagingDir, t.libDir); err != nil {
		installErr := fmt.Errorf("install staged library: %w", err)
		if t.hadLibrary {
			return errors.Join(installErr, t.rollback())
		}
		return installErr
	}
	return nil
}

func (t *libraryTransaction) rollback() error {
	var errs []error
	if err := t.removeAll(t.libDir); err != nil {
		errs = append(errs, fmt.Errorf("remove failed library: %w", err))
	}
	if t.hadLibrary {
		if err := t.rename(t.backupDir, t.libDir); err != nil {
			errs = append(errs, fmt.Errorf("restore previous library: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (t *libraryTransaction) commit() error {
	if !t.hadLibrary {
		return nil
	}
	if err := t.removeAll(t.backupDir); err != nil {
		return err
	}
	return nil
}

func (i *installer) Upgrade(ctx context.Context) error {
	if err := i.cleanupLib(ctx); err != nil {
		return fmt.Errorf("lib: %w", err)
	}
	err := i.downloadLibrary(ctx, i.LibDir())
	if err != nil {
		return fmt.Errorf("lib: %w", err)
	}
	err = i.recordVersion(ctx)
	if err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	err = i.installPythonDependencies(ctx, ".")
	if err != nil {
		return fmt.Errorf("python dependencies: %w", err)
	}
	err = i.runInstallHook(ctx)
	if err != nil {
		return fmt.Errorf("installer: %w", err)
	}
	return nil
}

func (i *installer) warning(s string) {
	i.cmd.PrintErrln(cmdio.Yellow(i.cmd.Context(), s))
}

func (i *installer) cleanupLib(ctx context.Context) error {
	libDir := i.LibDir()
	if err := os.RemoveAll(libDir); err != nil {
		return fmt.Errorf("remove all: %w", err)
	}
	if err := os.MkdirAll(libDir, ownerRWXworldRX); err != nil {
		return fmt.Errorf("create library directory: %w", err)
	}
	return nil
}

func (i *installer) recordVersion(ctx context.Context) error {
	return i.writeVersionFile(ctx, i.version)
}

func (i *installer) login(ctx context.Context) (*databricks.WorkspaceClient, error) {
	cfg, err := i.metaEntrypoint(ctx).validLogin(i.cmd)
	if errors.Is(err, ErrNoLoginConfig) {
		cfg, err = i.Installer.envAwareConfig(ctx)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("valid: %w", err)
	}
	if !i.HasAccountLevelCommands() && cfg.ConfigType() == config.AccountConfig {
		return nil, errors.New("got account-level client, but no account-level commands")
	}
	lc := &loginConfig{Entrypoint: i.Installer.Entrypoint}
	w, err := lc.askWorkspace(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("ask for workspace: %w", err)
	}
	err = lc.save(ctx)
	if err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}
	return w, nil
}

func (i *installer) downloadLibrary(ctx context.Context, libTarget string) error {
	sp := cmdio.NewSpinner(ctx)
	defer sp.Close()
	// we may support wheels, jars, and golang binaries. but those are not zipballs
	if i.IsZipball() {
		sp.Update("Downloading and unpacking zipball for " + i.version)
		return i.downloadAndUnpackZipball(ctx, libTarget)
	}
	return errors.New("we only support zipballs for now")
}

func (i *installer) downloadAndUnpackZipball(ctx context.Context, libTarget string) error {
	raw, err := github.DownloadZipball(ctx, "databrickslabs", i.Name, i.version)
	if err != nil {
		return fmt.Errorf("download zipball from GitHub: %w", err)
	}
	zipball := unpack.GitHubZipball{Reader: bytes.NewBuffer(raw)}
	log.Debugf(ctx, "Unpacking zipball to: %s", libTarget)
	return zipball.UnpackTo(libTarget)
}

func (i *installer) setupPythonVirtualEnvironment(ctx context.Context, w *databricks.WorkspaceClient) error {
	if !i.HasPython() {
		return nil
	}
	sp := cmdio.NewSpinner(ctx)
	defer sp.Close()
	sp.Update("Detecting all installed Python interpreters on the system")
	pythonInterpreters, err := DetectInterpreters(ctx)
	if err != nil {
		return fmt.Errorf("detect: %w", err)
	}
	py, err := pythonInterpreters.AtLeast(i.MinPython)
	if err != nil {
		return fmt.Errorf("min version: %w", err)
	}
	log.Debugf(ctx, "Detected Python %s at: %s", py.Version, py.Path)
	venvPath := i.virtualEnvPath(ctx)
	log.Debugf(ctx, "Creating Python Virtual Environment at: %s", venvPath)
	sp.Update("Creating Virtual Environment with Python " + py.Version)
	_, err = process.Background(ctx, []string{py.Path, "-m", "venv", venvPath})
	if err != nil {
		return fmt.Errorf("create venv: %w", err)
	}
	if i.Installer != nil && i.Installer.RequireDatabricksConnect {
		sp.Update("Determining Databricks Connect version")
		cluster, err := w.Clusters.Get(ctx, compute.GetClusterRequest{
			ClusterId: w.Config.ClusterID,
		})
		if err != nil {
			return fmt.Errorf("cluster: %w", err)
		}
		runtimeVersion, ok := cfgpickers.GetRuntimeVersion(*cluster)
		if !ok {
			return fmt.Errorf("unsupported runtime: %s", cluster.SparkVersion)
		}
		sp.Update("Installing Databricks Connect v" + runtimeVersion)
		pipSpec := "databricks-connect==" + runtimeVersion
		err = i.installPythonDependencies(ctx, pipSpec)
		if err != nil {
			return fmt.Errorf("dbconnect: %w", err)
		}
	}
	sp.Update("Installing Python library dependencies")
	if i.Installer.Extras != "" {
		// install main and optional dependencies
		return i.installPythonDependencies(ctx, fmt.Sprintf(".[%s]", i.Installer.Extras))
	}
	return i.installPythonDependencies(ctx, ".")
}

func (i *installer) installPythonDependencies(ctx context.Context, spec string) error {
	if !i.IsPythonProject() {
		return nil
	}
	libDir := i.LibDir()
	log.Debugf(ctx, "Installing Python dependencies for: %s", libDir)
	// maybe we'll need to add call one of the two scripts:
	// - python3 -m ensurepip --default-pip
	// - curl -o https://bootstrap.pypa.io/get-pip.py | python3
	var buf bytes.Buffer
	// Ensure latest version(s) is installed with the `--upgrade` and `--upgrade-strategy eager` flags
	// https://pip.pypa.io/en/stable/cli/pip_install/#cmdoption-U
	_, err := process.Background(ctx,
		[]string{i.virtualEnvPython(ctx), "-m", "pip", "install", "--upgrade", "--upgrade-strategy", "eager", spec},
		process.WithCombinedOutput(&buf),
		process.WithDir(libDir))
	if err != nil {
		i.warning(buf.String())
		return fmt.Errorf("failed to install dependencies of %s", spec)
	}
	return nil
}

func (i *installer) runInstallHook(ctx context.Context) error {
	if i.Installer == nil {
		return nil
	}
	if i.Installer.Script == "" {
		return nil
	}
	log.Debugf(ctx, "Launching installer script %s in %s", i.Installer.Script, i.LibDir())
	return i.Installer.runHook(i.cmd)
}
