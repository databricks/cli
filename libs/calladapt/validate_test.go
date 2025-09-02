package calladapt_test

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/calladapt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testIface interface {
	Foo()
	Bar()
}

type partialType struct{}

func (*partialType) Foo() {}
func (*partialType) baz() {}

type goodType struct{}

func (*goodType) Foo() {}
func (*goodType) Bar() {}
func (*goodType) baz() {}

type badType struct{}

func (*badType) Foo()   {}
func (*badType) Bar()   {}
func (*badType) Extra() {}

func TestEnsureNoExtraMethods_AllowsPartial(t *testing.T) {
	typedNil := (*partialType)(nil)
	err := calladapt.EnsureNoExtraMethods(typedNil, calladapt.TypeOf[testIface]())
	require.NoError(t, err)
}

func TestEnsureNoExtraMethods_AllowsGood(t *testing.T) {
	typedNil := (*goodType)(nil)
	err := calladapt.EnsureNoExtraMethods(typedNil, calladapt.TypeOf[testIface]())
	require.NoError(t, err)
}

func TestEnsureNoExtraMethods_RejectsExtra(t *testing.T) {
	typedNil := (*badType)(nil)
	err := calladapt.EnsureNoExtraMethods(typedNil, calladapt.TypeOf[testIface]())
	require.Error(t, err)
	assert.Equal(t, "unexpected exported method Extra on *calladapt_test.badType; only methods from calladapt_test.testIface are allowed", err.Error())
}

func TestEnsureNoExtraMethods_InvalidInterfaceArg(t *testing.T) {
	typedNil := (*goodType)(nil)
	err := calladapt.EnsureNoExtraMethods(typedNil, reflect.TypeOf(0))
	require.Error(t, err)
}
