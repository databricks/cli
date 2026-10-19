package testserver

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRootPath = "/Workspace/Users/someone/orders"

func createTestPipelineWithTests(t *testing.T, workspace *FakeWorkspace) string {
	response := workspace.PipelineCreate(Request{Body: []byte(`{"name": "orders", "root_path": "` + testRootPath + `"}`)})
	require.Equal(t, 0, response.StatusCode)
	workspace.files[testRootPath+"/tests/test_orders.py"] = FileEntry{Data: []byte("def test_ok():\n    pass\n\ndef test_fails():\n    pass\n\n    def test_nested():\n        pass\n")}
	workspace.files[testRootPath+"/tests/test_slow.py"] = FileEntry{Data: []byte("def test_hang():\n    pass\n")}
	workspace.files[testRootPath+"/tests/helpers.py"] = FileEntry{Data: []byte("def test_helper():\n    pass\n")}
	return response.Body.(pipelines.CreatePipelineResponse).PipelineId
}

func startTestUpdate(t *testing.T, workspace *FakeWorkspace, pipelineID, body string) string {
	response := workspace.PipelineStartUpdate(pipelineID, []byte(body))
	require.Equal(t, 0, response.StatusCode, response.Body)
	return response.Body.(pipelines.StartUpdateResponse).UpdateId
}

func getUpdateState(workspace *FakeWorkspace, pipelineID, updateID string) pipelines.UpdateInfoState {
	return workspace.PipelineGetUpdate(pipelineID, updateID).Body.(pipelines.GetUpdateResponse).Update.State
}

func listEvents(t *testing.T, workspace *FakeWorkspace, pipelineID string, query url.Values) ([]fakePipelineEvent, string) {
	response := workspace.PipelineEvents(pipelineID, query)
	require.Equal(t, 0, response.StatusCode, response.Body)
	body := response.Body.(map[string]any)
	token, _ := body["next_page_token"].(string)
	return body["events"].([]fakePipelineEvent), token
}

func TestCollectPipelineTests(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	createTestPipelineWithTests(t, workspace)

	assert.Equal(t, []string{"tests/test_orders.py::test_ok", "tests/test_orders.py::test_fails", "tests/test_slow.py::test_hang"},
		workspace.collectPipelineTests(testRootPath, nil))
	assert.Equal(t, []string{"tests/test_orders.py::test_fails"},
		workspace.collectPipelineTests(testRootPath, []string{testRootPath + "/tests/test_orders.py::test_fails", testRootPath + "/tests/test_orders.py::test_fails"}))
	assert.Equal(t, []string{"tests/helpers.py::test_helper"},
		workspace.collectPipelineTests(testRootPath, []string{testRootPath + "/tests/helpers.py"}))
	assert.Equal(t, []string{testRootPath + "/tests/test_slow.py::test_hang"},
		workspace.collectPipelineTests("", []string{testRootPath + "/tests/test_slow.py"}))
	assert.Empty(t, workspace.collectPipelineTests("", nil))
}

func TestPipelineTestUpdate(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	pipelineID := createTestPipelineWithTests(t, workspace)
	updateID := startTestUpdate(t, workspace, pipelineID, `{"test_only": true, "test_options": {"test_selection": ["`+testRootPath+`/tests/test_orders.py"]}}`)
	query := url.Values{"filter": {"update_id='" + updateID + "'"}, "order_by": {"timestamp asc"}}

	events, _ := listEvents(t, workspace, pipelineID, query)
	assert.Empty(t, events)

	assert.Equal(t, pipelines.UpdateInfoStateWaitingForResources, getUpdateState(workspace, pipelineID, updateID))
	assert.Equal(t, pipelines.UpdateInfoStateRunning, getUpdateState(workspace, pipelineID, updateID))
	events, _ = listEvents(t, workspace, pipelineID, query)
	var statuses []string
	for _, e := range events {
		statuses = append(statuses, e.Origin.TestCaseId+" "+e.Details["test_case_progress"].(map[string]any)["status"].(string))
	}
	assert.Equal(t, []string{
		"tests/test_orders.py::test_ok PENDING",
		"tests/test_orders.py::test_fails PENDING",
		"tests/test_orders.py::test_ok RUNNING",
		"tests/test_orders.py::test_ok PASSED",
		"tests/test_orders.py::test_fails RUNNING",
		"tests/test_orders.py::test_fails FAILED",
	}, statuses)

	assert.Equal(t, pipelines.UpdateInfoStateFailed, getUpdateState(workspace, pipelineID, updateID))
	query.Set("since_event_id", events[len(events)-1].Id)
	events, _ = listEvents(t, workspace, pipelineID, query)
	require.Len(t, events, 1)
	assert.Equal(t, map[string]any{"total": 2, "passed": 1, "failed": 1, "skipped": 0, "errors": 0}, events[0].Details["test_run_summary"])
}

func TestPipelineEventsPages(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	pipelineID := createTestPipelineWithTests(t, workspace)
	updateID := startTestUpdate(t, workspace, pipelineID, `{"test_only": true}`)
	for range 3 {
		getUpdateState(workspace, pipelineID, updateID)
	}

	all, token := listEvents(t, workspace, pipelineID, url.Values{"order_by": {"timestamp asc"}, "max_results": {"100"}})
	assert.Empty(t, token)
	require.Len(t, all, 8)

	var paged []fakePipelineEvent
	query := url.Values{"filter": {"update_id='" + updateID + "'"}, "order_by": {"timestamp asc"}, "max_results": {"3"}}
	for {
		events, token := listEvents(t, workspace, pipelineID, query)
		paged = append(paged, events...)
		if token == "" {
			break
		}
		query = url.Values{"page_token": {token}, "max_results": {"3"}}
	}
	assert.Equal(t, all, paged)

	// Events are newest first by default.
	newest, _ := listEvents(t, workspace, pipelineID, url.Values{"max_results": {"1"}})
	assert.Equal(t, all[len(all)-1].Id, newest[0].Id)

	response := workspace.PipelineEvents(pipelineID, url.Values{"page_token": {"x"}, "filter": {"level='INFO'"}})
	assert.Equal(t, 400, response.StatusCode)
}

func TestPipelineTestUpdateConflictAndStop(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	pipelineID := createTestPipelineWithTests(t, workspace)
	updateID := startTestUpdate(t, workspace, pipelineID, `{"test_only": true, "test_options": {"test_selection": ["`+testRootPath+`/tests/test_slow.py"]}}`)

	response := workspace.PipelineStartUpdate(pipelineID, nil)
	assert.Equal(t, 409, response.StatusCode)
	assert.Equal(t, "An active update '"+updateID+"' already exists for pipeline '"+pipelineID+"'.", response.Body.(map[string]string)["message"])

	workspace.PipelineStop(pipelineID)
	for range 2 {
		getUpdateState(workspace, pipelineID, updateID)
	}
	assert.Equal(t, pipelines.UpdateInfoStateCanceled, getUpdateState(workspace, pipelineID, updateID))
	startTestUpdate(t, workspace, pipelineID, `{}`)
}

func TestPipelineStartUpdateRejectsTestOptionsWithoutTestOnly(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	pipelineID := createTestPipelineWithTests(t, workspace)

	response := workspace.PipelineStartUpdate(pipelineID, []byte(`{"test_options": {"pytest_args": ["-k", "x"]}}`))
	assert.Equal(t, 400, response.StatusCode)
}
