package aircmd

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// relaunchServer records, in arrival order, the calls that decide whether a
// relaunch may write, and lets a test choose what Me answers.
type relaunchServer struct {
	mu sync.Mutex
	// userName is what Me returns; meStatus, when set, makes Me fail instead.
	userName string
	meStatus int
	// awaitSpeculation makes Me wait until the speculative probe and validation
	// have arrived, and awaitExperimentDirRead until the experiment directory
	// read has, so the test sees them before Me, not by scheduling luck.
	awaitSpeculation       bool
	awaitExperimentDirRead bool
	// experimentDir, when set, is recorded as "expdir read" and "expdir mkdirs"
	// instead of "probe" and "mkdirs".
	experimentDir  string
	events         []string
	validateBodies []string
	submitPaths    []string
}

func (s *relaunchServer) record(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

// take returns the events recorded since the last call.
func (s *relaunchServer) take() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.events
	s.events = nil
	return events
}

func (s *relaunchServer) setUser(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userName = name
}

func (s *relaunchServer) setMeStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meStatus = status
}

func (s *relaunchServer) hasEvent(event string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(s.events, event)
}

// isExperimentDir reports whether p is the experiment directory under test.
func (s *relaunchServer) isExperimentDir(p string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.experimentDir != "" && p == s.experimentDir
}

// awaitBeforeMe blocks until the requests the test expects before Me have
// arrived, or gives up after a few seconds so a regression fails instead of
// hanging.
func (s *relaunchServer) awaitBeforeMe() {
	s.mu.Lock()
	var want []string
	if s.awaitSpeculation {
		want = append(want, "probe", "validate")
	}
	if s.awaitExperimentDirRead {
		want = append(want, "expdir read")
	}
	s.mu.Unlock()
	missing := func(event string) bool { return !s.hasEvent(event) }
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && slices.ContainsFunc(want, missing) {
		time.Sleep(time.Millisecond)
	}
}

func newRelaunchClient(t *testing.T) (*databricks.WorkspaceClient, *relaunchServer) {
	t.Helper()
	s := &relaunchServer{userName: "a@example.com"}
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// Register before AddDefaultHandlers: the router is first-wins.
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(testserver.Request) any {
		s.awaitBeforeMe()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.events = append(s.events, "me")
		if s.meStatus != 0 {
			return testserver.Response{StatusCode: s.meStatus, Body: `{"error_code":"PERMISSION_DENIED","message":"denied"}`}
		}
		return iam.User{UserName: s.userName}
	})
	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		p := req.URL.Query().Get("path")
		if s.isExperimentDir(p) {
			s.record("expdir read")
		} else {
			s.record("probe")
		}
		return req.Workspace.WorkspaceGetStatus(p, false)
	})
	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(req testserver.Request) any {
		var request workspace.Mkdirs
		require.NoError(t, json.Unmarshal(req.Body, &request))
		if s.isExperimentDir(request.Path) {
			s.record("expdir mkdirs")
		} else {
			s.record("mkdirs")
		}
		req.Workspace.WorkspaceMkdirs(request)
		return struct{}{}
	})
	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
		s.record("import " + req.Vars["path"])
		return req.Workspace.WorkspaceFilesImportFile(req.Vars["path"], req.Body, req.URL.Query().Get("overwrite") == "true")
	})
	server.Handle("POST", validateConfigPath, func(req testserver.Request) any {
		s.record("validate")
		s.mu.Lock()
		s.validateBodies = append(s.validateBodies, string(req.Body))
		s.mu.Unlock()
		return validateConfigResponse{}
	})
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		s.record("submit")
		var got jobs.SubmitRun
		require.NoError(t, json.Unmarshal(req.Body, &got))
		s.mu.Lock()
		s.submitPaths = append(s.submitPaths, got.Tasks[0].AiRuntimeTask.Deployments[0].CommandPath)
		s.mu.Unlock()
		return jobs.SubmitRunResponse{RunId: 777}
	})
	testserver.AddDefaultHandlers(server)

	// A fixed workspace ID keeps the run-link lookup from issuing its own Me call.
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token", WorkspaceID: "1"})
	require.NoError(t, err)
	return w, s
}

func submitMinimal(t *testing.T, w *databricks.WorkspaceClient, key string) error {
	t.Helper()
	return submitConfig(t, w, minimalConfig, key)
}

func submitConfig(t *testing.T, w *databricks.WorkspaceClient, content, key string) error {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", content)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	_, _, err = submitWorkload(t.Context(), w, cfg, cfgPath, key, false)
	return err
}

// A relaunch probes and validates its remembered dir before Me returns, and makes
// no write before Me: nothing is uploaded, and the submit follows Me.
func TestRelaunchSpeculatesBeforeMe(t *testing.T) {
	w, s := newRelaunchClient(t)
	require.NoError(t, submitMinimal(t, w, "idem-1"))
	s.take()

	s.mu.Lock()
	s.awaitSpeculation = true
	s.mu.Unlock()
	require.NoError(t, submitMinimal(t, w, "idem-2"))
	events := s.take()

	me := slices.Index(events, "me")
	require.GreaterOrEqual(t, me, 0, events)
	assert.Contains(t, events[:me], "probe")
	assert.Contains(t, events[:me], "validate")
	assert.NotContains(t, events, "mkdirs")
	for _, event := range events {
		assert.False(t, strings.HasPrefix(event, "import"), "relaunch uploaded: %s", event)
	}
	assert.Greater(t, slices.Index(events, "submit"), me, events)
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.submitPaths, 2)
	assert.Contains(t, s.submitPaths[1], "/Workspace/Users/a@example.com/.air/cli_launch/")
}

// A remembered dir under a different name is discarded once Me answers: the
// speculative reads and validation stay under the old name, every write goes to
// the new home, and the index is re-pointed there.
func TestRelaunchMismatchedUserDiscardsSpeculation(t *testing.T) {
	w, s := newRelaunchClient(t)
	require.NoError(t, submitMinimal(t, w, "idem-1"))
	s.take()

	s.setUser("b@example.com")
	s.mu.Lock()
	s.awaitSpeculation = true
	s.mu.Unlock()
	require.NoError(t, submitMinimal(t, w, "idem-2"))
	events := s.take()

	me := slices.Index(events, "me")
	require.GreaterOrEqual(t, me, 0, events)
	assert.Contains(t, events[:me], "probe")
	assert.Contains(t, events[:me], "validate")
	var uploads []string
	for _, event := range events {
		if p, ok := strings.CutPrefix(event, "import "); ok {
			uploads = append(uploads, p)
		}
	}
	require.NotEmpty(t, uploads, events)
	for _, p := range uploads {
		assert.True(t, strings.HasPrefix(p, "Workspace/Users/b@example.com/"), p)
	}
	assert.Greater(t, slices.Index(events, "submit"), me, events)
	// Validation order: the first run, the speculative check under the remembered
	// home, then the real check under b.
	s.mu.Lock()
	require.Len(t, s.validateBodies, 3)
	assert.Contains(t, s.validateBodies[1], "/Workspace/Users/a@example.com/", "speculative validation used the remembered home")
	assert.Contains(t, s.validateBodies[2], "/Workspace/Users/b@example.com/")
	s.mu.Unlock()

	// The re-pointed entry now belongs to b, so the next relaunch under b is a hit.
	require.NoError(t, submitMinimal(t, w, "idem-3"))
	events = s.take()
	assert.NotContains(t, events, "mkdirs")
	for _, event := range events {
		assert.False(t, strings.HasPrefix(event, "import"), "relaunch under b uploaded: %s", event)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.submitPaths, 3)
	assert.Contains(t, s.submitPaths[2], "/Workspace/Users/b@example.com/")
}

// A failing Me fails the submission exactly as before, and nothing written or
// submitted follows it.
func TestRelaunchMeFailureWritesNothing(t *testing.T) {
	w, s := newRelaunchClient(t)
	require.NoError(t, submitMinimal(t, w, "idem-1"))
	s.take()

	s.setMeStatus(403)
	s.mu.Lock()
	s.awaitSpeculation = true
	s.mu.Unlock()
	err := submitMinimal(t, w, "idem-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve current user")
	events := s.take()
	assert.NotContains(t, events, "submit")
	assert.NotContains(t, events, "mkdirs")
	for _, event := range events {
		assert.False(t, strings.HasPrefix(event, "import"), "wrote after failed Me: %s", event)
	}
}

const testExperimentDir = "/Workspace/Users/a@example.com/exp"

// newExperimentDirClient serves a launch whose config sets a missing
// mlflow_experiment_directory, with Me held until the directory read arrives.
func newExperimentDirClient(t *testing.T) (*databricks.WorkspaceClient, *relaunchServer) {
	t.Helper()
	w, s := newRelaunchClient(t)
	s.mu.Lock()
	s.experimentDir = testExperimentDir
	s.awaitExperimentDirRead = true
	s.mu.Unlock()
	return w, s
}

const experimentDirConfig = minimalConfig + "mlflow_experiment_directory: " + testExperimentDir + "\n"

// The experiment directory is read before Me returns, and created only after Me
// and before the submit.
func TestExperimentDirReadBeforeMe(t *testing.T) {
	w, s := newExperimentDirClient(t)
	require.NoError(t, submitConfig(t, w, experimentDirConfig, "idem-1"))
	events := s.take()

	me := slices.Index(events, "me")
	require.GreaterOrEqual(t, me, 0, events)
	assert.Contains(t, events[:me], "expdir read")
	mkdirs := slices.Index(events, "expdir mkdirs")
	assert.Greater(t, mkdirs, me, events)
	assert.Greater(t, slices.Index(events, "submit"), mkdirs, events)
}

// A failing Me stops the launch before the experiment directory is created.
func TestExperimentDirMeFailureCreatesNothing(t *testing.T) {
	w, s := newExperimentDirClient(t)
	s.setMeStatus(403)
	err := submitConfig(t, w, experimentDirConfig, "idem-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve current user")
	events := s.take()
	assert.Contains(t, events, "expdir read")
	assert.NotContains(t, events, "expdir mkdirs")
	assert.NotContains(t, events, "submit")
}
