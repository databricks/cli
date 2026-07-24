package aircmd

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// seenNano keys the dedup set. Distinct lines can share a nano (each rank stamps
// from its own clock), so the body disambiguates them.
type seenNano struct {
	nano int64
	body string
}

// seenSet is an insertion-ordered set bounded to a capacity, evicting the
// oldest-inserted entry first.
type seenSet struct {
	cap   int
	items map[seenNano]*list.Element
	order *list.List
}

func newSeenSet(capacity int) *seenSet {
	return &seenSet{
		cap:   capacity,
		items: make(map[seenNano]*list.Element),
		order: list.New(),
	}
}

func (s *seenSet) has(nano int64, body string) bool {
	_, ok := s.items[seenNano{nano, body}]
	return ok
}

func (s *seenSet) add(nano int64, body string) {
	key := seenNano{nano, body}
	if _, ok := s.items[key]; ok {
		return
	}
	s.items[key] = s.order.PushBack(key)
	if s.order.Len() > s.cap {
		oldest := s.order.Front()
		s.order.Remove(oldest)
		delete(s.items, oldest.Value.(seenNano))
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

// emitLogLine writes one log line: raw in text mode, or a JSONL LOG event under
// --json. In --json mode a line matching a fatal-failure pattern also emits an
// ALERT event first, giving an agent an immediate actionable signal.
func emitLogLine(out io.Writer, req logRequest, body string) {
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
