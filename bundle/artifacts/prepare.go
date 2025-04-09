package artifacts

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/python"
)

func Prepare() bundle.Mutator {
	return &prepare{}
}

type prepare struct{}

func (m *prepare) Name() string {
	return "artifacts.Prepare"
}

func (m *prepare) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	err := InsertPythonArtifact(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	removeFolders := make(map[string]bool, len(b.Config.Artifacts))
	cleanupWheelFolders := make(map[string]bool, len(b.Config.Artifacts))

	for _, artifactName := range sortedKeys(b.Config.Artifacts) {
		artifact := b.Config.Artifacts[artifactName]
		b.Metrics.AddBoolValue(metrics.ArtifactBuildCommandIsSet, artifact.BuildCommand != "")
		b.Metrics.AddBoolValue(metrics.ArtifactFilesIsSet, len(artifact.Files) != 0)

		if artifact.Type == "whl" {
			if artifact.BuildCommand == "" && len(artifact.Files) == 0 {
				artifact.BuildCommand = python.GetExecutable() + " setup.py bdist_wheel"
			}
		}

		l := b.Config.GetLocation("artifacts." + artifactName)
		dirPath := filepath.Dir(l.File)

		// Check if source paths are absolute, if not, make them absolute
		for k := range artifact.Files {
			f := &artifact.Files[k]
			if !filepath.IsAbs(f.Source) {
				f.Source = filepath.Join(dirPath, f.Source)
			}
		}

		if artifact.Path == "" {
			artifact.Path = b.BundleRootPath
		}

		if !filepath.IsAbs(artifact.Path) {
			artifact.Path = filepath.Join(dirPath, artifact.Path)
		}

		if artifact.Type == "whl" && artifact.BuildCommand != "" {
			dir := artifact.Path
			removeFolders[filepath.Join(dir, "dist")] = true
			cleanupWheelFolders[dir] = true
		}

		if artifact.BuildCommand == "" && len(artifact.Files) == 0 {
			diags = diags.Extend(diag.Errorf("misconfigured artifact: please specify 'build' or 'files' property"))
		}

		if len(artifact.Files) > 0 && artifact.BuildCommand == "" {
			diags = diags.Extend(bundle.Apply(ctx, b, expandGlobs{name: artifactName}))
		}

		if diags.HasError() {
			break
		}
	}

	if diags.HasError() {
		return diags
	}

	for _, dir := range sortedKeys(removeFolders) {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Infof(ctx, "Failed to remove %s: %s", dir, err)
		}
	}

	for _, dir := range sortedKeys(cleanupWheelFolders) {
		log.Infof(ctx, "Cleaning up Python build artifacts in %s", dir)
		python.CleanupWheelFolder(dir)
	}

	return diags
}

func InsertPythonArtifact(ctx context.Context, b *bundle.Bundle) error {
	if b.Config.Artifacts != nil {
		log.Debugf(ctx, "artifacts block is defined, skipping auto-detecting")
		return nil
	}

	tasks := libraries.FindTasksWithLocalLibraries(b)
	if len(tasks) == 0 {
		log.Infof(ctx, "No local tasks in databricks.yml config, skipping auto detect")
		return nil
	}

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
		return err
	}

	b.Config.Artifacts = make(map[string]*config.Artifact)
	b.Config.Artifacts["python_artifact"] = &config.Artifact{
		Path: pkgPath,
		Type: config.ArtifactPythonWheel,
		// BuildCommand will be auto set by later code
	}

	return nil
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
