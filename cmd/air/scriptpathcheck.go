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
// as a script path. It's the CLI's own heuristic for an advisory warning: any
// other first word (echo, env, uv, cd, a custom wrapper) may resolve paths
// itself, so it never warns.
var scriptLaunchers = [][]string{{"python"}, {"python3"}, {"torchrun"}, {"accelerate", "launch"}, {"deepspeed"}, {"bash"}, {"sh"}}

// relativeScriptInCommand returns the script argument when the whole command is
// one launcher invocation, such as `NCCL_DEBUG=INFO python train.py`, whose
// script is a file in snapshotFiles given by a relative path, or "". It warns
// only when the warning is almost certainly right: a command with more than one
// line, a shell operator, a subshell or a command substitution may change
// directory or resolve paths in ways this check can't see, so it never warns.
func relativeScriptInCommand(command string, snapshotFiles []string) string {
	command = strings.TrimSpace(command)
	if strings.ContainsAny(command, "\n;&|()`") {
		return ""
	}
	return scriptArgument(commandWords(command), snapshotFiles)
}

// commandWords returns the unquoted words of command before any `#` comment,
// without its leading NAME=value assignments.
func commandWords(command string) []string {
	var words []string
	for word := range strings.FieldsSeq(command) {
		if strings.HasPrefix(word, "#") {
			break
		}
		word = strings.Trim(word, `"'`)
		if len(words) == 0 && strings.Contains(word, "=") {
			continue
		}
		words = append(words, word)
	}
	return words
}

// scriptArgument returns the script argument of words when they start with a
// script launcher and the script is a snapshot file, or "". Flags are skipped,
// and so are words that can't be a path (no `.`, `/` or `$`), such as the 8 in
// `torchrun --nproc_per_node 8 train.py`. The first word that can be a path is
// the only one checked. `-m` and `-c` run a module or inline code, so there's no
// script; for bash and sh, so does any flag that contains `c`, such as `-lc`.
func scriptArgument(words, snapshotFiles []string) string {
	launcher, args, ok := launcherArgs(words)
	if !ok {
		return ""
	}
	shell := launcher == "bash" || launcher == "sh"
	for _, word := range args {
		switch {
		case word == "-m" || word == "-c" || shell && strings.HasPrefix(word, "-") && strings.Contains(word, "c"):
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

// launcherArgs returns the launcher's first word and the words after the
// launcher when words start with a script launcher.
func launcherArgs(words []string) (string, []string, bool) {
	for _, launcher := range scriptLaunchers {
		if len(words) > len(launcher) && slices.Equal(words[:len(launcher)], launcher) {
			return launcher[0], words[len(launcher):], true
		}
	}
	return "", nil, false
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
