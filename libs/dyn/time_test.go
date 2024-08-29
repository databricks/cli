package dyn_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestTimeValid(t *testing.T) {
	for _, tc := range []string{
		"2024-08-29",
		"2024-01-15T12:34:56.789012345Z",
	} {
		tm, ok := dyn.NewTime(tc)
		if assert.True(t, ok) {
			assert.NotEqual(t, time.Time{}, tm.Time())
			assert.Equal(t, tc, tm.String())
		}
	}
}

func TestTimeInvalid(t *testing.T) {
	tm, ok := dyn.NewTime("invalid")
	assert.False(t, ok)
	assert.Equal(t, dyn.Time{}, tm)
}

func TestTimeFromTime(t *testing.T) {
	tref := time.Now()
	t1 := dyn.FromTime(tref)

	// Verify that the underlying value is the same.
	assert.Equal(t, tref, t1.Time())

	// Verify that the string representation can be used to construct the same.
	t2, ok := dyn.NewTime(t1.String())
	assert.True(t, ok)
	assert.True(t, t1.Time().Equal(t2.Time()))
}
