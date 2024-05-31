package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type TraceEvent struct {
	Name      string                 `json:"name"`
	Phase     string                 `json:"ph"`
	Timestamp int64                  `json:"ts"`
	PID       int                    `json:"pid"`
	TID       int                    `json:"tid"`
	Duration  int64                  `json:"dur,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
}

type TraceEvents struct {
	TraceEvents []TraceEvent `json:"traceEvents"`
}

var (
	events []TraceEvent
	// processID  = 1 // Arbitrary process ID
	// threadID   = 1 // Arbitrary thread ID
	// startTimes = make(map[string]int64)
)

// func startEvent(name string) {
// 	startTimes[name] = time.Now().UnixNano() / 1000 // Convert to microseconds
// }

// func endEvent(name string) {
// 	startTime, ok := startTimes[name]
// 	if !ok {
// 		return // No matching start event
// 	}
// 	endTime := time.Now().UnixNano() / 1000 // Convert to microseconds
// 	duration := endTime - startTime

// 	events = append(events, TraceEvent{
// 		Name:      name,
// 		Phase:     "X", // Complete event
// 		Timestamp: startTime,
// 		PID:       processID,
// 		TID:       threadID,
// 		Duration:  duration,
// 	})
// }

func writeTraceFile(filename string) error {
	trace := TraceEvents{
		TraceEvents: events,
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(trace)
}

// DuplicateKeysMap is a custom structure to hold values of duplicate keys
type DuplicateKeysMap map[string][]any

// UnmarshalJSON custom unmarshaler to handle duplicate keys
func (d *DuplicateKeysMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	_, err := dec.Token() // consume the opening '{'
	if err != nil {
		return err
	}

	if *d == nil {
		*d = make(DuplicateKeysMap)
	}

	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}

		key := t.(string)

		var value any
		if err := dec.Decode(&value); err != nil {
			return err
		}

		(*d)[key] = append((*d)[key], value)
	}

	_, err = dec.Token() // consume the closing '}'
	return err

}

func main() {
	var entries []DuplicateKeysMap

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	dec := json.NewDecoder(f)
	for {
		var entry = new(DuplicateKeysMap)
		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		entries = append(entries, *entry)
	}

	startTimes := make(map[string]int64)
	for _, entry := range entries {
		var mutators []string
		var t time.Time
		var msg string
		var level string
		var sdk bool

		for key, values := range entry {
			switch key {
			case "mutator":
				for _, value := range values {
					mutators = append(mutators, value.(string))
				}
			case "time":
				// Parse the following:
				// "2024-05-30T18:19:29.172811239Z"
				t, err = time.Parse(time.RFC3339Nano, values[0].(string))
			case "msg":
				msg = values[0].(string)
			case "level":
				level = values[0].(string)
			case "sdk":
				sdk = values[0].(bool)
			}
		}

		if len(mutators) == 0 {
			log.Printf("No mutators found in entry: %v", entry)
			continue
		}

		key := strings.Join(mutators, ", ")
		if level == "TRACE" {
			switch msg {
			case "mutator:entry":
				startTimes[key] = t.UnixNano()
			case "mutator:exit":
				startTime, ok := startTimes[key]
				if !ok {
					panic("No matching start event")
				}
				durationMicros := (t.UnixNano() - startTime) / 1000
				events = append(events, TraceEvent{
					Name:      mutators[len(mutators)-1],
					Phase:     "X", // Complete event
					Timestamp: startTime / 1000,
					PID:       1, // Arbitrary process ID
					TID:       1, // Arbitrary thread ID
					Duration:  durationMicros,
				})
			}
			continue
		}

		if level == "DEBUG" && sdk {
			lines := strings.Split(msg, "\n")
			events = append(events, TraceEvent{
				Name:      lines[0],
				Phase:     "i", // Complete event
				Timestamp: t.UnixMicro(),
				PID:       1, // Arbitrary process ID
				TID:       1, // Arbitrary thread ID
			})
		}

		// Emit start and stop events
	}

	// // fmt.Printf("%d entries!\n", len(entries))

	// // Start custom event
	// startEvent("MyCustomEvent")

	// // Simulate some work
	// time.Sleep(1 * time.Second)

	// startEvent("otherevent")

	// time.Sleep(1 * time.Second)

	// endEvent("otherevent")

	// time.Sleep(1 * time.Second)

	// // End custom event
	// endEvent("MyCustomEvent")

	// Write trace file
	if err := writeTraceFile("trace.json"); err != nil {
		panic(err)
	}
}
