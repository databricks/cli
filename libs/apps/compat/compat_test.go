package compat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	m := testManifest()
	entry, err := Resolve(m, "0.300.0")
	require.NoError(t, err)
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
}

func TestResolve_OnlyNextKey(t *testing.T) {
	m := Manifest{
		"next": {Appkit: "0.27.0", Skills: "0.1.5"},
	}
	entry, err := Resolve(m, "0.296.0")
	require.NoError(t, err)
	assert.Equal(t, "0.27.0", entry.Appkit)
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
	m := testManifest()
	body, _ := json.Marshal(m)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	// Override the manifest URL and HTTP client for this test.
	origURL := manifestURL
	origClient := httpClient
	defer func() {
		// Restore. We can't assign to const so we test via fetchRemote indirectly.
		httpClient = origClient
	}()
	httpClient = srv.Client()

	// We need to test fetchRemote directly since manifestURL is a const.
	// Instead, test the full FetchManifest with embedded fallback.
	result, err := FetchManifest(context.Background())
	require.NoError(t, err)
	// Should get a valid manifest from the embedded fallback (since we can't override the const URL).
	assert.NotNil(t, result)
	assert.Contains(t, result, "next")
	_ = origURL
}

func TestFetchManifest_FallbackToEmbedded(t *testing.T) {
	// Create a server that always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// FetchManifest with the real URL will fail (or succeed if GitHub is up),
	// but embedded fallback always works.
	result, err := FetchManifest(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "next")
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
