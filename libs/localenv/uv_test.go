package localenv

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const uvPythonInstall312Pattern = `^uv(?:\.exe)? python install 3\.12$`

// writePipConf writes conf to the primary OS-specific pip config path under a
// fresh temp home and returns a context rooted at that home. On Windows the
// primary path is %APPDATA%\pip\pip.ini, so APPDATA is pointed inside the temp
// home too; without this the file would land in a location pipConfPaths never
// probes on that OS and the test would read an empty index-url.
func writePipConf(t *testing.T, conf string) context.Context {
	t.Helper()
	tmp := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), tmp)
	if runtime.GOOS == "windows" {
		ctx = env.Set(ctx, "APPDATA", filepath.Join(tmp, "AppData", "Roaming"))
	}
	paths := pipConfPaths(ctx)
	require.NotEmpty(t, paths)
	primary := paths[0]
	require.NoError(t, os.MkdirAll(filepath.Dir(primary), 0o755))
	require.NoError(t, os.WriteFile(primary, []byte(conf), 0o644))
	return ctx
}

// uvStubCommand returns the normalized command spelling produced by
// processStub.Commands on the current platform.
func uvStubCommand(args string) string {
	bin := "uv"
	if runtime.GOOS == "windows" {
		bin = "uv.exe"
	}
	return bin + " " + args
}

func TestUvArgs(t *testing.T) {
	m := &uvManager{bin: "uv"}
	assert.Equal(t, []string{"sync", "--python", "3.12"}, m.syncArgs("3.12"))
	assert.Equal(t, []string{"python", "install", "3.12"}, m.pythonInstallArgs("3.12"))
	assert.Equal(t, []string{
		"python", "find", "--system", "--no-python-downloads", "cpython@==3.12.*",
	}, m.pythonFindArgs("3.12"))
	assert.Equal(t, []string{"pip", "install", "pip", "--python", "/p/.venv/bin/python"}, m.pipSeedArgs("/p/.venv/bin/python"))
}

func TestEnsurePythonStopsAfterSuccessfulInstall(t *testing.T) {
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(_ *exec.Cmd) error { return nil })
	m := &uvManager{bin: "uv"}

	selection, err := m.EnsurePython(ctx, "3.12")

	require.NoError(t, err)
	assert.Equal(t, PythonSelection{
		Executable: "3.12",
		Resolution: PythonResolutionUVInstallSucceeded,
	}, selection)
	assert.Equal(t, []string{uvStubCommand("python install 3.12")}, stub.Commands())
}

func TestEnsurePythonFallsBackToInstalledInterpreter(t *testing.T) {
	ctx, stub := process.WithStub(t.Context())
	stub.WithFailureFor(uvPythonInstall312Pattern, errors.New("download blocked"))
	stub.WithStdoutFor(`python find`, "/system/python3.12\n")
	m := &uvManager{bin: "uv"}

	selection, err := m.EnsurePython(ctx, "3.12")

	require.NoError(t, err)
	assert.Equal(t, PythonSelection{
		Executable: "/system/python3.12",
		Resolution: PythonResolutionInstalledFallback,
	}, selection)
	assert.Equal(t, []string{
		uvStubCommand("python install 3.12"),
		uvStubCommand("python find --system --no-python-downloads cpython@==3.12.*"),
	}, stub.Commands())
}

func TestEnsurePythonFallbackPreservesIndexURLBridge(t *testing.T) {
	// CI sets UV_INDEX_URL. Remove it so this test exercises pip.conf bridging;
	// t.Setenv registers cleanup that restores the original value.
	t.Setenv("UV_INDEX_URL", "")
	os.Unsetenv("UV_INDEX_URL")
	ctx := writePipConf(t, "[global]\nindex-url = https://proxy.example/simple\n")
	ctx, stub := process.WithStub(ctx)
	stub.WithFailureFor(uvPythonInstall312Pattern, errors.New("download blocked"))
	stub.WithStdoutFor(`python find`, "/system/python3.12\n")

	selection, err := (&uvManager{bin: "uv"}).EnsurePython(ctx, "3.12")

	require.NoError(t, err)
	assert.Equal(t, "/system/python3.12", selection.Executable)
	assert.Equal(t, "https://proxy.example/simple", stub.LookupEnv("UV_INDEX_URL"))
}

func TestEnsurePythonStopsFallbackWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	ctx, stub := process.WithStub(ctx)
	stub.WithCallback(func(_ *exec.Cmd) error {
		cancel()
		return context.Canceled
	})

	_, err := (&uvManager{bin: "uv"}).EnsurePython(ctx, "3.12")

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{uvStubCommand("python install 3.12")}, stub.Commands())
}

func TestEnsurePythonFallbackErrorSurfacesSearchStderr(t *testing.T) {
	ctx, stub := process.WithStub(t.Context())
	stub.WithStderrFor(uvPythonInstall312Pattern, "error: Connection refused")
	stub.WithFailureFor(uvPythonInstall312Pattern, errors.New("exit status 1"))
	stub.WithStderrFor(`python find`, "error: unexpected argument '--no-python-downloads'")
	stub.WithFailureFor(`python find`, errors.New("exit status 2"))
	m := &uvManager{bin: "uv"}

	_, err := m.EnsurePython(ctx, "3.12")

	require.Error(t, err)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	// Both the download reason and the search command's own stderr must be
	// reachable; the latter distinguishes "nothing installed" from a uv that
	// rejected the find flags.
	assert.Contains(t, pe.Msg, "Connection refused")
	assert.Contains(t, pe.Msg, "unexpected argument '--no-python-downloads'")
}

func TestEnsurePythonReportsCancellationDuringFallbackSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	ctx, stub := process.WithStub(ctx)
	// Install fails normally (ctx still live), so the search runs; the search is
	// then interrupted. The error must report the interruption, not claim that no
	// interpreter is installed.
	stub.WithFailureFor(uvPythonInstall312Pattern, errors.New("download blocked"))
	stub.WithCallback(func(_ *exec.Cmd) error {
		cancel()
		return context.Canceled
	})

	_, err := (&uvManager{bin: "uv"}).EnsurePython(ctx, "3.12")

	require.ErrorIs(t, err, context.Canceled)
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.NotContains(t, pe.Msg, "no compatible installed Python found")
	assert.Equal(t, []string{
		uvStubCommand("python install 3.12"),
		uvStubCommand("python find --system --no-python-downloads cpython@==3.12.*"),
	}, stub.Commands())
}

func TestEnsurePythonFailsWhenFallbackFindsNothing(t *testing.T) {
	ctx, stub := process.WithStub(t.Context())
	stub.WithFailureFor(uvPythonInstall312Pattern, errors.New("download blocked"))
	stub.WithFailureFor(`python find`, errors.New("No interpreter found"))
	m := &uvManager{bin: "uv"}

	selection, err := m.EnsurePython(ctx, "3.12")

	require.Error(t, err)
	assert.Empty(t, selection)
	assert.Equal(t, []string{
		uvStubCommand("python install 3.12"),
		uvStubCommand("python find --system --no-python-downloads cpython@==3.12.*"),
	}, stub.Commands())
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrPythonInstall, pe.Code)
}

func TestVenvPythonPath(t *testing.T) {
	// Validate invokes this interpreter directly (not via `uv run`) so it observes
	// exactly the .venv that was provisioned, ignoring any active VIRTUAL_ENV.
	got := venvPython(filepath.Join("p", "proj"))
	var want string
	if runtime.GOOS == "windows" {
		want = filepath.Join("p", "proj", ".venv", "Scripts", "python.exe")
	} else {
		want = filepath.Join("p", "proj", ".venv", "bin", "python")
	}
	assert.Equal(t, want, got)
}

func TestDiscoverUvFindsBinOnPath(t *testing.T) {
	dir := t.TempDir()
	// exec.LookPath only resolves a bare "uv" to a file with a PATHEXT extension
	// on Windows, so the on-PATH binary must be named uv.exe there.
	exe := "uv"
	if runtime.GOOS == "windows" {
		exe = "uv.exe"
	}
	bin := filepath.Join(dir, exe)
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)
	got, err := discoverUv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, bin, got)
}

func TestDiscoverUvFindsBinInLocalBin(t *testing.T) {
	// The installer drops uv (uv.exe on Windows) under ~/.local/bin; discoverUv
	// must find it there when it is not on PATH, using the OS-appropriate name.
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	exe := "uv"
	if runtime.GOOS == "windows" {
		exe = "uv.exe"
	}
	bin := filepath.Join(binDir, exe)
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))

	t.Setenv("PATH", t.TempDir()) // no uv on PATH
	ctx := env.WithUserHomeDir(t.Context(), home)
	ctx = env.Set(ctx, "XDG_BIN_HOME", "")
	got, err := discoverUv(ctx)
	require.NoError(t, err)
	assert.Equal(t, bin, got)
}

func TestDiscoverUvSkipsRelativeCandidatesWhenHomeUnset(t *testing.T) {
	// Regression: when HOME and XDG_BIN_HOME are unset the candidate paths
	// collapse to relative "uv" / ".local/bin/uv". discoverUv must not os.Stat
	// them against the CWD and return a stray ./uv as if it were the real binary.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "uv"), []byte("#!/bin/sh\n"), 0o755))
	t.Chdir(dir)
	t.Setenv("PATH", t.TempDir()) // a PATH with no uv, so LookPath falls through

	ctx := env.WithUserHomeDir(t.Context(), "")
	ctx = env.Set(ctx, "XDG_BIN_HOME", "")
	got, err := discoverUv(ctx)
	if err == nil {
		assert.True(t, filepath.IsAbs(got), "discoverUv must not return a relative path; got %q", got)
	}
}

func TestPipConfIndexURLReadsOSSpecificPath(t *testing.T) {
	// The primary pip config path for the current OS must be honored, not just
	// the Linux/XDG ~/.config/pip/pip.conf location.
	tmp := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), tmp)
	if runtime.GOOS == "windows" {
		ctx = env.Set(ctx, "APPDATA", filepath.Join(tmp, "AppData", "Roaming"))
	}

	paths := pipConfPaths(ctx)
	require.NotEmpty(t, paths)
	primary := paths[0]
	require.NoError(t, os.MkdirAll(filepath.Dir(primary), 0o755))
	require.NoError(t, os.WriteFile(primary, []byte("[global]\nindex-url = https://proxy.example/simple\n"), 0o644))

	assert.Equal(t, "https://proxy.example/simple", pipConfIndexURL(ctx))
}

func TestRedactURLCredentials(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://user:pass@proxy.example/simple", "https://proxy.example/simple"},
		{"https://token@proxy.example/simple", "https://proxy.example/simple"},
		{"https://proxy.example/simple", "https://proxy.example/simple"},
		{"not a url", "not a url"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, redactURLCredentials(tc.in))
	}
}

func TestPipConfIndexURLReadsLegacyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("legacy ~/.pip path is Unix/macOS only")
	}
	// The legacy ~/.pip/pip.conf location pip still reads must be honored when the
	// newer per-user locations are absent.
	tmp := t.TempDir()
	legacy := filepath.Join(tmp, ".pip")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "pip.conf"), []byte("[global]\nindex-url = https://legacy.example/simple\n"), 0o644))

	ctx := env.WithUserHomeDir(t.Context(), tmp)
	assert.Equal(t, "https://legacy.example/simple", pipConfIndexURL(ctx))
}

func TestPipConfIndexURL(t *testing.T) {
	t.Run("returns_url_from_pip_conf", func(t *testing.T) {
		ctx := writePipConf(t, "[global]\nindex-url = https://proxy.example/simple\n")
		got := pipConfIndexURL(ctx)
		assert.Equal(t, "https://proxy.example/simple", got)
	})

	t.Run("returns_empty_when_no_pip_conf", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := env.WithUserHomeDir(t.Context(), tmp)
		got := pipConfIndexURL(ctx)
		assert.Empty(t, got)
	})

	t.Run("returns_empty_when_no_index_url_in_conf", func(t *testing.T) {
		ctx := writePipConf(t, "[global]\nextra-index-url = https://other.example/simple\n")
		got := pipConfIndexURL(ctx)
		assert.Empty(t, got)
	})
}

func TestResolveIndexURLRespectsExistingEnv(t *testing.T) {
	m := &uvManager{}

	t.Run("returns_empty_when_UV_INDEX_URL_already_set", func(t *testing.T) {
		// Set up a pip.conf that would otherwise be used.
		ctx := writePipConf(t, "[global]\nindex-url = https://proxy.example/simple\n")
		// When UV_INDEX_URL is in ctx, resolveIndexURL must not override it.
		ctx = env.Set(ctx, "UV_INDEX_URL", "https://explicit.example/simple")

		got := m.resolveIndexURL(ctx)
		assert.Empty(t, got)
	})

	t.Run("returns_pip_conf_url_when_UV_INDEX_URL_unset", func(t *testing.T) {
		// CI sets UV_INDEX_URL to a package mirror in the process environment, and
		// env.Lookup falls back to os.LookupEnv, so it must actually be removed
		// (setting it to "" still reports ok=true) for resolveIndexURL to fall
		// through to pip.conf. t.Setenv registers the cleanup that restores it.
		t.Setenv("UV_INDEX_URL", "")
		os.Unsetenv("UV_INDEX_URL")

		ctx := writePipConf(t, "[global]\nindex-url = https://proxy.example/simple\n")
		got := m.resolveIndexURL(ctx)
		assert.Equal(t, "https://proxy.example/simple", got)
	})
}

func TestUvFailureIncludesStderr(t *testing.T) {
	t.Run("includes_stderr_when_present", func(t *testing.T) {
		underlying := &process.ProcessError{
			Command: "uv sync",
			Err:     errors.New("exit status 2"),
			Stderr:  "error: Connection refused\n",
		}
		pe := uvFailure(ErrProvision, underlying, "uv sync")
		assert.Equal(t, ErrProvision, pe.Code)
		assert.Contains(t, pe.Msg, "Connection refused")
		assert.NotEqual(t, '\n', pe.Msg[len(pe.Msg)-1], "Msg must not end with a newline")
	})

	t.Run("omits_stderr_suffix_when_empty", func(t *testing.T) {
		underlying := &process.ProcessError{
			Command: "uv sync",
			Err:     errors.New("exit status 2"),
			Stderr:  "",
		}
		pe := uvFailure(ErrProvision, underlying, "uv sync")
		assert.Equal(t, ErrProvision, pe.Code)
		assert.Equal(t, "uv sync failed", pe.Msg)
	})

	t.Run("non_process_error_uses_action_only", func(t *testing.T) {
		pe := uvFailure(ErrProvision, errors.New("some other error"), "uv sync")
		assert.Equal(t, ErrProvision, pe.Code)
		assert.Equal(t, "uv sync failed", pe.Msg)
	})
}

func TestConfirmUvInstall(t *testing.T) {
	t.Run("opt_in_env_var_consents_without_prompt", func(t *testing.T) {
		// Non-interactive context, but the opt-in env var grants consent.
		ctx := env.Set(t.Context(), EnvAutoInstallUv, "1")
		assert.True(t, confirmUvInstall(ctx))
	})

	t.Run("non_interactive_without_opt_in_declines", func(t *testing.T) {
		// SetupTest defaults to PromptSupported=false, i.e. non-interactive.
		ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})
		assert.False(t, confirmUvInstall(ctx))
	})

	t.Run("missing_cmdio_declines_without_panic", func(t *testing.T) {
		// A context with no cmdio (library entry point) must not panic in
		// IsPromptSupported; it declines like any other non-interactive run.
		assert.NotPanics(t, func() {
			assert.False(t, confirmUvInstall(t.Context()))
		})
	})

	t.Run("falsey_opt_in_does_not_consent_when_non_interactive", func(t *testing.T) {
		ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})
		ctx = env.Set(ctx, EnvAutoInstallUv, "0")
		assert.False(t, confirmUvInstall(ctx))
	})
}

func TestLineWithPrefix(t *testing.T) {
	// A stray leading line (as uv might emit) must not shift the parse: the
	// value is located by prefix, not position.
	out := "warning: something\nPYVER:3.12\nDBCVER:17.2.0\n"

	pyVer, ok := lineWithPrefix(out, validatePyPrefix)
	assert.True(t, ok)
	assert.Equal(t, "3.12", pyVer)

	dbcVer, ok := lineWithPrefix(out, validateDBCPrefix)
	assert.True(t, ok)
	assert.Equal(t, "17.2.0", dbcVer)

	// An empty DBC value (databricks-connect absent) is found but blank.
	empty, ok := lineWithPrefix("PYVER:3.12\nDBCVER:\n", validateDBCPrefix)
	assert.True(t, ok)
	assert.Empty(t, empty)

	// A missing prefix reports not-found rather than a wrong line.
	_, ok = lineWithPrefix("PYVER:3.12\n", validateDBCPrefix)
	assert.False(t, ok)
}
