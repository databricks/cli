package artifacts

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/python"
	"github.com/databricks/cli/libs/utils"
)

func Prepare() bundle.Mutator {
	return &prepare{}
}

type prepare struct{}

func (m *prepare) Name() string {
	return "artifacts.Prepare"
}

func (m *prepare) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := InsertPythonArtifact(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, artifactName := range utils.SortedKeys(b.Config.Artifacts) {
		artifact := b.Config.Artifacts[artifactName]
		if artifact == nil {
			l := b.Config.GetLocation("artifacts." + artifactName)
			logdiag.LogDiag(ctx, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Artifact not properly configured",
				Detail:    "please specify artifact properties",
				Locations: []dyn.Location{l},
			})
			continue
		}
		b.Metrics.AddBoolValue(metrics.ArtifactBuildCommandIsSet, artifact.BuildCommand != "")
		b.Metrics.AddBoolValue(metrics.ArtifactFilesIsSet, len(artifact.Files) != 0)

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

		if artifact.Type == "whl" {
			if artifact.BuildCommand == "" && len(artifact.Files) == 0 {
				artifact.BuildCommand = python.GetExecutable() + " setup.py bdist_wheel"
			}

			// Wheel builds write to `./dist`. Pick up all wheel files by default if nothing is specified.
			if len(artifact.Files) == 0 {
				artifact.Files = []config.ArtifactFile{
					{
						Source: filepath.Join(artifact.Path, "dist", "*.whl"),
					},
				}
			}
		}

		if !filepath.IsAbs(artifact.Path) {
			artifact.Path = filepath.Join(dirPath, artifact.Path)
		}

		if artifact.BuildCommand == "" && len(artifact.Files) == 0 {
			logdiag.LogError(ctx, errors.New("misconfigured artifact: please specify 'build' or 'files' property"))
		}

		if len(artifact.Files) > 0 && artifact.BuildCommand == "" {
			bundle.ApplyContext(ctx, b, expandGlobs{name: artifactName})
		}

		if logdiag.HasError(ctx) {
			break
		}
	}

	return nil
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
