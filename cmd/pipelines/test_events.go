package pipelines

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

const (
	testCaseProgressEventType = "test_case_progress"
	testRunSummaryEventType   = "test_run_summary"
)

type testCaseStatus string

const (
	testCasePending testCaseStatus = "PENDING"
	testCaseRunning testCaseStatus = "RUNNING"
	testCasePassed  testCaseStatus = "PASSED"
	testCaseFailed  testCaseStatus = "FAILED"
	testCaseSkipped testCaseStatus = "SKIPPED"
	testCaseError   testCaseStatus = "ERROR"
)

// parseTestCaseStatus accepts both the short status and the proto enum value name.
func parseTestCaseStatus(s string) testCaseStatus {
	return testCaseStatus(strings.TrimPrefix(s, "PIPELINE_TEST_CASE_STATUS_"))
}

func (s testCaseStatus) isFinal() bool {
	switch s {
	case testCasePassed, testCaseFailed, testCaseSkipped, testCaseError:
		return true
	default:
		return false
	}
}

type testRunSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

// testPipelineEvent is an entry of List Pipeline Events. The Go SDK PipelineEvent has no
// details or origin.test_case_id, so test events are decoded from the raw response.
// Once the SDK has those fields, remove this type and testEventReader and switch to
// w.Pipelines.ListPipelineEvents.
type testPipelineEvent struct {
	Id        string               `json:"id"`
	EventType string               `json:"event_type"`
	Timestamp string               `json:"timestamp"`
	Level     pipelines.EventLevel `json:"level"`
	Message   string               `json:"message"`
	Origin    struct {
		TestCaseId string `json:"test_case_id"`
	} `json:"origin"`
	Error      *pipelines.ErrorDetail `json:"error"`
	Truncation *pipelines.Truncation  `json:"truncation"`
	Details    struct {
		TestCaseProgress *struct {
			Status string `json:"status"`
		} `json:"test_case_progress"`
		TestRunSummary *testRunSummary `json:"test_run_summary"`
	} `json:"details"`
}

func (e testPipelineEvent) isTruncated(field string) bool {
	if e.Truncation == nil {
		return false
	}
	for _, f := range e.Truncation.TruncatedFields {
		if f.FieldName == field {
			return true
		}
	}
	return false
}

// testEventsPageSize is the largest page List Pipeline Events returns.
const testEventsPageSize = 250

// testEventReader reads the events of one update.
type testEventReader struct {
	apiClient  *client.DatabricksClient
	headers    map[string]string
	pipelineID string
	updateID   string
}

func newTestEventReader(w *databricks.WorkspaceClient, pipelineID, updateID string) (*testEventReader, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return &testEventReader{
		apiClient:  apiClient,
		headers:    auth.WorkspaceIDHeaders(w.Config),
		pipelineID: pipelineID,
		updateID:   updateID,
	}, nil
}

// read returns the update's events in timestamp order, following every page. With
// sinceTimestamp it returns only the events at or after that timestamp, which must be a
// timestamp string exactly as the API returned it.
func (r *testEventReader) read(ctx context.Context, sinceTimestamp string) ([]testPipelineEvent, error) {
	path := fmt.Sprintf("/api/2.0/pipelines/%s/events", r.pipelineID)
	filter := fmt.Sprintf("update_id='%s'", r.updateID)
	if sinceTimestamp != "" {
		filter += fmt.Sprintf(" AND timestamp >= '%s'", sinceTimestamp)
	}
	request := pipelines.ListPipelineEventsRequest{
		Filter:     filter,
		OrderBy:    []string{"timestamp asc"},
		MaxResults: testEventsPageSize,
	}

	var events []testPipelineEvent
	for {
		var response struct {
			Events        []testPipelineEvent `json:"events"`
			NextPageToken string              `json:"next_page_token"`
		}
		err := r.apiClient.Do(ctx, http.MethodGet, path, r.headers, nil, request, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to read events of update %s: %w", r.updateID, err)
		}
		events = append(events, response.Events...)
		if response.NextPageToken == "" {
			return events, nil
		}
		// A request with page_token must not repeat filter or order_by; the token carries
		// them. The SDK iterator repeats them, which is another reason not to use it here.
		request = pipelines.ListPipelineEventsRequest{
			PageToken:  response.NextPageToken,
			MaxResults: testEventsPageSize,
		}
	}
}

// testCaseResult is what the events report about one test case.
type testCaseResult struct {
	// id is origin.test_case_id: the pytest test ID, relative to pytest's rootdir, such as
	// tests/test_orders.py::test_backfill.
	id     string
	status testCaseStatus
	// finals counts final outcomes; a complete result has exactly one per case.
	finals           int
	started          time.Time
	finished         time.Time
	errorDetail      *pipelines.ErrorDetail
	detailsTruncated bool
}

// duration returns the time between the RUNNING and the final event, if both are known.
func (c *testCaseResult) duration() (time.Duration, bool) {
	if c.started.IsZero() || c.finished.IsZero() || c.finished.Before(c.started) {
		return 0, false
	}
	return c.finished.Sub(c.started), true
}

// testRun accumulates the events of a test update, ignoring events it has already seen.
type testRun struct {
	seen      map[string]bool
	cases     []*testCaseResult
	byID      map[string]*testCaseResult
	summaries []testRunSummary
	// unreadable counts test events whose outcome is missing.
	unreadable int
	// fatalError is the first fatal error that the update reported, other than the test verdict.
	fatalError string
	// verdictError is the service's verdict that tests failed, if the update reported it.
	verdictError string
}

// testVerdictErrorPrefix starts the message of the error by which the service reports that tests
// failed, such as "[PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed (1 passed, 0 skipped).".
const testVerdictErrorPrefix = "[PIPELINE_TEST_FAILED]"

// isTestVerdict reports whether an update error is the service's verdict that tests failed.
func isTestVerdict(e *pipelines.ErrorDetail) bool {
	for _, exception := range e.Exceptions {
		if strings.HasPrefix(exception.Message, testVerdictErrorPrefix) {
			return true
		}
	}
	return false
}

func newTestRun() *testRun {
	return &testRun{seen: map[string]bool{}, byID: map[string]*testCaseResult{}}
}

// add records an event and returns the case it gave a final outcome to, if any.
func (r *testRun) add(e testPipelineEvent) *testCaseResult {
	if r.seen[e.Id] {
		return nil
	}
	r.seen[e.Id] = true

	if e.Error != nil && e.Error.Fatal {
		switch {
		case isTestVerdict(e.Error):
			r.verdictError = errorDetailText(e.Error)
		case r.fatalError == "":
			r.fatalError = errorDetailText(e.Error)
		}
	}

	switch e.EventType {
	case testRunSummaryEventType:
		if e.Details.TestRunSummary == nil {
			r.unreadable++
			return nil
		}
		r.summaries = append(r.summaries, *e.Details.TestRunSummary)
	case testCaseProgressEventType:
		progress := e.Details.TestCaseProgress
		if progress == nil || e.Origin.TestCaseId == "" {
			r.unreadable++
			return nil
		}
		status := parseTestCaseStatus(progress.Status)
		if !status.isFinal() && status != testCasePending && status != testCaseRunning {
			r.unreadable++
			return nil
		}

		c, ok := r.byID[e.Origin.TestCaseId]
		if !ok {
			c = &testCaseResult{id: e.Origin.TestCaseId, status: testCasePending}
			r.byID[c.id] = c
			r.cases = append(r.cases, c)
		}
		timestamp, _ := time.Parse(time.RFC3339Nano, e.Timestamp)
		switch {
		case status == testCaseRunning:
			c.started = timestamp
			if !c.status.isFinal() {
				c.status = testCaseRunning
			}
		case status.isFinal():
			c.finals++
			c.status = status
			c.finished = timestamp
			c.errorDetail = e.Error
			c.detailsTruncated = e.isTruncated("error")
			if c.finals == 1 {
				return c
			}
		}
	}
	return nil
}

// counts returns the number of cases with each final outcome.
func (r *testRun) counts() testRunSummary {
	var s testRunSummary
	for _, c := range r.cases {
		if !c.status.isFinal() {
			continue
		}
		s.Total++
		switch c.status {
		case testCasePassed:
			s.Passed++
		case testCaseFailed:
			s.Failed++
		case testCaseSkipped:
			s.Skipped++
		case testCaseError:
			s.Errors++
		default:
		}
	}
	return s
}

// incompleteReason explains why the events of an ended update are not a complete result,
// or returns "" if they are: one summary and one final outcome per case, matching its counts.
func (r *testRun) incompleteReason(state pipelines.UpdateInfoState) string {
	if r.unreadable > 0 {
		return "some test results could not be read"
	}
	switch len(r.summaries) {
	case 0:
		switch state {
		case pipelines.UpdateInfoStateFailed:
			return "the update failed before all test results were received"
		case pipelines.UpdateInfoStateCanceled:
			return "the update was canceled before all test results were received"
		default:
			return "the update ended without a test summary"
		}
	case 1:
	default:
		return fmt.Sprintf("received %d test summaries instead of one", len(r.summaries))
	}

	for _, c := range r.cases {
		if c.finals > 1 {
			return fmt.Sprintf("test %s has more than one result", c.id)
		}
	}
	summary := r.summaries[0]
	received := r.counts()
	if received.Total != summary.Total {
		return fmt.Sprintf("received %d of %d test results", received.Total, summary.Total)
	}
	if received != summary {
		return "the test results do not match the test summary"
	}
	for _, c := range r.cases {
		if c.finals == 0 {
			return fmt.Sprintf("test %s has no result", c.id)
		}
	}
	return ""
}

// updateError returns the error that the update reported: its own, or else the test verdict.
func (r *testRun) updateError() string {
	if r.fatalError != "" {
		return r.fatalError
	}
	return r.verdictError
}

func errorDetailText(detail *pipelines.ErrorDetail) string {
	var parts []string
	for _, e := range detail.Exceptions {
		switch {
		case e.ClassName != "" && e.Message != "":
			parts = append(parts, e.ClassName+": "+e.Message)
		case e.Message != "":
			parts = append(parts, e.Message)
		case e.ClassName != "":
			parts = append(parts, e.ClassName)
		}
	}
	return strings.Join(parts, "\n")
}
