package project

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/log"
)

func projectInDevMode(ctx context.Context) (*Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	folder, err := folders.FindDirWithLeaf(cwd, "labs.yml")
	if err != nil {
		return nil, err
	}
	log.Debugf(ctx, "Found project under development in: %s", cwd)
	return Load(ctx, filepath.Join(folder, "labs.yml"))
}

func Installed(ctx context.Context) (projects []*Project, err error) {
	root, err := PathInLabs(ctx)
	if errors.Is(err, env.ErrNoHomeEnv) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	labsDir, err := os.ReadDir(root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	projectDev, err := projectInDevMode(ctx)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		projects = append(projects, projectDev)
	}
	for _, v := range labsDir {
		if !v.IsDir() {
			continue
		}
		if projectDev != nil && v.Name() == projectDev.Name {
			continue
		}
		labsYml := filepath.Join(root, v.Name(), "lib", "labs.yml")
		prj, err := Load(ctx, labsYml)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", v.Name(), err)
		}
		projects = append(projects, prj)
	}
	return projects, nil
}
