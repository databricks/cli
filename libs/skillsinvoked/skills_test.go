package skillsinvoked

import (
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"single", "databricks-core", []string{"databricks-core"}},
		{"multiple", "a,b,c", []string{"a", "b", "c"}},
		{"trims and lowercases", " Databricks-Core , JOBS ", []string{"databricks-core", "jobs"}},
		{"drops empty entries", ",,a,,", []string{"a"}},
		{"plugin scoped name", "databricks:databricks-core", []string{"databricks:databricks-core"}},
		{"allows underscore dot and colon", "a_b.c:d-e", []string{"a_b.c:d-e"}},
		{"drops invalid characters", "ok,bad name,semi;colon,a/b", []string{"ok"}},
		{"drops leading punctuation", "-a,.b,:c,_d", nil},
		{"dedupes preserving first occurrence", "b,a,b,a,c", []string{"b", "a", "c"}},
		{"dedupes after normalization", "Core,core, CORE", []string{"core"}},
		{"drops entry longer than 64 bytes", strings.Repeat("a", 65) + ",ok", []string{"ok"}},
		{"keeps entry of exactly 64 bytes", strings.Repeat("a", 64), []string{strings.Repeat("a", 64)}},
		{"caps at 20 entries", strings.Join(skillNames(25), ","), skillNames(20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Parse(t.Context(), tt.raw))
		})
	}
}

// skillNames returns n distinct valid skill names.
func skillNames(n int) []string {
	names := make([]string, 0, n)
	for i := range n {
		names = append(names, "skill"+strings.Repeat("x", i))
	}
	return names
}

func TestApplySetsHeaderFromContext(t *testing.T) {
	cfg := &config.Config{}
	Apply(cfg)

	ctx := WithSkills(t.Context(), []string{"a", "b"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/api", nil)
	require.NoError(t, err)
	require.NoError(t, cfg.Headers(req))
	assert.Equal(t, "a,b", req.Header.Get(Header))
}

func TestApplyNoHeaderWhenEmpty(t *testing.T) {
	cfg := &config.Config{}
	Apply(cfg)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.test/api", nil)
	require.NoError(t, err)
	require.NoError(t, cfg.Headers(req))
	assert.Empty(t, req.Header.Values(Header))
}

func TestApplyKeepsExistingHeaders(t *testing.T) {
	cfg := &config.Config{Headers: config.StaticHeaders(map[string]string{"X-Other": "1"})}
	Apply(cfg)

	ctx := WithSkills(t.Context(), []string{"a"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/api", nil)
	require.NoError(t, err)
	require.NoError(t, cfg.Headers(req))
	assert.Equal(t, "1", req.Header.Get("X-Other"))
	assert.Equal(t, "a", req.Header.Get(Header))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestTransportSetsHeaderWithoutMutatingRequest(t *testing.T) {
	var sent *http.Request
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sent = r
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	client := &http.Client{Transport: Transport(base)}

	ctx := WithSkills(t.Context(), []string{"databricks:jobs"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/api/2.0/fs/files", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "databricks:jobs", sent.Header.Get(Header))
	assert.Empty(t, req.Header.Values(Header))
}

func TestTransportNoHeaderWhenEmpty(t *testing.T) {
	var sent *http.Request
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sent = r
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	client := &http.Client{Transport: Transport(base)}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.test/api", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Empty(t, sent.Header.Values(Header))
}
