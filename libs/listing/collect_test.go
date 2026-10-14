package listing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollect(t *testing.T) {
	a, b := 1, 2
	got, err := Collect(func(yield func(*int, error) bool) {
		_ = yield(&a, nil) && yield(&b, nil)
	})
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, got)

	_, err = Collect(func(yield func(*int, error) bool) {
		_ = yield(&a, nil) && yield(nil, errors.New("boom"))
	})
	assert.EqualError(t, err, "boom")
}
