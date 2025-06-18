package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/python"
	"github.com/databricks/databricks-sdk-go/logger"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

const ownerRW = 0o600

func Load(ctx context.Context, labsYml string) (*Project, error) {
	raw, err := os.ReadFile(labsYml)
	if err != nil {
		return nil, fmt.Errorf("read labs.yml: %w", err)
	}
	project, err := readFromBytes(ctx, raw)
	if err != nil {
		return nil, err
	}
	project.folder = filepath.Dir(labsYml)
	return project, nil
}

func readFromBytes(ctx context.Context, labsYmlRaw []byte) (*Project, error) {
	var project Project
	err := yaml.Unmarshal(labsYmlRaw, &project)
	if err != nil {
		return nil, fmt.Errorf("parse labs.yml: %w", err)
	}
	e := (&project).metaEntrypoint(ctx)
	if project.Installer != nil {
		project.Installer.Entrypoint = e
	}
	if project.Uninstaller != nil {
		project.Uninstaller.Entrypoint = e
	}
	rootDir, err := PathInLabs(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	project.rootDir = rootDir
	return &project, nil
}

type Project struct {
	SpecVersion int `yaml:"$version"`

	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Installer   *hook    `yaml:"install,omitempty"`
	Uninstaller *hook    `yaml:"uninstall,omitempty"`
	Main        string   `yaml:"entrypoint"`
	MinPython   string   `yaml:"min_python"`
	Commands    []*proxy `yaml:"commands,omitempty"`

	folder  string
	rootDir string
}

func (p *Project) IsZipball() bool {
	// the simplest way of running the project - download ZIP file from github
	return true
}

func (p *Project) HasPython() bool {
	if strings.HasSuffix(p.Main, ".py") {
		return true
	}
	if p.Installer != nil && p.Installer.HasPython() {
		return true
	}
	if p.Uninstaller != nil && p.Uninstaller.HasPython() {
		return true
	}
	return p.MinPython != ""
}

func (p *Project) metaEntrypoint(ctx context.Context) *Entrypoint {
	return &Entrypoint{
		Project:               p,
		RequireRunningCluster: p.requireRunningCluster(),
	}
}

func (p *Project) requireRunningCluster() bool {
	if p.Installer != nil && p.Installer.RequireRunningCluster() {
		return true
	}
	for _, v := range p.Commands {
		if v.RequireRunningCluster {
			return true
		}
	}
	return false
}

func (p *Project) fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func (p *Project) projectFilePath(name string) string {
	return filepath.Join(p.EffectiveLibDir(), name)
}

func (p *Project) IsPythonProject() bool {
	if p.fileExists(p.projectFilePath("setup.py")) {
		return true
	}
	if p.fileExists(p.projectFilePath("pyproject.toml")) {
		return true
	}
	return false
}

func (p *Project) IsDeveloperMode() bool {
	return p.folder != "" && !strings.HasPrefix(p.LibDir(), p.folder)
}

func (p *Project) HasAccountLevelCommands() bool {
	for _, v := range p.Commands {
		if v.IsAccountLevel {
			return true
		}
	}
	return false
}

func (p *Project) IsBundleAware() bool {
	for _, v := range p.Commands {
		if v.IsBundleAware {
			return true
		}
	}
	return false
}

func (p *Project) Register(parent *cobra.Command) {
	group := &cobra.Command{
		Use:     p.Name,
		Short:   p.Description,
		GroupID: "labs",
	}
	parent.AddCommand(group)
	for _, cp := range p.Commands {
		cp.register(group)
		cp.Entrypoint.Project = p
	}
}

func (p *Project) CacheDir() string {
	return filepath.Join(p.rootDir, "cache")
}

func (p *Project) ConfigDir() string {
	return filepath.Join(p.rootDir, "config")
}

func (p *Project) LibDir() string {
	return filepath.Join(p.rootDir, "lib")
}

func (p *Project) EffectiveLibDir() string {
	if p.IsDeveloperMode() {
		// developer is working on a local checkout, that is not inside of installed root
		return p.folder
	}
	return p.LibDir()
}

func (p *Project) StateDir() string {
	return filepath.Join(p.rootDir, "state")
}

func (p *Project) EnsureFoldersExist() error {
	dirs := []string{p.CacheDir(), p.ConfigDir(), p.LibDir(), p.StateDir()}
	for _, v := range dirs {
		err := os.MkdirAll(v, ownerRWXworldRX)
		if err != nil {
			return fmt.Errorf("folder %s: %w", v, err)
		}
	}
	return nil
}

func (p *Project) Uninstall(cmd *cobra.Command) error {
	if p.Uninstaller != nil {
		err := p.Uninstaller.runHook(cmd)
		if err != nil {
			return fmt.Errorf("uninstall hook: %w", err)
		}
	}
	ctx := cmd.Context()
	log.Infof(ctx, "Removing project: %s", p.Name)
	return os.RemoveAll(p.rootDir)
}

func (p *Project) virtualEnvPath(ctx context.Context) string {
	if p.IsDeveloperMode() {
		// When a virtual environment has been activated, the VIRTUAL_ENV environment variable
		// is set to the path of the environment. Since explicitly activating a virtual environment
		// is not required to use it, VIRTUAL_ENV cannot be relied upon to determine whether a virtual
		// environment is being used.
		//
		// See https://docs.python.org/3/library/venv.html#how-venvs-work
		activatedVenv := env.Get(ctx, "VIRTUAL_ENV")
		if activatedVenv != "" {
			logger.Debugf(ctx, "(development mode) using active virtual environment from: %s", activatedVenv)
			return activatedVenv
		}
		nonActivatedVenv, err := python.DetectVirtualEnvPath(p.EffectiveLibDir())
		if err == nil {
			logger.Debugf(ctx, "(development mode) using virtual environment from: %s", nonActivatedVenv)
			return nonActivatedVenv
		}
	}
	// by default, we pick Virtual Environment from DATABRICKS_LABS_STATE_DIR
	return filepath.Join(p.StateDir(), "venv")
}

func (p *Project) virtualEnvPython(ctx context.Context) string {
	overridePython := env.Get(ctx, "PYTHON_BIN")
	if overridePython != "" {
		return overridePython
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(p.virtualEnvPath(ctx), "Scripts", "python.exe")
	}
	return filepath.Join(p.virtualEnvPath(ctx), "bin", "python3")
}

func (p *Project) loginFile(ctx context.Context) string {
	if p.IsDeveloperMode() {
		// developers may not want to pollute the state in
		// ~/.databricks/labs/X/config while the version is not yet
		// released
		return p.projectFilePath(".databricks-login.json")
	}
	return filepath.Join(p.ConfigDir(), "login.json")
}

func (p *Project) loadLoginConfig(ctx context.Context) (*loginConfig, error) {
	loginFile := p.loginFile(ctx)
	log.Debugf(ctx, "Loading login configuration from: %s", loginFile)
	lc, err := tryLoadAndParseJSON[loginConfig](loginFile)
	if err != nil {
		return nil, fmt.Errorf("try load: %w", err)
	}
	lc.Entrypoint = p.metaEntrypoint(ctx)
	return lc, nil
}

func (p *Project) versionFile(ctx context.Context) string {
	return filepath.Join(p.StateDir(), "version.json")
}

func (p *Project) InstalledVersion(ctx context.Context) (*version, error) {
	if p.IsDeveloperMode() {
		return &version{
			Version: "*",
			Date:    time.Now(),
		}, nil
	}
	versionFile := p.versionFile(ctx)
	log.Debugf(ctx, "Loading installed version info from: %s", versionFile)
	return tryLoadAndParseJSON[version](versionFile)
}

func (p *Project) writeVersionFile(ctx context.Context, ver string) error {
	versionFile := p.versionFile(ctx)
	raw, err := json.Marshal(version{
		Version: ver,
		Date:    time.Now(),
	})
	if err != nil {
		return err
	}
	log.Debugf(ctx, "Writing installed version info to: %s", versionFile)
	return os.WriteFile(versionFile, raw, ownerRW)
}

// checkUpdates is called before every command of an installed project,
// giving users hints when they need to update their installations.
func (p *Project) checkUpdates(cmd *cobra.Command) error {
	ctx := cmd.Context()
	if p.IsDeveloperMode() {
		// skipping update check for projects in developer mode, that
		// might not be installed yet
		return nil
	}
	r := github.NewReleaseCache("databrickslabs", p.Name, p.CacheDir(), false)
	versions, err := r.Load(ctx)
	if err != nil {
		return err
	}
	installed, err := p.InstalledVersion(ctx)
	if err != nil {
		return err
	}
	latest := versions[0]
	if installed.Version == latest.Version {
		return nil
	}
	ago := time.Since(latest.PublishedAt)
	msg := "[UPGRADE ADVISED] Newer %s version was released %s ago. Please run `databricks labs upgrade %s` to upgrade: %s -> %s"
	cmd.PrintErrln(color.YellowString(msg, p.Name, p.timeAgo(ago), p.Name, installed.Version, latest.Version))
	return nil
}

func (p *Project) timeAgo(dur time.Duration) string {
	days := int(dur.Hours()) / 24
	hours := int(dur.Hours()) % 24
	minutes := int(dur.Minutes()) % 60
	if dur < time.Minute {
		return "minute"
	} else if dur < time.Hour {
		return fmt.Sprintf("%d minutes", minutes)
	} else if dur < (24 * time.Hour) {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d days", days)
}

func (p *Project) profileOverride(cmd *cobra.Command) string {
	profileFlag := cmd.Flag("profile")
	if profileFlag == nil {
		return ""
	}
	return profileFlag.Value.String()
}

type version struct {
	Version string    `json:"version"`
	Date    time.Time `json:"date"`
}
