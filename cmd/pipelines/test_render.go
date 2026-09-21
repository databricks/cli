package pipelines

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Event type strings emitted by a test_only pipeline update. They are matched
// client-side (rather than via the server-side event_type filter) because that
// filter is not reliable for external/PAT callers; see the M4 design note.
const (
	testCaseProgressEventType = "test_case_progress"
	testSummaryEventType      = "test_summary"
)

// Lifecycle status of a single test case (PipelineTestCaseProgress.status).
const (
	testStatusPending   = "PENDING"
	testStatusRunning   = "RUNNING"
	testStatusCompleted = "COMPLETED"
)

// Terminal outcome of a completed test case (PipelineTestCaseProgress.result).
const (
	testResultPassed  = "PASSED"
	testResultFailed  = "FAILED"
	testResultSkipped = "SKIPPED"
	testResultError   = "ERROR"
)

// testCaseProgress is the decoded details.test_case_progress payload of a
// pipeline event. Field names mirror the dogfood-verified emitter contract;
// the exact wire shape is still pending the event-shape 1DD (see worklog).
type testCaseProgress struct {
	NodeID     string `json:"node_id"`
	Path       string `json:"path"`
	Line       *int   `json:"line"`
	Status     string `json:"status"`
	Result     string `json:"result"`
	DurationMs *int64 `json:"duration_ms"`
	Message    string `json:"message"`
	Traceback  string `json:"traceback"`
	Truncated  bool   `json:"truncated"`
}

// testSummary is the decoded details.test_summary payload of a pipeline event.
type testSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

// testEventDetails is the details object of a pipeline event, keyed by event
// type. Only the fields relevant to test events are decoded.
type testEventDetails struct {
	TestCaseProgress *testCaseProgress `json:"test_case_progress,omitempty"`
	TestSummary      *testSummary      `json:"test_summary,omitempty"`
}

// testPipelineEvent is a decoded pipeline event carrying test details. The Go
// SDK's pipelines.PipelineEvent drops the details field, so the test command
// decodes the /events response into this type instead.
type testPipelineEvent struct {
	EventType string           `json:"event_type"`
	Timestamp string           `json:"timestamp,omitempty"`
	Details   testEventDetails `json:"details"`
}

// testRunResult is the folded outcome of a test_only update: the latest known
// state of each case in first-seen order, plus the run summary if present.
type testRunResult struct {
	Cases   []testCaseProgress
	Summary *testSummary
}

// reduceTestEvents folds a stream of pipeline events into a testRunResult. Case
// events are folded by node_id preserving first-seen (collection) order; a
// COMPLETED case is never regressed to a non-terminal status by a late event.
// The last test_summary wins.
func reduceTestEvents(events []testPipelineEvent) *testRunResult {
	result := &testRunResult{Cases: nil, Summary: nil}
	indexByNode := map[string]int{}

	for i := range events {
		evt := events[i]
		switch evt.EventType {
		case testSummaryEventType:
			if evt.Details.TestSummary != nil {
				summary := *evt.Details.TestSummary
				result.Summary = &summary
			}
		case testCaseProgressEventType:
			c := evt.Details.TestCaseProgress
			if c == nil {
				continue
			}
			idx, seen := indexByNode[c.NodeID]
			if !seen {
				indexByNode[c.NodeID] = len(result.Cases)
				result.Cases = append(result.Cases, *c)
				continue
			}
			// Don't let a late non-terminal event clobber a completed case.
			if result.Cases[idx].Status == testStatusCompleted && c.Status != testStatusCompleted {
				continue
			}
			result.Cases[idx] = *c
		default:
			// Ignore unrelated event types (the client-side filter should have
			// excluded them, but tolerate them defensively).
		}
	}

	return result
}

// completed returns the terminal (COMPLETED) cases in first-seen order.
func (r *testRunResult) completed() []testCaseProgress {
	var out []testCaseProgress
	for _, c := range r.Cases {
		if c.Status == testStatusCompleted {
			out = append(out, c)
		}
	}
	return out
}

// checkComplete enforces the result-completeness barrier: a terminal outcome is
// never reported without a summary and every case it counts. It mirrors the
// PIPELINE_TEST_NO_RESULT / PIPELINE_TEST_EVENTS_INCOMPLETE contract.
func (r *testRunResult) checkComplete() error {
	if r.Summary == nil {
		return errors.New("PIPELINE_TEST_NO_RESULT: update produced no test_summary event")
	}
	terminal := len(r.completed())
	if terminal != r.Summary.Total {
		return fmt.Errorf("PIPELINE_TEST_EVENTS_INCOMPLETE: summary.total=%d but only %d terminal case events are readable", r.Summary.Total, terminal)
	}
	return nil
}

// failed reports whether the run should exit non-zero (any failure or error).
func (r *testRunResult) failed() bool {
	return r.Summary != nil && (r.Summary.Failed > 0 || r.Summary.Errors > 0)
}

// resultLabel maps a case result to its short display label.
func resultLabel(result string) string {
	switch result {
	case testResultPassed:
		return "PASS"
	case testResultFailed:
		return "FAIL"
	case testResultSkipped:
		return "SKIP"
	case testResultError:
		return "ERROR"
	default:
		return result
	}
}

// durationSeconds converts a nullable millisecond duration to seconds.
func durationSeconds(ms *int64) float64 {
	if ms == nil {
		return 0
	}
	return float64(*ms) / 1000.0
}

// caseLocation renders "path:line" when a line is present, else "path".
func caseLocation(c testCaseProgress) string {
	if c.Line != nil {
		return c.Path + ":" + strconv.Itoa(*c.Line)
	}
	return c.Path
}

// renderTestResultsText writes a deterministic, pytest-style human summary:
// a per-case line, an expanded failures section, and a final tally. It carries
// no wall-clock time or color so the output is stable and golden-testable.
func renderTestResultsText(w io.Writer, pipelineID, updateID string, r *testRunResult) error {
	var b strings.Builder

	total := len(r.Cases)
	if r.Summary != nil {
		total = r.Summary.Total
	}
	fmt.Fprintf(&b, "Running %d pipeline tests for %s (update %s)\n\n", total, pipelineID, updateID)

	for _, c := range r.completed() {
		label := resultLabel(c.Result)
		if c.Result == testResultSkipped || c.Result == testResultError {
			fmt.Fprintf(&b, "%-5s  %s\n", label, c.NodeID)
		} else {
			fmt.Fprintf(&b, "%-5s  %s (%.2fs)\n", label, c.NodeID, durationSeconds(c.DurationMs))
		}
	}

	failures := failingCases(r)
	if len(failures) > 0 {
		fmt.Fprintf(&b, "\nFailures:\n")
		for _, c := range failures {
			fmt.Fprintf(&b, "\n%-5s %s (%s)\n", resultLabel(c.Result), c.NodeID, caseLocation(c))
			if c.Message != "" {
				fmt.Fprintf(&b, "  %s\n", c.Message)
			}
			if c.Traceback != "" {
				for ln := range strings.SplitSeq(strings.TrimRight(c.Traceback, "\n"), "\n") {
					fmt.Fprintf(&b, "  %s\n", ln)
				}
			}
			if c.Truncated {
				fmt.Fprintf(&b, "  (output truncated; see the pipeline event log)\n")
			}
		}
	}

	if r.Summary != nil {
		s := r.Summary
		fmt.Fprintf(&b, "\n%d tests: %d passed, %d failed, %d skipped, %d errors\n",
			s.Total, s.Passed, s.Failed, s.Skipped, s.Errors)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// failingCases returns the completed cases with a FAILED or ERROR result, in
// first-seen order.
func failingCases(r *testRunResult) []testCaseProgress {
	var out []testCaseProgress
	for _, c := range r.completed() {
		if c.Result == testResultFailed || c.Result == testResultError {
			out = append(out, c)
		}
	}
	return out
}

// testRunJSONOutput is the machine-readable (-o json) contract. It carries the
// typed summary plus the identifiers a CI caller needs.
type testRunJSONOutput struct {
	UpdateID        string      `json:"update_id"`
	PipelineID      string      `json:"pipeline_id"`
	Summary         testSummary `json:"summary"`
	DurationSeconds float64     `json:"duration_seconds"`
}

// toJSONOutput projects a folded result into the -o json contract.
func (r *testRunResult) toJSONOutput(pipelineID, updateID string) testRunJSONOutput {
	var summary testSummary
	if r.Summary != nil {
		summary = *r.Summary
	}
	var totalMs int64
	for _, c := range r.Cases {
		if c.DurationMs != nil {
			totalMs += *c.DurationMs
		}
	}
	return testRunJSONOutput{
		UpdateID:        updateID,
		PipelineID:      pipelineID,
		Summary:         summary,
		DurationSeconds: float64(totalMs) / 1000.0,
	}
}

// JUnit XML types. Field order is fixed so the marshaled output is stable.
type junitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Name       string          `xml:"name,attr"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Errors     int             `xml:"errors,attr"`
	Skipped    int             `xml:"skipped,attr"`
	Time       string          `xml:"time,attr"`
	Properties []junitProperty `xml:"properties>property"`
	TestCases  []junitTestCase `xml:"testcase"`
}

type junitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type junitTestCase struct {
	Classname string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitDetail  `xml:"failure,omitempty"`
	Error     *junitDetail  `xml:"error,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitDetail struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

// buildJUnitSuite builds the JUnit testsuite model from a folded result.
func buildJUnitSuite(pipelineID, updateID string, r *testRunResult) junitTestSuite {
	var summary testSummary
	if r.Summary != nil {
		summary = *r.Summary
	}
	var totalMs int64
	cases := r.completed()
	testCases := make([]junitTestCase, 0, len(cases))
	for _, c := range cases {
		if c.DurationMs != nil {
			totalMs += *c.DurationMs
		}
		tc := junitTestCase{
			Classname: strings.ReplaceAll(c.Path, "/", "."),
			Name:      c.NodeID,
			Time:      formatSeconds(durationSeconds(c.DurationMs)),
			Failure:   nil,
			Error:     nil,
			Skipped:   nil,
		}
		switch c.Result {
		case testResultFailed:
			tc.Failure = &junitDetail{Message: c.Message, Text: c.Traceback}
		case testResultError:
			tc.Error = &junitDetail{Message: c.Message, Text: c.Traceback}
		case testResultSkipped:
			tc.Skipped = &junitSkipped{Message: c.Message}
		default:
			// PASSED and any other outcome: no child element.
		}
		testCases = append(testCases, tc)
	}

	return junitTestSuite{
		XMLName:  xml.Name{Space: "", Local: "testsuite"},
		Name:     "sdp-pipeline-tests",
		Tests:    summary.Total,
		Failures: summary.Failed,
		Errors:   summary.Errors,
		Skipped:  summary.Skipped,
		Time:     formatSeconds(float64(totalMs) / 1000.0),
		Properties: []junitProperty{
			{Name: "pipeline_id", Value: pipelineID},
			{Name: "update_id", Value: updateID},
		},
		TestCases: testCases,
	}
}

// renderJUnitXML writes the JUnit XML report for a folded result.
func renderJUnitXML(w io.Writer, pipelineID, updateID string, r *testRunResult) error {
	suite := buildJUnitSuite(pipelineID, updateID, r)
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suite); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// formatSeconds renders a duration in the JUnit 3-decimal-seconds convention.
func formatSeconds(seconds float64) string {
	return strconv.FormatFloat(seconds, 'f', 3, 64)
}
