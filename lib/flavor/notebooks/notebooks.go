package notebooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/lib/fileset"
	"github.com/databricks/bricks/lib/flavor"
)

type Notebooks struct {
	Folder string `json:"folder"`
}

func (n *Notebooks) Detected(prj flavor.Project) bool {
	_, err := os.Stat(filepath.Join(prj.Root(), n.Folder))
	return err == nil
}

func (n *Notebooks) LocalArtifacts(ctx context.Context, prj flavor.Project) (flavor.Artifacts, error) {
	all := flavor.Artifacts{}
	found, err := fileset.RecursiveChildren(filepath.Join(prj.Root(), n.Folder), prj.Root())
	if err != nil {
		return nil, fmt.Errorf("list notebooks: %w", err)
	}
	for _, f := range found {
		if !f.MustMatch("# Databricks notebook source") {
			continue
		}
		all = append(all, flavor.Artifact{
			Notebook: &flavor.Notebook{
				LocalAbsolute:  f.Absolute,
				RemoteRelative: f.Relative, // TODO: TBD behavior with regards to isolation
			},
			Flavor: n,
		})
	}
	return all, nil
}
