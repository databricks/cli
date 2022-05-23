package python

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWheel(t *testing.T) {
	wheel, err := BuildWheel(context.Background(), "testdata/simple-python-wheel")
	assert.NoError(t, err)
	assert.Equal(t, "testdata/simple-python-wheel/dist/dummy-0.0.1-py3-none-any.whl", wheel)

	noFile(t, "testdata/simple-python-wheel/dummy.egg-info")
	noFile(t, "testdata/simple-python-wheel/__pycache__")
	noFile(t, "testdata/simple-python-wheel/build")
}

func noFile(t *testing.T, name string) {
	_, err := os.Stat(name)
	assert.Error(t, err, "file %s should exist", name)
}