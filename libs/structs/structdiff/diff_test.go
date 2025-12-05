package structdiff

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type B struct{ S string }

type A struct {
	XX int            `json:"xx"`
	X  int            `json:"x,omitempty"`
	B  B              `json:"b,omitempty"`
	P  *B             `json:"p,omitempty"`
	M  map[string]int `json:"m,omitempty"`
	L  []string       `json:"l,omitempty"`
}

type C struct {
	Name            string   `json:"name,omitempty"`
	Age             int      `json:"age,omitempty"`
	IsEnabled       bool     `json:"is_enabled,omitempty"`
	Title           string   `json:"title"` // no omitempty
	ForceSendFields []string `json:"-"`
}

type Embedded struct {
	EmbeddedString string `json:"embedded_field,omitempty"`
	EmbeddedInt    int    `json:"embedded_int,omitempty"`
}

type D struct {
	Embedded        // Anonymous embedded struct
	Name     string `json:"name,omitempty"`
}

type E struct {
	*Embedded        // Pointer to anonymous embedded struct
	Name      string `json:"name,omitempty"`
}

// ResolvedChange represents a change with the field path as a string (like the old Change struct)
type ResolvedChange struct {
	Field string
	Old   any
	New   any
}

// Helper function to convert []Change to []ResolvedChange for test comparison
func resolveChanges(changes []Change) []ResolvedChange {
	if len(changes) == 0 {
		return nil
	}
	resolved := make([]ResolvedChange, len(changes))
	for i, ch := range changes {
		resolved[i] = ResolvedChange{
			Field: ch.Path.String(),
			Old:   ch.Old,
			New:   ch.New,
		}
	}
	return resolved
}

func TestGetStructDiff(t *testing.T) {
	b1 := &B{S: "one"}
	b2 := &B{S: "two"}

	// An *invalid* reflect.Value (IsValid() == false)
	var invalidRV reflect.Value

	tests := []struct {
		name    string
		a, b    any
		want    []ResolvedChange
		wantErr bool
	}{
		{
			name:    "pointer vs non-pointer",
			a:       &A{},
			b:       A{},
			wantErr: true,
		},
		{
			name: "both typed nil pointers",
			a:    (*A)(nil),
			b:    (*A)(nil),
			want: nil,
		},
		{
			name: "both untyped nil pointers",
			a:    nil,
			b:    nil,
			want: nil,
		},
		{
			name: "typed nil vs non-nil pointer",
			a:    (*A)(nil),
			b:    &A{X: 1},
			want: []ResolvedChange{{Field: "", Old: (*A)(nil), New: &A{X: 1}}},
		},
		{
			name:    "different top-level types",
			a:       A{},
			b:       B{},
			wantErr: true,
		},
		{
			name:    "invalid reflect value input",
			a:       invalidRV,
			b:       A{},
			wantErr: true,
		},
		{
			name: "simple field change - omitempty",
			a:    A{X: 5},
			b:    A{},
			want: []ResolvedChange{{Field: "x", Old: 5, New: nil}},
		},
		{
			name: "simple field change - required",
			a:    A{XX: 5},
			b:    A{},
			want: []ResolvedChange{{Field: "xx", Old: 5, New: 0}},
		},
		{
			name: "nested struct field",
			a:    A{B: B{S: "one"}},
			b:    A{B: B{S: "two"}},
			want: []ResolvedChange{{Field: "b.S", Old: "one", New: "two"}},
		},
		{
			name: "pointer nil vs value",
			a:    A{P: b1},
			b:    A{},
			want: []ResolvedChange{{Field: "p", Old: b1, New: nil}},
		},
		{
			name: "pointer nested value diff",
			a:    A{P: b1},
			b:    A{P: b2},
			want: []ResolvedChange{{Field: "p.S", Old: "one", New: "two"}},
		},
		{
			name: "map diff",
			a:    A{M: map[string]int{"a": 1}},
			b:    A{M: map[string]int{"a": 2}},
			want: []ResolvedChange{{Field: "m['a']", Old: 1, New: 2}},
		},
		{
			name: "slice diff",
			a:    A{L: []string{"a"}},
			b:    A{L: []string{"a", "b"}},
			want: []ResolvedChange{{Field: "l", Old: []string{"a"}, New: []string{"a", "b"}}},
		},

		// ForceSendFields with non-empty fields (omitempty)
		{
			name: "forcesend nonempty 1",
			a:    C{Name: "Hello", ForceSendFields: []string{"Name"}},
			b:    C{Name: "World"},
			want: []ResolvedChange{{Field: "name", Old: "Hello", New: "World"}},
		},
		{
			name: "forcesend noneempty 2",
			a:    C{Name: "Hello", ForceSendFields: []string{"Name"}},
			b:    C{Name: "World", ForceSendFields: []string{"Name"}},
			want: []ResolvedChange{{Field: "name", Old: "Hello", New: "World"}},
		},
		{
			name: "forcesend noneempty 3",
			a:    C{Name: "Hello"},
			b:    C{Name: "World", ForceSendFields: []string{"Name"}},
			want: []ResolvedChange{{Field: "name", Old: "Hello", New: "World"}},
		},

		// ForceSendFields with non-empty fields (required)
		{
			name: "forcesend nonempty required 1",
			a:    C{Title: "Hello", ForceSendFields: []string{"Title"}},
			b:    C{Title: "World"},
			want: []ResolvedChange{{Field: "title", Old: "Hello", New: "World"}},
		},
		{
			name: "forcesend noneempty required 2",
			a:    C{Title: "Hello", ForceSendFields: []string{"Title"}},
			b:    C{Title: "World", ForceSendFields: []string{"Title"}},
			want: []ResolvedChange{{Field: "title", Old: "Hello", New: "World"}},
		},
		{
			name: "forcesend noneempty required 3",
			a:    C{Title: "Hello"},
			b:    C{Title: "World", ForceSendFields: []string{"Title"}},
			want: []ResolvedChange{{Field: "title", Old: "Hello", New: "World"}},
		},

		// ForceSendFields with empty fields
		{
			name: "forcesend empty string diff",
			a:    C{ForceSendFields: []string{"Name"}}, // Name == "" zero, but forced
			b:    C{},
			want: []ResolvedChange{{Field: "name", Old: "", New: nil}},
		},
		{
			name: "forcesend empty int diff",
			a:    C{ForceSendFields: []string{"Age"}},
			b:    C{},
			want: []ResolvedChange{{Field: "age", Old: 0, New: nil}},
		},
		{
			name: "forcesend empty bool diff",
			a:    C{ForceSendFields: []string{"IsEnabled"}},
			b:    C{},
			want: []ResolvedChange{{Field: "is_enabled", Old: false, New: nil}},
		},
		{
			name: "forcesend empty all",
			a:    C{ForceSendFields: []string{"Name", "IsEnabled"}},
			b:    C{ForceSendFields: []string{"Age"}},
			want: []ResolvedChange{
				{Field: "name", Old: "", New: nil},
				{Field: "age", Old: nil, New: 0},
				{Field: "is_enabled", Old: false, New: nil},
			},
		},
		{
			name: "forcesend is different but field is non empty – no diff",
			a:    C{Name: "name", ForceSendFields: []string{"Name"}},
			b:    C{Name: "name"},
			want: nil,
		},
		{
			name: "forcesend on non-omitempty field – no diff",
			a:    C{Title: "", ForceSendFields: []string{"Title"}},
			b:    C{},
			want: nil,
		},

		// ForceSendFields inside slice
		{
			name: "slice of struct with only ForceSendFields diff",
			a:    []C{{Name: "hello", ForceSendFields: []string{"Name"}}},
			b:    []C{{Name: "hello"}},
			want: nil,
		},
		{
			name: "slice of struct with empty string and ForceSendFields diff",
			a:    []C{{Name: "", ForceSendFields: []string{"Name"}}},
			b:    []C{{Name: ""}},
			want: []ResolvedChange{{Field: "[0].name", Old: "", New: nil}},
		},

		// ForceSendFields inside map value
		{
			name: "forcesend inside map",
			a:    map[string]C{"key1": {Title: "title", ForceSendFields: []string{"Name", "IsEnabled", "Title"}}},
			b:    map[string]C{"key1": {Title: "title", ForceSendFields: []string{"Age"}}},
			want: []ResolvedChange{
				{Field: "['key1'].name", Old: "", New: nil},
				{Field: "['key1'].age", Old: nil, New: 0},
				{Field: "['key1'].is_enabled", Old: false, New: nil},
			},
		},

		// ForceSendFields inside map value pointer
		{
			name: "forcesend inside map",
			a:    map[string]*C{"key1": {Title: "title", ForceSendFields: []string{"Name", "IsEnabled", "Title"}}},
			b:    map[string]*C{"key1": {Title: "title", ForceSendFields: []string{"Age"}}},
			want: []ResolvedChange{
				{Field: "['key1'].name", Old: "", New: nil},
				{Field: "['key1'].age", Old: nil, New: 0},
				{Field: "['key1'].is_enabled", Old: false, New: nil},
			},
		},

		// Embedded struct tests
		{
			name: "embedded struct field change",
			a:    D{Embedded: Embedded{EmbeddedString: "old"}, Name: "test"},
			b:    D{Embedded: Embedded{EmbeddedString: "new"}, Name: "test"},
			want: []ResolvedChange{{Field: "embedded_field", Old: "old", New: "new"}},
		},
		{
			name: "embedded struct multiple field changes",
			a:    D{Embedded: Embedded{EmbeddedString: "old", EmbeddedInt: 5}, Name: "test"},
			b:    D{Embedded: Embedded{EmbeddedString: "new", EmbeddedInt: 10}, Name: "test"},
			want: []ResolvedChange{
				{Field: "embedded_field", Old: "old", New: "new"},
				{Field: "embedded_int", Old: 5, New: 10},
			},
		},
		{
			name: "embedded and non-embedded field changes",
			a:    D{Embedded: Embedded{EmbeddedString: "old"}, Name: "alice"},
			b:    D{Embedded: Embedded{EmbeddedString: "new"}, Name: "bob"},
			want: []ResolvedChange{
				{Field: "embedded_field", Old: "old", New: "new"},
				{Field: "name", Old: "alice", New: "bob"},
			},
		},
		{
			name: "embedded struct zero to non-zero",
			a:    D{Name: "test"},
			b:    D{Embedded: Embedded{EmbeddedString: "value"}, Name: "test"},
			want: []ResolvedChange{{Field: "embedded_field", Old: nil, New: "value"}},
		},
		{
			name: "embedded struct non-zero to zero",
			a:    D{Embedded: Embedded{EmbeddedString: "value"}, Name: "test"},
			b:    D{Name: "test"},
			want: []ResolvedChange{{Field: "embedded_field", Old: "value", New: nil}},
		},
		{
			name: "embedded struct both zero",
			a:    D{Name: "test"},
			b:    D{Name: "test"},
			want: nil,
		},
		{
			name: "embedded struct only non-embedded field changes",
			a:    D{Embedded: Embedded{EmbeddedString: "same"}, Name: "alice"},
			b:    D{Embedded: Embedded{EmbeddedString: "same"}, Name: "bob"},
			want: []ResolvedChange{{Field: "name", Old: "alice", New: "bob"}},
		},

		// Pointer embedded struct tests
		{
			name: "pointer embedded struct field change",
			a:    E{Embedded: &Embedded{EmbeddedString: "old"}, Name: "test"},
			b:    E{Embedded: &Embedded{EmbeddedString: "new"}, Name: "test"},
			want: []ResolvedChange{{Field: "embedded_field", Old: "old", New: "new"}},
		},
		{
			name: "pointer embedded struct multiple field changes",
			a:    E{Embedded: &Embedded{EmbeddedString: "old", EmbeddedInt: 5}, Name: "test"},
			b:    E{Embedded: &Embedded{EmbeddedString: "new", EmbeddedInt: 10}, Name: "test"},
			want: []ResolvedChange{
				{Field: "embedded_field", Old: "old", New: "new"},
				{Field: "embedded_int", Old: 5, New: 10},
			},
		},
		{
			name: "pointer embedded and non-embedded field changes",
			a:    E{Embedded: &Embedded{EmbeddedString: "old"}, Name: "alice"},
			b:    E{Embedded: &Embedded{EmbeddedString: "new"}, Name: "bob"},
			want: []ResolvedChange{
				{Field: "embedded_field", Old: "old", New: "new"},
				{Field: "name", Old: "alice", New: "bob"},
			},
		},
		{
			name: "pointer embedded struct nil to non-nil",
			a:    E{Name: "test"},
			b:    E{Embedded: &Embedded{EmbeddedString: "value"}, Name: "test"},
			want: []ResolvedChange{{Field: "", Old: (*Embedded)(nil), New: &Embedded{EmbeddedString: "value"}}},
		},
		{
			name: "pointer embedded struct non-nil to nil",
			a:    E{Embedded: &Embedded{EmbeddedString: "value"}, Name: "test"},
			b:    E{Name: "test"},
			want: []ResolvedChange{{Field: "", Old: &Embedded{EmbeddedString: "value"}, New: (*Embedded)(nil)}},
		},
		{
			name: "pointer embedded struct both nil",
			a:    E{Name: "test"},
			b:    E{Name: "test"},
			want: nil,
		},
		{
			name: "pointer embedded struct only non-embedded field changes",
			a:    E{Embedded: &Embedded{EmbeddedString: "same"}, Name: "alice"},
			b:    E{Embedded: &Embedded{EmbeddedString: "same"}, Name: "bob"},
			want: []ResolvedChange{{Field: "name", Old: "alice", New: "bob"}},
		},
		{
			name: "pointer embedded struct zero to non-zero int",
			a:    E{Name: "test"},
			b:    E{Embedded: &Embedded{EmbeddedInt: 42}, Name: "test"},
			want: []ResolvedChange{{Field: "", Old: (*Embedded)(nil), New: &Embedded{EmbeddedInt: 42}}},
		},
		{
			name: "pointer embedded struct non-zero to zero int",
			a:    E{Embedded: &Embedded{EmbeddedInt: 42}, Name: "test"},
			b:    E{Name: "test"},
			want: []ResolvedChange{{Field: "", Old: &Embedded{EmbeddedInt: 42}, New: (*Embedded)(nil)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.b, nil)

			assert.Equal(t, tt.want, resolveChanges(got))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})

		t.Run(tt.name+" mirror", func(t *testing.T) {
			got, err := GetStructDiff(tt.b, tt.a, nil)

			var mirrorWant []ResolvedChange
			for _, ch := range tt.want {
				mirrorWant = append(mirrorWant, ResolvedChange{
					Field: ch.Field,
					Old:   ch.New,
					New:   ch.Old,
				})
			}

			assert.Equal(t, mirrorWant, resolveChanges(got))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})

		t.Run(tt.name+" equal A", func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.a, nil)
			assert.NoError(t, err)
			assert.Nil(t, got)
		})

		t.Run(tt.name+" equal B", func(t *testing.T) {
			got, err := GetStructDiff(tt.b, tt.b, nil)
			assert.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}

type Task struct {
	TaskKey     string `json:"task_key,omitempty"`
	Description string `json:"description,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
}

type Job struct {
	Name  string `json:"name,omitempty"`
	Tasks []Task `json:"tasks,omitempty"`
}

func taskKeyFunc(task Task) (string, string) {
	return "task_key", task.TaskKey
}

func TestGetStructDiffSliceKeys(t *testing.T) {
	sliceKeys := map[string]KeyFunc{
		"tasks": taskKeyFunc,
	}

	tests := []struct {
		name string
		a, b any
		want []ResolvedChange
	}{
		{
			name: "slice with same keys same order",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}, {TaskKey: "b", Description: "two"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}, {TaskKey: "b", Description: "two"}}},
			want: nil,
		},
		{
			name: "slice with same keys different order",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}, {TaskKey: "b", Description: "two"}}},
			b:    Job{Tasks: []Task{{TaskKey: "b", Description: "two"}, {TaskKey: "a", Description: "one"}}},
			want: nil,
		},
		{
			name: "slice with same keys field change",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "changed"}}},
			want: []ResolvedChange{{Field: "tasks[task_key='a'].description", Old: "one", New: "changed"}},
		},
		{
			name: "slice element added",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}, {TaskKey: "b", Description: "two"}}},
			want: []ResolvedChange{{Field: "tasks[task_key='b']", Old: nil, New: Task{TaskKey: "b", Description: "two"}}},
		},
		{
			name: "slice element removed",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}, {TaskKey: "b", Description: "two"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}}},
			want: []ResolvedChange{{Field: "tasks[task_key='b']", Old: Task{TaskKey: "b", Description: "two"}, New: nil}},
		},
		{
			name: "slice element replaced",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}}},
			b:    Job{Tasks: []Task{{TaskKey: "b", Description: "two"}}},
			want: []ResolvedChange{
				{Field: "tasks[task_key='a']", Old: Task{TaskKey: "a", Description: "one"}, New: nil},
				{Field: "tasks[task_key='b']", Old: nil, New: Task{TaskKey: "b", Description: "two"}},
			},
		},
		{
			name: "multiple changes with reorder",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "one"}, {TaskKey: "b", Description: "two"}, {TaskKey: "c", Description: "three"}}},
			b:    Job{Tasks: []Task{{TaskKey: "c", Description: "changed"}, {TaskKey: "a", Description: "one"}}},
			want: []ResolvedChange{
				{Field: "tasks[task_key='b']", Old: Task{TaskKey: "b", Description: "two"}, New: nil},
				{Field: "tasks[task_key='c'].description", Old: "three", New: "changed"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.b, sliceKeys)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, resolveChanges(got))
		})
	}
}

type Nested struct {
	Items []Item `json:"items,omitempty"`
}

type Item struct {
	ID    string `json:"id,omitempty"`
	Value int    `json:"value,omitempty"`
}

type Root struct {
	Nested []Nested `json:"nested,omitempty"`
}

func itemKeyFunc(item Item) (string, string) {
	return "id", item.ID
}

func TestGetStructDiffNestedSliceKeys(t *testing.T) {
	sliceKeys := map[string]KeyFunc{
		"nested[*].items": itemKeyFunc,
	}

	tests := []struct {
		name string
		a, b any
		want []ResolvedChange
	}{
		{
			name: "nested slice with same keys different order",
			a:    Root{Nested: []Nested{{Items: []Item{{ID: "x", Value: 1}, {ID: "y", Value: 2}}}}},
			b:    Root{Nested: []Nested{{Items: []Item{{ID: "y", Value: 2}, {ID: "x", Value: 1}}}}},
			want: nil,
		},
		{
			name: "nested slice field change",
			a:    Root{Nested: []Nested{{Items: []Item{{ID: "x", Value: 1}}}}},
			b:    Root{Nested: []Nested{{Items: []Item{{ID: "x", Value: 99}}}}},
			want: []ResolvedChange{{Field: "nested[0].items[id='x'].value", Old: 1, New: 99}},
		},
		{
			name: "nested slice element added",
			a:    Root{Nested: []Nested{{Items: []Item{{ID: "x", Value: 1}}}}},
			b:    Root{Nested: []Nested{{Items: []Item{{ID: "x", Value: 1}, {ID: "y", Value: 2}}}}},
			want: []ResolvedChange{{Field: "nested[0].items[id='y']", Old: nil, New: Item{ID: "y", Value: 2}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.b, sliceKeys)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, resolveChanges(got))
		})
	}
}

func TestGetStructDiffSliceKeysInvalidFunc(t *testing.T) {
	tests := []struct {
		name    string
		keyFunc any
		errMsg  string
	}{
		{
			name:    "not a function",
			keyFunc: "not a function",
			errMsg:  "KeyFunc must be a function, got string",
		},
		{
			name:    "wrong number of parameters",
			keyFunc: func() (string, string) { return "", "" },
			errMsg:  "KeyFunc must have exactly 1 parameter, got 0",
		},
		{
			name:    "too many parameters",
			keyFunc: func(a, b Task) (string, string) { return "", "" },
			errMsg:  "KeyFunc must have exactly 1 parameter, got 2",
		},
		{
			name:    "wrong number of returns",
			keyFunc: func(t Task) string { return "" },
			errMsg:  "KeyFunc must return exactly 2 values, got 1",
		},
		{
			name:    "wrong first return type",
			keyFunc: func(t Task) (int, string) { return 0, "" },
			errMsg:  "KeyFunc must return (string, string), got (int, string)",
		},
		{
			name:    "wrong second return type",
			keyFunc: func(t Task) (string, int) { return "", 0 },
			errMsg:  "KeyFunc must return (string, string), got (string, int)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sliceKeys := map[string]KeyFunc{"tasks": tt.keyFunc}
			a := Job{Tasks: []Task{{TaskKey: "a"}}}
			b := Job{Tasks: []Task{{TaskKey: "a"}}}
			_, err := GetStructDiff(a, b, sliceKeys)
			assert.EqualError(t, err, tt.errMsg)
		})
	}
}

func TestGetStructDiffSliceKeysWrongArgType(t *testing.T) {
	// Function expects Item but slice contains Task
	sliceKeys := map[string]KeyFunc{
		"tasks": func(item Item) (string, string) {
			return "id", item.ID
		},
	}
	a := Job{Tasks: []Task{{TaskKey: "a"}}}
	b := Job{Tasks: []Task{{TaskKey: "b"}}}
	_, err := GetStructDiff(a, b, sliceKeys)
	assert.EqualError(t, err, "KeyFunc expects structdiff.Item, got structdiff.Task")
}

func TestGetStructDiffSliceKeysDuplicates(t *testing.T) {
	sliceKeys := map[string]KeyFunc{
		"tasks": taskKeyFunc,
	}

	tests := []struct {
		name string
		a, b Job
		want []ResolvedChange
	}{
		{
			name: "same duplicates no change",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "2"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "2"}}},
			want: nil,
		},
		{
			name: "duplicates with field change",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "2"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "changed"}}},
			want: []ResolvedChange{{Field: "tasks[task_key='a'].description", Old: "2", New: "changed"}},
		},
		{
			name: "extra in old is deleted",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "2"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}}},
			want: []ResolvedChange{{Field: "tasks[task_key='a']", Old: Task{TaskKey: "a", Description: "2"}, New: nil}},
		},
		{
			name: "extra in new is added",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "2"}}},
			want: []ResolvedChange{{Field: "tasks[task_key='a']", Old: nil, New: Task{TaskKey: "a", Description: "2"}}},
		},
		{
			name: "two in old one in new with change",
			a:    Job{Tasks: []Task{{TaskKey: "a", Description: "1"}, {TaskKey: "a", Description: "2"}}},
			b:    Job{Tasks: []Task{{TaskKey: "a", Description: "changed"}}},
			want: []ResolvedChange{
				{Field: "tasks[task_key='a'].description", Old: "1", New: "changed"},
				{Field: "tasks[task_key='a']", Old: Task{TaskKey: "a", Description: "2"}, New: nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.b, sliceKeys)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, resolveChanges(got))
		})
	}
}
