package agentconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
)

// cliCommand is one run of an agent CLI: its command line, and the options of the run, which set the variable that the
// CLI must see (see homeOpts).
type cliCommand struct {
	argv []string
	opts []process.ExecOption
}

// runCLI runs one agent CLI command. It is a variable so tests can run a stand-in for the CLI.
var runCLI = func(ctx context.Context, c cliCommand) error {
	return runWithTimeout(ctx, cliTimeout, c)
}

// cliTimeout bounds one add or remove. Each one edits a single config file and reads no input. A Node-based CLI can take
// seconds to start, so the bound is generous; it keeps a CLI that hangs from holding up add or remove.
const cliTimeout = 30 * time.Second

// cliWaitDelay bounds how long a run waits for the output of the CLI once the CLI has exited or been killed. A launcher
// can leave a child that holds the output open, as an npm shim or Gemini relaunching itself does, and without this
// bound the wait would outlast the timeout. The CLI is not put in a process group, since that would take it out of the
// terminal's foreground group: Ctrl-C would then stop databricks and leave the CLI running. So a child of the CLI can
// outlive the timeout. The read-back after the run reads the file whatever the CLI did.
const cliWaitDelay = 5 * time.Second

// runWithTimeout runs c with the environment of ctx. A run that exceeds timeout fails with context.DeadlineExceeded. It
// starts in an empty directory, so that a project config in the working directory does not change what the CLI does.
// The error of a timeout does not carry the process error, whose text is the whole argv.
func runWithTimeout(ctx context.Context, timeout time.Duration, c cliCommand) error {
	dir, err := os.MkdirTemp("", "databricks-agent-cli-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	opts := append([]process.ExecOption{process.WithWaitDelay(cliWaitDelay), process.WithDir(dir)}, c.opts...)
	_, err = process.Background(cctx, c.argv, opts...)
	// Only the timeout of this run is labelled: a deadline of the caller's ctx is returned as the process error.
	if err != nil && ctx.Err() == nil && errors.Is(cctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w after %s", context.DeadlineExceeded, timeout)
	}
	return err
}

// addViaCLI adds the server through the agent's own CLI, when the agent has an add command and the CLI can run here. It
// reports whether it finished the add. When it reports false with a nil error, the caller writes the entry itself.
//
// Once the CLI has run, the file decides what happens, whatever the CLI's exit status, and afterwards path holds the
// entry asked for:
//   - the entry is there and is the one asked for: done. A non-zero exit is logged.
//   - otherwise the direct writer writes the entry to path. An entry there that differs (another launch key, or another
//     type) was written by this run's CLI, since it was absent before, and is replaced. When the CLI wrote nothing to
//     path, it failed, or it wrote another file, the entry is added there.
//
// entry is the entry to be written, normalized to JSON values as a file holds them, so that it compares with what the
// CLI wrote whatever Go types the spec used for it.
func addViaCLI(ctx context.Context, a *agents.Agent, spec *mcpconfig.Spec, path, name, cmd string, args []string, env map[string]string, entry map[string]any) (bool, error) {
	if spec.AddArgs == nil || flagLike(name) || isSymlink(path) {
		return false, nil
	}
	sub := spec.AddArgs(name, cmd, args, env)
	if sub == nil {
		return false, nil
	}
	call, ok, err := cliCall(ctx, a, spec, sub)
	if !ok {
		return false, err
	}
	if runErr := runCLI(ctx, call); runErr != nil {
		log.Debugf(ctx, "%v", cliFailure(a, sub, runErr))
	}
	got, found, err := entryAt(spec, path, name)
	if err != nil {
		return true, err
	}
	if !found {
		log.Debugf(ctx, "%s did not write %s; the direct writer adds the entry", commandName(a, sub), filepath.ToSlash(path))
		return true, putDirect(spec, path, name, entry)
	}
	asked, err := isAsked(spec, got, entry)
	if err != nil {
		return true, err
	}
	if asked {
		return true, nil
	}
	return true, putDirect(spec, path, name, entry)
}

// removeViaCLI removes the server through the agent's own CLI, when the agent has a remove command, the CLI is on PATH,
// and path is the file the agent reads now. The caller has already checked that the entry is ours. It reports whether
// the entry is gone afterwards. When the CLI fails, or exits 0 and leaves the entry in path (it reads another file, say),
// it reports false, and the caller removes the entry with the direct writer: the entry is ours, and the user asked for
// it to go. The CLI's failure is logged.
func removeViaCLI(ctx context.Context, a *agents.Agent, spec *mcpconfig.Spec, path, name string) (bool, error) {
	if spec.RemoveArgs == nil || flagLike(name) || isSymlink(path) {
		return false, nil
	}
	current, err := configPath(ctx, spec)
	if err != nil {
		// Without the current file, the entry is not in the file the CLI reads.
		log.Debugf(ctx, "%v; the direct writer removes the entry", err)
		return false, nil
	}
	if path != current {
		// The entry is in a file the agent no longer reads, so the CLI would not find it.
		return false, nil
	}
	sub := spec.RemoveArgs(name)
	call, ok, err := cliCall(ctx, a, spec, sub)
	if !ok {
		return false, err
	}
	if err := runCLI(ctx, call); err != nil {
		log.Debugf(ctx, "%v; the direct writer removes the entry", cliFailure(a, sub, err))
	}
	_, found, err := entryAt(spec, path, name)
	if err != nil {
		return false, err
	}
	return !found, nil
}

// isSymlink reports whether path is a symbolic link. A CLI that renames a temp file over its config replaces the link with
// a regular file, so the direct writer makes the change instead: it writes through the link.
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// flagLike reports whether name starts with a dash. The CLIs would read such a name as one of their flags, so the direct
// writer makes the change.
func flagLike(name string) bool {
	return strings.HasPrefix(name, "-")
}

// isAsked reports whether got, the entry that the file holds after a CLI ran, is the one asked for: ours, with the
// launch keys of want. An agent may write "env": {} for a server with no variables, which counts as no env.
func isAsked(spec *mcpconfig.Spec, got, want map[string]any) (bool, error) {
	if !mcpconfig.IsDatabricksEntry(got) {
		return false, nil
	}
	keys := ownedKeys(spec)
	return sameEntry(launchOf(got, keys), launchOf(want, keys))
}

// launchOf returns the launch keys of entry, without an empty env table. An agent's CLI may write an empty env table
// for a server with no variables, and that counts as none.
func launchOf(entry map[string]any, keys []string) map[string]any {
	out := launchKeysOf(entry, keys)
	for _, key := range mcpconfig.EnvKeys {
		if vars, ok := out[key].(map[string]any); ok && len(vars) == 0 {
			delete(out, key)
		}
	}
	return out
}

// entryAt returns the entry named name in the config file at path, and whether the file holds one.
func entryAt(spec *mcpconfig.Spec, path, name string) (map[string]any, bool, error) {
	doc, err := load(spec, path)
	if err != nil {
		return nil, false, err
	}
	entry, found, err := doc.entry(rootPath(spec), name)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return entry, found, nil
}

// cliCall returns the run of agent a's CLI with the arguments sub. It reports false when the direct writer must make the
// change: the CLI is not on PATH, or it is a batch file that cannot take these arguments.
func cliCall(ctx context.Context, a *agents.Agent, spec *mcpconfig.Spec, sub []string) (cliCommand, bool, error) {
	bin, err := cliPath(ctx, a)
	if err != nil || bin == "" {
		return cliCommand{}, false, err
	}
	argv := append([]string{bin}, sub...)
	if !batchSafe(argv) {
		return cliCommand{}, false, nil
	}
	opts, err := homeOpts(ctx, spec)
	if err != nil {
		return cliCommand{}, false, err
	}
	return cliCommand{argv: argv, opts: opts}, true, nil
}

// homeOpts returns the option that gives the CLI the agent's config directory variable, or nil when the agent has none.
// The value is made absolute, since the CLI runs in an empty directory, and configPath resolved it against the directory
// of the caller. It is cleaned, as the config path that the read-back uses is, so that the CLI and the read-back name
// the same directory. An empty value is unset rather than passed on: a CLI that reads an empty variable as set may then write
// relative to its working directory.
func homeOpts(ctx context.Context, spec *mcpconfig.Spec) ([]process.ExecOption, error) {
	if spec.HomeEnv == "" {
		return nil, nil
	}
	home := env.Get(ctx, spec.HomeEnv)
	if home == "" {
		return []process.ExecOption{process.WithoutEnv(spec.HomeEnv)}, nil
	}
	abs, err := mcpconfig.AbsoluteAsWritten(home)
	if err != nil {
		return nil, err
	}
	return []process.ExecOption{process.WithEnv(spec.HomeEnv, filepath.Clean(abs))}, nil
}

// cliPath returns the path of agent a's CLI, or "" when no CLI of that name is on PATH. A CLI that resolves only
// through the current directory is treated the same: it is not run, and the direct writer makes the change.
func cliPath(ctx context.Context, a *agents.Agent) (string, error) {
	path, err := agents.ResolveBinary(a)
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, exec.ErrDot) {
		log.Debugf(ctx, "%s CLI is not usable on PATH: %v", a.DisplayName, err)
		return "", nil
	}
	return path, err
}

// batchSafe reports whether argv reaches its program intact. npm installs many agent CLIs as .cmd shims, and Go hands
// the arguments of a batch file to cmd.exe unquoted (see os/exec). A double quote, as in the JSON that claude add-json
// takes, or a character that cmd.exe expands, splits or ends a line on, does not survive that. Nor does a space: Go
// quotes an argument that has one, and cmd.exe then strips the first and last quote, which may be the ones around
// argv[0] when the path of the shim has a space.
func batchSafe(argv []string) bool {
	ext := strings.ToLower(filepath.Ext(argv[0]))
	if ext != ".cmd" && ext != ".bat" {
		return true
	}
	return !slices.ContainsFunc(argv, func(arg string) bool {
		return strings.ContainsAny(arg, " \"%!^&|<>\r\n")
	})
}

// cliError is a failed agent CLI command. Its text names the command and the stderr the CLI printed. It does not print
// the argv, which a process error would: for claude add-json that is the whole server entry as JSON. So it is not
// the installer's stderrOf, whose fallback is the process error's own text.
type cliError struct {
	command string
	err     error
}

func (e *cliError) Error() string {
	perr, ok := errors.AsType[*process.ProcessError](e.err)
	if !ok {
		return e.command + " failed: " + e.err.Error()
	}
	if s := strings.TrimSpace(perr.Stderr); s != "" {
		return e.command + " failed: " + s
	}
	return e.command + " failed: " + perr.Err.Error()
}

func (e *cliError) Unwrap() error { return e.err }

// cliFailure reports a CLI command that failed. err stays in the chain, so that a timeout or a process error can be
// matched with errors.Is or errors.As.
func cliFailure(a *agents.Agent, sub []string, err error) error {
	return &cliError{command: commandName(a, sub), err: err}
}

// commandName names an agent CLI command for an error: the binary and its subcommand words, such as "claude mcp
// add-json". The first two arguments of sub are those words (see mcpconfig.Spec).
func commandName(a *agents.Agent, sub []string) string {
	return a.Binary + " " + strings.Join(sub[:2], " ")
}
