package knowledge_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/knowledge"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedRequest struct {
	method string
	path   string
	query  map[string]string
	body   map[string]any
}

// newTestClient returns a client for a server that records the requests to the Knowledge
// Pages API and answers each with the given status and body.
func newTestClient(t *testing.T, status int, response string) (*knowledge.Client, *[]recordedRequest) {
	var requests []recordedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The SDK also probes the host for its configuration.
		if r.URL.Path == "/.well-known/databricks-config" {
			http.NotFound(w, r)
			return
		}

		recorded := recordedRequest{method: r.Method, path: r.URL.Path, query: map[string]string{}}
		for key := range r.URL.Query() {
			recorded.query[key] = r.URL.Query().Get(key)
		}
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if len(raw) > 0 {
			assert.NoError(t, json.Unmarshal(raw, &recorded.body))
		}
		requests = append(requests, recorded)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	client, err := knowledge.NewClient(w)
	require.NoError(t, err)
	return client, &requests
}

func TestGetPage(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"name": "pages/1234", "display_name": "Chargeback", "state": "DRAFT", "description": ""}`)

	page, err := client.GetPage(t.Context(), "pages/1234")
	require.NoError(t, err)
	assert.Equal(t, "pages/1234", page.Name)
	assert.Equal(t, "Chargeback", page.DisplayName)
	assert.Equal(t, knowledge.StateDraft, page.State)

	require.Len(t, *requests, 1)
	assert.Equal(t, http.MethodGet, (*requests)[0].method)
	assert.Equal(t, "/api/2.0/knowledge/pages/1234", (*requests)[0].path)
}

func TestGetPageMissing(t *testing.T) {
	client, _ := newTestClient(t, http.StatusNotFound, `{"error_code": "NOT_FOUND", "message": "Knowledge page with id '1234' does not exist"}`)

	_, err := client.GetPage(t.Context(), "pages/1234")
	require.Error(t, err)
	assert.True(t, apierr.IsMissing(err))
}

func TestCreatePage(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"name": "pages/1234"}`)

	page, err := client.CreatePage(t.Context(), knowledge.Page{
		Domain:      "domains/5678",
		DisplayName: "Chargeback",
		Body:        "# Chargeback\n",
	})
	require.NoError(t, err)
	assert.Equal(t, "pages/1234", page.Name)

	require.Len(t, *requests, 1)
	request := (*requests)[0]
	assert.Equal(t, http.MethodPost, request.method)
	assert.Equal(t, "/api/2.0/knowledge/pages", request.path)
	// The page is the body itself, and unset fields are left to the API to default.
	assert.Equal(t, map[string]any{
		"domain":       "domains/5678",
		"display_name": "Chargeback",
		"body":         "# Chargeback\n",
	}, request.body)
	// The API requires the request ID to be a UUIDv4.
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, request.query["request_id"])
}

func TestCreatePageUsesNewRequestIDPerCall(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"name": "pages/1234"}`)

	for range 2 {
		_, err := client.CreatePage(t.Context(), knowledge.Page{Domain: "domains/5678", DisplayName: "Chargeback"})
		require.NoError(t, err)
	}

	// Two pages with the same content are two pages, not a retry of one request.
	require.Len(t, *requests, 2)
	assert.NotEqual(t, (*requests)[0].query["request_id"], (*requests)[1].query["request_id"])
}

func TestUpdatePage(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"name": "pages/1234"}`)

	_, err := client.UpdatePage(t.Context(), knowledge.Page{
		Name:        "pages/1234",
		Domain:      "domains/5678",
		DisplayName: "Chargeback",
		State:       knowledge.StatePublished,
	}, []string{"display_name", "body", "synonyms", "description", "state"})
	require.NoError(t, err)

	require.Len(t, *requests, 1)
	request := (*requests)[0]
	assert.Equal(t, http.MethodPatch, request.method)
	assert.Equal(t, "/api/2.0/knowledge/pages/1234", request.path)
	assert.Equal(t, "display_name,body,synonyms,description,state", request.query["update_mask"])
	// Only the fields in the mask are sent, and an empty one is sent as empty so that it
	// is cleared.
	assert.Equal(t, map[string]any{
		"display_name": "Chargeback",
		"body":         "",
		"synonyms":     []any{},
		"description":  "",
		"state":        "PUBLISHED",
	}, request.body)
}

func TestDeletePage(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{}`)

	err := client.DeletePage(t.Context(), "pages/1234")
	require.NoError(t, err)

	require.Len(t, *requests, 1)
	assert.Equal(t, http.MethodDelete, (*requests)[0].method)
	assert.Equal(t, "/api/2.0/knowledge/pages/1234", (*requests)[0].path)
}
