package structaccess

import (
	"fmt"
	"reflect"
	"strings"
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
	Conn   *inner            `json:"connection"`
	Items  []inner           `json:"items"`
	Labels map[string]string `json:"labels"`
	// Unexported or no-json-tag fields should be ignored
	GoOnly string // no json tag: should NOT be accessible
}

type outerWithFSF struct {
	Conn   *inner            `json:"connection"`
	Items  []inner           `json:"items"`
	Labels map[string]string `json:"labels"`
	GoOnly string            // no json tag: should NOT be accessible
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

		// Errors common to both
		{
			name:   "wildcard not supported",
			path:   "items[*].id",
			errFmt: "invalid path: items[*].id",
		},
		{
			name:   "missing field",
			path:   "connection.missing",
			errFmt: "field \"missing\" not found in struct structaccess.inner",
		},
		{
			name:   "wrong index target",
			path:   "connection[0]",
			errFmt: "expected slice/array to index [0], found struct",
		},
		{
			name:   "out of range index",
			path:   "items[5]",
			errFmt: "index out of range [5] with length 2",
		},
		{
			name:   "no json tag field should not be accessible",
			path:   "goOnly",
			errFmt: "field \"goOnly\" not found in struct %s",
		},
		{
			name:   "key on slice not supported",
			path:   "items.id",
			errFmt: "key \"id\" cannot be applied to slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(obj, tt.path)
			if tt.errFmt != "" {
				expected := tt.errFmt
				if strings.Contains(expected, "%") {
					expected = fmt.Sprintf(expected, typeName)
				}
				require.EqualError(t, err, expected)
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
}

func TestGet_Common_WithFSF(t *testing.T) {
	runCommonTests(t, makeOuterWithFSF())
}

// Dedicated tests for cases with different outcomes between types
func TestGet_NoFSF_NilPointer(t *testing.T) {
	in := outerNoFSF{
		Items: []inner{
			{ID: "i0"},
		},
		Labels: map[string]string{
			"env": "dev",
		},
	}
	_, err := Get(in, "connection.id")
	require.EqualError(t, err, "nil found at connection")
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

// ForceSendFields with scalar types
type fsfScalars struct {
	B bool   `json:"b"`
	I int    `json:"i"`
	S string `json:"s"`

	ForceSendFields []string
}

// Scalars with omitempty
type fsfScalarsOmit struct {
	B bool   `json:"b,omitempty"`
	I int    `json:"i,omitempty"`
	S string `json:"s,omitempty"`

	ForceSendFields []string
}

func TestGet_WithFSF_ScalarZeroValues(t *testing.T) {
	in := fsfScalars{
		B: false,
		I: 0,
		S: "",
		ForceSendFields: []string{
			"B",
			"I",
			"S",
		},
	}

	gotB, err := Get(in, "b")
	require.NoError(t, err)
	require.Equal(t, false, gotB)

	gotI, err := Get(in, "i")
	require.NoError(t, err)
	require.Equal(t, 0, gotI)

	gotS, err := Get(in, "s")
	require.NoError(t, err)
	require.Equal(t, "", gotS)
}

func TestGet_NoFSF_ScalarZeroValues(t *testing.T) {
	in := fsfScalars{
		B: false,
		I: 0,
		S: "",
	}

	gotB, err := Get(in, "b")
	require.NoError(t, err)
	require.Equal(t, false, gotB)

	gotI, err := Get(in, "i")
	require.NoError(t, err)
	require.Equal(t, 0, gotI)

	gotS, err := Get(in, "s")
	require.NoError(t, err)
	require.Equal(t, "", gotS)
}

// omitempty behavior
func TestGet_OmitEmpty_NoFSF_ReturnsNil(t *testing.T) {
	in := fsfScalarsOmit{
		B: false,
		I: 0,
		S: "",
	}

	gotB, err := Get(in, "b")
	require.NoError(t, err)
	require.Nil(t, gotB)

	gotI, err := Get(in, "i")
	require.NoError(t, err)
	require.Nil(t, gotI)

	gotS, err := Get(in, "s")
	require.NoError(t, err)
	require.Nil(t, gotS)
}

func TestGet_OmitEmpty_WithFSF_ReturnsZero(t *testing.T) {
	in := fsfScalarsOmit{
		B: false,
		I: 0,
		S: "",
		ForceSendFields: []string{
			"B",
			"I",
			"S",
		},
	}

	gotB, err := Get(in, "b")
	require.NoError(t, err)
	require.Equal(t, false, gotB)

	gotI, err := Get(in, "i")
	require.NoError(t, err)
	require.Equal(t, 0, gotI)

	gotS, err := Get(in, "s")
	require.NoError(t, err)
	require.Equal(t, "", gotS)
}

func TestGet_NoOmitEmpty_ZeroIsZero(t *testing.T) {
	in := fsfScalars{
		B: false,
		I: 0,
		S: "",
	}

	gotB, err := Get(in, "b")
	require.NoError(t, err)
	require.Equal(t, false, gotB)

	gotI, err := Get(in, "i")
	require.NoError(t, err)
	require.Equal(t, 0, gotI)

	gotS, err := Get(in, "s")
	require.NoError(t, err)
	require.Equal(t, "", gotS)
}
