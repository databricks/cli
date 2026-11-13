package agentconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sleepingCLIEnv tells TestSleepingCLI to block, so that the timeout test can run it as a child process.
const sleepingCLIEnv = "AGENTCONFIG_TEST_SLEEP"

func TestMain(m *testing.M) {
	// No agent CLI is on PATH unless a test puts one there, so no test runs a host CLI.
	agents.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	m.Run()
}

// cliAgent is an agent whose config is the JSON file at path, and whose CLI has the add and remove commands of a
// stand-in: `mcp add <name> -- <command> <args>` and `mcp remove <name>`.
func cliAgent(path string) *agents.Agent {
	return &agents.Agent{
		Name:        "fakeagent",
		DisplayName: "Fake agent",
		Binary:      "fakeagent",
		MCP: &mcpconfig.Spec{
			Path:   func(context.Context) (string, error) { return path, nil },
			Format: mcpconfig.FormatJSON,
			Root:   "mcpServers",
			Entry:  testEntry,
			AddArgs: func(name, cmd string, args []string, _ map[string]string) []string {
				return append([]string{"mcp", "add", name, "--", cmd}, args...)
			},
			RemoveArgs: func(name string) []string { return []string{"mcp", "remove", name} },
		},
	}
}

// stubCLI puts a CLI named bin on PATH, and replaces the run of that CLI with run. It returns the argv of each run.
func stubCLI(t *testing.T, bin string, run func(argv []string) error) *[][]string {
	t.Helper()
	var calls [][]string
	origLookPath, origRunCLI := agents.LookPath, runCLI
	t.Cleanup(func() { agents.LookPath, runCLI = origLookPath, origRunCLI })
	agents.LookPath = func(string) (string, error) { return bin, nil }
	runCLI = func(_ context.Context, c cliCommand) error {
		calls = append(calls, c.argv)
		return run(c.argv)
	}
	return &calls
}

// writeServers writes a JSON config with the given server entries under mcpServers, as an agent's CLI would.
func writeServers(t *testing.T, path string, servers map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func TestUpsertAddsThroughCLI(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	bin := filepath.Join(t.TempDir(), "fakeagent")
	entry := testEntry(testCmd, testArgs("test"), nil)
	writeServers(t, path, map[string]any{"other": map[string]any{"command": "/usr/bin/other"}})
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	var written []byte
	calls := stubCLI(t, bin, func([]string) error {
		// The stand-in writes its own bytes, so a direct write after the CLI would be visible.
		data, err := json.MarshalIndent(map[string]any{"mcpServers": map[string]any{"other": map[string]any{"command": "/usr/bin/other"}, testName: entry}}, "", "  ")
		if err != nil {
			return err
		}
		written = append(data, '\n')
		return os.WriteFile(path, written, 0o600)
	})

	gotPath, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, path, gotPath)
	assert.Equal(t, StatusAdded, status)
	assert.Equal(t, [][]string{append([]string{bin, "mcp", "add", testName, "--", testCmd}, testArgs("test")...)}, *calls)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, written, after)
	assert.NotEqual(t, before, after)
}

func TestUpsertWritesDirectlyWhenCLIIsNotOnPath(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")

	_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	entries, err := ReadAt(cliAgent(path), path)
	require.NoError(t, err)
	assert.Contains(t, entries, testName)
}

func TestUpsertWritesDirectlyWhenCLIResolvesOnlyFromCurrentDirectory(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	origLookPath, origRunCLI := agents.LookPath, runCLI
	t.Cleanup(func() { agents.LookPath, runCLI = origLookPath, origRunCLI })
	agents.LookPath = func(string) (string, error) { return filepath.Join(".", "fakeagent"), exec.ErrDot }
	runCLI = func(context.Context, cliCommand) error {
		t.Fatal("a ./fakeagent must never run")
		return nil
	}

	_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	entries, err := ReadAt(cliAgent(path), path)
	require.NoError(t, err)
	assert.Contains(t, entries, testName)
}

// TestUpsertWritesDirectlyWhenCLIWritesNothing covers a CLI that does not write the entry to the file: it fails, for
// example because it is too old for the command or a different program has the same name, or it hangs, or it exits 0
// and writes another file or nothing. The direct writer adds the entry.
func TestUpsertWritesDirectlyWhenCLIWritesNothing(t *testing.T) {
	failures := map[string]error{
		"exit 0":  nil,
		"exit 1":  &process.ProcessError{Command: "fakeagent", Err: errors.New("exit status 1"), Stderr: "unknown command\n"},
		"timeout": fmt.Errorf("%w after 30s", context.DeadlineExceeded),
	}
	for name, failure := range failures {
		t.Run(name, func(t *testing.T) {
			ctx := testHome(t)
			path := filepath.Join(t.TempDir(), "agent.json")
			calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return failure })

			_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
			require.NoError(t, err)
			assert.Equal(t, StatusAdded, status)
			assert.Len(t, *calls, 1)
			entries, err := ReadAt(cliAgent(path), path)
			require.NoError(t, err)
			assert.Contains(t, entries, testName)
		})
	}
}

func TestUpsertKeepsEntryWhenCLIFailsAfterWriting(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	failure := &process.ProcessError{Command: "fakeagent", Err: errors.New("exit status 1"), Stderr: "warning\n"}
	stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func(argv []string) error {
		writeServers(t, path, map[string]any{testName: testEntry(testCmd, testArgs("test"), nil)})
		return failure
	})

	_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err, "the entry the CLI wrote is the one asked for, so its exit status does not matter")
	assert.Equal(t, StatusAdded, status)
}

func TestUpsertDoesNotRunCLIForEntryWeOwn(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)

	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })
	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("other"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Empty(t, *calls, "an update of our entry keeps the direct writer")
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Contains(t, entries[testName]["args"], "other")
}

func TestUpsertDoesNotRunCLIForForeignEntry(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	writeServers(t, path, map[string]any{testName: map[string]any{"command": "/usr/bin/other", "args": []string{"x"}}})
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	_, _, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorIs(t, err, ErrNotOwned)
	assert.Empty(t, *calls, "a refused entry is not touched by the CLI")

	_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, true)
	require.NoError(t, err)
	assert.Equal(t, StatusUpdated, status)
	assert.Empty(t, *calls, "--force replaces the entry directly")
}

func TestUpsertAddsEntryWhenCLIWritesOtherFile(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	other := filepath.Join(t.TempDir(), "elsewhere.json")
	a := cliAgent(path)
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error {
		writeServers(t, other, map[string]any{testName: testEntry(testCmd, testArgs("test"), nil)})
		return nil
	})

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Len(t, *calls, 1, "the CLI is not run again to undo its add")
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Contains(t, entries, testName, "the agent's file holds the entry asked for")
}

// TestUpsertReplacesEntryWhenCLIWritesDifferentEntry covers a CLI that writes the name with other launch keys, such as
// a different type. This run's CLI wrote the entry, so the direct writer replaces it with the one asked for, and the
// add is reported as added.
func TestUpsertReplacesEntryWhenCLIWritesDifferentEntry(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	a.MCP.Entry = func(cmd string, args []string, env map[string]string) map[string]any {
		entry := testEntry(cmd, args, env)
		entry["type"] = "local"
		return entry
	}
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error {
		entry := testEntry(testCmd, testArgs("test"), nil)
		entry["type"] = "stdio"
		writeServers(t, path, map[string]any{testName: entry})
		return nil
	})

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Len(t, *calls, 1, "the CLI is not run again")
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Equal(t, "local", entries[testName]["type"])
	assert.Equal(t, testCmd, entries[testName]["command"])
}

func TestUpsertAcceptsCLIEntryWhenSpecWritesEmptyEnvAsTyped(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	// Like claudeMCPEntry: env is a typed map, empty when there are no variables.
	a.MCP.Entry = func(cmd string, args []string, env map[string]string) map[string]any {
		vars := map[string]string{}
		maps.Copy(vars, env)
		return map[string]any{"command": cmd, "args": args, "env": vars}
	}
	stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error {
		entry := testEntry(testCmd, testArgs("test"), nil)
		entry["env"] = map[string]any{}
		writeServers(t, path, map[string]any{testName: entry})
		return nil
	})

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
}

func TestUpsertWritesDirectlyWhenBatchFileCannotTakeArguments(t *testing.T) {
	ctx := testHome(t)
	args := append(testArgs("test"), `say "hi"`)
	dir := t.TempDir()

	path := filepath.Join(dir, "batch.json")
	calls := stubCLI(t, filepath.Join(dir, "fakeagent.cmd"), func([]string) error { return nil })
	_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, args, nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Empty(t, *calls, "a batch file would garble the quotes, so the direct writer does the change")

	path = filepath.Join(dir, "binary.json")
	calls = stubCLI(t, filepath.Join(dir, "fakeagent.exe"), func([]string) error { return nil })
	_, status, err = Upsert(ctx, cliAgent(path), testName, testCmd, args, nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Len(t, *calls, 1, "an executable runs the CLI, which here writes nothing, and the direct writer adds the entry")
}

func TestRemoveThroughCLI(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)

	bin := filepath.Join(t.TempDir(), "fakeagent")
	calls := stubCLI(t, bin, func([]string) error {
		writeServers(t, path, map[string]any{})
		return nil
	})
	removed, err := RemoveAt(ctx, a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, [][]string{{bin, "mcp", "remove", testName}}, *calls)
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRemoveWritesDirectlyWhenFileMoved(t *testing.T) {
	ctx := testHome(t)
	current := filepath.Join(t.TempDir(), "current.json")
	recorded := filepath.Join(t.TempDir(), "recorded.json")
	a := cliAgent(current)
	writeServers(t, recorded, map[string]any{testName: testEntry(testCmd, testArgs("test"), nil)})
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	removed, err := RemoveAt(ctx, a, recorded, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Empty(t, *calls, "the CLI reads the current file, not the recorded one")
	entries, err := ReadAt(a, recorded)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRemoveRefusesForeignEntryBeforeCLI(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	writeServers(t, path, map[string]any{testName: map[string]any{"command": "/usr/bin/other", "args": []string{"x"}}})
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	_, err := RemoveAt(ctx, cliAgent(path), path, testName)
	assert.ErrorIs(t, err, ErrNotOwned)
	assert.Empty(t, *calls)
}

func TestRemoveDoesNotRunCLIWhenEntryIsAbsent(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	removed, err := RemoveAt(ctx, cliAgent(path), path, testName)
	require.NoError(t, err)
	assert.False(t, removed)
	assert.Empty(t, *calls)
}

func TestRemoveDirectWhenCLILeavesEntry(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	removed, err := RemoveAt(ctx, a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Len(t, *calls, 1, "the CLI ran once, and the direct writer removed what it left")
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRemoveDirectWhenCLIFails(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	failure := &process.ProcessError{Command: "fakeagent", Err: errors.New("exit status 2"), Stderr: "no such server\n"}
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return failure })

	removed, err := RemoveAt(ctx, a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Len(t, *calls, 1)
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// launcherEnv tells TestLauncherCLI to start the sleeping CLI as a child and wait for it, as npm shims and Gemini do.
const launcherEnv = "AGENTCONFIG_TEST_LAUNCHER"

// TestLauncherCLI is a CLI that starts another process and waits for it. The child inherits the output of the launcher,
// so the output stays open after the launcher is killed.
func TestLauncherCLI(t *testing.T) {
	if os.Getenv(launcherEnv) == "" {
		t.Skip("runs only as the launcher of TestRunWithTimeoutStopsLauncherThatHoldsOutput")
	}
	child := exec.Command(os.Args[0], "-test.run=^TestSleepingCLI$")
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	_ = child.Run()
}

func TestRunWithTimeoutStopsLauncherThatHoldsOutput(t *testing.T) {
	ctx := env.Set(env.Set(t.Context(), launcherEnv, "1"), sleepingCLIEnv, "1")
	argv := []string{os.Args[0], "-test.run=^TestLauncherCLI$"}

	start := time.Now()
	err := runWithTimeout(ctx, 500*time.Millisecond, cliCommand{argv: argv})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 30*time.Second, "the child kept the run open past its timeout")
}

func TestRunWithTimeoutStopsCLIThatHangs(t *testing.T) {
	ctx := env.Set(t.Context(), sleepingCLIEnv, "1")
	argv := []string{os.Args[0], "-test.run=^TestSleepingCLI$"}

	err := runWithTimeout(ctx, 200*time.Millisecond, cliCommand{argv: argv})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.NotContains(t, err.Error(), os.Args[0], "the timeout error does not carry the argv")
}

func TestRunWithTimeoutDoesNotLabelCallerDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(env.Set(t.Context(), sleepingCLIEnv, "1"), 200*time.Millisecond)
	defer cancel()
	argv := []string{os.Args[0], "-test.run=^TestSleepingCLI$"}

	err := runWithTimeout(ctx, time.Minute, cliCommand{argv: argv})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, context.DeadlineExceeded, "the caller's deadline is not the timeout of this run")
}

// TestSleepingCLI is the child process of the timeout tests. It blocks when run as that child, for just over the wait
// delay, so that a launcher's wait for it ends at the delay and the test does not leave it running long.
func TestSleepingCLI(t *testing.T) {
	if os.Getenv(sleepingCLIEnv) == "" {
		t.Skip("runs only as the child process of the timeout tests")
	}
	time.Sleep(cliWaitDelay + time.Second)
}

func TestBatchSafe(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want bool
	}{
		{"executable takes JSON", []string{`C:\bin\claude.exe`, `{"type":"stdio"}`}, true},
		{"batch file takes plain arguments", []string{`C:\npm\codex.cmd`, "mcp", "remove", "github"}, true},
		{"executable with a path that has spaces", []string{`C:\Program Files\claude\claude.exe`, "mcp", "remove", "github"}, true},
		{"batch file with a path that has spaces", []string{`C:\Users\John Doe\npm\codex.CMD`, "mcp", "remove", "github"}, false},
		{"batch file takes an argument with a space", []string{`C:\npm\codex.cmd`, "mcp", "add", "my server"}, false},
		{"batch file takes a double quote", []string{`C:\npm\claude.cmd`, `{"type":"stdio"}`}, false},
		{"batch file takes a percent sign", []string{`C:\npm\codex.bat`, "%PATH%"}, false},
		{"batch file takes an ampersand", []string{`C:\npm\codex.bat`, "a&b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, batchSafe(tt.argv))
		})
	}
}

func TestCLIFailureNamesCommandNotArgv(t *testing.T) {
	argv := []string{"/bin/fakeagent", "mcp", "add-json", testName, `{"command":"/bin/databricks"}`}
	failed := &process.ProcessError{Command: strings.Join(argv, " "), Err: errors.New("exit status 1")}

	err := cliFailure(cliAgent(""), argv[1:], failed)
	assert.EqualError(t, err, "fakeagent mcp add-json failed: exit status 1")
	assert.ErrorIs(t, err, failed)

	failed.Stderr = " no such server \n"
	assert.EqualError(t, cliFailure(cliAgent(""), argv[1:], failed), "fakeagent mcp add-json failed: no such server")
}

func TestUpsertWritesDirectlyForNameWithDash(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	_, status, err := Upsert(ctx, a, "-h", testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	_, err = RemoveAt(ctx, a, path, "-h")
	require.NoError(t, err)
	assert.Empty(t, *calls, "a name with a dash would reach the CLI as one of its flags")
}

func TestUpsertTreatsEmptyEnvFromCLIAsUnchanged(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error {
		entry := testEntry(testCmd, testArgs("test"), nil)
		entry["env"] = map[string]any{}
		writeServers(t, path, map[string]any{testName: entry})
		return nil
	})

	_, status, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	entries, err := ReadAt(cliAgent(path), path)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, entries[testName]["env"], "the entry the CLI wrote is kept as it wrote it")
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	_, status, err = Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusUnchanged, status)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

// homeOutEnv names the file that TestWriteHomeVariable writes the agent home variable to.
const homeOutEnv = "AGENTCONFIG_TEST_HOME_OUT"

// testHomeVar is the agent home variable of the tests. Its value is what TestRunCLISeesHomeVariable sets.
const testHomeVar = "FAKEAGENT_HOME"

// TestWriteHomeVariable is the child process of TestRunCLISeesHomeVariable. It writes the agent home variable it sees, or
// "unset" when it does not see the variable.
func TestWriteHomeVariable(t *testing.T) {
	out := os.Getenv(homeOutEnv)
	if out == "" {
		t.Skip("runs only as the child process of TestRunCLISeesHomeVariable")
	}
	value, ok := os.LookupEnv(testHomeVar)
	if !ok {
		value = "unset"
	}
	require.NoError(t, os.WriteFile(out, []byte(value), 0o600))
}

// runHomeChild runs the child TestWriteHomeVariable as an agent CLI would be run, with the agent home variable set to
// value, and returns what the child saw.
func runHomeChild(t *testing.T, value string) string {
	out := filepath.Join(t.TempDir(), "home")
	ctx := env.Set(env.Set(t.Context(), testHomeVar, value), homeOutEnv, out)
	spec := &mcpconfig.Spec{HomeEnv: testHomeVar}
	opts, err := homeOpts(ctx, spec)
	require.NoError(t, err)
	argv := []string{os.Args[0], "-test.run=^TestWriteHomeVariable$"}
	require.NoError(t, runWithTimeout(ctx, time.Minute, cliCommand{argv: argv, opts: opts}))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	return string(data)
}

func TestRunCLISeesHomeVariable(t *testing.T) {
	t.Run("relative value is made absolute", func(t *testing.T) {
		want, err := mcpconfig.AbsoluteAsWritten("relative-home")
		require.NoError(t, err)
		assert.Equal(t, want, runHomeChild(t, "relative-home"), "the CLI runs in an empty directory")
	})
	t.Run("empty value is unset", func(t *testing.T) {
		assert.Equal(t, "unset", runHomeChild(t, ""), "a CLI that reads an empty variable as set writes relative to its directory")
	})
}

// TestHomeOptsPassesCleanedHome checks that the CLI gets the directory the read-back reads: a ".." is cleaned, as the
// config path is, so the CLI and the read-back do not name different directories.
func TestHomeOptsPassesCleanedHome(t *testing.T) {
	sep := string(filepath.Separator)
	home := t.TempDir() + sep + "link" + sep + ".." + sep + "codex"
	ctx := env.Set(t.Context(), testHomeVar, home)

	opts, err := homeOpts(ctx, &mcpconfig.Spec{HomeEnv: testHomeVar})
	require.NoError(t, err)
	cmd := &exec.Cmd{}
	for _, opt := range opts {
		require.NoError(t, opt(ctx, cmd))
	}
	assert.Equal(t, []string{testHomeVar + "=" + filepath.Clean(home)}, cmd.Env)
}

func TestSymlinkedConfigIsWrittenThroughWithoutCLI(t *testing.T) {
	ctx := testHome(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "agent.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o700))
	writeServers(t, target, map[string]any{})
	path := filepath.Join(dir, "agent.json")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	a := cliAgent(path)
	calls := stubCLI(t, filepath.Join(dir, "fakeagent"), func([]string) error { return nil })

	_, status, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	assert.Equal(t, StatusAdded, status)
	assert.Empty(t, *calls, "a CLI that renames over the config would replace the link")

	_, err = RemoveAt(ctx, a, path, testName)
	require.NoError(t, err)
	assert.Empty(t, *calls)

	info, err := os.Lstat(path)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "the link is kept")
	entries, err := ReadAt(a, target)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRemoveWritesDirectlyWhenCurrentPathCannotBeResolved(t *testing.T) {
	ctx := testHome(t)
	path := filepath.Join(t.TempDir(), "agent.json")
	a := cliAgent(path)
	_, _, err := Upsert(ctx, a, testName, testCmd, testArgs("test"), nil, false)
	require.NoError(t, err)
	a.MCP.Path = func(context.Context) (string, error) { return "", errors.New("no home directory") }
	calls := stubCLI(t, filepath.Join(t.TempDir(), "fakeagent"), func([]string) error { return nil })

	removed, err := RemoveAt(ctx, a, path, testName)
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Empty(t, *calls, "the recorded file is not the current one, so the CLI is not run")
	entries, err := ReadAt(a, path)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestUpsertRefusesDanglingSymlinkBeforeCLI(t *testing.T) {
	ctx := testHome(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	if err := os.Symlink(filepath.Join(dir, "missing.json"), path); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	calls := stubCLI(t, filepath.Join(dir, "fakeagent"), func([]string) error { return nil })

	_, _, err := Upsert(ctx, cliAgent(path), testName, testCmd, testArgs("test"), nil, false)
	assert.ErrorContains(t, err, "symlink to a file that does not exist")
	assert.Empty(t, *calls, "the CLI would write through a link with no target")
}

// listingEnv names the file that TestWriteCLIDirListing writes the names in its working directory to.
const listingEnv = "AGENTCONFIG_TEST_LISTING"

// TestWriteCLIDirListing is the child process of TestRunWithTimeoutStartsInEmptyDirectory. It lists its working directory.
func TestWriteCLIDirListing(t *testing.T) {
	out := os.Getenv(listingEnv)
	if out == "" {
		t.Skip("runs only as the child process of TestRunWithTimeoutStartsInEmptyDirectory")
	}
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.NoError(t, os.WriteFile(out, []byte(strings.Join(names, "\n")), 0o600))
}

func TestRunWithTimeoutStartsInEmptyDirectory(t *testing.T) {
	out := filepath.Join(t.TempDir(), "listing")
	ctx := env.Set(t.Context(), listingEnv, out)
	argv := []string{os.Args[0], "-test.run=^TestWriteCLIDirListing$"}

	require.NoError(t, runWithTimeout(ctx, 30*time.Second, cliCommand{argv: argv}))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Empty(t, data, "a project config in the working directory would change what the CLI does")
}
