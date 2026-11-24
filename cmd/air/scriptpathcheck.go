package aircmd

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// relativeScriptWarning returns a warning when command (or a container's
// command) refers to a snapshot file by a path relative to code_source.root_path,
// such as `python train.py`, or "" when none seems to. snapshotFiles are the
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

// scriptLaunchers are the commands whose first non-flag argument this check reads
// as a script path. It's the CLI's own heuristic for an advisory warning: other
// first words (echo, env, uv, custom wrappers) may resolve paths themselves, so
// they never warn.
var scriptLaunchers = [][]string{{"python"}, {"python3"}, {"torchrun"}, {"accelerate", "launch"}, {"deepspeed"}, {"bash"}, {"sh"}}

// relativeScriptInCommand returns the script argument of the first simple
// command (split on shell operators) that runs a script launcher on a file in
// snapshotFiles by a relative path, or "". Scanning stops at a `#` comment and at
// a `cd` or `pushd`: once the command changes directory, as in
// `cd "$CODE_SOURCE_PATH" && python train.py`, later relative paths are resolved
// against a directory this check can't know.
func relativeScriptInCommand(command string, snapshotFiles []string) string {
	for simple := range strings.FieldsFuncSeq(command, isShellOperator) {
		words, comment := commandWords(simple)
		if len(words) > 0 && (words[0] == "cd" || words[0] == "pushd") {
			return ""
		}
		if rel := scriptArgument(words, snapshotFiles); rel != "" || comment {
			return rel
		}
	}
	return ""
}

// commandWords returns the unquoted words of a simple command without its
// leading NAME=value assignments, and whether a `#` comment ended it.
func commandWords(simple string) (words []string, comment bool) {
	for word := range strings.FieldsSeq(simple) {
		if strings.HasPrefix(word, "#") {
			return words, true
		}
		word = strings.Trim(word, `"'`)
		if len(words) == 0 && strings.Contains(word, "=") {
			continue
		}
		words = append(words, word)
	}
	return words, false
}

// scriptArgument returns the script argument of words when they start with a
// script launcher and the script is a snapshot file, or "". Flags are skipped,
// and so are words that can't be a path (no `.`, `/` or `$`), such as the 8 in
// `torchrun --nproc_per_node 8 train.py`. `-m` and `-c` run a module or inline
// code, so there's no script.
func scriptArgument(words, snapshotFiles []string) string {
	args, ok := launcherArgs(words)
	if !ok {
		return ""
	}
	for _, word := range args {
		switch {
		case word == "-m" || word == "-c":
			return ""
		case strings.HasPrefix(word, "-") || !strings.ContainsAny(word, "./$"):
			continue
		case !strings.Contains(word, "$") && !strings.HasPrefix(word, "/") && slices.Contains(snapshotFiles, path.Clean(word)):
			return word
		default:
			return ""
		}
	}
	return ""
}

func launcherArgs(words []string) ([]string, bool) {
	for _, launcher := range scriptLaunchers {
		if len(words) > len(launcher) && slices.Equal(words[:len(launcher)], launcher) {
			return words[len(launcher):], true
		}
	}
	return nil, false
}

func isShellOperator(r rune) bool {
	return strings.ContainsRune(";&|()", r)
}

func relativeScriptMessage(field, rel string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Warning: %s may refer to %q relative to code_source.root_path. If it does, \"$CODE_SOURCE_PATH/%s\" is safer. If the command already handles its working directory, ignore this. Docs on command paths and $CODE_SOURCE_PATH:", field, rel, path.Clean(rel))
	for _, d := range configDocsBases {
		fmt.Fprintf(&b, "\n  %-6s %s%s", d.cloud+":", d.base, yamlConfigDocsPath)
	}
	return b.String()
}

// yamlConfigDocsPath is the docs page that describes command paths and
// $CODE_SOURCE_PATH; see configDocsBases.
const yamlConfigDocsPath = "/machine-learning/ai-runtime/connecting/cli/yaml-config#command"
