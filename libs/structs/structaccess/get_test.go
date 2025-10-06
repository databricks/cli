package structaccess

import (
	"reflect"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

// unexported global test case type
type testCase struct {
	name        string
	path        string
	want        any
	wantSelf    bool
	errFmt      string
	typeHasPath bool
}

func testGet(t *testing.T, obj any, path string, want any) {
	t.Helper()
	got, err := GetByString(obj, path)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

type inner struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type Key string

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
	POmit      *int              `json:"p_omit,omitempty"`
	QOmit      *inner            `json:"q_omit,omitempty"`
	MapInt     map[int]string    `json:"map_int"`
	AliasMap   map[Key]string    `json:"alias_map"`
	Ignored    string            `json:"-"`
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
	POmit      *int              `json:"p_omit,omitempty"`
	QOmit      *inner            `json:"q_omit,omitempty"`
	MapInt     map[int]string    `json:"map_int"`
	AliasMap   map[Key]string    `json:"alias_map"`
	Ignored    string            `json:"-"`
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
		MapInt:   map[int]string{1: "a"},
		AliasMap: map[Key]string{"foo": "bar"},
		Ignored:  "x",
		GoOnly:   "hidden",
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
		MapInt:   map[int]string{1: "a"},
		AliasMap: map[Key]string{"foo": "bar"},
		Ignored:  "x",
		GoOnly:   "hidden",
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
			name: "map alias key",
			path: "alias_map.foo",
			want: "bar",
		},
		{
			name: "struct field with bracket notation",
			path: "['connection']['id']",
			want: "abc",
		},
		{
			name: "map key with bracket notation",
			path: "labels['env']",
			want: "dev",
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
		{
			name: "nil struct",
			path: "connection_not_set",
			want: nil,
		},

		// Errors common to both
		{
			name:   "wildcard not supported",
			path:   "items[*].id",
			errFmt: "wildcards not supported: items[*].id",
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
			name:        "out of range index",
			path:        "items[5]",
			errFmt:      "items[5]: index out of range, length is 2",
			typeHasPath: true,
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
			name:        "nil pointer access",
			path:        "connection_not_set.id",
			errFmt:      "connection_not_set: cannot access nil value",
			typeHasPath: true,
		},
		{
			name:   "map non-string key type",
			path:   "map_int.any",
			errFmt: "map_int.any: map key must be string, got int",
		},
		{
			name:        "map missing key",
			path:        "labels.missing",
			errFmt:      "labels.missing: key \"missing\" not found in map",
			typeHasPath: true,
		},
		{
			name:   "json dash ignored",
			path:   "ignored",
			errFmt: "ignored: field \"ignored\" not found in " + typeName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasPathError := ValidateByString(reflect.TypeOf(obj), tt.path)
			if tt.errFmt == "" || tt.typeHasPath {
				require.NoError(t, hasPathError)
			} else {
				require.EqualError(t, hasPathError, tt.errFmt)
			}

			got, err := GetByString(obj, tt.path)
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
	obj.ForceSendFields = []string{"BOmit", "IOmit", "SOmit", "POmit", "QOmit"}
	// prepare zero pointers for pointer-omitempty fields
	zi := 0
	obj.POmit = &zi
	obj.QOmit = &inner{}
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
		{
			name: "pointer int omitempty",
			path: "p_omit",
			want: func() any {
				if wantNil {
					return nil
				}
				v := 0
				return &v
			}(),
		},
		{
			name: "pointer struct omitempty",
			path: "q_omit",
			want: func() any {
				if wantNil {
					return nil
				}
				return &inner{}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testGet(t, obj, tt.path, tt.want)
		})
	}
}

func TestGet_Embedded_NilPointerAnonymousNotDescended(t *testing.T) {
	type embedded struct {
		Hidden string `json:"hidden"`
	}
	type host struct {
		*embedded
	}
	require.NoError(t, ValidateByString(reflect.TypeOf(host{}), "hidden"))
	_, err := GetByString(host{}, "hidden")
	require.EqualError(t, err, "hidden: field \"hidden\" not found in structaccess.host")
}

func TestGet_Embedded_ValueAnonymousResolved(t *testing.T) {
	type embedded struct {
		Hidden string `json:"hidden"`
	}
	type host struct {
		embedded
	}
	in := host{embedded: embedded{Hidden: "x"}}
	require.NoError(t, ValidateByString(reflect.TypeOf(in), "hidden"))
	testGet(t, in, "hidden", "x")
}

func TestGet_InterfaceRoot_Unwraps(t *testing.T) {
	v := any(makeOuterNoFSF())
	testGet(t, v, "items[0].id", "i0")
}

func TestGet_BundleTag_SkipsDirect(t *testing.T) {
	type S struct {
		A string `json:"a" bundle:"readonly"`
		B string `json:"b" bundle:"internal"`
		C string `json:"c"`
	}

	// Direct readonly/internal fields should be invisible
	_, err := GetByString(S{A: "x", B: "y", C: "z"}, "a")
	require.EqualError(t, err, "a: field \"a\" not found in structaccess.S")
	require.EqualError(t, ValidateByString(reflect.TypeOf(S{}), "a"), "a: field \"a\" not found in structaccess.S")

	_, err = GetByString(S{}, "b")
	require.EqualError(t, err, "b: field \"b\" not found in structaccess.S")
	require.EqualError(t, ValidateByString(reflect.TypeOf(S{}), "b"), "b: field \"b\" not found in structaccess.S")

	// Visible field works
	testGet(t, S{C: "z"}, "c", "z")
	require.NoError(t, ValidateByString(reflect.TypeOf(S{}), "c"))
}

func TestGet_BundleTag_SkipsPromoted(t *testing.T) {
	type embedded struct {
		Hidden string `json:"hidden" bundle:"readonly"`
	}
	type host struct {
		embedded
	}
	// Promoted readonly field should be invisible
	_, err := GetByString(host{embedded: embedded{Hidden: "x"}}, "hidden")
	require.EqualError(t, err, "hidden: field \"hidden\" not found in structaccess.host")
	require.EqualError(t, ValidateByString(reflect.TypeOf(host{}), "hidden"), "hidden: field \"hidden\" not found in structaccess.host")
}

func TestGet_EmbeddedStructForceSendFields(t *testing.T) {
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

	objWithOuterFSF := Outer{
		OuterFieldOmit:   "",
		OuterFieldNoOmit: "",
		Inner: Inner{
			InnerFieldOmit:   "",
			InnerFieldNoOmit: "",
			// Presence of "OuterFieldOmit" does not have any effect
			// on OuterFieldOmit because it's in the wrong struct
			ForceSendFields: []string{"OuterFieldOmit"},
		},
	}

	testGet(t, objWithOuterFSF, "outer_field_omit", nil)
	testGet(t, objWithOuterFSF, "outer_field_no_omit", "")
	testGet(t, objWithOuterFSF, "inner_field_omit", nil)
	testGet(t, objWithOuterFSF, "inner_field_no_omit", "")

	objWithInnerFSF := Outer{
		OuterFieldOmit:   "",
		OuterFieldNoOmit: "",
		Inner: Inner{
			InnerFieldOmit:   "",
			InnerFieldNoOmit: "",
			ForceSendFields:  []string{"InnerFieldOmit"},
		},
	}

	testGet(t, objWithInnerFSF, "outer_field_omit", nil)
	testGet(t, objWithInnerFSF, "outer_field_no_omit", "")
	testGet(t, objWithInnerFSF, "inner_field_omit", "")
	testGet(t, objWithInnerFSF, "inner_field_no_omit", "")
}

func TestGet_MultipleEmbeddedStructsForceSendFields(t *testing.T) {
	type First struct {
		FirstFieldOmit   string   `json:"first_field_omit,omitempty"`
		FirstFieldNoOmit string   `json:"first_field_no_omit"`
		ForceSendFields  []string `json:"-"`
	}

	type Second struct {
		SecondFieldOmit   string   `json:"second_field_omit,omitempty"`
		SecondFieldNoOmit string   `json:"second_field_no_omit"`
		ForceSendFields   []string `json:"-"`
	}

	type Outer struct {
		OuterFieldOmit   string `json:"outer_field_omit,omitempty"`
		OuterFieldNoOmit string `json:"outer_field_no_omit"`
		First
		Second
	}

	objWithFirstFSF := Outer{
		OuterFieldOmit:   "",
		OuterFieldNoOmit: "",
		First: First{
			FirstFieldOmit:   "",
			FirstFieldNoOmit: "",
			ForceSendFields:  []string{"FirstFieldOmit"},
		},
		Second: Second{
			SecondFieldOmit:   "",
			SecondFieldNoOmit: "",
		},
	}

	testGet(t, objWithFirstFSF, "outer_field_omit", nil)
	testGet(t, objWithFirstFSF, "outer_field_no_omit", "")
	testGet(t, objWithFirstFSF, "first_field_omit", "")
	testGet(t, objWithFirstFSF, "first_field_no_omit", "")
	testGet(t, objWithFirstFSF, "second_field_omit", nil)
	testGet(t, objWithFirstFSF, "second_field_no_omit", "")

	objWithSecondFSF := Outer{
		OuterFieldOmit:   "",
		OuterFieldNoOmit: "",
		First: First{
			FirstFieldOmit:   "",
			FirstFieldNoOmit: "",
		},
		Second: Second{
			SecondFieldOmit:   "",
			SecondFieldNoOmit: "",
			ForceSendFields:   []string{"SecondFieldOmit"},
		},
	}

	testGet(t, objWithSecondFSF, "outer_field_omit", nil)
	testGet(t, objWithSecondFSF, "outer_field_no_omit", "")
	testGet(t, objWithSecondFSF, "first_field_omit", nil)
	testGet(t, objWithSecondFSF, "first_field_no_omit", "")
	testGet(t, objWithSecondFSF, "second_field_omit", "")
	testGet(t, objWithSecondFSF, "second_field_no_omit", "")
}

func TestGet_MixedForceSendFields(t *testing.T) {
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

	objWithOuterFSF := Outer{
		OuterFieldOmit:   "",
		OuterFieldNoOmit: "",
		ForceSendFields:  []string{"OuterFieldOmit"},
		First: First{
			FirstFieldOmit:   "",
			FirstFieldNoOmit: "",
		},
		Second: Second{
			SecondFieldOmit:   "",
			SecondFieldNoOmit: "",
		},
	}

	testGet(t, objWithOuterFSF, "outer_field_omit", "")
	testGet(t, objWithOuterFSF, "outer_field_no_omit", "")
	testGet(t, objWithOuterFSF, "first_field_omit", nil)
	testGet(t, objWithOuterFSF, "first_field_no_omit", "")
	testGet(t, objWithOuterFSF, "second_field_omit", nil)
	testGet(t, objWithOuterFSF, "second_field_no_omit", "")

	objWithSecondFSF := Outer{
		OuterFieldOmit:   "",
		OuterFieldNoOmit: "",
		First: First{
			FirstFieldOmit:   "",
			FirstFieldNoOmit: "",
		},
		Second: Second{
			SecondFieldOmit:   "",
			SecondFieldNoOmit: "",
			ForceSendFields:   []string{"SecondFieldOmit"},
		},
	}

	testGet(t, objWithSecondFSF, "outer_field_omit", nil)
	testGet(t, objWithSecondFSF, "outer_field_no_omit", "")
	testGet(t, objWithSecondFSF, "first_field_omit", nil)
	testGet(t, objWithSecondFSF, "first_field_no_omit", "")
	testGet(t, objWithSecondFSF, "second_field_omit", "")
	testGet(t, objWithSecondFSF, "second_field_no_omit", "")
}

func TestGet_FieldByNameBugRegressionTest(t *testing.T) {
	type BaseResource struct {
		ID string `json:"id,omitempty"`
	}

	type JobSettings struct {
		Name            string   `json:"name"`
		ForceSendFields []string `json:"-"`
	}

	type Job struct {
		BaseResource
		JobSettings
		PermissionsOmit   []string `json:"permissions_omit,omitempty"`
		PermissionsNoOmit []string `json:"permissions_no_omit"`
	}

	objWithNameValue := Job{
		BaseResource: BaseResource{
			ID: "resource_id",
		},
		JobSettings: JobSettings{
			Name:            "test-job",
			ForceSendFields: []string{"Name"},
		},
		PermissionsOmit:   []string{"read", "write"},
		PermissionsNoOmit: []string{"read", "write"},
	}

	testGet(t, objWithNameValue, "id", "resource_id")
	testGet(t, objWithNameValue, "name", "test-job")
	testGet(t, objWithNameValue, "permissions_omit", []string{"read", "write"})
	testGet(t, objWithNameValue, "permissions_no_omit", []string{"read", "write"})

	objWithEmptyFields := Job{
		BaseResource: BaseResource{
			ID: "",
		},
		JobSettings: JobSettings{
			Name:            "",
			ForceSendFields: []string{"Name"},
		},
		PermissionsOmit:   []string{},
		PermissionsNoOmit: []string{},
	}

	testGet(t, objWithEmptyFields, "id", nil)
	testGet(t, objWithEmptyFields, "name", "")
	testGet(t, objWithEmptyFields, "permissions_omit", nil)
	testGet(t, objWithEmptyFields, "permissions_no_omit", []string{})
}

func TestGet_PointerToStructWithZeroValues(t *testing.T) {
	type NestedStruct struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	type Container struct {
		// Pointer with omitempty - this is the key case
		NestedOmit *NestedStruct `json:"nested_omit,omitempty"`
		// Pointer without omitempty for comparison
		NestedNoOmit *NestedStruct `json:"nested_no_omit"`
	}

	// Test case: explicitly set pointer to struct with zero values
	obj := Container{
		NestedOmit:   &NestedStruct{ID: 0, Name: "", Count: 0}, // All zero values but pointer is explicitly set
		NestedNoOmit: &NestedStruct{ID: 0, Name: "", Count: 0},
	}

	// The pointer was explicitly set, so it should return the struct even with zero values
	testGet(t, obj, "nested_omit", &NestedStruct{ID: 0, Name: "", Count: 0})
	testGet(t, obj, "nested_omit.id", int64(0))
	testGet(t, obj, "nested_omit.name", "")
	testGet(t, obj, "nested_omit.count", 0)

	// Non-omit field should definitely work
	testGet(t, obj, "nested_no_omit", &NestedStruct{ID: 0, Name: "", Count: 0})
	testGet(t, obj, "nested_no_omit.id", int64(0))

	// Test case: nil pointer should return nil
	objNil := Container{
		NestedOmit:   nil,
		NestedNoOmit: nil,
	}
	testGet(t, objNil, "nested_omit", nil)
	testGet(t, objNil, "nested_no_omit", nil)
}

func TestGetJobSettings(t *testing.T) {
	jobSettings := jobs.JobSettings{
		Name: "job foo",
		// Tasks []Task `json:"tasks,omitempty"`
		Tasks: []jobs.Task{
			{
				TaskKey: "job_task",
				// RunJobTask *RunJobTask `json:"run_job_task,omitempty"`
				RunJobTask: &jobs.RunJobTask{
					// JobId int64 `json:"job_id"`
					JobId: 0,
				},
			},
		},
	}

	testGet(t, &jobSettings, "tasks[0].run_job_task", &jobs.RunJobTask{})
	testGet(t, &jobSettings, "tasks[0].run_job_task.job_id", int64(0))
}

func TestPipeline(t *testing.T) {
	p := pipelines.CreatePipeline{}

	v, err := GetByString(&p, "ingestion_definition.connectin_name")
	require.Error(t, err)
	require.Equal(t, "ingestion_definition: cannot access nil value", err.Error())
	require.Nil(t, v)
}
