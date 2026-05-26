package dresources

import (
	"errors"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/retries"
)

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	if e, ok := errors.AsType[*retries.Err](err); ok {
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

// truncateAtIndex truncates a field path at the first bracket index (e.g. "[0]", "[*]",
// "[key=value]"). Most update_mask APIs only support referencing entire collection
// fields, not individual elements within them.
// Examples: "resources[0].name" -> "resources", "description" -> "description",
// "config.env[0].name" -> "config.env".
func truncateAtIndex(path string) string {
	p, err := structpath.ParsePath(path)
	if err != nil {
		return path
	}
	return p.Prefix(1).String()
}
