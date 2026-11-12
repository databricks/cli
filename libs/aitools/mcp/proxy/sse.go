package proxy

import (
	"bufio"
	"bytes"
	"io"
)

// messageEvent is the event type of a JSON-RPC message. An event with no type is a message too.
const messageEvent = "message"

// readEvents calls fn with the data of each event in a text/event-stream body, in order, until fn returns
// false. Multi-line data is joined with newlines, and an event left open at the end is still delivered. Only
// events of type message, or with no type, are delivered. An event with empty data is skipped, such as the
// priming event a server sends before a resumable stream.
func readEvents(r io.Reader, fn func([]byte) (bool, error)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, maxLineBytes)
	var ev event
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			if more, err := ev.dispatch(fn); err != nil || !more {
				return err
			}
			continue
		}
		if line[0] == ':' {
			continue
		}
		field, value, _ := bytes.Cut(line, []byte{':'})
		value = bytes.TrimPrefix(value, []byte{' '})
		switch string(field) {
		case "data":
			ev.add(value)
		case "event":
			ev.typ = string(value)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	_, err := ev.dispatch(fn)
	return err
}

// event collects the data lines and the type of one text/event-stream event.
type event struct {
	typ     string
	data    []byte
	hasData bool
}

// add appends one data line.
func (e *event) add(line []byte) {
	if e.hasData {
		e.data = append(e.data, '\n')
	}
	e.data = append(e.data, line...)
	e.hasData = true
}

// dispatch passes the event's data to fn and clears the event. It reports whether fn wants more events.
func (e *event) dispatch(fn func([]byte) (bool, error)) (bool, error) {
	data, typ := e.data, e.typ
	*e = event{}
	if (typ != "" && typ != messageEvent) || len(bytes.TrimSpace(data)) == 0 {
		return true, nil
	}
	return fn(data)
}
