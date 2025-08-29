package calladapt

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Data struct {
	X int
}

type NewData struct {
	Y int
}

type MyStruct struct {
	State int
}

func (t *MyStruct) DoCreate(_ context.Context, _ *Data) (string, error) {
	return "123", nil
}

func (t *MyStruct) PMethodVoid() {}

func (t *MyStruct) PMethodAcceptData(data Data) error {
	if data.X == 1 {
		return errors.New("X cannot be 1")
	}
	return nil
}

func (t *MyStruct) PMethodAcceptAny(v any) error { // for nil interface coverage
	if v == nil {
		return nil
	}
	return nil
}

func (t *MyStruct) PMethodAcceptSlice(s []int) int { // for nil slice coverage
	return len(s)
}

func (t *MyStruct) PMethodAcceptDataPtr(data *Data) error {
	if data == nil {
		return errors.New("data is nil")
	}
	if data.X == 1 {
		return errors.New("X cannot be 1")
	}
	return nil
}

func (t *MyStruct) PMethodTransformData(data Data) (NewData, error) {
	if t == nil {
		return NewData{}, errors.New("t is nil")
	}
	if data.X == 1 {
		return NewData{}, errors.New("X cannot be 1")
	}
	t.State += 10
	return NewData{Y: data.X + t.State}, nil
}

func (t *MyStruct) PMethodTransformDataPtr(data *Data) (*NewData, error) {
	if t == nil {
		return nil, errors.New("t is nil")
	}
	if data == nil {
		return nil, errors.New("data is nil")
	}
	if data.X == 1 {
		return nil, errors.New("X cannot be 1")
	}
	t.State += 10
	return &NewData{Y: data.X + t.State}, nil
}

func (t MyStruct) VMethodAcceptData(data Data) error {
	if data.X == 1 {
		return errors.New("X cannot be 1")
	}
	return nil
}

func (t MyStruct) VMethodTransformNoError(data Data) NewData {
	return NewData{Y: data.X + t.State}
}

func (t *MyStruct) PMethodTransformPtrNoError(data *Data) *NewData {
	return &NewData{Y: data.X + t.State}
}

// Additional methods for edge case testing
func (t *MyStruct) BadMethod() (int, string)    { return 0, "" }
func (t *MyStruct) GetCustomError() CustomError { return CustomError{} }

// CustomError for error type testing
type CustomError struct{}

func (CustomError) Error() string { return "custom" }

func TestPrepareCallErrors(t *testing.T) {
	cases := []struct {
		name           string
		recv           any
		ifaceType      reflect.Type
		method         string
		errMsg         string
		methodNotFound bool
		unexpected     bool
	}{
		{
			name:      "void method is supported",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodVoid() }](),
			method:    "PMethodVoid",
		},
		{
			name:      "correct number of args - concrete matching argument type",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(data Data) error }](),
			method:    "PMethodAcceptData",
		},
		{
			name:      "correct number of args - interface argument is any",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(data any) error }](),
			method:    "PMethodAcceptData",
		},
		{
			name:      "correct number of args - concrete mismatching  argument type",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(data NewData) error }](),
			method:    "PMethodAcceptData",
			errMsg:    "interface { PMethodAcceptData(calladapt.NewData) error }.PMethodAcceptData: param 0 mismatch: iface calladapt.NewData, concrete calladapt.Data",
		},
		{
			name:      "incorrect number of args",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData() error }](),
			method:    "PMethodAcceptData",
			errMsg:    "interface { PMethodAcceptData() error }.PMethodAcceptData: param count mismatch: iface 0, concrete 2 (incl. recv)",
		},
		{
			name:       "incorrect number of return values",
			recv:       (*MyStruct)(nil),
			ifaceType:  TypeOf[interface{ PMethodAcceptData(any) (any, error) }](),
			method:     "PMethodAcceptData",
			errMsg:     "interface { PMethodAcceptData(interface {}) (interface {}, error) }.PMethodAcceptData: return count mismatch: iface 2, concrete 1",
			unexpected: true,
		},
		{
			name:      "error return convertible to any",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(any) any }](),
			method:    "PMethodAcceptData",
		},
		{
			name:      "nil interface type",
			recv:      (*MyStruct)(nil),
			ifaceType: nil,
			method:    "Missing",
			errMsg:    "second argument must be an interface reflect.Type",
		},
		{
			name:      "untyped nil receiver",
			recv:      nil,
			ifaceType: TypeOf[interface{ PMethodAcceptData(any) (any, error) }](),
			method:    "PMethodAcceptData",
			errMsg:    "first argument must not be untyped nil",
		},
		{
			name:      "method is not on interface",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[any](),
			method:    "PMethodAcceptData",
			errMsg:    "interface {} has no method \"PMethodAcceptData\"",
		},
		{
			name:           "method is not on receiver",
			recv:           (*MyStruct)(nil),
			ifaceType:      TypeOf[interface{ Hello(any) (any, error) }](),
			method:         "Hello",
			methodNotFound: true,
		},
		{
			name:      "any instead of error allowed",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(data Data) any }](),
			method:    "PMethodAcceptData",
		},
		{
			name:       "error type mismatch",
			recv:       (*MyStruct)(nil),
			ifaceType:  TypeOf[interface{ GetCustomError() error }](),
			method:     "GetCustomError",
			errMsg:     "interface { GetCustomError() error }.GetCustomError: result 0 mismatch: iface error, concrete calladapt.CustomError",
			unexpected: true,
		},
		{
			name:      "two returns without error are supported",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ BadMethod() (int, string) }](),
			method:    "BadMethod",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := PrepareCall(tc.recv, tc.ifaceType, tc.method)
			if tc.methodNotFound {
				// Method not found on receiver now returns nil, nil
				assert.NoError(t, err)
				assert.Nil(t, c)
				return
			}
			if tc.errMsg == "" {
				assert.NoError(t, err)
				assert.NotNil(t, c)
				return
			}
			require.Error(t, err)
			var cae *CallAdaptError
			require.ErrorAs(t, err, &cae)
			assert.Equal(t, tc.errMsg, cae.Error())
		})
	}
}

func TestCall(t *testing.T) {
	my := MyStruct{State: 10}

	cases := []struct {
		name      string
		recv      any
		ifaceType reflect.Type
		method    string
		args      []any
		errMsg    string
		expect    []any
	}{
		{
			name:      "nil receiver - PMethodAcceptData ok",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(data Data) error }](),
			method:    "PMethodAcceptData",
			args:      []any{Data{}},
			expect:    []any{},
		},
		{
			name:      "error return",
			recv:      (*MyStruct)(nil),
			ifaceType: TypeOf[interface{ PMethodAcceptData(data Data) error }](),
			method:    "PMethodAcceptData",
			args:      []any{Data{1}},
			errMsg:    "X cannot be 1",
		},
		{
			name:      "value return",
			recv:      my,
			ifaceType: TypeOf[interface{ VMethodTransformNoError(any) any }](),
			method:    "VMethodTransformNoError",
			args:      []any{Data{2}},
			expect:    []any{NewData{Y: 12}},
		},
		{
			name:      "value return with ptr args",
			recv:      &my,
			ifaceType: TypeOf[interface{ PMethodTransformPtrNoError(any) any }](),
			method:    "PMethodTransformPtrNoError",
			args:      []any{&Data{2}},
			expect:    []any{&NewData{Y: 12}},
		},
		{
			name:      "any+error return",
			recv:      &my,
			ifaceType: TypeOf[interface{ PMethodTransformData(data Data) (any, error) }](),
			method:    "PMethodTransformData",
			args:      []any{Data{2}},
			expect:    []any{NewData{Y: 22}},
		},
		{
			name:      "any+error return, error case",
			recv:      &MyStruct{State: 0},
			ifaceType: TypeOf[interface{ PMethodTransformData(data Data) (any, error) }](),
			method:    "PMethodTransformData",
			args:      []any{Data{1}},
			errMsg:    "X cannot be 1",
		},
		{
			name:      "ptr any+error return",
			recv:      &MyStruct{State: 0},
			ifaceType: TypeOf[interface{ PMethodTransformDataPtr(data *Data) (any, error) }](),
			method:    "PMethodTransformDataPtr",
			args:      []any{&Data{2}},
			expect:    []any{&NewData{Y: 12}},
		},
		{
			name:      "ptr any+error return, error case (nil)",
			recv:      &MyStruct{State: 0},
			ifaceType: TypeOf[interface{ PMethodTransformDataPtr(data *Data) (any, error) }](),
			method:    "PMethodTransformDataPtr",
			args:      []any{nil},
			errMsg:    "PMethodTransformDataPtr: arg 0 type mismatch: want *calladapt.Data, got nil",
		},
		{
			name:      "void method call returns no outs",
			recv:      &my,
			ifaceType: TypeOf[interface{ PMethodVoid() }](),
			method:    "PMethodVoid",
			args:      []any{},
			expect:    []any{},
		},
		{
			name:      "too many args error",
			recv:      my,
			ifaceType: TypeOf[interface{ VMethodTransformNoError(data Data) any }](),
			method:    "VMethodTransformNoError",
			args:      []any{Data{1}, Data{2}},
			errMsg:    "VMethodTransformNoError: want 1 args, got 2",
		},
		{
			name:      "wrong arg type error (different pointer)",
			recv:      &my,
			ifaceType: TypeOf[interface{ PMethodTransformPtrNoError(data *Data) any }](),
			method:    "PMethodTransformPtrNoError",
			args:      []any{&NewData{}},
			errMsg:    "PMethodTransformPtrNoError: arg 0 type mismatch: want *calladapt.Data, got *calladapt.NewData",
		},
		{
			name:      "nil interface param allowed",
			recv:      &my,
			ifaceType: TypeOf[interface{ PMethodAcceptAny(v any) error }](),
			method:    "PMethodAcceptAny",
			args:      []any{nil},
			errMsg:    "PMethodAcceptAny: arg 0 type mismatch: want interface {}, got nil",
		},
		{
			name:      "nil slice param allowed",
			recv:      &my,
			ifaceType: TypeOf[interface{ PMethodAcceptSlice(s []int) int }](),
			method:    "PMethodAcceptSlice",
			args:      []any{nil},
			errMsg:    "PMethodAcceptSlice: arg 0 type mismatch: want []int, got nil",
		},
		{
			name: "DoCreate returns id",
			recv: &my,
			ifaceType: TypeOf[interface {
				DoCreate(ctx context.Context, data *Data) (string, error)
			}](),
			method: "DoCreate",
			args:   []any{context.Background(), &Data{2}},
			expect: []any{"123"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := PrepareCall(tc.recv, tc.ifaceType, tc.method)
			require.NoError(t, err)
			outs, callErr := c.Call(tc.args...)
			if tc.errMsg != "" {
				require.Error(t, callErr)
				assert.Equal(t, tc.errMsg, callErr.Error())
				return
			}
			require.NoError(t, callErr)
			assert.Equal(t, tc.expect, outs)
		})
	}
}
