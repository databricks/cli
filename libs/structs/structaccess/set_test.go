package structaccess_test

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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

// mustParsePath is a helper to parse path strings in tests
func mustParsePath(path string) *structpath.PathNode {
	p, err := structpath.Parse(path)
	if err != nil {
		panic(err)
	}
	return p
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("age"),
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
					Path: mustParsePath("info.version"),
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
					Path: mustParsePath("info.build"),
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
					Path: mustParsePath("tags['version']"),
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
					Path: mustParsePath("tags['version']"),
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
					Path: mustParsePath("items[1]"),
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
					Path: mustParsePath("count"),
					Old:  nil, // structdiff reports this as interface{}(nil)
					New:  intPtr(42),
				},
			},
		},
		{
			name:  "set typedefed string with string",
			path:  "custom",
			value: "new custom",
			expectedChanges: []structdiff.Change{
				{
					Path: mustParsePath("custom"),
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
					Path: mustParsePath("custom"),
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
			errorMsg: "wildcards not supported",
		},
		{
			name:  "custom string to string field",
			path:  "name",
			value: CustomString("custom to regular"),
			expectedChanges: []structdiff.Change{
				{
					Path: mustParsePath("name"),
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
					Path: mustParsePath("age"),
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
					Path: mustParsePath("age"),
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
					Path: mustParsePath("active"),
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
					Path: mustParsePath("score"),
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
					Path: mustParsePath("score"),
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
					Path: mustParsePath("priority"),
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("name"),
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
					Path: mustParsePath("age"),
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
					Path: mustParsePath("priority"),
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
					Path: mustParsePath("nested_items[1].name"),
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
			errorMsg: "failed to navigate to parent nested_items[id='nonexistent']: nested_items[id='nonexistent']: no element found with id=\"nonexistent\"",
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

func intPtr(i int) *int {
	return &i
}

// testSet sets a value and gets it back, asserting they're equal (roundtrip)
func testSet(t *testing.T, obj any, path string, value any) {
	t.Helper()
	err := structaccess.SetByString(obj, path, value)
	require.NoError(t, err, "SetByString(%T, %q, %#v)", obj, path, value)
	got, err := structaccess.GetByString(obj, path)
	require.NoError(t, err, "GetByString(%T, %q)", obj, path)
	require.Equal(t, value, got, "SetByString(%T, %q, %#v) then GetByString should return same value", obj, path, value)
}

// testSetGet sets a value and gets it back, allowing different expected get value
func testSetGet(t *testing.T, obj any, path string, setValue, expectedGetValue any) {
	t.Helper()
	err := structaccess.SetByString(obj, path, setValue)
	require.NoError(t, err, "SetByString(%#v, %q, %#v)", obj, path, setValue)
	got, err := structaccess.GetByString(obj, path)
	require.NoError(t, err, "GetByString(%#v, %q)", obj, path)
	require.Equal(t, expectedGetValue, got, "SetByString(%#v, %q, %#v) then GetByString should return %#v", obj, path, setValue, expectedGetValue)
}

func TestSetJobSettings(t *testing.T) {
	jobSettings := jobs.JobSettings{
		Name: "job foo",
		Tasks: []jobs.Task{
			{
				TaskKey: "job_task",
				RunJobTask: &jobs.RunJobTask{
					JobId: 0, // This will be resolved from the reference
				},
			},
		},
	}

	err := structaccess.SetByString(&jobSettings, "tasks[0].run_job_task.job_id", "123")
	require.NoError(t, err)

	require.Equal(t, &jobs.JobSettings{
		Name: "job foo",
		Tasks: []jobs.Task{
			{
				TaskKey: "job_task",
				RunJobTask: &jobs.RunJobTask{
					JobId: 123,
				},
			},
		},
	}, &jobSettings)
}

func TestSet_EmbeddedStructForceSendFields(t *testing.T) {
	type Inner struct {
		InnerFieldOmit   string   `json:"inner_field_omit,omitempty"`
		InnerFieldNoOmit string   `json:"inner_field_no_omit"`
		ForceSendFields  []string `json:"-"`
	}

	type Outer struct {
		OuterFieldOmit   string `json:"outer_field_omit,omitempty"`
		OuterFieldNoOmit string `json:"outer_field_no_omit"`
		Inner
	}

	t.Run("set nil", func(t *testing.T) {
		obj := Outer{
			OuterFieldOmit:   "outer_value",
			OuterFieldNoOmit: "outer_no_omit",
			Inner: Inner{
				InnerFieldOmit:   "inner_value",
				InnerFieldNoOmit: "inner_no_omit",
				ForceSendFields:  []string{"OuterFieldOmit", "InnerFieldOmit"},
			},
		}

		// Set nil value for outer field - roundtrip nil -> nil
		// Outer has no ForceSendFields, so Inner.ForceSendFields unchanged
		testSet(t, &obj, "outer_field_omit", nil)
		assert.Equal(t, []string{"OuterFieldOmit", "InnerFieldOmit"}, obj.ForceSendFields)

		// Set nil value for outer field no-omit - roundtrip nil -> ""
		// Outer has no ForceSendFields, so Inner.ForceSendFields unchanged
		testSetGet(t, &obj, "outer_field_no_omit", nil, "")
		assert.Equal(t, []string{"OuterFieldOmit", "InnerFieldOmit"}, obj.ForceSendFields)

		// Set nil value for inner field no-omit - roundtrip nil -> ""
		// Inner has ForceSendFields but this field has no omitempty, so no change to ForceSendFields
		testSetGet(t, &obj, "inner_field_no_omit", nil, "")
		assert.Equal(t, []string{"OuterFieldOmit", "InnerFieldOmit"}, obj.ForceSendFields)

		// Set nil value for inner field omit - should clear and remove from Inner.ForceSendFields
		testSet(t, &obj, "inner_field_omit", nil)
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)

		// Repeat
		testSet(t, &obj, "inner_field_omit", nil)
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
	})

	t.Run("set empty", func(t *testing.T) {
		obj := Outer{
			OuterFieldOmit:   "outer_value",
			OuterFieldNoOmit: "outer_no_omit",
			Inner: Inner{
				InnerFieldOmit:   "inner_value",
				InnerFieldNoOmit: "inner_no_omit",
				ForceSendFields:  []string{},
			},
		}

		// Set empty string for outer field, but get nil back (omitempty, no ForceSendFields)
		testSetGet(t, &obj, "outer_field_omit", "", nil)
		// Outer has no ForceSendFields, so Inner.ForceSendFields unchanged
		assert.Equal(t, []string{}, obj.ForceSendFields)

		// Set empty string for outer field no-omit - roundtrip "" -> ""
		testSet(t, &obj, "outer_field_no_omit", "")
		// Outer has no ForceSendFields, so Inner.ForceSendFields unchanged
		assert.Equal(t, []string{}, obj.ForceSendFields)

		// Set empty string for inner field no-omit - roundtrip "" -> ""
		testSet(t, &obj, "inner_field_no_omit", "")
		// Inner has ForceSendFields but this field has no omitempty, so no change to ForceSendFields
		assert.Equal(t, []string{}, obj.ForceSendFields)

		// Set empty string for inner field omit - should set field and add to Inner.ForceSendFields
		testSet(t, &obj, "inner_field_omit", "")
		assert.Equal(t, []string{"InnerFieldOmit"}, obj.ForceSendFields)

		// Repeat - should not duplicate in ForceSendFields
		testSet(t, &obj, "inner_field_omit", "")
		assert.Equal(t, []string{"InnerFieldOmit"}, obj.ForceSendFields)
	})
}

func TestSet_MixedForceSendFields(t *testing.T) {
	type First struct {
		FirstFieldOmit   string `json:"first_field_omit,omitempty"`
		FirstFieldNoOmit string `json:"first_field_no_omit"`
	}

	type Second struct {
		SecondFieldOmit   string   `json:"second_field_omit,omitempty"`
		SecondFieldNoOmit string   `json:"second_field_no_omit"`
		ForceSendFields   []string `json:"-"`
	}

	type Outer struct {
		OuterFieldOmit   string   `json:"outer_field_omit,omitempty"`
		OuterFieldNoOmit string   `json:"outer_field_no_omit"`
		ForceSendFields  []string `json:"-"`
		First
		Second
	}

	t.Run("set nil", func(t *testing.T) {
		obj := Outer{
			OuterFieldOmit:   "outer_value",
			OuterFieldNoOmit: "outer_no_omit",
			ForceSendFields:  []string{"OuterFieldOmit", "FirstFieldOmit"},
			First: First{
				FirstFieldOmit:   "first_value",
				FirstFieldNoOmit: "first_no_omit",
			},
			Second: Second{
				SecondFieldOmit:   "second_value",
				SecondFieldNoOmit: "second_no_omit",
				ForceSendFields:   []string{"SecondFieldOmit"},
			},
		}

		// Set nil for outer field - should clear and remove from Outer.ForceSendFields
		testSet(t, &obj, "outer_field_omit", nil)
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields)

		// Set nil for outer field no-omit - roundtrip nil -> ""
		testSetGet(t, &obj, "outer_field_no_omit", nil, "")
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields)

		// Set nil for first field no-omit - roundtrip nil -> ""
		testSetGet(t, &obj, "first_field_no_omit", nil, "")
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields)

		// Set nil for first field omit - should clear field but NOT affect any ForceSendFields
		// (First has no ForceSendFields, so nothing to manage there)
		testSet(t, &obj, "first_field_omit", nil)
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields) // unchanged - First fields don't belong to Outer's ForceSendFields management
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields)

		// Set nil for second field no-omit - roundtrip nil -> ""
		testSetGet(t, &obj, "second_field_no_omit", nil, "")
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields)

		// Set nil for second field omit - should clear and remove from Second.ForceSendFields
		// (Second owns its own fields since it has ForceSendFields)
		testSet(t, &obj, "second_field_omit", nil)
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields) // unchanged
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)

		// Repeat operations - should be idempotent
		testSet(t, &obj, "first_field_omit", nil)
		testSet(t, &obj, "second_field_omit", nil)
		assert.Equal(t, []string{"FirstFieldOmit"}, obj.ForceSendFields) // unchanged
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)
	})

	t.Run("set empty", func(t *testing.T) {
		obj := Outer{
			OuterFieldOmit:   "outer_value",
			OuterFieldNoOmit: "outer_no_omit",
			ForceSendFields:  []string{},
			First: First{
				FirstFieldOmit:   "first_value",
				FirstFieldNoOmit: "first_no_omit",
			},
			Second: Second{
				SecondFieldOmit:   "second_value",
				SecondFieldNoOmit: "second_no_omit",
				ForceSendFields:   []string{},
			},
		}

		// Set empty for outer field omit - should zero and add to Outer.ForceSendFields
		testSet(t, &obj, "outer_field_omit", "")
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)

		// Set empty for outer field no-omit - roundtrip "" -> ""
		testSet(t, &obj, "outer_field_no_omit", "")
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)

		// Set empty for first field no-omit - roundtrip "" -> ""
		testSet(t, &obj, "first_field_no_omit", "")
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)

		// Set empty for first field omit - should zero field but get nil back (no ForceSendFields to manage)
		// (First has no ForceSendFields, so empty value won't survive roundtrip)
		testSetGet(t, &obj, "first_field_omit", "", nil)
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields) // unchanged - First fields don't belong to Outer's ForceSendFields management
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)

		// Set empty for second field no-omit - roundtrip "" -> ""
		testSet(t, &obj, "second_field_no_omit", "")
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{}, obj.Second.ForceSendFields)

		// Set empty for second field omit - should zero and add to Second.ForceSendFields
		// (Second owns its own fields since it has ForceSendFields)
		testSet(t, &obj, "second_field_omit", "")
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields)

		// Repeat operations - should not duplicate
		testSet(t, &obj, "second_field_omit", "")
		assert.Equal(t, []string{"OuterFieldOmit"}, obj.ForceSendFields)
		assert.Equal(t, []string{"SecondFieldOmit"}, obj.Second.ForceSendFields) // no duplicates
	})
}
