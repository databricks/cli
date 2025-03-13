package whl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/python"
)

type build struct {
	name string
}

func Build(name string) bundle.Mutator {
	return &build{
		name: name,
	}
}

func (m *build) Name() string {
	return fmt.Sprintf("artifacts.whl.Build(%s)", m.name)
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	artifact, ok := b.Config.Artifacts[m.name]
	if !ok {
		return diag.Errorf("artifact doesn't exist: %s", m.name)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Building %s...", m.name))

	out, err := artifact.Build(ctx)
	if err != nil {
		return diag.Errorf("build failed %s, error: %v, output: %s", m.name, err, out)
	}
	log.Infof(ctx, "Build succeeded")

	dir := artifact.Path
	distPath := filepath.Join(artifact.Path, "dist")
	wheels := python.FindFilesWithSuffixInPath(distPath, ".whl")
	if len(wheels) == 0 {
		return diag.Errorf("cannot find built wheel in %s for package %s", dir, m.name)
	}

	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	patchedWheelsDir := filepath.Join(cacheDir, "patched_wheels")
	err = os.MkdirAll(patchedWheelsDir, 0o700)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	for _, wheel := range wheels {
		patchedWheel := ""
		if artifact.DynamicVersion {
			// TODO: clean up previous versions
			patchedWheel, err = patchwheel.PatchWheel(ctx, wheel, patchedWheelsDir)
			log.Warnf(ctx, "Patching %s: %s %s", wheel, patchedWheel, err)
			if err != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Failed to patch wheel with dynamic version",
					Detail:   fmt.Sprintf("When patching %s encountered an error: %s", wheel, err),
					// TODO: Locations
					// Paths: []Path{"artifacts." + m.name},
				})
			}
		}
		artifact.Files = append(artifact.Files, config.ArtifactFile{
			Source:  wheel,
			Patched: patchedWheel,
		})
	}

	return diags
}
