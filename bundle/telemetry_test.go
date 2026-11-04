package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolPairSet(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []bool
		want   BoolPair
	}{
		{"not measured", nil, BoolPair{}},
		{"true", []bool{true}, BoolPair{True: true}},
		{"false", []bool{false}, BoolPair{False: true}},
		{"repeated", []bool{true, true}, BoolPair{True: true}},
		{"mixed", []bool{true, false}, BoolPair{True: true, False: true}},
		{"mixed reversed", []bool{false, true}, BoolPair{True: true, False: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p BoolPair
			for _, v := range tc.values {
				p.Set(v)
			}
			assert.Equal(t, tc.want, p)
		})
	}
}
