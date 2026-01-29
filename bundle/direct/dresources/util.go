package dresources

import (
	"github.com/databricks/cli/bundle/deployplan"
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

// collectUpdatePaths extracts field paths from Changes that have action=Update.
// This builds the precise update_mask for API requests, excluding immutable and unchanged fields.
func collectUpdatePaths(changes Changes) []string {
	var paths []string
	for path, change := range changes {
		if change.Action == deployplan.Update {
			paths = append(paths, path)
		}
	}
	return paths
}
