package testdiff

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortLineRuns(t *testing.T) {
	pattern := regexp.MustCompile(`^(Created|Updated|Deleted) `)

	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "sorts a run and leaves surrounding lines alone",
			input: "Uploading files...\nCreated volumes.v\nCreated jobs.a\nUpdated jobs.b\nFiles: 1 uploaded\n",
			want:  "Uploading files...\nCreated jobs.a\nCreated volumes.v\nUpdated jobs.b\nFiles: 1 uploaded\n",
		},
		{
			// Runs are sorted independently: a non-matching line between them is a
			// boundary, so lines never move across it.
			name:  "does not merge runs separated by other output",
			input: "Created jobs.z\nError: boom\nCreated jobs.a\n",
			want:  "Created jobs.z\nError: boom\nCreated jobs.a\n",
		},
		{
			// Same, with runs long enough to sort: each is ordered on its own, and the
			// second run stays below the boundary even though it sorts first overall.
			name:  "sorts each run independently",
			input: "Updated jobs.z\nUpdated jobs.y\nError: boom\nCreated jobs.b\nCreated jobs.a\n",
			want:  "Updated jobs.y\nUpdated jobs.z\nError: boom\nCreated jobs.a\nCreated jobs.b\n",
		},
		{
			name:  "sorts a run that ends the input",
			input: ">>> deploy\nDeleted jobs.b\nDeleted jobs.a",
			want:  ">>> deploy\nDeleted jobs.a\nDeleted jobs.b",
		},
		{
			name:  "leaves input without matches untouched",
			input: "Plan: 1 to add\n\nResources: 1 created\n",
			want:  "Plan: 1 to add\n\nResources: 1 created\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SortLineRuns(tc.input, pattern)
			assert.Equal(t, tc.want, got)
			// Sorting is idempotent, so goldens stay stable when re-recorded.
			assert.Equal(t, tc.want, SortLineRuns(got, pattern))
		})
	}
}
