package aircmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
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
	if env := cfg.Environment; env != nil {
		if env.UnityCatalogImage != "" {
			return ""
		}
		if slices.ContainsFunc(env.Dependencies.list, mayProvideTorch) {
			return ""
		}
	}
	// Resolve the version the way submission does, so a DATABRICKS_DL_RUNTIME_IMAGE
	// override is honored when environment.version is unset.
	cfgVersion, _ := cfg.runtimeVersion()
	version := strings.ToLower(dlRuntimeImage(ctx, cfgVersion))
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
	file, err := findTorchImport(ctx, repoPath, plan)
	if err != nil {
		log.Debugf(ctx, "air: skipping torch import check: %v", err)
		return ""
	}
	if file == "" {
		return ""
	}
	return fmt.Sprintf("Warning: %s contains a torch import, but Standard environment %s does not include PyTorch and no torch dependency was detected, so this code path may fail. Consider using %s%s or adding torch to dependencies.",
		file, version, databricksAIPrefix, version)
}

// mayProvideTorch reports whether an environment.dependencies entry could install
// PyTorch: torch itself, or a requirements file or editable package that the
// check can't see into. Options that only configure package lookup, such as
// --extra-index-url, install nothing.
func mayProvideTorch(dep string) bool {
	dep = strings.TrimSpace(dep)
	if !strings.HasPrefix(dep, "-") {
		return strings.HasPrefix(strings.ToLower(dep), "torch")
	}
	opt := strings.Fields(dep)[0]
	// Short options may be glued to their value, as in -rrequirements.txt.
	if strings.HasPrefix(opt, "-r") || strings.HasPrefix(opt, "-e") {
		return true
	}
	opt, _, _ = strings.Cut(opt, "=")
	return opt == "--requirement" || opt == "--editable"
}

// findTorchImport returns the snapshot-relative path of a .py file that imports
// torch, or "". It reads the same source submission packages: the pinned commit
// for git.commit and git.branch, else the working tree.
func findTorchImport(ctx context.Context, repoPath string, plan snapshotPlan) (string, error) {
	if plan.mode == modeGitArchive {
		return findTorchImportAtCommit(ctx, newGitRepo(repoPath), plan)
	}
	files, err := snapshotFiles(ctx, repoPath, plan.includePaths, plan.isGitRepo)
	if err != nil {
		return "", err
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
			return filepath.ToSlash(f.rel), nil
		}
	}
	return "", nil
}

// torchImportERE is torchImportRe in POSIX ERE syntax for git grep.
const torchImportERE = `^[[:space:]]*(import[[:space:]]+torch|from[[:space:]]+torch)([^[:alnum:]_]|$)`

// findTorchImportAtCommit greps the .py files under the snapshot root (and
// include_paths) at plan.commitSHA.
func findTorchImportAtCommit(ctx context.Context, g gitRepo, plan snapshotPlan) (string, error) {
	bases := []string{plan.subtreePrefix}
	if len(plan.includePaths) > 0 {
		bases = nil
		for _, p := range plan.includePaths {
			bases = append(bases, path.Join(plan.subtreePrefix, filepath.ToSlash(p)))
		}
	}
	args := []string{"grep", "-l", "-I", "--full-name", "-E", torchImportERE, plan.commitSHA, "--"}
	for _, b := range bases {
		args = append(args, ":(top,glob)"+path.Join(b, "**", "*.py"))
	}
	for _, p := range plan.includePaths {
		if strings.HasSuffix(p, ".py") {
			file := path.Join(plan.subtreePrefix, filepath.ToSlash(p))
			args = append(args, ":(top,literal)"+file)
		}
	}
	out, err := g.run(ctx, args...)
	if err != nil {
		// git grep exits 1 when nothing matches.
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	// Output is "<commit>:<repo-relative path>".
	_, file, _ := strings.Cut(first, ":")
	if rel, ok := strings.CutPrefix(file, plan.subtreePrefix); ok && plan.subtreePrefix != "" {
		file = strings.TrimPrefix(rel, "/")
	}
	return file, nil
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
