package github

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type closeRecordingBody struct {
	io.Reader
	closed *bool
}

func (b *closeRecordingBody) Close() error {
	*b.closed = true
	return nil
}

type stubTransport struct {
	status int
	closed bool
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Status:     fmt.Sprintf("%d %s", s.status, http.StatusText(s.status)),
		Header:     http.Header{},
		Body:       &closeRecordingBody{Reader: strings.NewReader("{}"), closed: &s.closed},
		Request:    req,
	}, nil
}

func TestGetPagedBytesClosesBodyOnHTTPError(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"not found", http.StatusNotFound},
		{"server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// getPagedBytes hardcodes http.DefaultClient, so swapping its
			// transport is the only seam to observe body closure.
			stub := &stubTransport{status: tt.status}
			prev := http.DefaultClient.Transport
			http.DefaultClient.Transport = stub
			t.Cleanup(func() { http.DefaultClient.Transport = prev })

			_, err := getPagedBytes(t.Context(), "GET", "https://api.github.test/x", nil)
			assert.Error(t, err)
			assert.True(t, stub.closed)
		})
	}
}

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		name       string
		linkHeader string
		expected   string
	}{
		// First and foremost, the well-formed cases that we can expect from real GitHub API.
		{
			name:       "no header",
			linkHeader: "",
			expected:   "",
		},
		{
			name:       "documentation example",
			linkHeader: `<https://api.github.com/repositories/1300192/issues?page=2>; rel="prev", <https://api.github.com/repositories/1300192/issues?page=4>; rel="next", <https://api.github.com/repositories/1300192/issues?page=515>; rel="last", <https://api.github.com/repositories/1300192/issues?page=1>; rel="first"`,
			expected:   "https://api.github.com/repositories/1300192/issues?page=4",
		},
		{
			name:       "with next only",
			linkHeader: `<https://api.github.com/repos/databricks/cli/issues?page=2>; rel="next"`,
			expected:   "https://api.github.com/repos/databricks/cli/issues?page=2",
		},
		{
			name:       "without next",
			linkHeader: `<https://api.github.com/repositories/1300192/issues?page=1>; rel="prev", <https://api.github.com/repositories/1300192/issues?page=1>; rel="first", <https://api.github.com/repositories/1300192/issues?page=515>; rel="last"`,
			expected:   "",
		},
		{
			name:       "next at beginning",
			linkHeader: `<https://api.github.com/repos/test/test?page=5>; rel="next", <https://api.github.com/repos/test/test?page=10>; rel="last"`,
			expected:   "https://api.github.com/repos/test/test?page=5",
		},
		{
			name:       "next at end",
			linkHeader: `<https://api.github.com/repos/test/test?page=10>; rel="last", <https://api.github.com/repos/test/test?page=5>; rel="next"`,
			expected:   "https://api.github.com/repos/test/test?page=5",
		},
		// Malformed cases to ensure robustness. (These should not occur in practice, but are here to demonstrate resilience.)
		{
			name:       "malformed no semicolon",
			linkHeader: `<https://api.github.com/repos/test/test?page=2> rel="next"`,
			expected:   "",
		},
		{
			name:       "malformed no angle-brackets",
			linkHeader: `https://api.github.com/repos/test/test?page=2; rel="next"`,
			expected:   "",
		},
		{
			name:       "malformed multiple parts",
			linkHeader: `<https://api.github.com/repos/test/test?page=2>; rel="next"; extra="value"`,
			expected:   "",
		},
		{
			name:       "malformed no url",
			linkHeader: `<>; rel="next"`,
			expected:   "",
		},
		{
			name:       "malformed empty link",
			linkHeader: `, <https://api.github.com/repos/test/test?page=5>; rel="next"`,
			expected:   "https://api.github.com/repos/test/test?page=5",
		},
		// Borderline case: some tolerance of whitespace.
		{
			name:       "tolerate whitespace",
			linkHeader: `  <https://api.github.com/repos/test/test?page=2>  ;  rel="next"  `,
			expected:   "https://api.github.com/repos/test/test?page=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNextLink(tt.linkHeader)
			assert.Equal(t, tt.expected, result)
		})
	}
}
