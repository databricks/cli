// Demo-only fake control plane for `databricks pipelines test`.
//
// It replays a captured DP emission (full /events wire shape) with a compressed
// update timeline: a short cluster wait, then the original inter-event gaps
// stretched into a visible pytest window. Not a 15-minute serverless spin-up.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8765", "listen address")
	eventsPath := flag.String("events", "cmd/pipelines/testdata/pipeline_test_events.json", "captured DP /events JSON")
	clusterWait := flag.Duration("cluster-wait", 3*time.Second, "compressed WAITING_FOR_RESOURCES+INITIALIZING (real run was ~15m)")
	eventWindow := flag.Duration("event-window", 4*time.Second, "stretch the original pytest event span (~127ms) into this window")
	flag.Parse()

	raw, err := os.ReadFile(*eventsPath)
	if err != nil {
		log.Fatalf("read events fixture: %v", err)
	}
	var fixture struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		log.Fatalf("parse events fixture: %v", err)
	}
	if len(fixture.Events) == 0 {
		log.Fatal("events fixture is empty")
	}

	updateID := originString(fixture.Events[0], "update_id")
	pipelineID := originString(fixture.Events[0], "pipeline_id")

	var mu sync.Mutex
	var run *replayRun

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/databricks-config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"oidc_endpoint": "http://" + r.Host + "/oidc",
			"workspace_id":  "4168071307939726",
		})
	})
	mux.HandleFunc("/api/2.0/pipelines/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/updates"):
			body, _ := io.ReadAll(r.Body)
			var req struct {
				TestOnly bool `json:"test_only"`
			}
			_ = json.Unmarshal(body, &req)
			next := newReplayRun(fixture.Events, *clusterWait, *eventWindow)
			mu.Lock()
			run = next
			mu.Unlock()
			log.Printf("StartUpdate test_only=%v cluster-wait=%s event-window=%s",
				req.TestOnly, *clusterWait, *eventWindow)
			go next.emitToLog(pipelineID, updateID)
			writeJSON(w, map[string]string{"update_id": updateID})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/updates/"):
			mu.Lock()
			current := run
			mu.Unlock()
			state := "WAITING_FOR_RESOURCES"
			if current != nil {
				state = current.state()
			}
			log.Printf("GetUpdate state=%s", state)
			writeJSON(w, map[string]any{
				"update": map[string]any{
					"update_id":   updateID,
					"pipeline_id": pipelineID,
					"state":       state,
				},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/events"):
			mu.Lock()
			current := run
			mu.Unlock()
			events := []map[string]any{}
			if current != nil {
				events = current.visibleEvents()
			}
			log.Printf("GET /events filter=%q -> %d/%s emitted", r.URL.Query().Get("filter"), len(events), eventCount(current))
			writeJSON(w, map[string]any{"events": events})
		default:
			http.NotFound(w, r)
		}
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	host := "http://" + ln.Addr().String()
	fmt.Fprintf(os.Stderr, `Replaying captured DP emission on %s
  pipeline_id=%s  update_id=%s  events=%d
  compressed cluster wait=%s (real ~15m skipped)
  pytest event window=%s (original span ~127ms, same order/gaps)

In another terminal:

  export DATABRICKS_HOST=%s
  export DATABRICKS_TOKEN=dbapi0aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

  ./databricks pipelines test --pipeline-id orders-pipeline-id

`, host, pipelineID, updateID, len(fixture.Events), *clusterWait, *eventWindow, host)

	log.Fatal(http.Serve(ln, mux))
}

type scheduledEvent struct {
	event map[string]any
	at    time.Duration
}

type replayRun struct {
	started     time.Time
	clusterWait time.Duration
	eventEnd    time.Duration
	events      []scheduledEvent
}

func newReplayRun(events []map[string]any, clusterWait, eventWindow time.Duration) *replayRun {
	offsets := eventOffsets(events)
	span := time.Duration(0)
	if len(offsets) > 0 {
		span = offsets[len(offsets)-1]
	}
	out := make([]scheduledEvent, len(events))
	for i, e := range events {
		at := clusterWait
		if span > 0 {
			at += time.Duration(float64(eventWindow) * float64(offsets[i]) / float64(span))
		}
		out[i] = scheduledEvent{event: e, at: at}
	}
	return &replayRun{
		started:     time.Now(),
		clusterWait: clusterWait,
		eventEnd:    clusterWait + eventWindow,
		events:      out,
	}
}

func eventOffsets(events []map[string]any) []time.Duration {
	var origin time.Time
	out := make([]time.Duration, len(events))
	for i, e := range events {
		ts, _ := e["timestamp"].(string)
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			t, err = time.Parse(time.RFC3339, ts)
		}
		if err != nil {
			continue
		}
		if origin.IsZero() {
			origin = t
		}
		out[i] = t.Sub(origin)
	}
	return out
}

func (r *replayRun) elapsed() time.Duration {
	return time.Since(r.started)
}

func (r *replayRun) state() string {
	e := r.elapsed()
	switch {
	case e < r.clusterWait/2:
		return "WAITING_FOR_RESOURCES"
	case e < r.clusterWait:
		return "INITIALIZING"
	case e < r.eventEnd:
		return "RUNNING"
	default:
		return "COMPLETED"
	}
}

func (r *replayRun) visibleEvents() []map[string]any {
	now := r.elapsed()
	var out []map[string]any
	for _, se := range r.events {
		if se.at <= now {
			out = append(out, se.event)
		}
	}
	return out
}

func (r *replayRun) emitToLog(pipelineID, updateID string) {
	fmt.Fprintf(os.Stderr, "\n======== DP emission replay pipeline=%s update=%s ========\n", pipelineID, updateID)
	fmt.Fprintf(os.Stderr, "  t=0        WAITING_FOR_RESOURCES (compressed; real run waited ~15m)\n")
	time.Sleep(r.clusterWait / 2)
	fmt.Fprintf(os.Stderr, "  t=%-8s INITIALIZING\n", r.elapsed().Truncate(time.Millisecond))
	time.Sleep(time.Until(r.started.Add(r.clusterWait)))
	fmt.Fprintf(os.Stderr, "  t=%-8s RUNNING — emitting captured DP events in original order\n", r.elapsed().Truncate(time.Millisecond))

	for _, se := range r.events {
		wait := time.Until(r.started.Add(se.at))
		if wait > 0 {
			time.Sleep(wait)
		}
		printEvent(se.event)
	}
	fmt.Fprintf(os.Stderr, "======== %d events delivered; update COMPLETED ========\n\n", len(r.events))
}

func printEvent(e map[string]any) {
	ts, _ := e["timestamp"].(string)
	level, _ := e["level"].(string)
	typ, _ := e["event_type"].(string)
	msg, _ := e["message"].(string)
	fmt.Fprintf(os.Stderr, "%s %s %s  %s\n", ts, level, typ, msg)
	if pretty, err := json.MarshalIndent(e, "  ", "  "); err == nil {
		fmt.Fprintf(os.Stderr, "  %s\n", pretty)
	}
}

func eventCount(r *replayRun) string {
	if r == nil {
		return "0"
	}
	return fmt.Sprintf("%d", len(r.events))
}

func originString(event map[string]any, key string) string {
	origin, _ := event["origin"].(map[string]any)
	if origin == nil {
		return ""
	}
	s, _ := origin[key].(string)
	return s
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("write json: %v", err)
		return
	}
	_, _ = w.Write(buf.Bytes())
}
