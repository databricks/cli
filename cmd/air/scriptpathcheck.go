package aircmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// relativeScriptWarning returns a warning when command refers to a file by a
// path relative to code_source.root_path, such as `python train.py`, or "" when
// it doesn't or the check doesn't apply. The runtime working directory is not the
// extracted code directory, so such a run only fails after it waits for a GPU.
func relativeScriptWarning(ctx context.Context, cfg *runConfig, configPath string) string {
	if cfg.Command == nil || cfg.CodeSource == nil || cfg.CodeSource.Snapshot == nil {
		return ""
	}
	repoPath, err := resolveRootPath(ctx, cfg.CodeSource.Snapshot.RootPath, filepath.Dir(configPath))
	if err != nil {
		return ""
	}
	for word := range strings.FieldsSeq(*cfg.Command) {
		rel := strings.Trim(word, `"'`)
		if rel == "" || strings.ContainsAny(rel, "$=") || filepath.IsAbs(rel) {
			continue
		}
		info, err := os.Stat(filepath.Join(repoPath, rel))
		if err != nil || info.IsDir() {
			continue
		}
		return fmt.Sprintf("Warning: command refers to %q, a path relative to code_source.root_path, but at runtime the working directory is not your code directory, so this run may fail with \"No such file or directory\". Use \"$CODE_SOURCE_PATH/%s\".", rel, filepath.ToSlash(filepath.Clean(rel)))
	}
	return ""
}
