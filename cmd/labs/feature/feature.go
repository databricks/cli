package feature

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

type Feature struct {
	Name        string `json:"name"`
	Context     string `json:"context,omitempty"` // auth context
	Description string `json:"description"`
	Hooks       struct {
		Install   string `json:"install,omitempty"`
		Uninstall string `json:"uninstall,omitempty"`
	}
	Entrypoint string `json:"entrypoint"`
	Commands   []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Flags       []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"flags,omitempty"`
	} `json:"commands,omitempty"`

	path     string
	checkout *git.Repository
}

func NewFeature(name string) (*Feature, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".databricks", "labs", name)
	checkout, err := git.NewRepository(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	feat := &Feature{
		Name:     name,
		path:     path,
		checkout: checkout,
	}
	raw, err := os.ReadFile(filepath.Join(path, "labs.yml"))
	if err != nil {
		return nil, fmt.Errorf("read labs.yml: %w", err)
	}
	err = yaml.Unmarshal(raw, feat)
	if err != nil {
		return nil, fmt.Errorf("parse labs.yml: %w", err)
	}
	return feat, nil
}

type release struct {
	TagName     string    `json:"tag_name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

func (i *Feature) LatestVersion(ctx context.Context) (*release, error) {
	var tags []release
	url := fmt.Sprintf("https://api.github.com/repos/databrickslabs/%s/releases", i.Name)
	err := httpCall(ctx, url, &tags)
	if err != nil {
		return nil, err
	}
	return &tags[0], nil
}

const CacheDir = ".databricks"

type pythonInstallation struct {
	Version string
	Binary  string
}

func (i *Feature) pythonExecutables(ctx context.Context) ([]pythonInstallation, error) {
	found := []pythonInstallation{}
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, candidate := range paths {
		bin := filepath.Join(candidate, "python3")
		_, err := os.Stat(bin)
		if err != nil && os.IsNotExist(err) {
			continue
		}
		out, err := i.cmd(ctx, bin, "--version")
		if err != nil {
			return nil, err
		}
		words := strings.Split(out, " ")
		found = append(found, pythonInstallation{
			Version: words[len(words)-1],
			Binary:  bin,
		})
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no python3 executables found")
	}
	sort.Slice(found, func(i, j int) bool {
		a := found[i].Version
		b := found[j].Version
		cmp := semver.Compare(a, b)
		if cmp != 0 {
			return cmp < 0
		}
		return a < b
	})
	return found, nil
}

func (i *Feature) installVirtualEnv(ctx context.Context) error {
	_, err := os.Stat(filepath.Join(i.path, "setup.py"))
	if err != nil {
		return err
	}
	pys, err := i.pythonExecutables(ctx)
	if err != nil {
		return err
	}
	python3 := pys[0].Binary
	log.Debugf(ctx, "Creating python virtual environment in %s/%s", i.path, CacheDir)
	_, err = i.cmd(ctx, python3, "-m", "venv", CacheDir)
	if err != nil {
		return fmt.Errorf("create venv: %w", err)
	}

	log.Debugf(ctx, "Installing dependencies from setup.py")
	venvPip := filepath.Join(i.path, CacheDir, "bin", "pip")
	_, err = i.cmd(ctx, venvPip, "install", ".")
	if err != nil {
		return fmt.Errorf("pip install: %w", err)
	}
	return nil
}

func (i *Feature) Run(ctx context.Context, raw []byte) error {
	err := i.installVirtualEnv(ctx)
	if err != nil {
		return err
	}
	// TODO: detect virtual env (also create it on installation),
	// because here we just assume that virtual env is installed.
	python3 := filepath.Join(i.path, CacheDir, "bin", "python")

	// make sure to sync on writing to stdout
	reader, writer := io.Pipe()
	go io.CopyBuffer(os.Stdout, reader, make([]byte, 128))
	defer reader.Close()
	defer writer.Close()

	// pass command parameters down to script as the first arg
	cmd := exec.Command(python3, i.Entrypoint, string(raw))
	cmd.Dir = i.path
	cmd.Stdout = writer
	cmd.Stderr = writer

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go io.CopyBuffer(stdin, os.Stdin, make([]byte, 128))
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func (i *Feature) cmd(ctx context.Context, args ...string) (string, error) {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "running: %s", commandStr)
	cmd := exec.Command(args[0], args[1:]...)
	stdout := &bytes.Buffer{}
	cmd.Dir = i.path
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", commandStr, stdout.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (i *Feature) Install(ctx context.Context) error {
	if i.checkout != nil {
		curr, err := i.cmd(ctx, "git", "tag", "--points-at", "HEAD")
		if err != nil {
			return err
		}
		return fmt.Errorf("%s (%s) is already installed", i.Name, curr)
	}
	url := fmt.Sprintf("https://github.com/databrickslabs/%s", i.Name)
	release, err := i.LatestVersion(ctx)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Installing %s (%s) into %s", url, release.TagName, i.path)
	return git.Clone(ctx, url, release.TagName, i.path)
}
