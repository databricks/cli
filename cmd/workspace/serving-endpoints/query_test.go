package serving_endpoints

import (
	"bytes"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
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

func queryCmdWithFlags(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	var jf flags.JsonFlag
	var stream bool
	cmd.Flags().Var(&jf, "json", "")
	cmd.Flags().BoolVar(&stream, "stream", false, "")
	require.NoError(t, cmd.ParseFlags(args))
	return cmd
}

func TestQueryWantsStream(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"no flags", nil, false},
		{"stream flag", []string{"--stream"}, true},
		{"stream false flag", []string{"--stream=false"}, false},
		{"json stream true", []string{"--json", `{"stream": true}`}, true},
		{"json stream false", []string{"--json", `{"stream": false}`}, false},
		{"json without stream", []string{"--json", `{"inputs": [1]}`}, false},
		{"invalid json", []string{"--json", `{"stream": `}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := queryWantsStream(queryCmdWithFlags(t, tt.args...))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
