package dresources

import (
	"github.com/databricks/databricks-sdk-go/retries"
)

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	e := err.(*retries.Err)
	if e == nil {
		return false
	}
	return !e.Halt
}
