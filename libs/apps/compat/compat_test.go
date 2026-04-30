package compat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc adapts a function into an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// redirectToServer replaces the package-level httpClient with one whose
// transport rewrites every request URL to point at srv.
func redirectToServer(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			target, _ := url.Parse(srv.URL)
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
	t.Cleanup(func() { httpClient = orig })
}

func testManifest() Manifest {
	return Manifest{
		"next":    {Appkit: "0.27.0", Skills: "0.1.5"},
		"0.296.0": {Appkit: "0.27.0", Skills: "0.1.5"},
		"0.295.0": {Appkit: "0.27.0", Skills: "0.1.5"},
		"0.290.0": {Appkit: "0.24.0", Skills: "0.1.5"},
		"0.288.0": {Appkit: "0.24.0", Skills: "0.1.4"},
	}
}

func TestResolve_ExactMatch(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.296.0")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.Appkit)
	assert.Equal(t, "0.1.5", entry.Skills)
}

func TestResolve_NearestLower(t *testing.T) {
	m := testManifest()
	// 0.293.0 is between 0.290.0 and 0.295.0 → should use 0.290.0's entry
	entry, err := Resolve(m, "0.293.0")
	require.NoError(t, err)
	assert.Equal(t, "0.24.0", entry.Appkit)
	assert.Equal(t, "0.1.5", entry.Skills)
}

func TestResolve_NewerThanAll(t *testing.T) {
	m := Manifest{
		"next":    {Appkit: "0.99.0", Skills: "0.9.9"},
		"0.296.0": {Appkit: "0.27.0", Skills: "0.1.5"},
		"0.290.0": {Appkit: "0.24.0", Skills: "0.1.5"},
	}
	entry, err := Resolve(m, "0.300.0")
	require.NoError(t, err)
	// Nearest-lower returns the highest versioned entry, not "next".
	assert.Equal(t, "0.27.0", entry.Appkit)
	assert.Equal(t, "0.1.5", entry.Skills)
}

func TestResolve_DevBuild(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.0.0-dev+abc123def")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.Appkit)
	assert.Equal(t, "0.1.5", entry.Skills)
}

func TestResolve_OlderThanAll(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.280.0")
	require.NoError(t, err)
	// Falls back to "next" (best effort)
	assert.Equal(t, "0.27.0", entry.Appkit)
	assert.Equal(t, "0.1.5", entry.Skills)
}

func TestResolve_OnlyNextKey(t *testing.T) {
	m := Manifest{
		"next": {Appkit: "0.27.0", Skills: "0.1.5"},
	}
	entry, err := Resolve(m, "0.296.0")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.Appkit)
	assert.Equal(t, "0.1.5", entry.Skills)
}

func TestResolve_LowestEntryExactMatch(t *testing.T) {
	m := testManifest()
	entry, err := Resolve(m, "0.288.0")
	require.NoError(t, err)
	assert.Equal(t, "0.24.0", entry.Appkit)
	assert.Equal(t, "0.1.4", entry.Skills)
}

func TestResolve_EmptyManifest(t *testing.T) {
	m := Manifest{}
	_, err := Resolve(m, "0.296.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty compatibility manifest")
}

func TestResolve_MissingNextKey(t *testing.T) {
	m := Manifest{
		"0.296.0": {Appkit: "0.27.0", Skills: "0.1.5"},
	}
	_, err := Resolve(m, "0.296.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `missing "next" key`)
}

func TestFetchManifest_RemoteSuccess(t *testing.T) {
	want := Manifest{
		"next":    {Appkit: "0.99.0", Skills: "0.9.9"},
		"0.296.0": {Appkit: "0.99.0", Skills: "0.9.9"},
	}
	body, _ := json.Marshal(want)

	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write(body)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(context.Background())
	require.NoError(t, err)
	assert.True(t, called, "test server should have been called")
	// Verify we got the remote manifest, not the embedded one.
	assert.Equal(t, "0.99.0", result["next"].Appkit)
}

func TestFetchManifest_FallbackToEmbedded(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(context.Background())
	require.NoError(t, err)
	assert.True(t, called, "test server should have been called")
	// Should fall back to embedded manifest.
	assert.Contains(t, result, "next")
	assert.Equal(t, "0.27.0", result["next"].Appkit)
}

func TestFetchManifest_RemoteReturnsInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all"))
	}))
	defer srv.Close()
	redirectToServer(t, srv)

	result, err := FetchManifest(context.Background())
	require.NoError(t, err)
	// Should fall back to embedded manifest.
	assert.Contains(t, result, "next")
	assert.Equal(t, "0.27.0", result["next"].Appkit)
}

func TestParseManifest_Valid(t *testing.T) {
	data := `{"next":{"appkit":"0.27.0","skills":"0.1.5"},"0.296.0":{"appkit":"0.27.0","skills":"0.1.5"}}`
	m, err := parseManifest([]byte(data))
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", m["next"].Appkit)
	assert.Equal(t, "0.27.0", m["0.296.0"].Appkit)
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	_, err := parseManifest([]byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest JSON")
}

func TestParseManifest_MissingNext(t *testing.T) {
	data := `{"0.296.0":{"appkit":"0.27.0","skills":"0.1.5"}}`
	_, err := parseManifest([]byte(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `missing "next" key`)
}

func TestParseManifest_Empty(t *testing.T) {
	_, err := parseManifest([]byte("{}"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty compatibility manifest")
}
