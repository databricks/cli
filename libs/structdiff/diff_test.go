package structdiff

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type B struct{ S string }

type A struct {
	X int            `json:"x,omitempty"`
	B B              `json:"b,omitempty"`
	P *B             `json:"p,omitempty"`
	M map[string]int `json:"m,omitempty"`
	L []string       `json:"l,omitempty"`
}

type C struct {
	Name            string   `json:"name,omitempty"`
	Age             int      `json:"age,omitempty"`
	IsEnabled       bool     `json:"is_enabled,omitempty"`
	Title           string   `json:"title"` // no omitempty
	ForceSendFields []string `json:"-"`
}

func TestGetStructDiff(t *testing.T) {
	b1 := &B{S: "one"}
	b2 := &B{S: "two"}

	// An *invalid* reflect.Value (IsValid() == false)
	var invalidRV reflect.Value

	tests := []struct {
		name    string
		a, b    any
		want    []Change
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
			want: []Change{{Field: "", Old: (*A)(nil), New: &A{X: 1}}},
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
			name: "simple field change",
			a:    A{X: 5},
			b:    A{},
			want: []Change{{Field: ".X", Old: 5, New: 0}},
		},
		{
			name: "nested struct field",
			a:    A{B: B{S: "one"}},
			b:    A{B: B{S: "two"}},
			want: []Change{{Field: ".B.S", Old: "one", New: "two"}},
		},
		{
			name: "pointer nil vs value",
			a:    A{P: b1},
			b:    A{},
			want: []Change{{Field: ".P", Old: b1, New: (*B)(nil)}},
		},
		{
			name: "pointer nested value diff",
			a:    A{P: b1},
			b:    A{P: b2},
			want: []Change{{Field: ".P.S", Old: "one", New: "two"}},
		},
		{
			name: "map diff",
			a:    A{M: map[string]int{"a": 1}},
			b:    A{M: map[string]int{"a": 2}},
			want: []Change{{Field: ".M[\"a\"]", Old: 1, New: 2}},
		},
		{
			name: "slice diff",
			a:    A{L: []string{"a"}},
			b:    A{L: []string{"a", "b"}},
			want: []Change{{Field: ".L", Old: []string{"a"}, New: []string{"a", "b"}}},
		},

		// ForceSendFields related cases
		{
			name: "forcesend empty string diff",
			a:    C{ForceSendFields: []string{"Name"}}, // Name == "" zero, but forced
			b:    C{},
			want: []Change{{Field: ".Name", Old: "", New: nil}},
		},
		{
			name: "forcesend empty int diff",
			a:    C{ForceSendFields: []string{"Age"}},
			b:    C{},
			want: []Change{{Field: ".Age", Old: 0, New: nil}},
		},
		{
			name: "forcesend empty bool diff",
			a:    C{ForceSendFields: []string{"IsEnabled"}},
			b:    C{},
			want: []Change{{Field: ".IsEnabled", Old: false, New: nil}},
		},
		{
			name: "forcesend empty all",
			a:    C{ForceSendFields: []string{"Name", "IsEnabled"}},
			b:    C{ForceSendFields: []string{"Age"}},
			want: []Change{
				{Field: ".Name", Old: "", New: nil},
				{Field: ".Age", Old: nil, New: 0},
				{Field: ".IsEnabled", Old: false, New: nil},
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
			want: []Change{{Field: "[0].Name", Old: "", New: nil}},
		},

		// ForceSendFields inside map value
		{
			name: "forcesend inside map",
			a:    map[string]C{"key1": {Title: "title", ForceSendFields: []string{"Name", "IsEnabled", "Title"}}},
			b:    map[string]C{"key1": {Title: "title", ForceSendFields: []string{"Age"}}},
			want: []Change{
				{Field: "[\"key1\"].Name", Old: "", New: nil},
				{Field: "[\"key1\"].Age", Old: nil, New: 0},
				{Field: "[\"key1\"].IsEnabled", Old: false, New: nil},
			},
		},

		// ForceSendFields inside map value pointer
		{
			name: "forcesend inside map",
			a:    map[string]*C{"key1": {Title: "title", ForceSendFields: []string{"Name", "IsEnabled", "Title"}}},
			b:    map[string]*C{"key1": {Title: "title", ForceSendFields: []string{"Age"}}},
			want: []Change{
				{Field: "[\"key1\"].Name", Old: "", New: nil},
				{Field: "[\"key1\"].Age", Old: nil, New: 0},
				{Field: "[\"key1\"].IsEnabled", Old: false, New: nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.b)
			assert.Equal(t, tt.want, got)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})

		t.Run(tt.name+" mirror", func(t *testing.T) {
			got, err := GetStructDiff(tt.b, tt.a)

			var mirrorWant []Change
			for _, ch := range tt.want {
				mirrorWant = append(mirrorWant, Change{
					Field: ch.Field,
					Old:   ch.New,
					New:   ch.Old,
				})
			}
			assert.Equal(t, mirrorWant, got)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})

		t.Run(tt.name+" equal A", func(t *testing.T) {
			got, err := GetStructDiff(tt.a, tt.a)
			assert.NoError(t, err)
			assert.Nil(t, got)
		})

		t.Run(tt.name+" equal B", func(t *testing.T) {
			got, err := GetStructDiff(tt.b, tt.b)
			assert.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}
