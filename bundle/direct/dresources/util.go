package dresources

import (
	"errors"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/retries"
)

type StateLifecycle struct {
	Started *bool `json:"started,omitempty"`
}

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	var e *retries.Err
	if errors.As(err, &e) {
		return !e.Halt
	}
	return false
}

// collectUpdatePathsWithPrefix extracts field paths from Changes that have action=Update,
// adding a prefix to each path. This is used when the state type has a flattened structure
// but the API expects paths relative to a nested object (e.g., "spec.display_name").
func collectUpdatePathsWithPrefix(changes Changes, prefix string) []string {
	var paths []string
	for path, change := range changes {
		if change.Action == deployplan.Update {
			paths = append(paths, prefix+path)
		}
	}
	return paths
}
