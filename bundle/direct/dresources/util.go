package dresources

import (
	"fmt"
	"regexp"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/retries"
)

// postgresNamePattern matches hierarchical Postgres resource names:
// - projects/{project_id}
// - projects/{project_id}/branches/{branch_id}
// - projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}
var postgresNamePattern = regexp.MustCompile(`^projects/([^/]+)(?:/branches/([^/]+)(?:/endpoints/([^/]+))?)?$`)

// PostgresNameComponents holds the extracted components from a Postgres resource name.
type PostgresNameComponents struct {
	ProjectID  string
	BranchID   string
	EndpointID string
}

// ParsePostgresName extracts project, branch, and endpoint IDs from a hierarchical Postgres resource name.
// Returns an error if the name doesn't match the expected format.
func ParsePostgresName(name string) (PostgresNameComponents, error) {
	matches := postgresNamePattern.FindStringSubmatch(name)
	if matches == nil {
		return PostgresNameComponents{}, fmt.Errorf("invalid postgres resource name format: %q", name)
	}

	return PostgresNameComponents{
		ProjectID:  matches[1],
		BranchID:   matches[2],
		EndpointID: matches[3],
	}, nil
}

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
	return collectUpdatePathsWithPrefix(changes, "")
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
