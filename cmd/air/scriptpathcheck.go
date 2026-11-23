package aircmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// relativeScriptWarning returns a warning when command (or a container's
// command) refers to a file by a path relative to code_source.root_path, such as
// `python train.py`, or "" when none does or the check doesn't apply. The runtime
// working directory is not the extracted code directory, so such a run only fails
// after it waits for a GPU.
func relativeScriptWarning(ctx context.Context, cfg *runConfig, configPath string) string {
	if cfg.CodeSource == nil || cfg.CodeSource.Snapshot == nil {
		return ""
	}
	repoPath, err := resolveRootPath(ctx, cfg.CodeSource.Snapshot.RootPath, filepath.Dir(configPath))
	if err != nil {
		return ""
	}
	if cfg.Command != nil {
		if rel := relativeScriptInCommand(repoPath, *cfg.Command); rel != "" {
			return relativeScriptMessage("command", rel)
		}
	}
	for _, c := range cfg.Containers {
		if c.Command == nil {
			continue
		}
		if rel := relativeScriptInCommand(repoPath, *c.Command); rel != "" {
			return relativeScriptMessage(fmt.Sprintf("containers[%s].command", c.Name), rel)
		}
	}
	return ""
}

// relativeScriptInCommand returns the first word of command that names an
// existing file under repoPath, or "". Scanning stops at a `cd`: once the command
// changes directory, as in `cd "$CODE_SOURCE_PATH" && python train.py`, later
// relative paths are resolved against a directory this check can't know.
func relativeScriptInCommand(repoPath, command string) string {
	for word := range strings.FieldsSeq(command) {
		rel := strings.Trim(word, `"'`)
		if rel == "cd" {
			return ""
		}
		if rel == "" || strings.ContainsAny(rel, "$=") || filepath.IsAbs(rel) {
			continue
		}
		info, err := os.Stat(filepath.Join(repoPath, rel))
		if err != nil || info.IsDir() {
			continue
		}
		return rel
	}
	return ""
}

func relativeScriptMessage(field, rel string) string {
	return fmt.Sprintf("Warning: %s refers to %q, a path relative to code_source.root_path, but at runtime the working directory is not your code directory, so this run may fail with \"No such file or directory\". Use \"$CODE_SOURCE_PATH/%s\".", field, rel, filepath.ToSlash(filepath.Clean(rel)))
}
