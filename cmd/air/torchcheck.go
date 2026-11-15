package aircmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/log"
)

// The Standard environment dropped PyTorch in this version.
const firstStandardVersionWithoutTorch = 5

// Bounds keep the check cheap on large code sources; it is a hint, not a guarantee.
const (
	torchScanMaxFiles     = 200
	torchScanMaxFileBytes = 1 << 20
)

var torchImportRe = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^\s*(import\s+torch\b|from\s+torch\b)`)
})

// missingTorchWarning returns a warning when the code_source imports torch but
// the environment can't provide it, or "" when it can or the check doesn't apply.
// A run like that fails only after it waits for a GPU and then retries, so
// catching it locally saves the whole queue-and-retry cycle.
func missingTorchWarning(ctx context.Context, cfg *runConfig, configPath string) string {
	if len(cfg.Containers) > 0 || cfg.CodeSource == nil || cfg.CodeSource.Snapshot == nil {
		return ""
	}
	version := "6"
	if env := cfg.Environment; env != nil {
		if env.UnityCatalogImage != "" {
			return ""
		}
		if env.Version.set {
			version = strings.ToLower(env.Version.raw)
		}
		for _, dep := range env.Dependencies.list {
			// A requirements file or an explicit torch entry may provide it.
			if strings.HasPrefix(dep, "-") || strings.HasPrefix(strings.ToLower(dep), "torch") {
				return ""
			}
		}
	}
	if strings.HasPrefix(version, databricksAIPrefix) {
		return ""
	}
	if major, err := strconv.Atoi(version); err != nil || major < firstStandardVersionWithoutTorch {
		return ""
	}

	snap := cfg.CodeSource.Snapshot
	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return ""
	}
	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repoPath), snap.Git, snap.IncludePaths)
	if err != nil {
		return ""
	}
	files, err := snapshotFiles(ctx, repoPath, plan.includePaths, plan.isGitRepo)
	if err != nil {
		log.Debugf(ctx, "air: skipping torch import check: %v", err)
		return ""
	}
	scanned := 0
	for _, f := range files {
		if !strings.HasSuffix(f.rel, ".py") || f.size > torchScanMaxFileBytes {
			continue
		}
		if scanned++; scanned > torchScanMaxFiles {
			break
		}
		if importsTorch(filepath.Join(repoPath, f.rel)) {
			return fmt.Sprintf("Warning: %s imports torch, but environment version %s (Standard) does not include PyTorch, so the run would fail after waiting for a GPU. Set environment.version: %s6, or add torch to environment.dependencies.",
				filepath.ToSlash(f.rel), version, databricksAIPrefix)
		}
	}
	return ""
}

// importsTorch reports whether a top-level import line in the file imports torch.
func importsTorch(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if torchImportRe().MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}
