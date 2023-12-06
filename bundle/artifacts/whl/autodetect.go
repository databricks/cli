package whl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

type detectPkg struct {
}

func DetectPackage() bundle.Mutator {
	return &detectPkg{}
}

func (m *detectPkg) Name() string {
	return "artifacts.whl.AutoDetect"
}

func (m *detectPkg) Apply(ctx context.Context, b *bundle.Bundle) error {
	wheelTasks := libraries.FindAllWheelTasksWithLocalLibraries(b)
	if len(wheelTasks) == 0 {
		log.Infof(ctx, "No local wheel tasks in databricks.yml config, skipping auto detect")
		return nil
	}
	cmdio.LogString(ctx, "Detecting Python wheel project...")

	// checking if there is setup.py in the bundle root
	setupPy := filepath.Join(b.Config.Path, "setup.py")
	_, err := os.Stat(setupPy)
	if err != nil {
		cmdio.LogString(ctx, "No Python wheel project found at bundle root folder")
		return nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("Found Python wheel project at %s", b.Config.Path))
	module := extractModuleName(setupPy)

	if b.Config.Artifacts == nil {
		b.Config.Artifacts = make(map[string]*config.Artifact)
	}

	pkgPath, err := filepath.Abs(b.Config.Path)
	if err != nil {
		return err
	}
	b.Config.Artifacts[module] = &config.Artifact{
		Path: pkgPath,
		Type: config.ArtifactPythonWheel,
	}

	return nil
}

func extractModuleName(setupPy string) string {
	bytes, err := os.ReadFile(setupPy)
	if err != nil {
		return randomName()
	}

	content := string(bytes)
	r := regexp.MustCompile(`name=['"](.*)['"]`)
	matches := r.FindStringSubmatch(content)
	if len(matches) == 0 {
		return randomName()
	}
	return matches[1]
}

func randomName() string {
	return fmt.Sprintf("artifact%d", time.Now().Unix())
}
