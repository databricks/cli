package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrependPath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{"/a", "/b"}, string(os.PathListSeparator)))
	prependPath(t.Context(), "/b") // existing entry moves to the front (deduped)
	assert.Equal(t, []string{"/b", "/a"}, filepath.SplitList(os.Getenv("PATH")))
	prependPath(t.Context(), "/new")
	assert.Equal(t, []string{"/new", "/b", "/a"}, filepath.SplitList(os.Getenv("PATH")))
	prependPath(t.Context(), "/a") // a deeper existing entry also moves to the front (deduped)
	assert.Equal(t, []string{"/a", "/new", "/b"}, filepath.SplitList(os.Getenv("PATH")))
}

func TestRemovePath(t *testing.T) {
	t.Setenv("PATH", strings.Join([]string{"/a", "/b", "/a"}, string(os.PathListSeparator)))
	removePath(t.Context(), "/a")
	assert.Equal(t, []string{"/b"}, filepath.SplitList(os.Getenv("PATH")))
}

func TestAcquireSetupLock(t *testing.T) {
	home := t.TempDir()
	lockPath := filepath.Join(home, agentRootDir, setupLockName)

	// Acquire creates the sentinel; release removes it.
	unlock, err := acquireSetupLock(t.Context(), home)
	require.NoError(t, err)
	_, statErr := os.Stat(lockPath)
	require.NoError(t, statErr, "lock sentinel should exist while held")
	unlock()
	_, statErr = os.Stat(lockPath)
	assert.ErrorIs(t, statErr, fs.ErrNotExist, "lock sentinel should be gone after release")

	// A lock left behind by a dead process is reclaimed once stale, not waited on
	// forever.
	require.NoError(t, os.WriteFile(lockPath, nil, 0o644))
	stale := time.Now().Add(-2 * setupLockStaleAfter)
	require.NoError(t, os.Chtimes(lockPath, stale, stale))
	unlock2, err := acquireSetupLock(t.Context(), home)
	require.NoError(t, err)
	unlock2()
}

func TestNodeArchiveSpec(t *testing.T) {
	x64Spec := nodeArchiveSpec("amd64")
	assert.Contains(t, x64Spec.archiveURL, "nodejs.org/dist")
	assert.Equal(t, "npm", x64Spec.binaryName)
	assert.Equal(t, nodeX64LinuxChecksum, x64Spec.checksum)
	assert.Equal(t, "node", x64Spec.dirName)

	arm64Spec := nodeArchiveSpec("arm64")
	assert.Contains(t, arm64Spec.archiveURL, "nodejs.org/dist")
	assert.Equal(t, "npm", arm64Spec.binaryName)
	assert.Equal(t, nodeArm64LinuxChecksum, arm64Spec.checksum)
	assert.Equal(t, "node", arm64Spec.dirName)

	unknownSpec := nodeArchiveSpec("riscv")
	assert.Empty(t, unknownSpec.archiveURL)
	assert.Equal(t, "npm", unknownSpec.binaryName)
	assert.Empty(t, unknownSpec.checksum)
	assert.Equal(t, "node", unknownSpec.dirName)
}

func TestUvArchiveSpec(t *testing.T) {
	x64Spec := uvArchiveSpec("amd64")
	assert.Contains(t, x64Spec.archiveURL, "github.com/astral-sh/uv/releases")
	assert.Equal(t, "uv", x64Spec.binaryName)
	assert.Equal(t, uvX64LinuxChecksum, x64Spec.checksum)
	assert.Equal(t, "uv", x64Spec.dirName)

	arm64Spec := uvArchiveSpec("arm64")
	assert.Contains(t, arm64Spec.archiveURL, "github.com/astral-sh/uv/releases")
	assert.Equal(t, "uv", arm64Spec.binaryName)
	assert.Equal(t, uvArm64LinuxChecksum, arm64Spec.checksum)
	assert.Equal(t, "uv", arm64Spec.dirName)

	unknownSpec := uvArchiveSpec("riscv")
	assert.Empty(t, unknownSpec.archiveURL)
	assert.Equal(t, "uv", unknownSpec.binaryName)
	assert.Empty(t, unknownSpec.checksum)
	assert.Equal(t, "uv", unknownSpec.dirName)
}

func TestDownloadVerified(t *testing.T) {
	payload := []byte("fake node tarball")
	sum := sha256.Sum256(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	t.Run("matching checksum writes the file", func(t *testing.T) {
		path, err := downloadVerified(t.Context(), archiveSpec{
			archiveURL: srv.URL,
			binaryName: "npm",
			checksum:   hex.EncodeToString(sum[:]),
			dirName:    "node",
		})
		require.NoError(t, err)
		defer os.Remove(path)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, payload, got)
	})

	t.Run("mismatched checksum errors and leaves no file", func(t *testing.T) {
		_, err := downloadVerified(t.Context(), archiveSpec{
			archiveURL: srv.URL,
			binaryName: "npm",
			checksum:   strings.Repeat("0", 64),
			dirName:    "node",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})
}

// tarGz returns a gzipped tar archive holding a single member at name.
func tarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestEnsureBinary(t *testing.T) {
	// ensureBinary uses unix specific tooling, and agent-shim is only intended for Linux,
	// so skip tests for Windows
	if runtime.GOOS == "windows" {
		return
	}

	t.Run("unsupported architecture errors", func(t *testing.T) {
		_, err := ensureBinary(t.Context(), t.TempDir(), nodeArchiveSpec("riscv"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported architecture")
	})

	nodeArchive := tarGz(t, "node/bin/npm", []byte("#!/bin/sh\necho npm\n"))
	nodeSum := sha256.Sum256(nodeArchive)
	uvArchive := tarGz(t, "uv/uv", []byte("#!/bin/sh\necho uv\n"))
	uvSum := sha256.Sum256(uvArchive)

	var downloads int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads++
		switch r.URL.Path {
		case "/node.tar.gz":
			_, _ = w.Write(nodeArchive)
		case "/uv.tar.gz":
			_, _ = w.Write(uvArchive)
		default:
			panic("unsupported archive " + r.URL.Path)
		}
	}))
	defer srv.Close()

	home := t.TempDir()
	for _, c := range []struct {
		spec       archiveSpec
		wantBinDir string
	}{
		{
			spec: archiveSpec{
				archiveURL: srv.URL + "/node.tar.gz",
				binaryName: "npm",
				checksum:   hex.EncodeToString(nodeSum[:]),
				dirName:    "node",
				binDirName: "bin",
			}, wantBinDir: filepath.Join(home, agentDepsDir, "node", "bin"),
		},
		{
			spec: archiveSpec{
				archiveURL: srv.URL + "/uv.tar.gz",
				binaryName: "uv",
				checksum:   hex.EncodeToString(uvSum[:]),
				dirName:    "uv",
			}, wantBinDir: filepath.Join(home, agentDepsDir, "uv"),
		},
	} {
		t.Run("downloads, extracts, and returns the bin dir ("+c.spec.binaryName+")", func(t *testing.T) {
			beforeDownloads := downloads
			binDir, err := ensureBinary(t.Context(), home, c.spec)
			require.NoError(t, err)
			assert.Equal(t, c.wantBinDir, binDir)
			_, err = os.Stat(filepath.Join(binDir, c.spec.binaryName))
			require.NoError(t, err)
			assert.Equal(t, beforeDownloads+1, downloads)
		})

		t.Run("no-op when the binary is already installed ("+c.spec.binaryName+")", func(t *testing.T) {
			beforeDownloads := downloads
			binDir, err := ensureBinary(t.Context(), home, c.spec)
			require.NoError(t, err)
			assert.Equal(t, c.wantBinDir, binDir)
			assert.Equal(t, beforeDownloads, downloads, "an existing install must not re-download")
		})
	}
}

// writeFakeExec drops an executable POSIX shell script named name into dir.
func writeFakeExec(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body), 0o755))
}

// fakeUvRecorder is the body of a stand-in `uv` that records how ensureToolchain
// invoked it (args, the UV_TOOL_* overrides, and whether the setup lock was held)
// to $FAKE_UV_LOG. It uses only shell builtins so it needs nothing on PATH.
const fakeUvRecorder = `{
  echo "args: $*"
  echo "UV_TOOL_DIR=$UV_TOOL_DIR"
  echo "UV_TOOL_BIN_DIR=$UV_TOOL_BIN_DIR"
  [ -f "$FAKE_LOCK_PATH" ] && echo "lock-held"
} > "$FAKE_UV_LOG"`

func TestEnsureToolchain(t *testing.T) {
	// ensureToolchain shells out to POSIX tools and installs a Linux-only toolchain,
	// so the fake executables below only make sense off Windows.
	if runtime.GOOS == "windows" {
		return
	}

	lockPath := func(home string) string { return filepath.Join(home, agentRootDir, setupLockName) }

	t.Run("no-op when the toolchain is already present", func(t *testing.T) {
		home := t.TempDir()
		binDir := t.TempDir()
		writeFakeExec(t, binDir, "ucode", ":")
		writeFakeExec(t, binDir, "npm", ":")
		// A uv that records if it ever runs: a ready toolchain must install nothing.
		writeFakeExec(t, binDir, "uv", fakeUvRecorder)
		marker := filepath.Join(t.TempDir(), "uv.log")
		t.Setenv("FAKE_UV_LOG", marker)
		t.Setenv("PATH", binDir)

		require.NoError(t, ensureToolchain(cmdio.MockDiscard(t.Context()), home, "amd64"))
		assert.NoFileExists(t, marker, "uv must not run when the toolchain is already present")
	})

	t.Run("installs the Unity Gateway CLI via uv when uv is already present", func(t *testing.T) {
		home := t.TempDir()
		binDir := t.TempDir()
		// uv and npm are present; ucode is not, so only ucode gets installed and no
		// download happens (which keeps the test hermetic).
		writeFakeExec(t, binDir, "uv", fakeUvRecorder)
		writeFakeExec(t, binDir, "npm", ":")
		marker := filepath.Join(t.TempDir(), "uv.log")
		t.Setenv("FAKE_UV_LOG", marker)
		t.Setenv("FAKE_LOCK_PATH", lockPath(home))
		t.Setenv("PATH", binDir)

		require.NoError(t, ensureToolchain(cmdio.MockDiscard(t.Context()), home, "amd64"))

		out, err := os.ReadFile(marker)
		require.NoError(t, err)
		assert.Contains(t, string(out), "tool install git+https://github.com/"+ugRepo+"@"+ugCommit)
		assert.Contains(t, string(out), "UV_TOOL_DIR="+filepath.Join(home, uvToolDir))
		assert.Contains(t, string(out), "UV_TOOL_BIN_DIR="+filepath.Join(home, uvToolBinDir))
		assert.Contains(t, string(out), "lock-held", "the install must run while holding the setup lock")
		// The tool bin dir is prepended to PATH so the freshly installed ucode is found.
		assert.Contains(t, filepath.SplitList(os.Getenv("PATH")), filepath.Join(home, uvToolBinDir))
		assert.NoFileExists(t, lockPath(home), "the setup lock is released once setup finishes")
	})

	t.Run("recheck after acquiring the lock skips a redundant install", func(t *testing.T) {
		home := t.TempDir()
		binDir := t.TempDir()
		// uv is present and records if it ever installs; it must not be called once a
		// concurrent client finishes the toolchain while we wait for the lock.
		writeFakeExec(t, binDir, "uv", fakeUvRecorder)
		marker := filepath.Join(t.TempDir(), "uv.log")
		t.Setenv("FAKE_UV_LOG", marker)
		t.Setenv("FAKE_LOCK_PATH", lockPath(home))
		t.Setenv("PATH", binDir) // ucode+npm absent → not ready yet

		// Hold the setup lock to stand in for another client mid-install.
		unlock, err := acquireSetupLock(t.Context(), home)
		require.NoError(t, err)

		ctx := cmdio.MockDiscard(t.Context())
		done := make(chan error, 1)
		go func() { done <- ensureToolchain(ctx, home, "amd64") }()

		// The other client finishes: publish ucode+npm into the PATH dir, then release
		// the lock so the waiting ensureToolchain can proceed.
		writeFakeExec(t, binDir, "ucode", ":")
		writeFakeExec(t, binDir, "npm", ":")
		unlock()

		require.NoError(t, <-done)
		assert.NoFileExists(t, marker, "the post-lock readiness recheck must skip installing")
	})
}

func TestSupportedAgents(t *testing.T) {
	names := SupportedAgentNames()
	assert.Equal(t, []string{"claude", "codex"}, names)

	// Each agent injects context by exactly one mechanism (flag OR home file);
	// the two are never both set.
	expect := map[string]struct{ flag, home string }{
		"claude": {flag: "--append-system-prompt-file"},
		"codex":  {home: ".codex/AGENTS.md"},
	}
	for name, want := range expect {
		a, ok := agentByName(name)
		require.True(t, ok)
		assert.Equal(t, want.flag, a.contextFlag, "%s contextFlag", name)
		assert.Equal(t, want.home, a.contextHomeFile, "%s contextHomeFile", name)
	}

	_, ok := agentByName("not-an-agent")
	assert.False(t, ok)
}

func TestInjectAgentContext(t *testing.T) {
	claude, _ := agentByName("claude")
	codex, _ := agentByName("codex")
	noContext := agentSpec{name: "none"} // an agent with neither mechanism

	t.Run("flag agent writes a scratch file and returns the flag", func(t *testing.T) {
		home := t.TempDir()
		args, err := injectAgentContext(t.Context(), home, claude)
		require.NoError(t, err)
		require.Len(t, args, 2)
		assert.Equal(t, "--append-system-prompt-file", args[0])
		data, err := os.ReadFile(args[1])
		require.NoError(t, err)
		assert.Contains(t, string(data), "databricks ssh connect")
	})

	t.Run("home-file agent writes the global instructions file, no args", func(t *testing.T) {
		home := t.TempDir()
		args, err := injectAgentContext(t.Context(), home, codex)
		require.NoError(t, err)
		assert.Nil(t, args)
		data, err := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "databricks ssh connect")
	})

	t.Run("home-file agent does not clobber an existing file", func(t *testing.T) {
		home := t.TempDir()
		target := filepath.Join(home, ".codex", "AGENTS.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
		require.NoError(t, os.WriteFile(target, []byte("user's own instructions"), 0o644))
		args, err := injectAgentContext(t.Context(), home, codex)
		require.NoError(t, err)
		assert.Nil(t, args)
		data, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, "user's own instructions", string(data))
	})

	t.Run("agent without a mechanism writes nothing", func(t *testing.T) {
		home := t.TempDir()
		args, err := injectAgentContext(t.Context(), home, noContext)
		require.NoError(t, err)
		assert.Nil(t, args)
		entries, err := os.ReadDir(home)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestAgentSystemContext(t *testing.T) {
	assert.Contains(t, agentSystemContext("/Workspace/Users/me@example.com"), "working directory is /Workspace/Users/me@example.com")
	// Falls back to a generic phrase when the workspace home is unknown.
	assert.Contains(t, agentSystemContext(""), "working directory is the user's Databricks workspace home directory")
}

func newProbeClient(t *testing.T, host string) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(&config.Config{
		Host:  host,
		Token: "test-token",
		// The fake gateway server has no /.well-known/databricks-config, so skip
		// the metadata fetch that would otherwise log a noisy resolve warning.
		HostMetadataResolver: func(context.Context, string) (*config.HostMetadata, error) {
			return nil, nil
		},
	}))
	require.NoError(t, err)
	return w
}

// newGatewayServer routes the model-services (v3) and legacy endpoints (v2) probe
// requests to per-API handlers, 404ing anything else (e.g. the SDK's config
// discovery). A nil handler means that API is not served (404).
func newGatewayServer(t *testing.T, modelServices, endpoints http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/2.1/unity-catalog/model-services") && modelServices != nil:
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			modelServices(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/ai-gateway/v2/endpoints") && endpoints != nil:
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			endpoints(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("{}"))
		}
	}))
}

// jsonHandler replies with status and body for every request it receives.
func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestProbeAIGateway(t *testing.T) {
	const modelServicesBody = `{"model_services":[{"name":"model-services/system.ai.gpt-5"}]}`
	const endpointsBody = `{"endpoints":[{"name":"databricks-gpt-5"}]}`
	const emptyBody = `{}`

	t.Run("model service available: connected, legacy not probed", func(t *testing.T) {
		var legacyCalls int
		srv := newGatewayServer(t,
			jsonHandler(http.StatusOK, modelServicesBody),
			func(w http.ResponseWriter, r *http.Request) {
				legacyCalls++
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(emptyBody))
			},
		)
		defer srv.Close()
		require.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
		assert.Zero(t, legacyCalls, "legacy endpoints must not be probed once a model service is found")
	})

	t.Run("model service found across pages", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page_token") == "" {
				_, _ = w.Write([]byte(`{"next_page_token":"cursor-1"}`))
				return
			}
			_, _ = w.Write([]byte(modelServicesBody))
		}, nil)
		defer srv.Close()
		require.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
	})

	t.Run("empty model service but legacy reachable: proceeds", func(t *testing.T) {
		// v3 200-empty and v2 200-empty: both reachable, no resources → proceed (warn).
		srv := newGatewayServer(t, jsonHandler(http.StatusOK, emptyBody), jsonHandler(http.StatusOK, emptyBody))
		defer srv.Close()
		assert.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
	})

	t.Run("legacy-only workspace (v3 404, v2 reachable): proceeds", func(t *testing.T) {
		srv := newGatewayServer(t,
			jsonHandler(http.StatusNotFound, `{"message":"not found"}`),
			jsonHandler(http.StatusOK, endpointsBody),
		)
		defer srv.Close()
		assert.NoError(t, probeAIGateway(t.Context(), newProbeClient(t, srv.URL)))
	})

	t.Run("auth failure (401) fails fast without probing legacy", func(t *testing.T) {
		var legacyCalls int
		srv := newGatewayServer(t,
			jsonHandler(http.StatusUnauthorized, `{"message":"unauthorized"}`),
			func(w http.ResponseWriter, r *http.Request) {
				legacyCalls++
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(endpointsBody))
			},
		)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rejected the access token")
		assert.Zero(t, legacyCalls, "a definitive auth failure must not fall back to the legacy probe")
	})

	t.Run("missing OAuth scope on both paths routes to re-auth", func(t *testing.T) {
		scope := jsonHandler(http.StatusForbidden, `{"error_code":"PERMISSION_DENIED","message":"Provided OAuth token does not have required scopes: unity-catalog"}`)
		srv := newGatewayServer(t, scope, scope)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing an OAuth scope")
		assert.Contains(t, err.Error(), "databricks auth login")
	})

	t.Run("plain 403 on both paths is a permission failure", func(t *testing.T) {
		forbidden := jsonHandler(http.StatusForbidden, `{"message":"forbidden"}`)
		srv := newGatewayServer(t, forbidden, forbidden)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model service access could not be verified")
	})

	t.Run("neither gateway available: not enabled", func(t *testing.T) {
		notFound := jsonHandler(http.StatusNotFound, `{"message":"not found"}`)
		srv := newGatewayServer(t, notFound, notFound)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled on this workspace")
	})

	t.Run("transient failure (5xx both) is retryable, not disabled", func(t *testing.T) {
		unavailable := jsonHandler(http.StatusServiceUnavailable, `{"message":"try later"}`)
		srv := newGatewayServer(t, unavailable, unavailable)
		defer srv.Close()
		err := probeAIGateway(t.Context(), newProbeClient(t, srv.URL))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transient error")
		assert.NotContains(t, err.Error(), "not enabled")
	})
}

func TestProbeModelServicesPageError(t *testing.T) {
	// A failure while paging surfaces as a probe error: the iterator can't tell a
	// first-page failure from a later-page one, so any page error is not treated
	// as a reachable "empty".
	srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page_token") == "" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"next_page_token":"cursor-1"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}, nil)
	defer srv.Close()

	apiClient, err := newProbeAPIClient(newProbeClient(t, srv.URL).Config)
	require.NoError(t, err)
	probe := probeModelServices(t.Context(), apiClient)
	assert.False(t, probe.reachable)
	assert.False(t, probe.resourceAvailable)
	assert.Error(t, probe.err)
}
