package artifacts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/python"
)

func Build() bundle.Mutator {
	return &build{}
}

type build struct{}

func (m *build) Name() string {
	return "artifacts.Build"
}

func (m *build) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Failed to set up cache directory",
		})
	}

	for _, artifactName := range sortedKeys(b.Config.Artifacts) {
		a := b.Config.Artifacts[artifactName]

		if a.BuildCommand != "" {
			err := doBuild(ctx, artifactName, a)
			if err != nil {
				diags = diags.Extend(diag.FromErr(err))
				break
			}

			if a.Type == "whl" {
				dir := a.Path
				distPath := filepath.Join(a.Path, "dist")
				wheels := python.FindFilesWithSuffixInPath(distPath, ".whl")
				if len(wheels) == 0 {
					diags = diags.Extend(diag.Errorf("cannot find built wheel in %s for package %s", dir, artifactName))
					break
				}
				for _, wheel := range wheels {
					a.Files = append(a.Files, config.ArtifactFile{
						Source: wheel,
					})
				}
			} else {
				log.Warnf(ctx, "%s: Build succeeded", artifactName)
			}

			// We need to expand glob reference after build mutator is applied because
			// if we do it before, any files that are generated by build command will
			// not be included into artifact.Files and thus will not be uploaded.
			// We only do it if BuildCommand was specified because otherwise it should have been done already by artifacts.Prepare()
			diags = diags.Extend(bundle.Apply(ctx, b, expandGlobs{name: artifactName}))

			// After bundle.Apply is called, all of b.Config is recreated and all pointers are invalidated (!)
			a = b.Config.Artifacts[artifactName]

			if diags.HasError() {
				break
			}

		}

		if a.Type == "whl" && a.DynamicVersion && cacheDir != "" {
			b.Metrics.AddSetField("artifact.dynamic_version")
			for ind, artifactFile := range a.Files {
				patchedWheel, extraDiags := makePatchedWheel(ctx, cacheDir, artifactName, artifactFile.Source)
				log.Debugf(ctx, "Patching ind=%d artifactName=%s Source=%s patchedWheel=%s", ind, artifactName, artifactFile.Source, patchedWheel)
				diags = append(diags, extraDiags...)
				if patchedWheel != "" {
					a.Files[ind].Patched = patchedWheel
				}
				if extraDiags.HasError() {
					break
				}
			}
		}
	}

	return diags
}

func doBuild(ctx context.Context, artifactName string, a *config.Artifact) error {
	cmdio.LogString(ctx, fmt.Sprintf("Building %s...", artifactName))

	var executor *exec.Executor
	var err error
	if a.Executable != "" {
		executor, err = exec.NewCommandExecutorWithExecutable(a.Path, a.Executable)
	} else {
		executor, err = exec.NewCommandExecutor(a.Path)
		a.Executable = executor.ShellType()
	}

	if err != nil {
		return err
	}

	out, err := executor.Exec(ctx, a.BuildCommand)
	if err != nil {
		return fmt.Errorf("build failed %s, error: %v, output: %s", artifactName, err, out)
	}

	return nil
}

func makePatchedWheel(ctx context.Context, cacheDir, artifactName, wheel string) (string, diag.Diagnostics) {
	msg := "Failed to patch wheel with dynamic version"
	info, err := patchwheel.ParseWheelFilename(wheel)
	if err != nil {
		return "", []diag.Diagnostic{{
			Severity: diag.Warning,
			Summary:  msg,
			Detail:   fmt.Sprintf("When parsing filename \"%s\" encountered an error: %s", wheel, err),
		}}
	}

	dir := filepath.Join(cacheDir, "patched_wheels", artifactName+"_"+info.Distribution)
	createAndCleanupDirectory(ctx, dir)
	patchedWheel, isBuilt, err := patchwheel.PatchWheel(wheel, dir)
	if err != nil {
		return "", []diag.Diagnostic{{
			Severity: diag.Warning,
			Summary:  msg,
			Detail:   fmt.Sprintf("When patching %s encountered an error: %s", wheel, err),
		}}
	}

	if isBuilt {
		log.Infof(ctx, "Patched wheel (built) %s -> %s", wheel, patchedWheel)
	} else {
		log.Infof(ctx, "Patched wheel (cache) %s -> %s", wheel, patchedWheel)
	}

	return patchedWheel, nil
}

func createAndCleanupDirectory(ctx context.Context, dir string) {
	err := os.MkdirAll(dir, 0o700)
	if err != nil {
		log.Infof(ctx, "Failed to create %s: %s", dir, err)
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Infof(ctx, "Failed to clean up %s: %s", dir, err)
		// ReadDir can return partial results, so continue here
	}
	for _, entry := range files {
		path := filepath.Join(dir, entry.Name())
		err := os.Remove(path)
		if err != nil {
			log.Infof(ctx, "Failed to clean up %s: %s", path, err)
		}
	}
}
