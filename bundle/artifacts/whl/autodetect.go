package whl

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type detectPkg struct{}

func DetectPackage() bundle.Mutator {
	return &detectPkg{}
}

func (m *detectPkg) Name() string {
	return "artifacts.whl.AutoDetect"
}

func (m *detectPkg) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Artifacts != nil {
		log.Debugf(ctx, "artifacts block is defined, skipping auto-detecting")
		return nil
	}

	tasks := libraries.FindTasksWithLocalLibraries(b)
	if len(tasks) == 0 {
		log.Infof(ctx, "No local tasks in databricks.yml config, skipping auto detect")
		return nil
	}

	log.Infof(ctx, "Detecting Python wheel project...")

	// checking if there is setup.py in the bundle root
	setupPy := filepath.Join(b.BundleRootPath, "setup.py")
	_, err := os.Stat(setupPy)
	if err != nil {
		log.Infof(ctx, "No Python wheel project found at bundle root folder")
		return nil
	}

	log.Infof(ctx, "Found Python wheel project at %s", b.BundleRootPath)

	pkgPath, err := filepath.Abs(b.BundleRootPath)
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Artifacts = make(map[string]*config.Artifact)
	b.Config.Artifacts["python_artifact"] = &config.Artifact{
		Path: pkgPath,
		Type: config.ArtifactPythonWheel,
		// BuildCommand will be set by bundle/artifacts/whl/infer.go to "python3 setup.py bdist_wheel"
	}

	return nil
}
