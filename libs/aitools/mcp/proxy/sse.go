package proxy

import (
	"bufio"
	"bytes"
	"io"
)

// readEvents calls fn with the data of each event in a text/event-stream body, in order, until fn returns
// false. Multi-line data is joined with newlines. An event left open at the end of the stream is still
// delivered. An event whose data is empty is skipped: a server sends one with an id and no data first, to
// prime a client to resume (https://modelcontextprotocol.io/specification/2025-11-25/basic/transports#sending-messages-to-the-server). id,
// event and comment lines are ignored.
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
		field, value, _ := bytes.Cut(line, []byte{':'})
		if line[0] == ':' || string(field) != "data" {
			continue
		}
		ev.add(bytes.TrimPrefix(value, []byte{' '}))
	}
	if err := sc.Err(); err != nil {
		return err
	}
	_, err := ev.dispatch(fn)
	return err
}

// event collects the data lines of one text/event-stream event.
type event struct {
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
	data := e.data
	*e = event{}
	if len(bytes.TrimSpace(data)) == 0 {
		return true, nil
	}
	return fn(data)
}
