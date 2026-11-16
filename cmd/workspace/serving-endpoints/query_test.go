package serving_endpoints

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyEventStream(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "data lines until done",
			in:   "data: {\"a\":1}\n\ndata:{\"a\":2}\n\ndata: [DONE]\n\ndata: {\"after\":true}\n",
			want: "{\"a\":1}\n{\"a\":2}\n",
		},
		{
			name: "skips comments and other SSE fields",
			in:   ": keep-alive\nevent: message\nid: 7\nretry: 1000\ndata: {\"a\":1}\n\n",
			want: "{\"a\":1}\n",
		},
		{
			name: "plain JSON body is copied as is",
			in:   "{\"predictions\":[1]}\n",
			want: "{\"predictions\":[1]}\n",
		},
		{
			name: "stream without done sentinel",
			in:   "data: {\"a\":1}\n",
			want: "{\"a\":1}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			require.NoError(t, copyEventStream(&out, strings.NewReader(tt.in)))
			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestCopyEventStreamLongLine(t *testing.T) {
	long := strings.Repeat("x", 1024*1024)
	var out bytes.Buffer
	require.NoError(t, copyEventStream(&out, strings.NewReader("data: \""+long+"\"\n")))
	assert.Equal(t, "\""+long+"\"\n", out.String())
}
