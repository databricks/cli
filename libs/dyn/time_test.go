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
		tm, err := dyn.NewTime(tc)
		if assert.NoError(t, err) {
			assert.NotEqual(t, time.Time{}, tm.Time())
			assert.Equal(t, tc, tm.String())
		}
	}
}

func TestTimeInvalid(t *testing.T) {
	tm, err := dyn.NewTime("invalid")
	assert.Error(t, err)
	assert.Equal(t, dyn.Time{}, tm)
}

func TestTimeFromTime(t *testing.T) {
	tref := time.Now()
	t1 := dyn.FromTime(tref)

	// Verify that the underlying value is the same.
	assert.Equal(t, tref, t1.Time())

	// Verify that the string representation can be used to construct the same.
	t2, err := dyn.NewTime(t1.String())
	assert.NoError(t, err)
	assert.True(t, t1.Time().Equal(t2.Time()))
}
