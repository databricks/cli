package aircmd

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"
)

// relativeScriptWarning returns a warning when command (or a container's
// command) refers to a snapshot file by a path relative to code_source.root_path,
// such as `python train.py`, or "" when none does. snapshotFiles are the
// snapshot-relative paths that would be packaged. Whether such a path resolves
// depends on the runtime, which the CLI doesn't own, so the warning suggests
// $CODE_SOURCE_PATH and links the docs rather than asserting how the run behaves.
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
// snapshotFiles, or "". Words are split on whitespace and shell operators, so
// `&&cd` is seen as `cd`. A `--flag=value` word is checked by its value; a
// `NAME=value` word is an environment assignment and skipped. Scanning stops at a
// `#` comment and at a `cd` or `pushd`: once the command changes directory, as in
// `cd "$CODE_SOURCE_PATH" && python train.py`, later relative paths are resolved
// against a directory this check can't know.
func relativeScriptInCommand(command string, snapshotFiles []string) string {
	for word := range strings.FieldsFuncSeq(command, isShellWordBreak) {
		word = strings.Trim(word, `"'`)
		if word == "cd" || word == "pushd" || strings.HasPrefix(word, "#") {
			return ""
		}
		if strings.HasPrefix(word, "-") {
			_, word, _ = strings.Cut(word, "=")
			word = strings.Trim(word, `"'`)
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

func isShellWordBreak(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(";&|()", r)
}

func relativeScriptMessage(field, rel string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Warning: %s refers to %q, a path relative to code_source.root_path. Consider \"$CODE_SOURCE_PATH/%s\" instead. See how command paths and $CODE_SOURCE_PATH work:", field, rel, path.Clean(rel))
	for _, d := range configDocsBases {
		fmt.Fprintf(&b, "\n  %-6s %s%s", d.cloud+":", d.base, yamlConfigDocsPath)
	}
	return b.String()
}

// yamlConfigDocsPath is the docs page that describes command paths and
// $CODE_SOURCE_PATH; see configDocsBases.
const yamlConfigDocsPath = "/machine-learning/ai-runtime/connecting/cli/yaml-config#command"
