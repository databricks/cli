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
	var e Event

	e = newEventStart(0, []string{"put"}, []string{"delete"})
	assert.Equal(t, "Action: PUT: put, DELETE: delete", e.String())

	e = newEventStart(1, []string{"put"}, []string{})
	assert.Equal(t, "Action: PUT: put", e.String())

	e = newEventStart(2, []string{}, []string{"delete"})
	assert.Equal(t, "Action: DELETE: delete", e.String())

	e = newEventStart(3, []string{}, []string{})
	assert.Equal(t, "", e.String())
}

func TestEventStartJSON(t *testing.T) {
	var e Event

	e = newEventStart(0, []string{"put"}, []string{"delete"})
	jsonEqual(t, `{"seq": 0, "type": "start", "put": ["put"], "delete": ["delete"]}`, e)

	e = newEventStart(1, []string{"put"}, []string{})
	jsonEqual(t, `{"seq": 1, "type": "start", "put": ["put"]}`, e)

	e = newEventStart(2, []string{}, []string{"delete"})
	jsonEqual(t, `{"seq": 2, "type": "start", "delete": ["delete"]}`, e)

	e = newEventStart(3, []string{}, []string{})
	jsonEqual(t, `{"seq": 3, "type": "start"}`, e)
}

func TestEventProgress(t *testing.T) {
	var e Event

	// Empty string if no progress has been made.
	e = newEventProgress(0, EventActionPut, "path", 0.0)
	assert.Equal(t, "", e.String())

	e = newEventProgress(1, EventActionPut, "path", 1.0)
	assert.Equal(t, "Uploaded path", e.String())

	// Empty string if no progress has been made.
	e = newEventProgress(2, EventActionDelete, "path", 0.0)
	assert.Equal(t, "", e.String())

	e = newEventProgress(3, EventActionDelete, "path", 1.0)
	assert.Equal(t, "Deleted path", e.String())
}

func TestEventProgressJSON(t *testing.T) {
	var e Event

	e = newEventProgress(0, EventActionPut, "path", 0.0)
	jsonEqual(t, `{"seq": 0, "type": "progress", "action": "put", "path": "path", "progress": 0.0}`, e)

	e = newEventProgress(1, EventActionPut, "path", 1.0)
	jsonEqual(t, `{"seq": 1, "type": "progress", "action": "put", "path": "path", "progress": 1.0}`, e)

	e = newEventProgress(2, EventActionDelete, "path", 0.0)
	jsonEqual(t, `{"seq": 2, "type": "progress", "action": "delete", "path": "path", "progress": 0.0}`, e)

	e = newEventProgress(3, EventActionDelete, "path", 1.0)
	jsonEqual(t, `{"seq": 3, "type": "progress", "action": "delete", "path": "path", "progress": 1.0}`, e)
}

func TestEventComplete(t *testing.T) {
	var e Event

	e = newEventComplete(0, []string{"put"}, []string{"delete"})
	assert.Equal(t, "Initial Sync Complete", e.String())

	e = newEventComplete(1, []string{"put"}, []string{})
	assert.Equal(t, "Complete", e.String())

	e = newEventComplete(2, []string{}, []string{"delete"})
	assert.Equal(t, "Complete", e.String())

	e = newEventComplete(3, []string{}, []string{})
	assert.Equal(t, "", e.String())
}

func TestEventCompleteJSON(t *testing.T) {
	var e Event

	e = newEventComplete(0, []string{"put"}, []string{"delete"})
	jsonEqual(t, `{"seq": 0, "type": "complete", "put": ["put"], "delete": ["delete"]}`, e)

	e = newEventComplete(1, []string{"put"}, []string{})
	jsonEqual(t, `{"seq": 1, "type": "complete", "put": ["put"]}`, e)

	e = newEventComplete(2, []string{}, []string{"delete"})
	jsonEqual(t, `{"seq": 2, "type": "complete", "delete": ["delete"]}`, e)

	e = newEventComplete(3, []string{}, []string{})
	jsonEqual(t, `{"seq": 3, "type": "complete"}`, e)
}
