package calladapt_test

import (
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
func (*partialType) baz() {} //nolint:unused

type goodType struct{}

func (*goodType) Foo() {}
func (*goodType) Bar() {}
func (*goodType) baz() {} //nolint:unused

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
	assert.Equal(t, "unexpected method Extra on *calladapt_test.badType; only methods from [calladapt_test.testIface] are allowed", err.Error())
}

func TestEnsureNoExtraMethods_NoInterfaces(t *testing.T) {
	typedNil := (*goodType)(nil)
	err := calladapt.EnsureNoExtraMethods(typedNil)
	require.Error(t, err)
	assert.Equal(t, "unexpected method Bar on *calladapt_test.goodType; only methods from [] are allowed", err.Error())
}
