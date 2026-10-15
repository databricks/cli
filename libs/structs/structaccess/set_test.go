package structaccess_test

import (
	"encoding/json"
	"reflect"
	"testing"

	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	CustomString string
	CustomInt    int
)

type NestedInfo struct {
	Version string `json:"version"`
	Build   int    `json:"build"`
}

type NestedItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TestStruct struct {
	Name        string            `json:"name"`
	Age         int               `json:"age"`
	Score       float64           `json:"score"`
	Active      bool              `json:"active"`
	Priority    uint8             `json:"priority"`
	Tags        map[string]string `json:"tags"`
	Items       []string          `json:"items"`
	NestedItems []NestedItem      `json:"nested_items"`
	Count       *int              `json:"count,omitempty"`
	Custom      CustomString      `json:"custom"`
	Info        NestedInfo        `json:"info"`
	Internal    string            `json:"-"`
}

// newTestStruct creates a fresh TestStruct instance for testing
func newTestStruct() *TestStruct {
	return &TestStruct{
		Name:     "OldName",
		Age:      25,
		Score:    85.5,
		Active:   true,
		Priority: 10,
		Tags: map[string]string{
			"env": "old_env",
		},
		Items: []string{"old_a", "old_b", "old_c"},
		NestedItems: []NestedItem{
			{ID: "item1", Name: "first"},
			{ID: "item2", Name: "second"},
		},
		Count:  nil,
		Custom: CustomString("old custom"),
		Info: NestedInfo{
			Version: "old_version",
			Build:   100,
		},
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		value           any
		expectedChanges []structdiff.Change
		errorMsg        string // if set, test expects an error containing this message
	}{
		{
			name:  "set struct field by dot notation",
			path:  "name",
			value: "NewName",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "NewName",
				},
			},
		},
		{
			name:  "set struct field by bracket notation",
			path:  "['name']",
			value: "BracketName",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "BracketName",
				},
			},
		},
		{
			name:  "set top-level int field",
			path:  "age",
			value: 30,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("age"),
					Old:  25,
					New:  30,
				},
			},
		},
		{
			name:  "set nested struct field",
			path:  "info.version",
			value: "new_version",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("info.version"),
					Old:  "old_version",
					New:  "new_version",
				},
			},
		},
		{
			name:  "set nested struct field with bracket notation",
			path:  "info['build']",
			value: 200,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("info.build"),
					Old:  100,
					New:  200,
				},
			},
		},
		{
			name:  "set map value by bracket notation",
			path:  "tags['version']",
			value: "new_map_value",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("tags['version']"),
					Old:  nil, // new key
					New:  "new_map_value",
				},
			},
		},
		{
			name:  "set map value by dot notation",
			path:  "tags.version",
			value: "dot_map_value",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("tags['version']"),
					Old:  nil, // new key
					New:  "dot_map_value",
				},
			},
		},
		{
			name:  "set array element",
			path:  "items[1]",
			value: "new_item",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("items[1]"),
					Old:  "old_b",
					New:  "new_item",
				},
			},
		},
		{
			name:  "set pointer field",
			path:  "count",
			value: 42,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("count"),
					Old:  nil,
					New:  42, // diffValues dereferences non-nil pointers when the other side is nil/omitted
				},
			},
		},
		{
			name:  "set typedefed string with string",
			path:  "custom",
			value: "new custom",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("custom"),
					Old:  CustomString("old custom"),
					New:  CustomString("new custom"),
				},
			},
		},
		{
			name:  "set typedefed string with typedefed string",
			path:  "custom",
			value: CustomString("typed custom"),
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("custom"),
					Old:  CustomString("old custom"),
					New:  CustomString("typed custom"),
				},
			},
		},
		{
			name:     "error on non-existent field",
			path:     "nonexistent",
			value:    "value",
			errorMsg: "field \"nonexistent\" not found in structaccess_test.TestStruct",
		},
		{
			name:     "error on array index out of bounds",
			path:     "items[5]",
			value:    "value",
			errorMsg: "index 5 out of range, length is 3",
		},
		{
			name:     "error on setting root",
			path:     "",
			value:    "value",
			errorMsg: "cannot set empty path",
		},
		{
			name:     "error on wildcard",
			path:     "items[*]",
			value:    "value",
			errorMsg: "wildcards not allowed in path",
		},
		{
			name:  "custom string to string field",
			path:  "name",
			value: CustomString("custom to regular"),
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "custom to regular",
				},
			},
		},
		{
			name:  "int to custom int field",
			path:  "age",
			value: CustomInt(35),
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("age"),
					Old:  25,
					New:  35,
				},
			},
		},
		{
			name:     "error on incompatible slice to string",
			path:     "name",
			value:    []int{1, 2, 3},
			errorMsg: "cannot convert []int to string",
		},
		{
			name:     "error on string to int field",
			path:     "age",
			value:    "not a number",
			errorMsg: "cannot parse \"not a number\" as int: strconv.ParseInt: parsing \"not a number\": invalid syntax",
		},
		{
			name:  "set numeric string to int field",
			path:  "age",
			value: "42",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("age"),
					Old:  25,
					New:  42,
				},
			},
		},
		{
			name:            "set string 'true' to bool field (no change)",
			path:            "active",
			value:           "true",
			expectedChanges: nil, // No changes because true → true
		},
		{
			name:  "set string 'false' to bool field",
			path:  "active",
			value: "false",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("active"),
					Old:  true,
					New:  false,
				},
			},
		},
		{
			name:     "error on invalid string to bool field",
			path:     "active",
			value:    "bla",
			errorMsg: "cannot parse \"bla\" as bool: strconv.ParseBool: parsing \"bla\": invalid syntax",
		},
		{
			name:  "set numeric string to float field",
			path:  "score",
			value: "3.14",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("score"),
					Old:  85.5,
					New:  3.14,
				},
			},
		},
		{
			name:  "set zero string to float field",
			path:  "score",
			value: "0",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("score"),
					Old:  85.5,
					New:  0.0,
				},
			},
		},
		{
			name:     "error on invalid string to float field",
			path:     "score",
			value:    "bla",
			errorMsg: "cannot parse \"bla\" as float64: strconv.ParseFloat: parsing \"bla\": invalid syntax",
		},
		{
			name:  "set valid numeric string to uint8 field",
			path:  "priority",
			value: "200",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("priority"),
					Old:  uint8(10),
					New:  uint8(200),
				},
			},
		},
		{
			name:     "error on overflow string to uint8 field",
			path:     "priority",
			value:    "256",
			errorMsg: "value 256 overflows uint8",
		},
		{
			name:     "error on negative string to uint8 field",
			path:     "priority",
			value:    "-1",
			errorMsg: "cannot parse \"-1\" as uint8: strconv.ParseUint: parsing \"-1\": invalid syntax",
		},
		{
			name:  "set int value to string field",
			path:  "name",
			value: 42,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "42",
				},
			},
		},
		{
			name:  "set bool true to string field",
			path:  "name",
			value: true,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "true",
				},
			},
		},
		{
			name:  "set bool false to string field",
			path:  "name",
			value: false,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "false",
				},
			},
		},
		{
			name:  "set float64 to string field",
			path:  "name",
			value: 3.14,
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "3.14",
				},
			},
		},
		{
			name:  "set uint8 to string field",
			path:  "name",
			value: uint8(200),
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("name"),
					Old:  "OldName",
					New:  "200",
				},
			},
		},
		{
			name:  "set negative string to int field",
			path:  "age",
			value: "-10",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("age"),
					Old:  25,
					New:  -10,
				},
			},
		},
		{
			name:  "set zero string to uint8 field",
			path:  "priority",
			value: "0",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("priority"),
					Old:  uint8(10),
					New:  uint8(0),
				},
			},
		},

		// Key-value selector tests
		{
			name:  "set field via key-value selector",
			path:  "nested_items[id='item2'].name",
			value: "updated",
			expectedChanges: []structdiff.Change{
				{
					Path: structpath.MustParsePath("nested_items[1].name"),
					Old:  "second",
					New:  "updated",
				},
			},
		},
		{
			name:     "cannot set key-value selector itself",
			path:     "nested_items[id='item1']",
			value:    "new value",
			errorMsg: "cannot set value at key-value selector [id='item1'] - key-value syntax can only be used for path traversal, not as a final target",
		},
		{
			name:     "key-value no matching element",
			path:     "nested_items[id='nonexistent'].name",
			value:    "value",
			errorMsg: "failed to navigate to parent nested_items[id='nonexistent']: nested_items[id='nonexistent']: no element found with id=\"nonexistent\"", //nolint:dupword
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh instance for this test
			original := newTestStruct()
			target := newTestStruct()

			err := structaccess.SetByString(target, tt.path, tt.value)

			if tt.errorMsg != "" {
				// Test expects an error
				require.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				return
			}

			// Test expects success
			require.NoError(t, err)

			// Compare the actual changes using structdiff
			changes, err := structdiff.GetStructDiff(original, target, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedChanges, changes)
		})
	}
}

func TestSet_EmbedTag(t *testing.T) {
	type Item struct {
		Name  string `json:"name"`
		Level string `json:"level,omitempty"`
	}

	type Container struct {
		ObjectID      string `json:"object_id"`
		EmbeddedSlice []Item `json:"items,omitempty"`
	}

	c := &Container{
		ObjectID: "abc",
		EmbeddedSlice: []Item{
			{Name: "alice", Level: "admin"},
			{Name: "bob", Level: "reader"},
		},
	}

	// Set a field via EmbeddedSlice index.
	err := structaccess.SetByString(c, "[0].level", "writer")
	require.NoError(t, err)
	assert.Equal(t, "writer", c.EmbeddedSlice[0].Level)

	// Set non-embed field normally.
	err = structaccess.SetByString(c, "object_id", "def")
	require.NoError(t, err)
	assert.Equal(t, "def", c.ObjectID)

	// Set via key-value path traversal.
	err = structaccess.SetByString(c, "[name='bob'].level", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", c.EmbeddedSlice[1].Level)
}

func TestSetJobSettings(t *testing.T) {
	jobSettings := jobs.JobSettings{
		Name: new("job foo"),
		Tasks: []jobs.TaskSettings{
			{
				TaskKey:    new("job_task"),
				RunJobTask: &jobs.RunJobTask{}, // JobId will be resolved from the reference
			},
		},
	}

	err := structaccess.SetByString(&jobSettings, "tasks[0].run_job_task.job_id", "123")
	require.NoError(t, err)

	require.Equal(t, &jobs.JobSettings{
		Name: new("job foo"),
		Tasks: []jobs.TaskSettings{
			{
				TaskKey: new("job_task"),
				RunJobTask: &jobs.RunJobTask{
					JobId: new(int64(123)),
				},
			},
		},
	}, &jobSettings)
}

// A value that cannot be converted must leave the struct untouched.
func TestSet_FailedConversionLeavesStruct(t *testing.T) {
	job := &jobs.JobSettings{Name: new("n")} //exhaustruct:ignore

	require.Error(t, structaccess.SetByString(job, "max_concurrent_runs", ""))
	assert.Nil(t, job.MaxConcurrentRuns)
	assert.Equal(t, new("n"), job.Name)
}

// Setting a pointer-to-scalar field to a zero value allocates the pointer, so the
// zero value is marshaled; setting nil clears it.
func TestSet_PointerScalarZeroValue(t *testing.T) {
	job := &jobs.JobSettings{Name: new("n")} //exhaustruct:ignore

	require.NoError(t, structaccess.SetByString(job, "max_concurrent_runs", "0"))
	assert.Equal(t, new(0), job.MaxConcurrentRuns)

	blob, err := json.Marshal(job)
	require.NoError(t, err)
	assert.Contains(t, string(blob), `"max_concurrent_runs":0`)

	require.NoError(t, structaccess.SetByString(job, "description", ""))
	assert.Equal(t, new(""), job.Description)

	require.NoError(t, structaccess.SetByString(job, "max_concurrent_runs", nil))
	assert.Nil(t, job.MaxConcurrentRuns)

	got, err := structaccess.GetByString(job, "description")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// encoding/json resolves a name declared at two embedding depths in favour of the shallower
// one. Get and Set have to agree with it, so the embedded search goes level by level: a
// depth-first search would find Deep.Value first, since its embed is declared first.
type deepValue struct {
	Value string `json:"value"`
}

type deepEmbed struct {
	deepValue
}

type shallowEmbed struct {
	Value string `json:"value"`
}

type deeperEmbed struct {
	deepEmbed
}

type embedDepths struct {
	deepEmbed
	shallowEmbed
}

// The same name three levels down in the first member, against two levels down in a later
// one. json picks the shallower, so the search has to be breadth-first across the whole tree
// rather than depth-first per member.
type embedDepthsAcrossMembers struct {
	deeperEmbed
	deepEmbed
}

func TestSet_ShallowerEmbedWinsAcrossMembers(t *testing.T) {
	target := &embedDepthsAcrossMembers{}

	require.NoError(t, structaccess.SetByString(target, "value", "set"))
	assert.Equal(t, "set", target.Value)
	assert.Empty(t, target.deeperEmbed.Value)

	require.NoError(t, structaccess.ValidateByString(reflect.TypeOf(target), "value"))

	blob, err := json.Marshal(target)
	require.NoError(t, err)
	assert.JSONEq(t, `{"value":"set"}`, string(blob))
}

func TestSet_ShallowerEmbedWins(t *testing.T) {
	target := &embedDepths{}

	require.NoError(t, structaccess.SetByString(target, "value", "set"))
	assert.Equal(t, "set", target.Value)
	assert.Empty(t, target.deepEmbed.Value)

	got, err := structaccess.GetByString(target, "value")
	require.NoError(t, err)
	assert.Equal(t, "set", got)

	// The same field json.Marshal picks, which is the contract being matched.
	blob, err := json.Marshal(target)
	require.NoError(t, err)
	assert.JSONEq(t, `{"value":"set"}`, string(blob))
}

// dashNamed exercises the json:"-" vs json:"-,omitempty" distinction.
type dashNamed struct {
	Skipped string `json:"-"`
	Named   string `json:"-,omitempty"` //nolint:staticcheck // odd tag is the point
	Kept    string `json:"kept,omitempty"`
}

// TestGetSet_DashIsAFieldNameWhenTheTagHasOptions codifies that encoding/json
// treats json:"-" and json:"-,omitempty" differently, and that structaccess
// agrees. The test uses json.Marshal as the oracle.
func TestGetSet_DashIsAFieldNameWhenTheTagHasOptions(t *testing.T) {
	target := &dashNamed{Skipped: "s", Named: "n", Kept: "k"}

	// encoding/json oracle: exact json:"-" is omitted; json:"-,omitempty"
	// names the field "-" and serializes it.
	blob, err := json.Marshal(target)
	require.NoError(t, err)
	assert.JSONEq(t, `{"-":"n","kept":"k"}`, string(blob),
		"encoding/json must emit '-' from Named and omit Skipped")

	// structaccess must agree with the oracle.
	value, err := structaccess.GetByString(target, "-")
	require.NoError(t, err)
	assert.Equal(t, "n", value, "Get must resolve \"-\" to Named")

	require.NoError(t, structaccess.SetByString(target, "-", "set"))
	assert.Equal(t, "set", target.Named, "Set must write to Named")
	assert.Equal(t, "s", target.Skipped, "json:\"-\" field must stay out of reach")
}
