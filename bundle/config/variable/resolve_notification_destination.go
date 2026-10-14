package variable

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/notificationdestinations/v1"
)

type resolveNotificationDestination struct {
	name string
}

func (l resolveNotificationDestination) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	// Collect all notification destinations with the given name.
	var entities []*notificationdestinations.ListNotificationDestinationsResult
	for entity, err := range w.NotificationDestinations.ListNotificationDestinationsIter(ctx, notificationdestinations.ListNotificationDestinationsRequest{
		// The default page size for this API is 20.
		// We use a higher value to make fewer API calls.
		PageSize: new(int64(200)),
	}) {
		if err != nil {
			return "", err
		}
		if utils.Deref(entity.DisplayName) == l.name {
			entities = append(entities, entity)
		}
	}

	// Return the ID of the first matching notification destination.
	switch len(entities) {
	case 0:
		return "", fmt.Errorf("notification destination named %q does not exist", l.name)
	case 1:
		return utils.Deref(entities[0].Id), nil
	default:
		return "", fmt.Errorf("there are %d instances of clusters named %q", len(entities), l.name)
	}
}

func (l resolveNotificationDestination) String() string {
	return "notification-destination: " + l.name
}
