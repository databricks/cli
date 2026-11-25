package proxy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditInitialize(t *testing.T) {
	cases := []struct {
		name         string
		msg          string
		instructions string
		listChanged  bool
		want         string
	}{
		{
			name:         "instructions created when absent",
			msg:          `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18"}}`,
			instructions: "Bound.",
			want:         `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","instructions":"Bound."}}`,
		},
		{
			name:         "instructions appended to existing text",
			msg:          `{"jsonrpc":"2.0","id":1,"result":{"instructions":"Base."}}`,
			instructions: "Bound.",
			want:         `{"jsonrpc":"2.0","id":1,"result":{"instructions":"Base.\n\nBound."}}`,
		},
		{
			name:        "listChanged set and capabilities kept",
			msg:         `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"x":1},"resources":{}}}}`,
			listChanged: true,
			want:        `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"resources":{},"tools":{"listChanged":true,"x":1}}}}`,
		},
		{
			name:        "capabilities null",
			msg:         `{"jsonrpc":"2.0","id":1,"result":{"capabilities":null}}`,
			listChanged: true,
			want:        `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"listChanged":true}}}}`,
		},
		{
			name:         "error response unchanged",
			msg:          `{"jsonrpc":"2.0","id":1,"error":{"code":-32006,"message":"sign in"}}`,
			instructions: "Bound.",
			listChanged:  true,
			want:         `{"jsonrpc":"2.0","id":1,"error":{"code":-32006,"message":"sign in"}}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := editInitialize([]byte(tc.msg), tc.instructions, tc.listChanged)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(got))
		})
	}
}

func TestEditInitializeRejectsNonObjectResult(t *testing.T) {
	_, err := editInitialize([]byte(`{"jsonrpc":"2.0","id":1,"result":5}`), "Bound.", false)
	require.Error(t, err)
}

func TestErrorResponseWithoutIDUsesNull(t *testing.T) {
	got := errorResponse(nil, codeParseError, "Parse error")

	assert.JSONEq(t, `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`, string(got))
}

func TestMessageIsRequest(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{name: "request", line: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, want: true},
		{name: "string id", line: `{"jsonrpc":"2.0","id":"a","method":"tools/list"}`, want: true},
		{name: "notification", line: `{"jsonrpc":"2.0","method":"notifications/initialized"}`},
		{name: "null id", line: `{"jsonrpc":"2.0","id":null,"method":"tools/list"}`},
		{name: "client response", line: `{"jsonrpc":"2.0","id":1,"result":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := parseMessage([]byte(tc.line))
			require.NoError(t, err)
			assert.Equal(t, tc.want, m.isRequest())
		})
	}
}

// parseRequestWithID parses a tools/call request with the given id, written as JSON.
func parseRequestWithID(t *testing.T, id string) message {
	t.Helper()
	m, err := parseMessage([]byte(`{"jsonrpc":"2.0","id":` + id + `,"method":"tools/call"}`))
	require.NoError(t, err)
	return m
}

// parseResponseWithID parses a result response with the given id, written as JSON.
func parseResponseWithID(t *testing.T, id string) message {
	t.Helper()
	m, err := parseMessage([]byte(`{"jsonrpc":"2.0","id":` + id + `,"result":{}}`))
	require.NoError(t, err)
	return m
}

func TestIsResponseToComparesIDsByValue(t *testing.T) {
	cases := []struct {
		name     string
		request  string
		response string
		want     bool
	}{
		{name: "same string", request: `"a<b"`, response: `"a<b"`, want: true},
		{name: "string re-encoded", request: `"a<b"`, response: `"a\u003cb"`, want: true},
		{name: "number with fraction", request: `7`, response: `7.0`, want: true},
		{name: "string is not a number", request: `"7"`, response: `7`},
		{name: "different id", request: `1`, response: `2`},
		{name: "large integers that differ", request: `9007199254740993`, response: `9007199254740992`},
		{name: "large integers that are equal", request: `9007199254740993`, response: `9007199254740993.0`, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseResponseWithID(t, tc.response).isResponseTo(parseRequestWithID(t, tc.request)))
		})
	}
}

// TestIDKeyComparesNumbersByValue: numbers match by value in any spelling, and a large exponent is keyed by its
// digits, so the key stays short.
func TestIDKeyComparesNumbersByValue(t *testing.T) {
	same := [][2]string{
		{"7", "7.0"},
		{"7", "70e-1"},
		{"7", "7E0"},
		{"-0", "0"},
		{"0.0", "0e5"},
		{"1.5", "15e-1"},
		{"100", "1e2"},
		{"1e1000000", "1E1000000"},
	}
	for _, tc := range same {
		assert.Equal(t, idKey(json.RawMessage(tc[0])), idKey(json.RawMessage(tc[1])), tc)
	}
	different := [][2]string{
		{"9007199254740993", "9007199254740992"},
		{"1", "10"},
		{"-1", "1"},
		{"0.1", "0.10001"},
		{"1e1000000", "1e1000001"},
	}
	for _, tc := range different {
		assert.NotEqual(t, idKey(json.RawMessage(tc[0])), idKey(json.RawMessage(tc[1])), tc)
	}
	assert.Less(t, len(idKey(json.RawMessage("1e1000000"))), 32, "the key is built from the digits, not the expanded value")
}
