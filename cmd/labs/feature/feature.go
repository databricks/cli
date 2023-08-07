package feature

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/python"
	"gopkg.in/yaml.v2"
)

type Feature struct {
	Name        string `json:"name"`
	Context     string `json:"context,omitempty"` // auth context
	Description string `json:"description"`
	Hooks       struct {
		Install   string `json:"install,omitempty"`
		Uninstall string `json:"uninstall,omitempty"`
	} `json:"hooks,omitempty"`
	Entrypoint string `json:"entrypoint"`
	Commands   []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Flags       []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"flags,omitempty"`
	} `json:"commands,omitempty"`

	version  string
	path     string
	checkout *git.Repository
}

func NewFeature(name string) (*Feature, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	version := "latest"
	split := strings.Split(name, "@")
	if len(split) > 2 {
		return nil, fmt.Errorf("invalid coordinates: %s", name)
	}
	if len(split) == 2 {
		name = split[0]
		version = split[1]
	}
	path := filepath.Join(home, ".databricks", "labs", name)
	checkout, err := git.NewRepository(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return &Feature{
		Name:     name,
		path:     path,
		version:  version,
		checkout: checkout,
	}, nil
}

type release struct {
	TagName     string    `json:"tag_name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

func (i *Feature) loadMetadata() error {
	raw, err := os.ReadFile(filepath.Join(i.path, "labs.yml"))
	if err != nil {
		return fmt.Errorf("read labs.yml: %w", err)
	}
	err = yaml.Unmarshal(raw, i)
	if err != nil {
		return fmt.Errorf("parse labs.yml: %w", err)
	}
	return nil
}

func (i *Feature) fetchLatestVersion(ctx context.Context) (*release, error) {
	var tags []release
	url := fmt.Sprintf("https://api.github.com/repos/databrickslabs/%s/releases", i.Name)
	err := httpCall(ctx, url, &tags)
	if err != nil {
		return nil, err
	}
	return &tags[0], nil
}

func (i *Feature) requestedVersion(ctx context.Context) (string, error) {
	if i.version == "latest" {
		release, err := i.fetchLatestVersion(ctx)
		if err != nil {
			return "", err
		}
		return release.TagName, nil
	}
	return i.version, nil
}

func (i *Feature) Install(ctx context.Context) error {
	if i.hasFile(".git/HEAD") {
		curr, err := process.Background(ctx, []string{
			"git", "tag", "--points-at", "HEAD",
		}, process.WithDir(i.path))
		if err != nil {
			return err
		}
		return fmt.Errorf("%s (%s) is already installed", i.Name, curr)
	}
	url := fmt.Sprintf("https://github.com/databrickslabs/%s", i.Name)
	version, err := i.requestedVersion(ctx)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Installing %s (%s) into %s", url, version, i.path)
	err = git.Clone(ctx, url, version, i.path)
	if err != nil {
		return err
	}
	err = i.loadMetadata()
	if err != nil {
		return fmt.Errorf("labs.yml: %w", err)
	}
	if i.isPython() {
		err := i.installPythonTool(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

const CacheDir = ".databricks"

func (i *Feature) Run(ctx context.Context, raw []byte) error {
	// raw is a JSON-encoded payload that holds things like command name and flags
	return i.forwardPython(ctx, filepath.Join(i.path, i.Entrypoint), string(raw))
}

func (i *Feature) hasFile(name string) bool {
	_, err := os.Stat(filepath.Join(i.path, name))
	return err == nil
}

func (i *Feature) isPython() bool {
	return i.hasFile("setup.py") || i.hasFile("pyproject.toml")
}

func (i *Feature) venvBinDir() string {
	return filepath.Join(i.path, CacheDir, "bin")
}

func (i *Feature) forwardPython(ctx context.Context, pythonArgs ...string) error {
	args := []string{filepath.Join(i.venvBinDir(), "python")}
	args = append(args, pythonArgs...)
	return process.Forwarded(ctx, args,
		process.WithDir(i.path), // we may need to skip it for install step
		process.WithEnv("PYTHONPATH", i.path))
}

func (i *Feature) installPythonTool(ctx context.Context) error {
	pythons, err := python.DetectInterpreters(ctx)
	if err != nil {
		return err
	}
	interpreter := pythons.Latest()
	log.Debugf(ctx, "Creating Python %s virtual environment in %s", interpreter.Version, i.path)
	_, err = process.Background(ctx, []string{
		interpreter.Binary, "-m", "venv", CacheDir,
	}, process.WithDir(i.path))
	if err != nil {
		return fmt.Errorf("create venv: %w", err)
	}
	log.Debugf(ctx, "Installing dependencies via PIP")
	venvPip := filepath.Join(i.venvBinDir(), "pip")
	_, err = process.Background(ctx, []string{
		venvPip, "install", ".",
	}, process.WithDir(i.path))
	if err != nil {
		return fmt.Errorf("pip install: %w", err)
	}
	if i.Hooks.Install != "" {
		installer := filepath.Join(i.path, i.Hooks.Install)
		err = i.forwardPython(ctx, installer)
		if err != nil {
			return fmt.Errorf("%s: %w", i.Hooks.Install, err)
		}
	}
	return nil
}
