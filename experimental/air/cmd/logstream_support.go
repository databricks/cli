package aircmd

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// seenRecord keys the dedup set. Older responses without record IDs fall back
// to timestamp and body, since distinct lines can share a timestamp.
type seenRecord struct {
	recordID string
	nano     int64
	body     string
}

type seenEntry struct {
	key  seenRecord
	nano int64
}

// seenSet is an insertion-ordered set bounded to a capacity, evicting the
// oldest-inserted entry first.
type seenSet struct {
	cap   int
	items map[seenRecord]*list.Element
	order *list.List
}

func newSeenSet(capacity int) *seenSet {
	return &seenSet{
		cap:   capacity,
		items: make(map[seenRecord]*list.Element),
		order: list.New(),
	}
}

func seenRecordKey(record logRecord) seenRecord {
	if record.RecordID != "" {
		return seenRecord{recordID: record.RecordID}
	}
	return seenRecord{nano: record.nano(), body: record.Body}
}

func (s *seenSet) has(record logRecord) bool {
	_, ok := s.items[seenRecordKey(record)]
	return ok
}

func (s *seenSet) add(record logRecord) int64 {
	key := seenRecordKey(record)
	if _, ok := s.items[key]; ok {
		return 0
	}
	s.items[key] = s.order.PushBack(seenEntry{key: key, nano: record.nano()})
	if s.order.Len() > s.cap {
		oldest := s.order.Front()
		s.order.Remove(oldest)
		evicted := oldest.Value.(seenEntry)
		delete(s.items, evicted.key)
		return evicted.nano
	}
	return 0
}

func (s *seenSet) removeBefore(nano int64) {
	for element := s.order.Front(); element != nil; {
		next := element.Next()
		entry := element.Value.(seenEntry)
		if entry.nano != 0 && entry.nano < nano {
			s.order.Remove(element)
			delete(s.items, entry.key)
		}
		element = next
	}
}

// logEvent is one JSONL streaming event.
type logEvent struct {
	Type string `json:"type"`
	TS   string `json:"ts"`
	Node int    `json:"node"`
	Line string `json:"line"`
}

// printLogEvent writes a single JSONL event line for --json streaming output.
func printLogEvent(out io.Writer, eventType string, node int, line string) {
	b, err := json.Marshal(logEvent{
		Type: eventType,
		TS:   time.Now().UTC().Format(time.RFC3339),
		Node: node,
		Line: line,
	})
	if err != nil {
		return
	}
	fmt.Fprintln(out, string(b))
}

// submittedEvent is the JSONL event `air run --watch -o json` emits before the
// streamed log events, so a consumer sees the run id immediately.
type submittedEvent struct {
	Type         string `json:"type"`
	TS           string `json:"ts"`
	RunID        string `json:"run_id"`
	DashboardURL string `json:"dashboard_url"`
}

// printSubmittedEvent writes the SUBMITTED JSONL event.
func printSubmittedEvent(out io.Writer, runID, dashboardURL string) {
	b, err := json.Marshal(submittedEvent{
		Type:         "SUBMITTED",
		TS:           time.Now().UTC().Format(time.RFC3339),
		RunID:        runID,
		DashboardURL: dashboardURL,
	})
	if err != nil {
		return
	}
	fmt.Fprintln(out, string(b))
}

// statusEvent is a JSONL event emitted on each lifecycle transition while
// following a run with --watch.
type statusEvent struct {
	Type     string `json:"type"`
	TS       string `json:"ts"`
	Status   string `json:"status"`
	Previous string `json:"previous_status,omitempty"`
}

// printStatusEvent writes a STATUS JSONL event for a lifecycle transition.
func printStatusEvent(out io.Writer, current, previous string) {
	b, err := json.Marshal(statusEvent{
		Type:     "STATUS",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Status:   current,
		Previous: previous,
	})
	if err != nil {
		return
	}
	fmt.Fprintln(out, string(b))
}

// terminalEvent is the closing envelope `air run --watch -o json` emits after
// streaming, carrying the run's terminal status.
type terminalEvent struct {
	V    int       `json:"v"`
	TS   string    `json:"ts"`
	Data runResult `json:"data"`
}

// printTerminalEvent writes the closing terminal-status envelope, matching the
// shape of renderEnvelope(runResult).
func printTerminalEvent(out io.Writer, runID, status, dashboardURL string) {
	b, err := json.Marshal(terminalEvent{
		V:  envelopeVersion,
		TS: time.Now().UTC().Format(time.RFC3339),
		Data: runResult{
			Status:       status,
			RunID:        runID,
			DashboardURL: dashboardURL,
		},
	})
	if err != nil {
		return
	}
	fmt.Fprintln(out, string(b))
}

const (
	missingRequirementsNoticePrefix = "No co-located requirements.yaml at "
	missingRequirementsNoticeSuffix = "; skipping requirements.yaml install."
)

// suppressLogLine reports whether a backend log line should be omitted.
func suppressLogLine(body string) bool {
	// This backend-derived notice is non-actionable noise because requirements.yaml
	// is not supported by Databricks Air and is rejected earlier.
	return strings.HasPrefix(body, missingRequirementsNoticePrefix) &&
		strings.HasSuffix(body, missingRequirementsNoticeSuffix)
}

// emitLogLine writes one relevant log line.
func emitLogLine(out io.Writer, req logRequest, body string) {
	if suppressLogLine(body) {
		return
	}
	if !req.jsonOutput {
		fmt.Fprintln(out, body)
		return
	}
	if matchFatalPattern(body) {
		printLogEvent(out, "ALERT", req.node, body)
	}
	printLogEvent(out, "LOG", req.node, body)
}

// emitNoLogs reports that a run produced no logs. A terminal run gets its
// termination reason; a still-active run is reported as having no logs yet,
// since the MLflow fallback is a one-shot that does not follow it to completion.
// Under --json it is a JSONL ERROR, so a consumer never sees an empty stream.
func emitNoLogs(out io.Writer, req logRequest, status logRunStatus) {
	var msg string
	if status.terminal() {
		msg = fmt.Sprintf("No logs available for run %d. Run terminated in state %s", req.runID, status.displayState())
	} else {
		msg = fmt.Sprintf("No logs available yet for run %d, which is still in state %s", req.runID, status.displayState())
	}
	if status.stateMessage != "" {
		msg = fmt.Sprintf("%s: %s", msg, status.stateMessage)
	}
	if req.jsonOutput {
		printLogEvent(out, "ERROR", req.node, msg)
		return
	}
	fmt.Fprintln(out, msg)
}
