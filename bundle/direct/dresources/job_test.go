package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// TestJobRemote verifies that all fields from jobs.Job (except Settings and pagination/internal fields)
// are present in JobRemote.
func TestJobRemote(t *testing.T) {
	assertFieldsCovered(t, reflect.TypeOf(jobs.Job{}), reflect.TypeOf(JobRemote{}), map[string]bool{
		"Settings":        true, // Embedded as jobs.JobSettings
		"ForceSendFields": true, // Internal marshaling field
		"HasMore":         true, // Pagination field, not relevant for single job read
		"NextPageToken":   true, // Pagination field, not relevant for single job read
	})
}
