package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/settings"
)

type resolveNotificationDestination struct {
	name string
}

func (l resolveNotificationDestination) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	result, err := w.NotificationDestinations.ListAll(ctx, settings.ListNotificationDestinationsRequest{
		// The default page size for this API is 20.
		// We use a higher value to make fewer API calls.
		PageSize: 200,
	})
	if err != nil {
		return "", err
	}

	// Collect all notification destinations with the given name.
	var entities []settings.ListNotificationDestinationsResult
	for _, entity := range result {
		if entity.DisplayName == l.name {
			entities = append(entities, entity)
		}
	}

	// Return the ID of the first matching notification destination.
	switch len(entities) {
	case 0:
		return "", fmt.Errorf("notification destination named %q does not exist", l.name)
	case 1:
		return entities[0].Id, nil
	default:
		return "", fmt.Errorf("there are %d instances of clusters named %q", len(entities), l.name)
	}
}

func (l resolveNotificationDestination) String() string {
	return "notification-destination: " + l.name
}
