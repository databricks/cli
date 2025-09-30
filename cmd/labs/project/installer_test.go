package project_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/python"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ownerRWXworldRX = 0o755
	ownerRW         = 0o600
)

func zipballFromFolder(src string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	rootDir := path.Base(src) // this is required to emulate github ZIP downloads
	err := filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relpath, err := filepath.Rel(src, filePath)
		if err != nil {
			return err
		}
		relpath = path.Join(rootDir, relpath)
		if info.IsDir() {
			_, err = zw.Create(relpath + "/")
			return err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()
		f, err := zw.Create(relpath)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, file)
		return err
	})
	if err != nil {
		return nil, err
	}
	err = zw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func copyTestdata(t *testing.T, name string) string {
	// TODO: refactor fs.cp command into a reusable util
	tempDir := t.TempDir()
	name = strings.ReplaceAll(name, "/", string(os.PathSeparator))
	err := filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		dst := strings.TrimPrefix(path, name)
		if dst == "" {
			return nil
		}
		if d.IsDir() {
			err := os.MkdirAll(filepath.Join(tempDir, dst), ownerRWXworldRX)
			require.NoError(t, err)
			return nil
		}
		in, err := os.Open(path)
		require.NoError(t, err)
		defer in.Close()
		out, err := os.Create(filepath.Join(tempDir, dst))
		require.NoError(t, err)
		defer out.Close()
		_, err = io.Copy(out, in)
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)
	return tempDir
}

func installerContext(t *testing.T, server *httptest.Server) context.Context {
	ctx := context.Background()
	ctx = github.WithApiOverride(ctx, server.URL)
	ctx = github.WithUserContentOverride(ctx, server.URL)
	ctx = env.WithUserHomeDir(ctx, t.TempDir())
	// trick release cache to thing it went to github already
	cachePath, _ := project.PathInLabs(ctx, "blueprint", "cache")
	err := os.MkdirAll(cachePath, ownerRWXworldRX)
	require.NoError(t, err)
	bs := []byte(`{"refreshed_at": "2033-01-01T00:00:00.92857+02:00","data": [{"tag_name": "v0.3.15"}]}`)
	err = os.WriteFile(filepath.Join(cachePath, "databrickslabs-blueprint-releases.json"), bs, ownerRW)
	require.NoError(t, err)
	return ctx
}

func respondWithJSON(t *testing.T, w http.ResponseWriter, v any) {
	raw, err := json.Marshal(v)
	require.NoError(t, err)

	_, err = w.Write(raw)
	require.NoError(t, err)
}

type fileTree struct {
	Path     string
	MaxDepth int
}

func (ft fileTree) String() string {
	lines := ft.listFiles(ft.Path, ft.MaxDepth)
	return strings.Join(lines, "\n")
}

func (ft fileTree) listFiles(dir string, depth int) (lines []string) {
	if ft.MaxDepth > 0 && depth > ft.MaxDepth {
		return []string{fmt.Sprintf("deeper than %d levels", ft.MaxDepth)}
	}
	fileInfo, err := os.ReadDir(dir)
	if err != nil {
		return []string{err.Error()}
	}
	for _, entry := range fileInfo {
		lines = append(lines, fmt.Sprintf("%s%s", ft.getIndent(depth), entry.Name()))
		if entry.IsDir() {
			subdir := filepath.Join(dir, entry.Name())
			lines = append(lines, ft.listFiles(subdir, depth+1)...)
		}
	}
	return lines
}

func (ft fileTree) getIndent(depth int) string {
	return "│" + strings.Repeat(" ", depth*2) + "├─ "
}

func TestInstallerWorksForReleases(t *testing.T) {
	defer func() {
		if !t.Failed() {
			return
		}
		t.Logf("file tree:\n%s", fileTree{
			Path: filepath.Dir(t.TempDir()),
		})
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/databrickslabs/blueprint/v0.3.15/labs.yml" {
			raw, err := os.ReadFile("testdata/installed-in-home/.databricks/labs/blueprint/lib/labs.yml")
			assert.NoError(t, err)
			_, err = w.Write(raw)
			assert.NoError(t, err)
			return
		}
		if r.URL.Path == "/repos/databrickslabs/blueprint/zipball/v0.3.15" {
			raw, err := zipballFromFolder("testdata/installed-in-home/.databricks/labs/blueprint/lib")
			assert.NoError(t, err)
			w.Header().Add("Content-Type", "application/octet-stream")
			_, err = w.Write(raw)
			assert.NoError(t, err)
			return
		}
		if r.URL.Path == "/api/2.1/clusters/get" {
			respondWithJSON(t, w, &compute.ClusterDetails{
				State: compute.StateRunning,
			})
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		t.FailNow()
	}))
	defer server.Close()

	ctx := installerContext(t, server)

	ctx, stub := process.WithStub(ctx)
	stub.WithStdoutFor(`python[\S]+ --version`, "Python 3.10.5")
	// on Unix, we call `python3`, but on Windows it is `python.exe`
	stub.WithStderrFor(`python[\S]+ -m venv .*/.databricks/labs/blueprint/state/venv`, "[mock venv create]")
	stub.WithStderrFor(`python[\S]+ -m pip install --upgrade --upgrade-strategy eager .`, "[mock pip install]")
	stub.WithStdoutFor(`python[\S]+ install.py`, "setting up important infrastructure")

	// simulate the case of GitHub Actions
	ctx = env.Set(ctx, "DATABRICKS_HOST", server.URL)
	ctx = env.Set(ctx, "DATABRICKS_TOKEN", "...")
	ctx = env.Set(ctx, "DATABRICKS_CLUSTER_ID", "installer-cluster")
	ctx = env.Set(ctx, "DATABRICKS_WAREHOUSE_ID", "installer-warehouse")

	// After the installation, we'll have approximately the following state:
	// t.TempDir()
	// └── 001 <------------------------------------------------- env.UserHomeDir(ctx)
	//     ├── .databricks
	//     │   └── labs
	//     │       └── blueprint
	//     │           ├── cache <------------------------------- prj.CacheDir(ctx)
	//     │           │   └── databrickslabs-blueprint-releases.json
	//     │           ├── config
	//     │           ├── lib <--------------------------------- prj.LibDir(ctx)
	//     │           │   ├── install.py
	//     │           │   ├── labs.yml
	//     │           │   ├── main.py
	//     │           │   └── pyproject.toml
	//     │           └── state <------------------------------- prj.StateDir(ctx)
	//     │               ├── venv <---------------------------- prj.virtualEnvPath(ctx)
	//     │               │   ├── bin
	//     │               │   │   ├── pip
	//     │               │   │   ├── ...
	//     │               │   │   ├── python -> python3.9
	//     │               │   │   ├── python3 -> python3.9 <---- prj.virtualEnvPython(ctx)
	//     │               │   │   └── python3.9 -> (path to a detected python)
	//     │               │   ├── include
	//     │               │   ├── lib
	//     │               │   │   └── python3.9
	//     │               │   │       └── site-packages
	//     │               │   │           ├── ...
	//     │               │   │           ├── distutils-precedence.pth
	r := testcli.NewRunner(t, ctx, "labs", "install", "blueprint", "--debug")
	r.RunAndExpectOutput("setting up important infrastructure")
}

func TestOfflineInstallerWorksForReleases(t *testing.T) {
	// This cmd is useful in systems where there is internet restriction, the user should follow a set-up as follows:
	// install a labs project on a machine which has internet
	// zip and copy the file to the intended machine and
	// run databricks labs install --offline=true
	// it will look for the code in the same install directory and if present, install from there.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.1/clusters/get" {
			respondWithJSON(t, w, &compute.ClusterDetails{
				State: compute.StateRunning,
			})
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		t.FailNow()
	}))
	defer server.Close()

	ctx := installerContext(t, server)
	newHome := copyTestdata(t, "testdata/installed-in-home")
	ctx = env.WithUserHomeDir(ctx, newHome)

	ctx, stub := process.WithStub(ctx)
	stub.WithStdoutFor(`python[\S]+ --version`, "Python 3.10.5")
	// on Unix, we call `python3`, but on Windows it is `python.exe`
	stub.WithStderrFor(`python[\S]+ -m venv .*/.databricks/labs/blueprint/state/venv`, "[mock venv create]")
	stub.WithStderrFor(`python[\S]+ -m pip install --upgrade --upgrade-strategy eager .`, "[mock pip install]")
	stub.WithStdoutFor(`python[\S]+ install.py`, "setting up important infrastructure")

	// simulate the case of GitHub Actions
	ctx = env.Set(ctx, "DATABRICKS_HOST", server.URL)
	ctx = env.Set(ctx, "DATABRICKS_TOKEN", "...")
	ctx = env.Set(ctx, "DATABRICKS_CLUSTER_ID", "installer-cluster")
	ctx = env.Set(ctx, "DATABRICKS_WAREHOUSE_ID", "installer-warehouse")

	r := testcli.NewRunner(t, ctx, "labs", "install", "blueprint", "--offline=true", "--debug")
	r.RunAndExpectOutput("setting up important infrastructure")
}

func TestInstallerWorksForDevelopment(t *testing.T) {
	defer func() {
		if !t.Failed() {
			return
		}
		t.Logf("file tree:\n%s", fileTree{
			Path: filepath.Dir(t.TempDir()),
		})
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.1/clusters/list" {
			respondWithJSON(t, w, compute.ListClustersResponse{
				Clusters: []compute.ClusterDetails{
					{
						ClusterId:        "abc-id",
						ClusterName:      "first shared",
						DataSecurityMode: compute.DataSecurityModeUserIsolation,
						SparkVersion:     "12.2.x-whatever",
						State:            compute.StateRunning,
					},
					{
						ClusterId:        "bcd-id",
						ClusterName:      "second personal",
						DataSecurityMode: compute.DataSecurityModeSingleUser,
						SparkVersion:     "14.5.x-whatever",
						State:            compute.StateRunning,
						SingleUserName:   "serge",
					},
				},
			})
			return
		}
		if r.URL.Path == "/api/2.0/preview/scim/v2/Me" {
			respondWithJSON(t, w, iam.User{
				UserName: "serge",
			})
			return
		}
		if r.URL.Path == "/api/2.1/clusters/spark-versions" {
			respondWithJSON(t, w, compute.GetSparkVersionsResponse{
				Versions: []compute.SparkVersion{
					{
						Key:  "14.5.x-whatever",
						Name: "14.5 (Awesome)",
					},
				},
			})
			return
		}
		if r.URL.Path == "/api/2.1/clusters/get" {
			respondWithJSON(t, w, &compute.ClusterDetails{
				State: compute.StateRunning,
			})
			return
		}
		if r.URL.Path == "/api/2.0/sql/warehouses" {
			respondWithJSON(t, w, sql.ListWarehousesResponse{
				Warehouses: []sql.EndpointInfo{
					{
						Id:            "efg-id",
						Name:          "First PRO Warehouse",
						WarehouseType: sql.EndpointInfoWarehouseTypePro,
					},
				},
			})
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		t.FailNow()
	}))
	defer server.Close()

	wd, _ := os.Getwd()
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	devDir := copyTestdata(t, "testdata/installed-in-home/.databricks/labs/blueprint/lib")
	err := os.Chdir(devDir)
	require.NoError(t, err)

	ctx := installerContext(t, server)
	py, _ := python.DetectExecutable(ctx)
	py, _ = filepath.Abs(py)

	// development installer assumes it's in the active virtualenv
	ctx = env.Set(ctx, "PYTHON_BIN", py)
	home, _ := env.UserHomeDir(ctx)
	err = os.WriteFile(filepath.Join(home, ".databrickscfg"), []byte(fmt.Sprintf(`
[profile-one]
host = %s
token = ...

[acc]
host = %s
account_id = abc
	`, server.URL, server.URL)), ownerRW)
	require.NoError(t, err)

	// We have the following state at this point:
	// t.TempDir()
	// ├── 001 <------------------ $CWD, prj.EffectiveLibDir(ctx), prj.folder
	// │   ├── install.py
	// │   ├── labs.yml <--------- prj.IsDeveloperMode(ctx) == true
	// │   ├── main.py
	// │   └── pyproject.toml
	// └── 002 <------------------ env.UserHomeDir(ctx)
	// 	└── .databricks
	// 		└── labs
	// 			└── blueprint <--- project.PathInLabs(ctx, "blueprint"), prj.rootDir(ctx)
	// 				└── cache <--- prj.CacheDir(ctx)
	// 					└── databrickslabs-blueprint-releases.json

	// `databricks labs install .` means "verify this installer i'm developing does work"
	r := testcli.NewRunner(t, ctx, "labs", "install", ".")
	r.WithStdin()
	defer r.CloseStdin()

	r.RunBackground()
	r.WaitForTextPrinted("setting up important infrastructure", 5*time.Second)
}

func TestUpgraderWorksForReleases(t *testing.T) {
	defer func() {
		if !t.Failed() {
			return
		}
		t.Logf("file tree:\n%s", fileTree{
			Path: filepath.Dir(t.TempDir()),
		})
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/databrickslabs/blueprint/v0.4.0/labs.yml" {
			raw, err := os.ReadFile("testdata/installed-in-home/.databricks/labs/blueprint/lib/labs.yml")
			assert.NoError(t, err)
			_, err = w.Write(raw)
			assert.NoError(t, err)
			return
		}
		if r.URL.Path == "/repos/databrickslabs/blueprint/zipball/v0.4.0" {
			raw, err := zipballFromFolder("testdata/installed-in-home/.databricks/labs/blueprint/lib")
			assert.NoError(t, err)
			w.Header().Add("Content-Type", "application/octet-stream")
			_, err = w.Write(raw)
			assert.NoError(t, err)
			return
		}
		if r.URL.Path == "/api/2.1/clusters/get" {
			respondWithJSON(t, w, &compute.ClusterDetails{
				State: compute.StateRunning,
			})
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		t.FailNow()
	}))
	defer server.Close()

	ctx := installerContext(t, server)

	newHome := copyTestdata(t, "testdata/installed-in-home")
	ctx = env.WithUserHomeDir(ctx, newHome)

	// Install stubs for the python calls we need to ensure were run in the
	// upgrade process.
	ctx, stub := process.WithStub(ctx)
	stub.WithStderrFor(`python[\S]+ -m pip install --upgrade --upgrade-strategy eager .`, "[mock pip install]")
	stub.WithStdoutFor(`python[\S]+ install.py`, "setting up important infrastructure")

	py, _ := python.DetectExecutable(ctx)
	py, _ = filepath.Abs(py)
	ctx = env.Set(ctx, "PYTHON_BIN", py)

	cachePath, _ := project.PathInLabs(ctx, "blueprint", "cache")
	bs := []byte(`{"refreshed_at": "2033-01-01T00:00:00.92857+02:00","data": [{"tag_name": "v0.4.0"}]}`)
	err := os.WriteFile(filepath.Join(cachePath, "databrickslabs-blueprint-releases.json"), bs, ownerRW)
	require.NoError(t, err)

	// simulate the case of GitHub Actions
	ctx = env.Set(ctx, "DATABRICKS_HOST", server.URL)
	ctx = env.Set(ctx, "DATABRICKS_TOKEN", "...")
	ctx = env.Set(ctx, "DATABRICKS_CLUSTER_ID", "installer-cluster")
	ctx = env.Set(ctx, "DATABRICKS_WAREHOUSE_ID", "installer-warehouse")

	r := testcli.NewRunner(t, ctx, "labs", "upgrade", "blueprint")
	r.RunAndExpectOutput("setting up important infrastructure")

	// Check if the stub was called with the 'python -m pip install' command
	pi := false
	for _, call := range stub.Commands() {
		if strings.HasSuffix(call, "-m pip install --upgrade --upgrade-strategy eager .") {
			pi = true
			break
		}
	}
	if !pi {
		t.Logf(`Expected stub command 'python[\S]+ -m pip install --upgrade --upgrade-strategy eager .' not found`)
		t.FailNow()
	}
}
