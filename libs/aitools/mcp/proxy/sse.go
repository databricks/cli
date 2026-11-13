package proxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// messageEvent is the event type of a JSON-RPC message. An event with no type is a message too.
const messageEvent = "message"

// utf8BOM is the byte order mark that an event stream may start with.
var utf8BOM = []byte("\xEF\xBB\xBF")

// readEvents calls fn with the data of each event in a text/event-stream body, in order, until fn returns
// false. Multi-line data is joined with newlines, and an event left open at the end is still delivered. Only
// events of type message, or with no type, are delivered. An event with empty data is skipped, such as the
// priming event a server sends before a resumable stream.
func readEvents(r io.Reader, fn func([]byte) (bool, error)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, maxLineBytes)
	sc.Split(eventLines())
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
			if err := ev.add(value); err != nil {
				return err
			}
		case "event":
			ev.typ = string(value)
		}
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("%w: %w", errUpstreamTooLarge, err)
		}
		return err
	}
	_, err := ev.dispatch(fn)
	return err
}

// eventLines returns the split function for the lines of an event stream. A line ends at CR, LF or CRLF. A CR ends its
// line at once, so a blank line is read without waiting for the next byte, and the LF of a CRLF that follows is
// skipped. One byte order mark at the start of the stream is skipped.
//
// The skipped bytes are dropped in the same call that finds the line, never in a call of their own. Once the reader
// has reached EOF, bufio.Scanner stops at a call that returns no token, even if data remains.
func eventLines() bufio.SplitFunc {
	first, afterCR := true, false
	return func(data []byte, atEOF bool) (int, []byte, error) {
		skipped := 0
		if first {
			// Wait until the bytes can be told apart from a byte order mark.
			if len(data) < len(utf8BOM) && !atEOF && bytes.HasPrefix(utf8BOM, data) {
				return 0, nil, nil
			}
			first = false
			if bytes.HasPrefix(data, utf8BOM) {
				data = data[len(utf8BOM):]
				skipped += len(utf8BOM)
			}
		}
		if afterCR && len(data) > 0 {
			afterCR = false
			if data[0] == '\n' {
				data = data[1:]
				skipped++
			}
		}
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			afterCR = data[i] == '\r'
			return skipped + i + 1, data[:i], nil
		}
		if atEOF && len(data) > 0 {
			return skipped + len(data), data, nil
		}
		return skipped, nil, nil
	}
}

// event collects the data lines and the type of one text/event-stream event.
type event struct {
	typ     string
	data    []byte
	hasData bool
}

// add appends one data line. The data of an event is limited to maxLineBytes, as one line is.
func (e *event) add(line []byte) error {
	n := len(e.data) + len(line)
	if e.hasData {
		n++
	}
	if n > maxLineBytes {
		return errUpstreamTooLarge
	}
	if e.hasData {
		e.data = append(e.data, '\n')
	}
	e.data = append(e.data, line...)
	e.hasData = true
	return nil
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
