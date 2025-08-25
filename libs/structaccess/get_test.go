package structaccess

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// unexported global test case type
type testCase struct {
	name     string
	path     string
	want     any
	wantSelf bool
	errFmt   string
}

type inner struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type outerNoFSF struct {
	Conn       *inner            `json:"connection"`
	ConnNotSet *inner            `json:"connection_not_set"`
	Items      []inner           `json:"items"`
	Labels     map[string]string `json:"labels"`
	B          bool              `json:"b"`
	I          int               `json:"i"`
	S          string            `json:"s"`
	BOmit      bool              `json:"b_omit,omitempty"`
	IOmit      int               `json:"i_omit,omitempty"`
	SOmit      string            `json:"s_omit,omitempty"`
	// Unexported or no-json-tag fields should be ignored
	GoOnly string // no json tag: should NOT be accessible
}

type outerWithFSF struct {
	Conn       *inner            `json:"connection"`
	ConnNotSet *inner            `json:"connection_not_set"`
	Items      []inner           `json:"items"`
	Labels     map[string]string `json:"labels"`
	B          bool              `json:"b"`
	I          int               `json:"i"`
	S          string            `json:"s"`
	BOmit      bool              `json:"b_omit,omitempty"`
	IOmit      int               `json:"i_omit,omitempty"`
	SOmit      string            `json:"s_omit,omitempty"`
	GoOnly     string            // no json tag: should NOT be accessible
	// ForceSendFields allows forcing zero-values for specific fields
	ForceSendFields []string
}

func makeOuterNoFSF() outerNoFSF {
	return outerNoFSF{
		Conn: &inner{
			ID:   "abc",
			Name: "x",
		},
		Items: []inner{
			{ID: "i0"},
			{ID: "i1"},
		},
		Labels: map[string]string{
			"env": "dev",
		},
		GoOnly: "hidden",
	}
}

func makeOuterWithFSF() outerWithFSF {
	return outerWithFSF{
		Conn: &inner{
			ID:   "abc",
			Name: "x",
		},
		Items: []inner{
			{ID: "i0"},
			{ID: "i1"},
		},
		Labels: map[string]string{
			"env": "dev",
		},
		GoOnly: "hidden",
	}
}

func runCommonTests(t *testing.T, obj any) {
	t.Helper()

	// type name for error messages that include the struct type
	typeName := reflect.TypeOf(obj).String()

	tests := []testCase{
		{
			name:     "root empty path returns object",
			path:     "",
			wantSelf: true,
		},
		{
			name: "struct json field",
			path: "connection.id",
			want: "abc",
		},
		{
			name: "slice index then field",
			path: "items[1].id",
			want: "i1",
		},
		{
			name: "map string key",
			path: "labels.env",
			want: "dev",
		},
		{
			name: "leading dot allowed",
			path: ".connection.id",
			want: "abc",
		},

		// Regular scalar fields - always return zero values
		{
			name: "bool false",
			path: "b",
			want: false,
		},
		{
			name: "int zero",
			path: "i",
			want: 0,
		},
		{
			name: "string empty",
			path: "s",
			want: "",
		},

		// Errors common to both
		{
			name:   "wildcard not supported",
			path:   "items[*].id",
			errFmt: "invalid path: items[*].id",
		},
		{
			name:   "missing field",
			path:   "connection.missing",
			errFmt: "connection.missing: field \"missing\" not found in structaccess.inner",
		},
		{
			name:   "wrong index target",
			path:   "connection[0]",
			errFmt: "connection[0]: cannot index struct",
		},
		{
			name:   "out of range index",
			path:   "items[5]",
			errFmt: "items[5]: index out of range, length is 2",
		},
		{
			name:   "no json tag field should not be accessible",
			path:   "goOnly",
			errFmt: "goOnly: field \"goOnly\" not found in " + typeName,
		},
		{
			name:   "key on slice not supported",
			path:   "items.id",
			errFmt: "items.id: cannot access key \"id\" on slice",
		},
		{
			name:   "nil pointer access",
			path:   "connection_not_set.id",
			errFmt: "connection_not_set: cannot access nil value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(obj, tt.path)
			if tt.errFmt != "" {
				require.EqualError(t, err, tt.errFmt)
				return
			}
			require.NoError(t, err)
			if tt.wantSelf {
				require.Equal(t, obj, got)
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGet_Common_NoFSF(t *testing.T) {
	runCommonTests(t, makeOuterNoFSF())
	runOmitEmptyTests(t, makeOuterNoFSF(), true) // wantNil=true for NoFSF
}

func TestGet_Common_WithFSF(t *testing.T) {
	obj := makeOuterWithFSF()
	obj.ForceSendFields = []string{"BOmit", "IOmit", "SOmit"}
	runCommonTests(t, obj)
	runOmitEmptyTests(t, obj, false) // wantNil=false for WithFSF
}

func TestGet_Common_WithEmptyFSF(t *testing.T) {
	obj := makeOuterWithFSF()
	// obj.ForceSendFields = []string{} // empty slice
	runCommonTests(t, obj)
	runOmitEmptyTests(t, obj, true) // wantNil=true for empty FSF (behaves like NoFSF)
}

func runOmitEmptyTests(t *testing.T, obj any, wantNil bool) {
	t.Helper()

	tests := []testCase{
		{
			name: "bool omitempty",
			path: "b_omit",
			want: func() any {
				if wantNil {
					return nil
				}
				return false
			}(),
		},
		{
			name: "int omitempty",
			path: "i_omit",
			want: func() any {
				if wantNil {
					return nil
				}
				return 0
			}(),
		},
		{
			name: "string omitempty",
			path: "s_omit",
			want: func() any {
				if wantNil {
					return nil
				}
				return ""
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(obj, tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGet_WithFSF_NilPointerForced(t *testing.T) {
	in := outerWithFSF{
		Conn: nil,
		Items: []inner{
			{ID: "i0"},
		},
		Labels: map[string]string{
			"env": "dev",
		},
		ForceSendFields: []string{"Conn"},
	}
	got, err := Get(in, "connection.id")
	require.NoError(t, err)
	require.Equal(t, "", got)
}

// Additional tests for TODOs
// Final nil value should be supported and returned as nil when field is omitted due to omitempty
func TestGet_FinalNil_ReturnsNil(t *testing.T) {
	type withPtrOmit struct {
		P *inner `json:"p,omitempty"`
	}
	in := withPtrOmit{P: nil}
	got, err := Get(in, "p")
	require.NoError(t, err)
	require.Nil(t, got)
}

// Embedded anonymous struct with ForceSendFields should allow forcing a nil pointer-to-struct
// so that deeper fields can be accessed without error.
func TestGet_EmbeddedWithFSF_ForceNilPointerStruct(t *testing.T) {
	type embedded struct {
		P *inner `json:"p,omitempty"`
	}
	type outer struct {
		embedded
		ForceSendFields []string
	}

	in := outer{
		embedded:        embedded{P: nil},
		ForceSendFields: []string{"P"},
	}
	got, err := Get(in, "p.id")
	require.NoError(t, err)
	require.Equal(t, "", got)
}
