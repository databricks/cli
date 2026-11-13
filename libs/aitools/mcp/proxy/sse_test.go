package proxy

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fill is an endless reader of 'a' bytes, so a test can stream a body of any size without holding it.
type fill struct{}

func (fill) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}

func TestReadEvents(t *testing.T) {
	cases := []struct {
		name   string
		stream string
		want   []string
	}{
		{name: "two events", stream: "data: a\n\ndata: b\n\n", want: []string{"a", "b"}},
		{name: "multi-line data is joined", stream: "data: {\ndata: x}\n\n", want: []string{"{\nx}"}},
		{name: "comments and other fields ignored", stream: ": hi\nevent: message\nid: 3\ndata: 1\n\n", want: []string{"1"}},
		{name: "CRLF line endings", stream: "data: 1\r\n\r\ndata: 2\r\n\r\n", want: []string{"1", "2"}},
		{name: "event open at end of stream", stream: "data: 1\n\ndata: 2", want: []string{"1", "2"}},
		{name: "no data", stream: "event: x\n\n"},
		{name: "event with empty data is skipped", stream: "id: 1\ndata:\n\nid: 2\ndata: 2\n\n", want: []string{"2"}},
		{name: "empty data at end of stream is skipped", stream: "data: 1\n\nid: 2\ndata:", want: []string{"1"}},
		{name: "events of other types are skipped", stream: "event: ping\ndata: ping\n\nevent: message\ndata: 1\n\n", want: []string{"1"}},
		{name: "bare CR line endings", stream: "data: a\r\rdata: b\r\r", want: []string{"a", "b"}},
		{name: "CR and LF mixed", stream: "data: a\r\n\rdata: b\n\n", want: []string{"a", "b"}},
		{name: "one leading BOM is ignored", stream: "\xEF\xBB\xBFdata: 1\n\n", want: []string{"1"}},
		{name: "only one leading BOM is ignored", stream: "\xEF\xBB\xBF\xEF\xBB\xBFdata: 1\n\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			err := readEvents(strings.NewReader(tc.stream), func(data []byte) (bool, error) {
				got = append(got, string(data))
				return true, nil
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestReadEventsSplitReads: a BOM and a CRLF split across reads are read as they are in one read.
func TestReadEventsSplitReads(t *testing.T) {
	var got []string
	err := readEvents(iotest.OneByteReader(strings.NewReader("\xEF\xBB\xBFdata: 1\r\n\r\ndata: 2\r\r")), func(data []byte) (bool, error) {
		got = append(got, string(data))
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, got)
}

// TestReadEventsDispatchesBareCRBeforeNextBytes: a blank line ending in a bare CR dispatches its event without
// waiting for the next byte, so a server that keeps the stream open still gets its response delivered.
func TestReadEventsDispatchesBareCRBeforeNextBytes(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	got := make(chan string, 2)
	done := make(chan error, 1)
	go func() {
		done <- readEvents(pr, func(data []byte) (bool, error) {
			got <- string(data)
			return true, nil
		})
	}()

	_, err := io.WriteString(pw, "data: 1\r\r")
	require.NoError(t, err)
	select {
	case data := <-got:
		assert.Equal(t, "1", data)
	case <-time.After(10 * time.Second):
		t.Fatal("the event was not dispatched before the next bytes arrived")
	}
	_, err = io.WriteString(pw, "data: 2\r\r")
	require.NoError(t, err)
	require.NoError(t, pw.Close())
	require.NoError(t, <-done)
	assert.Equal(t, "2", <-got)
}

// TestReadEventsLimitsEventData: the 64 MiB limit applies to the whole event, not only to each of its lines.
func TestReadEventsLimitsEventData(t *testing.T) {
	var parts []io.Reader
	for range 70 {
		parts = append(parts, strings.NewReader("data: "), io.LimitReader(fill{}, 1<<20), strings.NewReader("\n"))
	}
	err := readEvents(io.MultiReader(parts...), func([]byte) (bool, error) { return true, nil })
	require.Error(t, err)
	assert.ErrorContains(t, err, "an upstream message exceeds the 64 MiB limit")
}

// TestReadEventsLimitsLine: a line over the limit fails with the limit named, and the scanner's error stays in the chain.
func TestReadEventsLimitsLine(t *testing.T) {
	stream := io.MultiReader(strings.NewReader("data: "), io.LimitReader(fill{}, maxLineBytes))
	err := readEvents(stream, func([]byte) (bool, error) { return true, nil })
	require.ErrorIs(t, err, bufio.ErrTooLong)
	assert.ErrorContains(t, err, "an upstream message exceeds the 64 MiB limit")
}

// TestReadEventsStopsWhenFnReturnsFalse: an event after the one that ends reading is not read.
func TestReadEventsStopsWhenFnReturnsFalse(t *testing.T) {
	var got []string
	err := readEvents(strings.NewReader("data: 1\n\ndata: 2\n\n"), func(data []byte) (bool, error) {
		got = append(got, string(data))
		return false, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"1"}, got)
}
