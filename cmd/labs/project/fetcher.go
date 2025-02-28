package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/libs/log"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type installable interface {
	Install(ctx context.Context) error
}

type devInstallation struct {
	*Project
	*cobra.Command
}

func (d *devInstallation) Install(ctx context.Context) error {
	if d.Installer == nil {
		return nil
	}
	_, err := d.Installer.validLogin(d.Command)
	if errors.Is(err, ErrNoLoginConfig) {
		cfg, err := d.Installer.envAwareConfig(ctx)
		if err != nil {
			return err
		}
		lc := &loginConfig{Entrypoint: d.Installer.Entrypoint}
		_, err = lc.askWorkspace(ctx, cfg)
		if err != nil {
			return fmt.Errorf("ask for workspace: %w", err)
		}
		err = lc.askAccountProfile(ctx, cfg)
		if err != nil {
			return fmt.Errorf("ask for account: %w", err)
		}
		err = lc.EnsureFoldersExist()
		if err != nil {
			return fmt.Errorf("folders: %w", err)
		}
		err = lc.save(ctx)
		if err != nil {
			return fmt.Errorf("save: %w", err)
		}
	}
	return d.Installer.runHook(d.Command)
}

func NewInstaller(cmd *cobra.Command, name string, offlineInstall bool) (installable, error) {
	if name == "." {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("working directory: %w", err)
		}
		prj, err := Load(cmd.Context(), filepath.Join(wd, "labs.yml"))
		if err != nil {
			return nil, fmt.Errorf("load: %w", err)
		}
		cmd.PrintErrln(color.YellowString("Installing %s in development mode from %s", prj.Name, wd))
		return &devInstallation{
			Project: prj,
			Command: cmd,
		}, nil
	}
	name, version, ok := strings.Cut(name, "@")
	if !ok {
		version = "latest"
	}
	f := &fetcher{name}

	version, err := f.checkReleasedVersions(cmd, version, offlineInstall)
	if err != nil {
		return nil, fmt.Errorf("version: %w", err)
	}

	prj, err := f.loadRemoteProjectDefinition(cmd, version, offlineInstall)
	if err != nil {
		return nil, fmt.Errorf("remote: %w", err)
	}

	return &installer{
		Project:        prj,
		version:        version,
		cmd:            cmd,
		offlineInstall: offlineInstall,
	}, nil
}

func NewUpgrader(cmd *cobra.Command, name string) (*installer, error) {
	f := &fetcher{name}
	version, err := f.checkReleasedVersions(cmd, "latest", false)
	if err != nil {
		return nil, fmt.Errorf("version: %w", err)
	}
	prj, err := f.loadRemoteProjectDefinition(cmd, version, false)
	if err != nil {
		return nil, fmt.Errorf("remote: %w", err)
	}
	prj.folder, err = PathInLabs(cmd.Context(), name)
	if err != nil {
		return nil, err
	}
	return &installer{
		Project: prj,
		version: version,
		cmd:     cmd,
	}, nil
}

type fetcher struct {
	name string
}

func (f *fetcher) checkReleasedVersions(cmd *cobra.Command, version string, offlineInstall bool) (string, error) {
	ctx := cmd.Context()
	cacheDir, err := PathInLabs(ctx, f.name, "cache")
	if err != nil {
		return "", err
	}
	// `databricks labs isntall X` doesn't know which exact version to fetch, so first
	// we fetch all versions and then pick the latest one dynamically.
	var versions github.Versions
	versions, err = github.NewReleaseCache("databrickslabs", f.name, cacheDir, offlineInstall).Load(ctx)
	if err != nil {
		return "", fmt.Errorf("versions: %w", err)
	}
	for _, v := range versions {
		if v.Version == version {
			return version, nil
		}
	}
	if version == "latest" && len(versions) > 0 {
		log.Debugf(ctx, "Latest %s version is: %s", f.name, versions[0].Version)
		return versions[0].Version, nil
	}
	cmd.PrintErrln(color.YellowString("[WARNING] Installing unreleased version: %s", version))
	return version, nil
}

func (i *fetcher) loadRemoteProjectDefinition(cmd *cobra.Command, version string, offlineInstall bool) (*Project, error) {
	ctx := cmd.Context()
	var raw []byte
	var err error
	if !offlineInstall {
		raw, err = github.ReadFileFromRef(ctx, "databrickslabs", i.name, version, "labs.yml")
		if err != nil {
			return nil, fmt.Errorf("read labs.yml from GitHub: %w", err)
		}
	} else {
		libDir, _ := PathInLabs(ctx, i.name, "lib")
		fileName := filepath.Join(libDir, "labs.yml")
		raw, err = os.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("read labs.yml from local path %s: %w", libDir, err)
		}
	}

	return readFromBytes(ctx, raw)
}
