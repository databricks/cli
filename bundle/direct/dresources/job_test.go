package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobRemote verifies that all fields from jobs.Job (except Settings and pagination/internal fields)
// are present in JobRemote.
func TestJobRemote(t *testing.T) {
	assertFieldsCovered(t, reflect.TypeFor[jobs.Job](), reflect.TypeFor[JobRemote](), map[string]bool{
		"Settings":        true, // Embedded as jobs.JobSettings
		"ForceSendFields": true, // Internal marshaling field
		"HasMore":         true, // Pagination field, not relevant for single job read
		"NextPageToken":   true, // Pagination field, not relevant for single job read
	})
}

func webhooks(ids ...string) []jobs.Webhook {
	result := make([]jobs.Webhook, len(ids))
	for i, id := range ids {
		result[i] = jobs.Webhook{Id: id}
	}
	return result
}

// TestJobWebhookNotificationsOrderInsensitive verifies that a webhook list
// reordered by the Jobs API produces no diff, but a changed set still does.
func TestJobWebhookNotificationsOrderInsensitive(t *testing.T) {
	keys := (&ResourceJob{}).KeyedSlices()

	config := jobs.JobSettings{
		WebhookNotifications: &jobs.WebhookNotifications{
			OnSuccess: webhooks("a", "b", "c"),
		},
		Tasks: []jobs.Task{
			{
				TaskKey: "t1",
				WebhookNotifications: &jobs.WebhookNotifications{
					OnFailure: webhooks("a", "b"),
				},
				ForEachTask: &jobs.ForEachTask{
					Task: jobs.Task{
						TaskKey: "inner",
						WebhookNotifications: &jobs.WebhookNotifications{
							OnStart: webhooks("x", "y"),
						},
					},
				},
			},
		},
	}

	remote := jobs.JobSettings{
		WebhookNotifications: &jobs.WebhookNotifications{
			OnSuccess: webhooks("a", "c", "b"),
		},
		Tasks: []jobs.Task{
			{
				TaskKey: "t1",
				WebhookNotifications: &jobs.WebhookNotifications{
					OnFailure: webhooks("b", "a"),
				},
				ForEachTask: &jobs.ForEachTask{
					Task: jobs.Task{
						TaskKey: "inner",
						WebhookNotifications: &jobs.WebhookNotifications{
							OnStart: webhooks("y", "x"),
						},
					},
				},
			},
		},
	}

	changes, err := structdiff.GetStructDiff(config, remote, keys)
	require.NoError(t, err)
	assert.Empty(t, changes)

	// A genuinely different destination set is still detected.
	remote.WebhookNotifications.OnSuccess = webhooks("a", "b", "d")
	changes, err = structdiff.GetStructDiff(config, remote, keys)
	require.NoError(t, err)
	assert.NotEmpty(t, changes)
}
