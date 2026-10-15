package apiclient_test

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryParamsDereferencePointers(t *testing.T) {
	var query string
	c := apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
	})
	var unset *bool
	err := c.Do(t.Context(), http.MethodPost, "/api/2.0/x", nil, map[string]any{"no_compute": new(true), "skip": unset}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "no_compute=true", query)
}
