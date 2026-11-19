package proxy

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const initializeResult = `{"protocolVersion":"2025-06-18","capabilities":{"tools":{"listChanged":false}},` +
	`"serverInfo":{"name":"genie_chat"},"instructions":"Base."}`

// answerInitialize replies to initialize with initializeResult and to anything else with an empty result.
func answerInitialize(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m, _ := parseMessage(body)
	w.Header().Set("Content-Type", "application/json")
	if m.Method == "initialize" {
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":`+initializeResult+`}`)
		return
	}
	answer(w, r)
}

// runOptions returns Options that send input from stdin to endpoint as profile logfood and agent claude-code, with
// testToken as the authorization. It also returns the buffers that receive stdout and stderr.
func runOptions(t *testing.T, endpoint string, cfg *config.Config, input string) (Options, *syncBuffer, *syncBuffer) {
	t.Helper()
	var out, errOut syncBuffer
	return Options{
		In:       strings.NewReader(input),
		Out:      &out,
		Err:      &errOut,
		Endpoint: endpoint,
		Config:   cfg,
		Authenticate: func(r *http.Request) error {
			r.Header.Set("Authorization", testToken)
			return nil
		},
		Profile: "logfood",
		Agent:   "claude-code",
	}, &out, &errOut
}

func TestRunSendsProxyUserAgentAndRoutingHeader(t *testing.T) {
	srv, fake := newFakeEndpoint(t, answerInitialize)
	cfg := &config.Config{Host: "https://workspace.test", WorkspaceID: "1965859176160743"}
	opts, _, _ := runOptions(t, srv.URL, cfg, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	require.NoError(t, Run(t.Context(), opts))

	reqs := fake.requests()
	require.Len(t, reqs, 1)
	ua := reqs[0].Header.Get("User-Agent")
	assert.Contains(t, ua, "aitools-mcp/proxy")
	assert.Contains(t, ua, "aitools-mcp-client/claude-code")
	assert.Equal(t, "1965859176160743", reqs[0].Header.Get("X-Databricks-Workspace-Id"))
	assert.Equal(t, testToken, reqs[0].Header.Get("Authorization"))
}

func TestRunOmitsRoutingHeaderWithoutWorkspaceID(t *testing.T) {
	srv, fake := newFakeEndpoint(t, answerInitialize)
	opts, _, _ := runOptions(t, srv.URL, &config.Config{Host: "https://workspace.test"}, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")
	opts.Agent = ""

	require.NoError(t, Run(t.Context(), opts))

	reqs := fake.requests()
	require.Len(t, reqs, 1)
	assert.Empty(t, reqs[0].Header.Values("X-Databricks-Workspace-Id"))
	assert.NotContains(t, reqs[0].Header.Get("User-Agent"), "aitools-mcp-client")
}

func TestRunNamesWorkspaceAndProfileInInstructions(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerInitialize)
	opts, out, _ := runOptions(t, srv.URL, &config.Config{Host: "https://workspace.test"},
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n")

	require.NoError(t, Run(t.Context(), opts))

	lines := outLines(out.String())
	require.Len(t, lines, 1)
	reply := parseReply(t, lines[0])
	assert.Equal(t,
		"Base.\n\nThis server is bound to Databricks workspace https://workspace.test (CLI profile \"logfood\"). Its tools act on that workspace only.",
		reply.Result["instructions"])
}
