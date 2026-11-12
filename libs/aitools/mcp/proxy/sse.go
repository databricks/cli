package proxy

import (
	"bufio"
	"bytes"
	"io"
)

// readEvents calls fn with the data of each event in a text/event-stream body, in order. Multi-line data
// is joined with newlines. An event left open at the end of the stream is still delivered. An event whose
// data is empty is skipped: a server sends one with an id and no data first, to prime a client to resume
// (MCP spec 2025-11-25, Streamable HTTP, "Sending Messages to the Server"). id, event and comment lines are
// ignored.
func readEvents(r io.Reader, fn func([]byte) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, maxLineBytes)
	var data []byte
	hasData := false
	dispatch := func() error {
		if len(bytes.TrimSpace(data)) == 0 {
			data, hasData = nil, false
			return nil
		}
		err := fn(data)
		data, hasData = nil, false
		return err
	}
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			if err := dispatch(); err != nil {
				return err
			}
			continue
		}
		field, value, _ := bytes.Cut(line, []byte{':'})
		if line[0] == ':' || string(field) != "data" {
			continue
		}
		value = bytes.TrimPrefix(value, []byte{' '})
		if hasData {
			data = append(data, '\n')
		}
		data = append(data, value...)
		hasData = true
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return dispatch()
}
