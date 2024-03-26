package dynassert

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func Equal(t assert.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool {
	ev, eok := expected.(dyn.Value)
	av, aok := actual.(dyn.Value)
	if eok && aok && ev.IsValid() && av.IsValid() {
		if !assert.Equal(t, ev.AsAny(), av.AsAny(), msgAndArgs...) {
			return false
		}

		// The values are equal on contents. Now compare the locations.
		if !assert.Equal(t, ev.Location(), av.Location(), msgAndArgs...) {
			return false
		}

		// Walk ev and av and compare the locations of each element.
		_, err := dyn.Walk(ev, func(p dyn.Path, evv dyn.Value) (dyn.Value, error) {
			avv, err := dyn.GetByPath(av, p)
			if assert.NoError(t, err, "unable to get value from actual value at path %v", p.String()) {
				assert.Equal(t, evv.Location(), avv.Location())
			}
			return evv, nil
		})
		return assert.NoError(t, err)
	}

	return assert.Equal(t, expected, actual, msgAndArgs...)
}

func EqualValues(t assert.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	return assert.EqualValues(t, expected, actual, msgAndArgs...)
}

func NotEqual(t assert.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool {
	return assert.NotEqual(t, expected, actual, msgAndArgs...)
}

func Len(t assert.TestingT, object interface{}, length int, msgAndArgs ...interface{}) bool {
	return assert.Len(t, object, length, msgAndArgs...)
}

func Empty(t assert.TestingT, object interface{}, msgAndArgs ...interface{}) bool {
	return assert.Empty(t, object, msgAndArgs...)
}

func Nil(t assert.TestingT, object interface{}, msgAndArgs ...interface{}) bool {
	return assert.Nil(t, object, msgAndArgs...)
}

func NotNil(t assert.TestingT, object interface{}, msgAndArgs ...interface{}) bool {
	return assert.NotNil(t, object, msgAndArgs...)
}

func NoError(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
	return assert.NoError(t, err, msgAndArgs...)
}

func Error(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
	return assert.Error(t, err, msgAndArgs...)
}

func EqualError(t assert.TestingT, theError error, errString string, msgAndArgs ...interface{}) bool {
	return assert.EqualError(t, theError, errString, msgAndArgs...)
}

func ErrorContains(t assert.TestingT, theError error, contains string, msgAndArgs ...interface{}) bool {
	return assert.ErrorContains(t, theError, contains, msgAndArgs...)
}

func ErrorIs(t assert.TestingT, theError, target error, msgAndArgs ...interface{}) bool {
	return assert.ErrorIs(t, theError, target, msgAndArgs...)
}

func True(t assert.TestingT, value bool, msgAndArgs ...interface{}) bool {
	return assert.True(t, value, msgAndArgs...)
}

func False(t assert.TestingT, value bool, msgAndArgs ...interface{}) bool {
	return assert.False(t, value, msgAndArgs...)
}

func Contains(t assert.TestingT, list interface{}, element interface{}, msgAndArgs ...interface{}) bool {
	return assert.Contains(t, list, element, msgAndArgs...)
}

func NotContains(t assert.TestingT, list interface{}, element interface{}, msgAndArgs ...interface{}) bool {
	return assert.NotContains(t, list, element, msgAndArgs...)
}

func ElementsMatch(t assert.TestingT, listA, listB interface{}, msgAndArgs ...interface{}) bool {
	return assert.ElementsMatch(t, listA, listB, msgAndArgs...)
}

func Panics(t assert.TestingT, f func(), msgAndArgs ...interface{}) bool {
	return assert.Panics(t, f, msgAndArgs...)
}

func PanicsWithValue(t assert.TestingT, expected interface{}, f func(), msgAndArgs ...interface{}) bool {
	return assert.PanicsWithValue(t, expected, f, msgAndArgs...)
}

func PanicsWithError(t assert.TestingT, errString string, f func(), msgAndArgs ...interface{}) bool {
	return assert.PanicsWithError(t, errString, f, msgAndArgs...)
}

func NotPanics(t assert.TestingT, f func(), msgAndArgs ...interface{}) bool {
	return assert.NotPanics(t, f, msgAndArgs...)
}
