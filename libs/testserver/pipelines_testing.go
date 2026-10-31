package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// Pipeline event types, which also name the event's details field.
const (
	pipelineEventUpdateProgress   = "update_progress"
	pipelineEventTestCaseProgress = "test_case_progress"
	pipelineEventTestRunSummary   = "test_run_summary"
)

// fakePipelineUpdate is a pipeline update whose state advances by one step on each Get Update call.
type fakePipelineUpdate struct {
	pipelineID string
	testOnly   bool
	states     []pipelines.UpdateInfoState
	polls      int

	// canceled is set by cancel. CANCELED then holds the index of a state the update never reached.
	canceled bool
}

// stateIndex returns the index of the state returned by the last Get Update call, or -1 before the
// first call.
func (u *fakePipelineUpdate) stateIndex() int {
	return min(u.polls, len(u.states)) - 1
}

// eventIndex returns the index of the last state whose events are visible. Events emitted for a
// later state are not visible yet, and never are for a state that a canceled update did not reach.
func (u *fakePipelineUpdate) eventIndex() int {
	if u.canceled {
		return len(u.states) - 2
	}
	return u.stateIndex()
}

// active reports whether the update has not ended yet: the state of the last Get Update call, or
// the first state before that call, is not terminal. Like the service, an update that will end on
// its own is active until it does.
func (u *fakePipelineUpdate) active() bool {
	return !isTerminalUpdateState(u.states[max(u.stateIndex(), 0)])
}

// cancel ends the update at its current state, as Stop does. CANCELED is the update's state from
// now on, even before the next Get Update call.
func (u *fakePipelineUpdate) cancel() {
	u.states = append(u.states[:u.stateIndex()+1], pipelines.UpdateInfoStateCanceled)
	u.polls = len(u.states)
	u.canceled = true
}

func isTerminalUpdateState(state pipelines.UpdateInfoState) bool {
	switch state {
	case pipelines.UpdateInfoStateCompleted, pipelines.UpdateInfoStateFailed, pipelines.UpdateInfoStateCanceled:
		return true
	default:
		return false
	}
}

type fakePipelineEventOrigin struct {
	PipelineId string `json:"pipeline_id"`
	UpdateId   string `json:"update_id"`
	TestCaseId string `json:"test_case_id,omitempty"`
}

// fakePipelineEvent is a pipeline event as returned by List Pipeline Events. The Go SDK
// PipelineEvent has no details or origin.test_case_id, so the fake defines its own shape.
type fakePipelineEvent struct {
	Id        string                  `json:"id"`
	EventType string                  `json:"event_type"`
	Timestamp string                  `json:"timestamp"`
	Level     pipelines.EventLevel    `json:"level"`
	Message   string                  `json:"message"`
	Origin    fakePipelineEventOrigin `json:"origin"`
	Error     *pipelines.ErrorDetail  `json:"error,omitempty"`
	Details   map[string]any          `json:"details"`

	// visibleAt is the index of the update state from which the event can be read.
	visibleAt int
}

// pipelineTestRequest holds the Start Update fields of a test run, which are not in the Go SDK yet.
type pipelineTestRequest struct {
	TestOnly    bool `json:"test_only"`
	TestOptions *struct {
		TestSelection []string `json:"test_selection"`
		PytestArgs    []string `json:"pytest_args"`
	} `json:"test_options"`
}

var pipelineTestFunctionRegexp = regexp.MustCompile(`(?m)^def (test_\w+)\(`)

// collectPipelineTests stands in for pytest collection: it finds the top-level "def test_*"
// functions in the workspace test_*.py files under each selection, or under rootPath without one.
// Like origin.test_case_id, a test ID is the absolute workspace path of its file followed by the
// pytest test ID, such as /Workspace/Users/someone/orders/tests/test_orders.py::test_backfill.
func (s *FakeWorkspace) collectPipelineTests(rootPath string, selection []string) []string {
	if len(selection) == 0 && rootPath != "" {
		selection = []string{rootPath}
	}

	var ids []string
	for _, target := range selection {
		targetPath, testName, _ := strings.Cut(target, "::")
		var files []string
		if _, ok := s.files[targetPath]; ok {
			files = []string{targetPath}
		} else {
			for filePath := range s.files {
				base := path.Base(filePath)
				if strings.HasPrefix(filePath, targetPath+"/") && strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
					files = append(files, filePath)
				}
			}
			slices.Sort(files)
		}

		for _, filePath := range files {
			for _, match := range pipelineTestFunctionRegexp.FindAllSubmatch(s.files[filePath].Data, -1) {
				name := string(match[1])
				id := filePath + "::" + name
				if (testName == "" || testName == name) && !slices.Contains(ids, id) {
					ids = append(ids, id)
				}
			}
		}
	}
	return ids
}

// startPipelineTestRun records the events of a test-only update. Tests do not run here; each
// test name selects its outcome:
//   - "fail": FAILED, "error": ERROR (a fixture failed), "skip": SKIPPED, otherwise PASSED.
//   - "crash": compute terminates when the case starts and the update fails without a summary.
//   - "hang": the case never finishes and the update stays RUNNING until the pipeline is stopped.
//   - "breaks_update": the case passes, but the update fails after the summary.
func (s *FakeWorkspace) startPipelineTestRun(pipelineID, updateID string, caseIDs []string) *fakePipelineUpdate {
	const (
		running = 1
		ended   = 2
	)
	update := &fakePipelineUpdate{
		pipelineID: pipelineID,
		testOnly:   true,
		states:     []pipelines.UpdateInfoState{pipelines.UpdateInfoStateWaitingForResources, pipelines.UpdateInfoStateRunning},
	}

	emit := func(visibleAt int, event fakePipelineEvent) {
		s.pipelineEventTime = s.pipelineEventTime.Add(time.Second)
		event.Id = nextUUID()
		event.Timestamp = s.pipelineEventTime.UTC().Format("2006-01-02T15:04:05.000Z")
		event.Origin.PipelineId = pipelineID
		event.Origin.UpdateId = updateID
		if event.Level == "" {
			event.Level = pipelines.EventLevelInfo
		}
		event.visibleAt = visibleAt
		s.pipelineEvents = append(s.pipelineEvents, event)
	}
	emitCase := func(id, status string, errorDetail *pipelines.ErrorDetail) {
		event := fakePipelineEvent{
			EventType: pipelineEventTestCaseProgress,
			Message:   fmt.Sprintf("Test case %s is %s.", id, status),
			Origin:    fakePipelineEventOrigin{TestCaseId: id},
			Error:     errorDetail,
			Details:   map[string]any{pipelineEventTestCaseProgress: map[string]any{"status": status}},
		}
		if errorDetail != nil {
			event.Level = pipelines.EventLevelError
		}
		emit(running, event)
	}
	exception := func(className, message string) *pipelines.ErrorDetail {
		return &pipelines.ErrorDetail{Exceptions: []pipelines.SerializedException{{ClassName: className, Message: message}}}
	}

	for _, id := range caseIDs {
		emitCase(id, "PENDING", nil)
	}

	counts := map[string]int{}
	breaksUpdate := false
	for _, id := range caseIDs {
		emitCase(id, "RUNNING", nil)
		_, name, _ := strings.Cut(id, "::")
		switch {
		case strings.Contains(name, "hang"):
			return update
		case strings.Contains(name, "crash"):
			emit(ended, fakePipelineEvent{
				EventType: pipelineEventUpdateProgress,
				Level:     pipelines.EventLevelError,
				Message:   fmt.Sprintf("Update %s is FAILED.", updateID[:6]),
				Error:     &pipelines.ErrorDetail{Fatal: true, Exceptions: []pipelines.SerializedException{{ClassName: "ComputeTerminatedException", Message: "Compute terminated unexpectedly."}}},
				Details:   map[string]any{pipelineEventUpdateProgress: map[string]any{"state": "FAILED"}},
			})
			update.states = append(update.states, pipelines.UpdateInfoStateFailed)
			return update
		case strings.Contains(name, "breaks_update"):
			emitCase(id, "PASSED", nil)
			counts["passed"]++
			breaksUpdate = true
		case strings.Contains(name, "fail"):
			emitCase(id, "FAILED", exception("AssertionError", "assert 100 == 150"))
			counts["failed"]++
		case strings.Contains(name, "error"):
			emitCase(id, "ERROR", exception("RuntimeError", "fixture 'orders' setup failed"))
			counts["errors"]++
		case strings.Contains(name, "skip"):
			emitCase(id, "SKIPPED", nil)
			counts["skipped"]++
		default:
			emitCase(id, "PASSED", nil)
			counts["passed"]++
		}
	}

	emit(ended, fakePipelineEvent{
		EventType: pipelineEventTestRunSummary,
		Message:   fmt.Sprintf("Collected %d tests.", len(caseIDs)),
		Details: map[string]any{pipelineEventTestRunSummary: map[string]any{
			"total":   len(caseIDs),
			"passed":  counts["passed"],
			"failed":  counts["failed"],
			"skipped": counts["skipped"],
			"errors":  counts["errors"],
		}},
	})
	// Like the service, a run whose tests failed or errored ends with the test verdict as the
	// update's error.
	if !breaksUpdate && (counts["failed"] > 0 || counts["errors"] > 0) {
		failed := counts["failed"] + counts["errors"]
		emit(ended, fakePipelineEvent{
			EventType: pipelineEventUpdateProgress,
			Level:     pipelines.EventLevelError,
			Message:   fmt.Sprintf("%d pipeline tests failed.", failed),
			Error: &pipelines.ErrorDetail{Fatal: true, Exceptions: []pipelines.SerializedException{{
				ClassName: "TestFailure",
				Message:   fmt.Sprintf("[PIPELINE_TEST_FAILED] %d of the pipeline's tests failed (%d passed, %d skipped).", failed, counts["passed"], counts["skipped"]),
			}}},
			Details: map[string]any{pipelineEventUpdateProgress: map[string]any{"state": "FAILED"}},
		})
	}
	if breaksUpdate {
		emit(ended, fakePipelineEvent{
			EventType: pipelineEventUpdateProgress,
			Level:     pipelines.EventLevelError,
			Message:   fmt.Sprintf("Update %s is FAILED.", updateID[:6]),
			Error:     &pipelines.ErrorDetail{Fatal: true, Exceptions: []pipelines.SerializedException{{ClassName: "ResultDeliveryException", Message: "Test results could not be verified."}}},
			Details:   map[string]any{pipelineEventUpdateProgress: map[string]any{"state": "FAILED"}},
		})
	}
	if len(caseIDs) == 0 || counts["failed"] > 0 || counts["errors"] > 0 || breaksUpdate {
		update.states = append(update.states, pipelines.UpdateInfoStateFailed)
	} else {
		update.states = append(update.states, pipelines.UpdateInfoStateCompleted)
	}
	return update
}

var (
	pipelineEventsUpdateIDFilter  = regexp.MustCompile(`update_id\s*=\s*'([^']*)'`)
	pipelineEventsTimestampFilter = regexp.MustCompile(`timestamp\s*>=\s*'([^']*)'`)
)

// pipelineEventsPageToken is the query of the first page plus the offset of the next one.
type pipelineEventsPageToken struct {
	Query  url.Values `json:"query"`
	Offset int        `json:"offset"`
}

// PipelineEvents lists the visible events of a pipeline. Only the update_id and timestamp >=
// conditions of the filter are applied.
func (s *FakeWorkspace) PipelineEvents(pipelineId string, query url.Values) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	invalid := func(message string) Response {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": message},
		}
	}

	maxResults := 25
	if value := query.Get("max_results"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return invalid("max_results must be a positive integer.")
		}
		maxResults = min(n, 250)
	}

	offset := 0
	if token := query.Get("page_token"); token != "" {
		for key := range query {
			if key != "page_token" && key != "max_results" {
				return invalid(fmt.Sprintf("page_token cannot be used with %s.", key))
			}
		}
		data, err := base64.URLEncoding.DecodeString(token)
		var decoded pipelineEventsPageToken
		if err == nil {
			err = json.Unmarshal(data, &decoded)
		}
		if err != nil {
			return invalid("Invalid page_token.")
		}
		query = decoded.Query
		offset = decoded.Offset
	}

	updateID := ""
	if match := pipelineEventsUpdateIDFilter.FindStringSubmatch(query.Get("filter")); match != nil {
		updateID = match[1]
	}
	var since time.Time
	if match := pipelineEventsTimestampFilter.FindStringSubmatch(query.Get("filter")); match != nil {
		var err error
		since, err = time.Parse(time.RFC3339Nano, match[1])
		if err != nil {
			return invalid(fmt.Sprintf("Invalid timestamp in filter: %s.", match[1]))
		}
	}

	var events []fakePipelineEvent
	for _, event := range s.pipelineEvents {
		update := s.PipelineUpdates[event.Origin.UpdateId]
		if update.pipelineID != pipelineId || event.visibleAt > update.eventIndex() {
			continue
		}
		if updateID != "" && event.Origin.UpdateId != updateID {
			continue
		}
		if timestamp, _ := time.Parse(time.RFC3339Nano, event.Timestamp); timestamp.Before(since) {
			continue
		}
		events = append(events, event)
	}

	// Events are returned newest first unless ascending order is requested.
	if query.Get("order_by") != "timestamp asc" {
		slices.Reverse(events)
	}

	page := events[min(offset, len(events)):]
	body := map[string]any{"events": append([]fakePipelineEvent{}, page[:min(maxResults, len(page))]...)}
	if len(page) > maxResults {
		data, _ := json.Marshal(pipelineEventsPageToken{Query: query, Offset: offset + maxResults})
		body["next_page_token"] = base64.URLEncoding.EncodeToString(data)
	}
	return Response{Body: body}
}
