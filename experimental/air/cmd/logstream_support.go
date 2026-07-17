package aircmd

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// seenNano keys the dedup set: a (nano, body) pair. time_unix_nano is nanosecond
// units and all ranks funnel into one stream stamping from their own clocks, so
// distinct log lines can share a nano — the body disambiguates them.
type seenNano struct {
	nano int64
	body string
}

// seenSet is an insertion-ordered set bounded to a capacity, evicting the
// oldest-inserted entry first. It lets the streamer re-query a boundary second
// on the next poll without re-printing records already shown, while a large
// initial drain can't grow without bound.
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

// logEvent is one JSONL streaming event, matching the Python CLI's
// print_jsonl_event shape ({type, ts, node, line}).
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
