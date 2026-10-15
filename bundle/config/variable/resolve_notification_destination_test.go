package variable

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/testserver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const notificationDestinationsPath = "GET /api/2.0/notification-destinations"

func TestResolveNotificationDestination_ResolveSuccess(t *testing.T) {
	w := newTestClients(t, map[string]any{
		notificationDestinationsPath: map[string]any{"results": []map[string]any{
			{"id": "1234", "display_name": "destination"},
		}},
	})
	l := resolveNotificationDestination{name: "destination"}
	result, err := l.Resolve(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "1234", result)
}

func TestResolveNotificationDestination_ResolveError(t *testing.T) {
	w := newTestClients(t, map[string]any{
		notificationDestinationsPath: testserver.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       map[string]string{"error_code": "INTERNAL_ERROR", "message": "bad"},
		},
	})
	l := resolveNotificationDestination{name: "destination"}
	_, err := l.Resolve(t.Context(), w)
	assert.ErrorContains(t, err, "bad")
}

func TestResolveNotificationDestination_ResolveNotFound(t *testing.T) {
	w := newTestClients(t, map[string]any{
		notificationDestinationsPath: map[string]any{},
	})
	l := resolveNotificationDestination{name: "destination"}
	_, err := l.Resolve(t.Context(), w)
	require.Error(t, err)
	assert.ErrorContains(t, err, `notification destination named "destination" does not exist`)
}

func TestResolveNotificationDestination_ResolveMultiple(t *testing.T) {
	w := newTestClients(t, map[string]any{
		notificationDestinationsPath: map[string]any{"results": []map[string]any{
			{"id": "1234", "display_name": "destination"},
			{"id": "5678", "display_name": "destination"},
		}},
	})
	l := resolveNotificationDestination{name: "destination"}
	_, err := l.Resolve(t.Context(), w)
	require.Error(t, err)
	assert.ErrorContains(t, err, `there are 2 instances of clusters named "destination"`)
}

func TestResolveNotificationDestination_String(t *testing.T) {
	l := resolveNotificationDestination{name: "name"}
	assert.Equal(t, "notification-destination: name", l.String())
}
