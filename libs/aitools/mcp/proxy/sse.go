package proxy

import (
	"bufio"
	"bytes"
	"io"
)

// readEvents calls fn with the data of each event in a text/event-stream body, in order. Multi-line data
// is joined with newlines. An event left open at the end of the stream is still delivered. id, event
// and comment lines are ignored.
func readEvents(r io.Reader, fn func([]byte) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, maxLineBytes)
	var data []byte
	hasData := false
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			if hasData {
				if err := fn(data); err != nil {
					return err
				}
				data, hasData = nil, false
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
	if hasData {
		return fn(data)
	}
	return nil
}
