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
	"github.com/databricks/cli/libs/cmdio"
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
	cmdio.LogString(ctx, "artifacts.whl.AutoDetect: Detecting Python wheel project...")

	// checking if there is setup.py in the bundle root
	setupPy := filepath.Join(b.Config.Path, "setup.py")
	_, err := os.Stat(setupPy)
	if err != nil {
		cmdio.LogString(ctx, "artifacts.whl.AutoDetect: No Python wheel project found at bundle root folder")
		return nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("artifacts.whl.AutoDetect: Found Python wheel project at %s", b.Config.Path))
	module := extractModuleName(setupPy)

	if b.Config.Artifacts == nil {
		b.Config.Artifacts = make(map[string]*config.Artifact)
	}

	b.Config.Artifacts[module] = &config.Artifact{
		Path: b.Config.Path,
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
