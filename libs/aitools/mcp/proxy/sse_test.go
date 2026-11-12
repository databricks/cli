package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			err := readEvents(strings.NewReader(tc.stream), func(data []byte) error {
				got = append(got, string(data))
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
