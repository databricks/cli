package jobstest

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestStringSlice(t *testing.T) {
	x := &jobs.JobEmailNotifications{
		OnFailure:       []string{},
		ForceSendFields: []string{"OnFailure"},
	}

	xs := jsonDump(t, x)

	// expecting this:
	//require.Equal(t, `{"on_failure":[]}`, xs)

	// but get this:
	require.Equal(t, `{}`, xs)
}

func TestBool(t *testing.T) {
	x := jobs.JobNotificationSettings{
		NoAlertForCanceledRuns: false,
		ForceSendFields:        []string{"NoAlertForCanceledRuns"},
	}

	require.Equal(t, `{"no_alert_for_canceled_runs":false}`, jsonDump(t, x))
}

func jsonDump(t *testing.T, x any) string {
	xb, err := json.Marshal(x)
	require.NoError(t, err)
	return string(xb)
}
