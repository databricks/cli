package sync

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonEqual(t *testing.T, expected string, e Event) {
	var expected_, e_ map[string]any

	buf, err := json.Marshal(e)
	require.NoError(t, err)

	err = json.Unmarshal([]byte(expected), &expected_)
	require.NoError(t, err)
	delete(expected_, "timestamp")

	err = json.Unmarshal(buf, &e_)
	require.NoError(t, err)
	delete(e_, "timestamp")

	assert.Equal(t, expected_, e_)
}

func TestEventStart(t *testing.T) {
	tests := []struct {
		name     string
		seq      int
		put      []string
		delete   []string
		dryRun   bool
		expected string
	}{
		{
			name:     "put and delete without dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: "Action: PUT: put, DELETE: delete",
		},
		{
			name:     "put and delete with dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: "Action: PUT: put, DELETE: delete",
		},
		{
			name:     "only put without dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   false,
			expected: "Action: PUT: put",
		},
		{
			name:     "only put with dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   true,
			expected: "Action: PUT: put",
		},
		{
			name:     "only delete without dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: "Action: DELETE: delete",
		},
		{
			name:     "only delete with dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: "Action: DELETE: delete",
		},
		{
			name:     "empty without dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   false,
			expected: "",
		},
		{
			name:     "empty with dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   true,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEventStart(tt.seq, tt.put, tt.delete, tt.dryRun)
			assert.Equal(t, tt.expected, e.String())
		})
	}
}

func TestEventStartJSON(t *testing.T) {
	tests := []struct {
		name     string
		seq      int
		put      []string
		delete   []string
		dryRun   bool
		expected string
	}{
		{
			name:     "put and delete without dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: `{"seq": 0, "type": "start", "put": ["put"], "delete": ["delete"]}`,
		},
		{
			name:     "put and delete with dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: `{"seq": 0, "type": "start", "dry_run": true, "put": ["put"], "delete": ["delete"]}`,
		},
		{
			name:     "only put without dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   false,
			expected: `{"seq": 1, "type": "start", "put": ["put"]}`,
		},
		{
			name:     "only put with dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   true,
			expected: `{"seq": 1, "type": "start", "dry_run": true, "put": ["put"]}`,
		},
		{
			name:     "only delete without dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: `{"seq": 2, "type": "start", "delete": ["delete"]}`,
		},
		{
			name:     "only delete with dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: `{"seq": 2, "type": "start", "dry_run": true, "delete": ["delete"]}`,
		},
		{
			name:     "empty without dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   false,
			expected: `{"seq": 3, "type": "start"}`,
		},
		{
			name:     "empty with dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   true,
			expected: `{"seq": 3, "type": "start", "dry_run": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEventStart(tt.seq, tt.put, tt.delete, tt.dryRun)
			jsonEqual(t, tt.expected, e)
		})
	}
}

func TestEventProgress(t *testing.T) {
	tests := []struct {
		name     string
		seq      int
		action   EventAction
		path     string
		progress float32
		dryRun   bool
		expected string
	}{
		{
			name:     "put no progress without dry run",
			seq:      0,
			action:   EventActionPut,
			path:     "path",
			progress: 0.0,
			dryRun:   false,
			expected: "",
		},
		{
			name:     "put no progress with dry run",
			seq:      0,
			action:   EventActionPut,
			path:     "path",
			progress: 0.0,
			dryRun:   true,
			expected: "",
		},
		{
			name:     "put completed without dry run",
			seq:      1,
			action:   EventActionPut,
			path:     "path",
			progress: 1.0,
			dryRun:   false,
			expected: "Uploaded path",
		},
		{
			name:     "put completed with dry run",
			seq:      1,
			action:   EventActionPut,
			path:     "path",
			progress: 1.0,
			dryRun:   true,
			expected: "Uploaded path",
		},
		{
			name:     "delete no progress without dry run",
			seq:      2,
			action:   EventActionDelete,
			path:     "path",
			progress: 0.0,
			dryRun:   false,
			expected: "",
		},
		{
			name:     "delete no progress with dry run",
			seq:      2,
			action:   EventActionDelete,
			path:     "path",
			progress: 0.0,
			dryRun:   true,
			expected: "",
		},
		{
			name:     "delete completed without dry run",
			seq:      3,
			action:   EventActionDelete,
			path:     "path",
			progress: 1.0,
			dryRun:   false,
			expected: "Deleted path",
		},
		{
			name:     "delete completed with dry run",
			seq:      3,
			action:   EventActionDelete,
			path:     "path",
			progress: 1.0,
			dryRun:   true,
			expected: "Deleted path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEventProgress(tt.seq, tt.action, tt.path, tt.progress, tt.dryRun)
			assert.Equal(t, tt.expected, e.String())
		})
	}
}

func TestEventProgressJSON(t *testing.T) {
	tests := []struct {
		name     string
		seq      int
		action   EventAction
		path     string
		progress float32
		dryRun   bool
		expected string
	}{
		{
			name:     "put no progress without dry run",
			seq:      0,
			action:   EventActionPut,
			path:     "path",
			progress: 0.0,
			dryRun:   false,
			expected: `{"seq": 0, "type": "progress", "action": "put", "path": "path", "progress": 0.0}`,
		},
		{
			name:     "put no progress with dry run",
			seq:      0,
			action:   EventActionPut,
			path:     "path",
			progress: 0.0,
			dryRun:   true,
			expected: `{"seq": 0, "type": "progress", "dry_run": true, "action": "put", "path": "path", "progress": 0.0}`,
		},
		{
			name:     "put half progress without dry run",
			seq:      0,
			action:   EventActionPut,
			path:     "path",
			progress: 0.5,
			dryRun:   false,
			expected: `{"seq": 0, "type": "progress", "action": "put", "path": "path", "progress": 0.5}`,
		},
		{
			name:     "put half progress with dry run",
			seq:      0,
			action:   EventActionPut,
			path:     "path",
			progress: 0.5,
			dryRun:   true,
			expected: `{"seq": 0, "type": "progress", "dry_run": true, "action": "put", "path": "path", "progress": 0.5}`,
		},
		{
			name:     "put completed without dry run",
			seq:      1,
			action:   EventActionPut,
			path:     "path",
			progress: 1.0,
			dryRun:   false,
			expected: `{"seq": 1, "type": "progress", "action": "put", "path": "path", "progress": 1.0}`,
		},
		{
			name:     "put completed with dry run",
			seq:      1,
			action:   EventActionPut,
			path:     "path",
			progress: 1.0,
			dryRun:   true,
			expected: `{"seq": 1, "type": "progress", "dry_run": true, "action": "put", "path": "path", "progress": 1.0}`,
		},
		{
			name:     "delete no progress without dry run",
			seq:      2,
			action:   EventActionDelete,
			path:     "path",
			progress: 0.0,
			dryRun:   false,
			expected: `{"seq": 2, "type": "progress", "action": "delete", "path": "path", "progress": 0.0}`,
		},
		{
			name:     "delete no progress with dry run",
			seq:      2,
			action:   EventActionDelete,
			path:     "path",
			progress: 0.0,
			dryRun:   true,
			expected: `{"seq": 2, "type": "progress", "dry_run": true, "action": "delete", "path": "path", "progress": 0.0}`,
		},
		{
			name:     "delete half progress without dry run",
			seq:      2,
			action:   EventActionDelete,
			path:     "path",
			progress: 0.5,
			dryRun:   false,
			expected: `{"seq": 2, "type": "progress", "action": "delete", "path": "path", "progress": 0.5}`,
		},
		{
			name:     "delete half progress with dry run",
			seq:      2,
			action:   EventActionDelete,
			path:     "path",
			progress: 0.5,
			dryRun:   true,
			expected: `{"seq": 2, "type": "progress", "dry_run": true, "action": "delete", "path": "path", "progress": 0.5}`,
		},
		{
			name:     "delete completed without dry run",
			seq:      3,
			action:   EventActionDelete,
			path:     "path",
			progress: 1.0,
			dryRun:   false,
			expected: `{"seq": 3, "type": "progress", "action": "delete", "path": "path", "progress": 1.0}`,
		},
		{
			name:     "delete completed with dry run",
			seq:      3,
			action:   EventActionDelete,
			path:     "path",
			progress: 1.0,
			dryRun:   true,
			expected: `{"seq": 3, "type": "progress", "dry_run": true, "action": "delete", "path": "path", "progress": 1.0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEventProgress(tt.seq, tt.action, tt.path, tt.progress, tt.dryRun)
			jsonEqual(t, tt.expected, e)
		})
	}
}

func TestEventComplete(t *testing.T) {
	tests := []struct {
		name     string
		seq      int
		put      []string
		delete   []string
		dryRun   bool
		expected string
	}{
		{
			name:     "initial sync without dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: "Initial Sync Complete",
		},
		{
			name:     "initial sync with dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: "Initial Sync Complete",
		},
		{
			name:     "only put without dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   false,
			expected: "Complete",
		},
		{
			name:     "only put with dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   true,
			expected: "Complete",
		},
		{
			name:     "only delete without dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: "Complete",
		},
		{
			name:     "only delete with dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: "Complete",
		},
		{
			name:     "empty without dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   false,
			expected: "",
		},
		{
			name:     "empty with dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   true,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEventComplete(tt.seq, tt.put, tt.delete, tt.dryRun)
			assert.Equal(t, tt.expected, e.String())
		})
	}
}

func TestEventCompleteJSON(t *testing.T) {
	tests := []struct {
		name     string
		seq      int
		put      []string
		delete   []string
		dryRun   bool
		expected string
	}{
		{
			name:     "put and delete without dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: `{"seq": 0, "type": "complete", "put": ["put"], "delete": ["delete"]}`,
		},
		{
			name:     "put and delete with dry run",
			seq:      0,
			put:      []string{"put"},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: `{"seq": 0, "type": "complete", "dry_run": true, "put": ["put"], "delete": ["delete"]}`,
		},
		{
			name:     "only put without dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   false,
			expected: `{"seq": 1, "type": "complete", "put": ["put"]}`,
		},
		{
			name:     "only put with dry run",
			seq:      1,
			put:      []string{"put"},
			delete:   []string{},
			dryRun:   true,
			expected: `{"seq": 1, "type": "complete", "dry_run": true, "put": ["put"]}`,
		},
		{
			name:     "only delete without dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   false,
			expected: `{"seq": 2, "type": "complete", "delete": ["delete"]}`,
		},
		{
			name:     "only delete with dry run",
			seq:      2,
			put:      []string{},
			delete:   []string{"delete"},
			dryRun:   true,
			expected: `{"seq": 2, "type": "complete", "dry_run": true, "delete": ["delete"]}`,
		},
		{
			name:     "empty without dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   false,
			expected: `{"seq": 3, "type": "complete"}`,
		},
		{
			name:     "empty with dry run",
			seq:      3,
			put:      []string{},
			delete:   []string{},
			dryRun:   true,
			expected: `{"seq": 3, "type": "complete", "dry_run": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEventComplete(tt.seq, tt.put, tt.delete, tt.dryRun)
			jsonEqual(t, tt.expected, e)
		})
	}
}
