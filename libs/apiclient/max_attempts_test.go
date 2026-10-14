package apiclient_test

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/stretchr/testify/assert"
)

func TestWithMaxAttemptsStopsRetrying(t *testing.T) {
	var calls atomic.Int32
	c := apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	ctx := apiclient.WithMaxAttempts(t.Context(), 2)
	err := c.Do(ctx, http.MethodGet, "/api/2.0/x", nil, nil, nil, nil)
	assert.True(t, apierrs.Is(err, apierrs.ErrTemporarilyUnavailable))
	assert.Equal(t, int32(2), calls.Load())
}
