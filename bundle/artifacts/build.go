package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

func BuildAll() bundle.Mutator {
	return &all{
		name: "Build",
		fn:   buildArtifactByName,
	}
}

type build struct {
	name string
}

func buildArtifactByName(name string) (bundle.Mutator, error) {
	return &build{name}, nil
}

func (m *build) Name() string {
	return fmt.Sprintf("artifacts.Build(%s)", m.name)
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	// Check if source paths are absolute, if not, make them absolute
	for k := range artifact.Files {
		f := &artifact.Files[k]
		if !filepath.IsAbs(f.Source) {
			dirPath := filepath.Dir(artifact.ConfigFilePath)
			f.Source = filepath.Join(dirPath, f.Source)
		}
	}

	// Skip building if build command is not specified or infered
	if artifact.BuildCommand == "" {
		// If no build command was specified or infered and there is no
		// artifact output files specified, artifact is misconfigured
		if len(artifact.Files) == 0 {
			return diag.Errorf("misconfigured artifact: please specify 'build' or 'files' property")
		}
		return nil
	}

	// If artifact path is not provided, use bundle root dir
	if artifact.Path == "" {
		artifact.Path = b.RootPath
	}

	if !filepath.IsAbs(artifact.Path) {
		dirPath := filepath.Dir(artifact.ConfigFilePath)
		artifact.Path = filepath.Join(dirPath, artifact.Path)
	}

	diags := bundle.Apply(ctx, b, getBuildMutator(artifact.Type, m.name))
	if diags.HasError() {
		return diags
	}

	// Expand any glob reference in files source path
	files := make([]config.ArtifactFile, 0, len(artifact.Files))
	for _, f := range artifact.Files {
		matches, err := filepath.Glob(f.Source)
		if err != nil {
			return diags.Extend(diag.Errorf("unable to find files for %s: %v", f.Source, err))
		}

		if len(matches) == 0 {
			return diags.Extend(diag.Errorf("no files found for %s", f.Source))
		}

		for _, match := range matches {
			files = append(files, config.ArtifactFile{
				Source: match,
			})
		}
	}

	artifact.Files = files
	return diags
}
