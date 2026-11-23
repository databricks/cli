package aircmd

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// relativeScriptWarning returns a warning when command (or a container's
// command) refers to a snapshot file by a path relative to code_source.root_path,
// such as `python train.py`, or "" when none does. snapshotFiles are the
// snapshot-relative paths that would be packaged. The runtime working directory
// is not the extracted code directory, so such a run only fails after it waits
// for a GPU.
func relativeScriptWarning(cfg *runConfig, snapshotFiles []string) string {
	if cfg.Command != nil {
		if rel := relativeScriptInCommand(*cfg.Command, snapshotFiles); rel != "" {
			return relativeScriptMessage("command", rel)
		}
	}
	for _, c := range cfg.Containers {
		if c.Command == nil {
			continue
		}
		if rel := relativeScriptInCommand(*c.Command, snapshotFiles); rel != "" {
			return relativeScriptMessage(fmt.Sprintf("containers[%s].command", c.Name), rel)
		}
	}
	return ""
}

// relativeScriptInCommand returns the first word of command that names a file in
// snapshotFiles, or "". A `--flag=value` word is checked by its value; a
// `NAME=value` word is an environment assignment and skipped. Scanning stops at a
// `cd` or `pushd`: once the command changes directory, as in
// `cd "$CODE_SOURCE_PATH" && python train.py`, later relative paths are resolved
// against a directory this check can't know.
func relativeScriptInCommand(command string, snapshotFiles []string) string {
	for word := range strings.FieldsSeq(command) {
		word = strings.Trim(word, `"'();&|`)
		if word == "cd" || word == "pushd" {
			return ""
		}
		if strings.HasPrefix(word, "-") {
			_, word, _ = strings.Cut(word, "=")
		} else if strings.Contains(word, "=") {
			continue
		}
		if word == "" || strings.Contains(word, "$") || strings.HasPrefix(word, "/") {
			continue
		}
		if slices.Contains(snapshotFiles, path.Clean(word)) {
			return word
		}
	}
	return ""
}

func relativeScriptMessage(field, rel string) string {
	return fmt.Sprintf("Warning: %s refers to %q, a path relative to code_source.root_path, but at runtime the working directory is not your code directory, so this run may fail with \"No such file or directory\". Use \"$CODE_SOURCE_PATH/%s\".", field, rel, path.Clean(rel))
}
