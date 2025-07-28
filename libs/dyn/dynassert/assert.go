package dynassert

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func Equal(t assert.TestingT, expected, actual any, msgAndArgs ...any) bool {
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
